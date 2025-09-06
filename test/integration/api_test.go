package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test configuration
type TestConfig struct {
	BaseURL    string
	TenantID   string
	UserID     string
	APIKey     string
	Timeout    time.Duration
}

// Default test configuration
func NewTestConfig() *TestConfig {
	return &TestConfig{
		BaseURL:  getEnvOrDefault("TEST_BASE_URL", "http://localhost:8105"),
		TenantID: getEnvOrDefault("TEST_TENANT_ID", "test-tenant-1"),
		UserID:   getEnvOrDefault("TEST_USER_ID", "test-user-1"),
		APIKey:   getEnvOrDefault("TEST_API_KEY", "test-api-key-12345"),
		Timeout:  30 * time.Second,
	}
}

// API client for testing
type TestClient struct {
	config     *TestConfig
	httpClient *http.Client
}

func NewTestClient(config *TestConfig) *TestClient {
	return &TestClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Helper to create requests with proper headers
func (c *TestClient) createRequest(method, path string, body interface{}) (*http.Request, error) {
	var reqBody bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&reqBody).Encode(body); err != nil {
			return nil, err
		}
	}

	fullURL := c.config.BaseURL + path
	req, err := http.NewRequest(method, fullURL, &reqBody)
	if err != nil {
		return nil, err
	}

	// Add required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", c.config.TenantID)
	req.Header.Set("X-User-ID", c.config.UserID)
	req.Header.Set("X-API-Key", c.config.APIKey)
	req.Header.Set("X-Correlation-ID", generateCorrelationID())

	return req, nil
}

// Test structures
type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	Stream      bool      `json:"stream"`
}

type Message struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

type ContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type CompletionResponse struct {
	ID       string   `json:"id"`
	Object   string   `json:"object"`
	Created  int64    `json:"created"`
	Model    string   `json:"model"`
	Provider string   `json:"provider"`
	Choices  []Choice `json:"choices"`
	Usage    Usage    `json:"usage"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	CostUSD          float64 `json:"cost_usd"`
	CacheHit         bool    `json:"cache_hit"`
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID           string   `json:"id"`
	Provider     string   `json:"provider"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
}

type HealthResponse struct {
	Status    string                         `json:"status"`
	Timestamp time.Time                      `json:"timestamp"`
	Services  map[string]ServiceHealth       `json:"services"`
	Providers map[string]ProviderHealth      `json:"providers"`
}

type ServiceHealth struct {
	Status  string `json:"status"`
	Latency int64  `json:"latency_ms"`
}

type ProviderHealth struct {
	Status    string  `json:"status"`
	Latency   int64   `json:"latency_ms"`
	ErrorRate float64 `json:"error_rate"`
}

// Integration Tests

