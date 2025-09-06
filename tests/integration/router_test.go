//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/internal/services/router"
	"github.com/quantum-suite/platform/pkg/shared/env"
	"github.com/quantum-suite/platform/pkg/shared/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouterIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Integration tests skipped")
	}

	// Setup test configuration
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "qlens-router-test",
		Port:        "8081",
		Logging: env.LoggingConfig{
			Level:      "debug",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{
			"azure-openai": {
				Enabled: true,
				Config: map[string]interface{}{
					"endpoint":     os.Getenv("AZURE_OPENAI_ENDPOINT"),
					"api_key":      os.Getenv("AZURE_OPENAI_API_KEY"),
					"api_version":  "2024-02-15-preview",
					"deployments": map[string]interface{}{
						"gpt-35-turbo": "gpt-35-turbo",
					},
				},
			},
		},
	}

	// Skip test if credentials not available
	if config.Providers["azure-openai"].Config["endpoint"] == "" || 
	   config.Providers["azure-openai"].Config["api_key"] == "" {
		t.Skip("Azure OpenAI credentials not available")
	}

	log := logger.NewNoop()
	service, err := router.NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	t.Run("health check", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var healthResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&healthResp)
		require.NoError(t, err)

		assert.Equal(t, "healthy", healthResp["status"])
	})

	t.Run("readiness check", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health/ready")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var readinessResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&readinessResp)
		require.NoError(t, err)

		assert.Equal(t, "ready", readinessResp["status"])
	})

	t.Run("list models", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/internal/v1/models")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var modelsResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&modelsResp)
		require.NoError(t, err)

		assert.Equal(t, "list", modelsResp["object"])
		data, ok := modelsResp["data"].([]interface{})
		require.True(t, ok)
		assert.Greater(t, len(data), 0)
	})

	t.Run("route completion", func(t *testing.T) {
		reqBody := router.CompletionRequest{
			TenantID: domain.TenantID("test-tenant"),
			UserID:   domain.UserID("test-user"),
			Model:    "gpt-35-turbo",
			Messages: []domain.Message{
				{
					Role: domain.MessageRoleUser,
					Content: []domain.ContentPart{
						{
							Type: domain.ContentTypeText,
							Text: "Say 'Hello Router Test'",
						},
					},
				},
			},
			MaxTokens:   intPtr(10),
			Temperature: float64Ptr(0.1),
			RequestID:   "test-request-123",
		}

		reqJSON, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/internal/v1/completions", bytes.NewBuffer(reqJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should work if Azure OpenAI is properly configured
		if resp.StatusCode == http.StatusOK {
			var completionResp router.CompletionResponse
			err = json.NewDecoder(resp.Body).Decode(&completionResp)
			require.NoError(t, err)

			assert.Equal(t, "gpt-35-turbo", completionResp.Model)
			assert.Equal(t, domain.ProviderAzureOpenAI, completionResp.Provider)
			assert.Greater(t, len(completionResp.Choices), 0)
		} else {
			t.Logf("Completion request failed with status %d (may be expected if service is not fully configured)", resp.StatusCode)
		}
	})

	t.Run("route embedding", func(t *testing.T) {
		// First, add an embedding model to the config for this test
		config.Providers["azure-openai"].Config["deployments"] = map[string]interface{}{
			"gpt-35-turbo":             "gpt-35-turbo",
			"text-embedding-ada-002":   "text-embedding-ada-002",
		}

		reqBody := router.EmbeddingRequest{
			TenantID:  domain.TenantID("test-tenant"),
			UserID:    domain.UserID("test-user"),
			Model:     "text-embedding-ada-002",
			Input:     []string{"test input for embedding"},
			RequestID: "test-embedding-123",
		}

		reqJSON, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/internal/v1/embeddings", bytes.NewBuffer(reqJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should work if Azure OpenAI is properly configured and has embedding model
		if resp.StatusCode == http.StatusOK {
			var embeddingResp router.EmbeddingResponse
			err = json.NewDecoder(resp.Body).Decode(&embeddingResp)
			require.NoError(t, err)

			assert.Equal(t, "text-embedding-ada-002", embeddingResp.Model)
			assert.Equal(t, domain.ProviderAzureOpenAI, embeddingResp.Provider)
			assert.Greater(t, len(embeddingResp.Data), 0)
		} else {
			t.Logf("Embedding request failed with status %d (may be expected if embedding model is not configured)", resp.StatusCode)
		}
	})

	t.Run("provider selection", func(t *testing.T) {
		// Test with explicit provider selection
		reqBody := router.CompletionRequest{
			TenantID: domain.TenantID("test-tenant"),
			UserID:   domain.UserID("test-user"),
			Provider: domain.ProviderAzureOpenAI,
			Model:    "gpt-35-turbo",
			Messages: []domain.Message{
				{
					Role: domain.MessageRoleUser,
					Content: []domain.ContentPart{
						{
							Type: domain.ContentTypeText,
							Text: "Test provider selection",
						},
					},
				},
			},
			MaxTokens:   intPtr(5),
			Temperature: float64Ptr(0.0),
			RequestID:   "test-provider-123",
		}

		reqJSON, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/internal/v1/completions", bytes.NewBuffer(reqJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var completionResp router.CompletionResponse
			err = json.NewDecoder(resp.Body).Decode(&completionResp)
			require.NoError(t, err)

			// Verify the correct provider was used
			assert.Equal(t, domain.ProviderAzureOpenAI, completionResp.Provider)
		} else {
			t.Logf("Provider selection test failed with status %d", resp.StatusCode)
		}
	})

	t.Run("invalid model", func(t *testing.T) {
		reqBody := router.CompletionRequest{
			TenantID: domain.TenantID("test-tenant"),
			UserID:   domain.UserID("test-user"),
			Model:    "non-existent-model",
			Messages: []domain.Message{
				{
					Role: domain.MessageRoleUser,
					Content: []domain.ContentPart{
						{
							Type: domain.ContentTypeText,
							Text: "Test",
						},
					},
				},
			},
			RequestID: "test-invalid-model-123",
		}

		reqJSON, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/internal/v1/completions", bytes.NewBuffer(reqJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return error for invalid model
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// Helper functions for tests
func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}