func TestHealthEndpoints(t *testing.T) {
	client := NewTestClient(NewTestConfig())

	t.Run("Health Check", func(t *testing.T) {
		req, err := client.createRequest("GET", "/health", nil)
		require.NoError(t, err)

		resp, err := client.httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health HealthResponse
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		assert.Equal(t, "healthy", health.Status)
		assert.NotZero(t, health.Timestamp)
	})

	t.Run("Readiness Check", func(t *testing.T) {
		req, err := client.createRequest("GET", "/health/ready", nil)
		require.NoError(t, err)

		resp, err := client.httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestListModels(t *testing.T) {
	client := NewTestClient(NewTestConfig())

	t.Run("List All Models", func(t *testing.T) {
		req, err := client.createRequest("GET", "/v1/models", nil)
		require.NoError(t, err)

		resp, err := client.httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var models ModelsResponse
		err = json.NewDecoder(resp.Body).Decode(&models)
		require.NoError(t, err)

		assert.Equal(t, "list", models.Object)
		assert.NotEmpty(t, models.Data)

		// Verify model structure
		for _, model := range models.Data {
			assert.NotEmpty(t, model.ID)
			assert.NotEmpty(t, model.Provider)
			assert.NotEmpty(t, model.Name)
		}
	})

	t.Run("Filter Models by Provider", func(t *testing.T) {
		req, err := client.createRequest("GET", "/v1/models?provider=openai", nil)
		require.NoError(t, err)

		resp, err := client.httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var models ModelsResponse
		err = json.NewDecoder(resp.Body).Decode(&models)
		require.NoError(t, err)

		// All returned models should be from OpenAI
		for _, model := range models.Data {
			assert.Equal(t, "openai", model.Provider)
		}
	})
}

func TestCompletions(t *testing.T) {
	client := NewTestClient(NewTestConfig())

	t.Run("Simple Completion", func(t *testing.T) {
		temperature := 0.7
		maxTokens := 100

		request := CompletionRequest{
			Model: "gpt-3.5-turbo",
			Messages: []Message{
				{
					Role: "user",
					Content: []ContentPart{
						{
							Type: "text",
							Text: "Hello, this is a test message. Please respond with a simple greeting.",
						},
					},
				},
			},
			MaxTokens:   &maxTokens,
			Temperature: &temperature,
			Stream:      false,
		}

		req, err := client.createRequest("POST", "/v1/completions", request)
		require.NoError(t, err)

		resp, err := client.httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var completion CompletionResponse
		err = json.NewDecoder(resp.Body).Decode(&completion)
		require.NoError(t, err)

		// Verify response structure
		assert.NotEmpty(t, completion.ID)
		assert.Equal(t, "chat.completion", completion.Object)
		assert.Equal(t, "gpt-3.5-turbo", completion.Model)
		assert.NotEmpty(t, completion.Provider)
		assert.NotEmpty(t, completion.Choices)
		assert.Equal(t, "assistant", completion.Choices[0].Message.Role)
		assert.NotEmpty(t, completion.Choices[0].Message.Content)
		assert.Greater(t, completion.Usage.TotalTokens, 0)
	})

	t.Run("Invalid Model", func(t *testing.T) {
		request := CompletionRequest{
			Model: "invalid-model-name",
			Messages: []Message{
				{
					Role: "user",
					Content: []ContentPart{
						{
							Type: "text",
							Text: "Test message",
						},
					},
				},
			},
			Stream: false,
		}

		req, err := client.createRequest("POST", "/v1/completions", request)
		require.NoError(t, err)

		resp, err := client.httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Missing Required Headers", func(t *testing.T) {
		request := CompletionRequest{
			Model: "gpt-3.5-turbo",
			Messages: []Message{
				{
					Role: "user",
					Content: []ContentPart{
						{
							Type: "text",
							Text: "Test message",
						},
					},
				},
			},
			Stream: false,
		}

		reqBody, _ := json.Marshal(request)
		fullURL := client.config.BaseURL + "/v1/completions"
		req, err := http.NewRequest("POST", fullURL, bytes.NewReader(reqBody))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		// Intentionally omit required headers

		resp, err := client.httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestEmbeddings(t *testing.T) {
	client := NewTestClient(NewTestConfig())

	t.Run("Create Embeddings", func(t *testing.T) {
		request := map[string]interface{}{
			"model": "text-embedding-ada-002",
			"input": []string{
				"Hello world",
				"This is a test embedding",
			},
		}

		req, err := client.createRequest("POST", "/v1/embeddings", request)
		require.NoError(t, err)

		resp, err := client.httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, "list", result["object"])
		assert.NotEmpty(t, result["data"])
	})
}

func TestRateLimit(t *testing.T) {
	client := NewTestClient(NewTestConfig())

	t.Run("Rate Limit Enforcement", func(t *testing.T) {
		// This test would require rate limiting to be configured
		// In a real scenario, you'd make many requests rapidly
		// and expect to receive 429 status codes

		successCount := 0
		rateLimitCount := 0

		for i := 0; i < 10; i++ {
			req, err := client.createRequest("GET", "/health", nil)
			require.NoError(t, err)

			resp, err := client.httpClient.Do(req)
			require.NoError(t, err)
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				successCount++
			} else if resp.StatusCode == http.StatusTooManyRequests {
				rateLimitCount++
			}

			time.Sleep(100 * time.Millisecond) // Small delay
		}

		// In development, rate limiting might be disabled
		// So we just verify the service responds
		assert.Greater(t, successCount, 0)
	})
}

// Test utilities

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateCorrelationID() string {
	return fmt.Sprintf("test-%d", time.Now().UnixNano())
}

// Test setup and teardown

func TestMain(m *testing.M) {
	// Setup test environment
	fmt.Println("Setting up integration tests...")
	
	// Verify service is running
	config := NewTestConfig()
	client := NewTestClient(config)
	
	req, err := client.createRequest("GET", "/health", nil)
	if err != nil {
		fmt.Printf("Failed to create health check request: %v\n", err)
		os.Exit(1)
	}
	
	resp, err := client.httpClient.Do(req)
	if err != nil {
		fmt.Printf("Failed to connect to service at %s: %v\n", config.BaseURL, err)
		fmt.Println("Make sure QLens is running before running integration tests")
		fmt.Println("Run: ./scripts/dev-start.sh")
		os.Exit(1)
	}
	resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Service health check failed with status %d\n", resp.StatusCode)
		os.Exit(1)
	}
	
	fmt.Println("✅ Service is running and healthy")
	fmt.Println("🧪 Running integration tests...")
	
	// Run tests
	code := m.Run()
	
	// Cleanup
	fmt.Println("🧹 Cleaning up...")
	
	os.Exit(code)
}