package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/shared/env"
	"github.com/quantum-suite/platform/pkg/shared/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRouterService(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-router",
		Port:        "8081",
		Logging: env.LoggingConfig{
			Level:      "debug",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{
			"openai": {
				Enabled: true,
				Config: map[string]interface{}{
					"api_key": "test-key",
				},
			},
		},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	assert.NotNil(t, service)
	
	err = service.Close()
	assert.NoError(t, err)
}

func TestRouterServiceHealthCheck(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-router",
		Port:        "8081",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{
			"openai": {
				Enabled: true,
				Config: map[string]interface{}{
					"api_key": "test-key",
				},
			},
		},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var healthResp HealthResponse
	err = json.NewDecoder(resp.Body).Decode(&healthResp)
	require.NoError(t, err)

	assert.NotEmpty(t, healthResp.Status)
	assert.NotZero(t, healthResp.Timestamp)
}

func TestRouterServiceReadiness(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-router",
		Port:        "8081",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{
			"openai": {
				Enabled: true,
				Config: map[string]interface{}{
					"api_key": "test-key",
				},
			},
		},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/health/ready")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var readinessResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&readinessResp)
	require.NoError(t, err)

	assert.Equal(t, "ready", readinessResp["status"])
}

func TestRouterServiceListModels(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-router",
		Port:        "8081",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{
			"openai": {
				Enabled: true,
				Config: map[string]interface{}{
					"api_key": "test-key",
				},
			},
		},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/internal/v1/models")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var modelsResp ModelsResponse
	err = json.NewDecoder(resp.Body).Decode(&modelsResp)
	require.NoError(t, err)

	assert.Equal(t, "list", modelsResp.Object)
	assert.NotNil(t, modelsResp.Data)
}

func TestRouterServiceCompletionRequest(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-router",
		Port:        "8081",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{
			"openai": {
				Enabled: true,
				Config: map[string]interface{}{
					"api_key": "test-key",
				},
			},
		},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	reqBody := CompletionRequest{
		TenantID: domain.TenantID("test-tenant"),
		UserID:   domain.UserID("test-user"),
		Model:    "gpt-35-turbo",
		Messages: []domain.Message{
			{
				Role: domain.MessageRoleUser,
				Content: []domain.ContentPart{
					{
						Type: domain.ContentTypeText,
						Text: "Hello",
					},
				},
			},
		},
		MaxTokens: intPtr(5),
		RequestID: "test-123",
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

	// With mock provider, this should still process the request structure
	// The actual response will depend on the mock implementation
	assert.True(t, resp.StatusCode >= 200 && resp.StatusCode < 500)
}

func TestRouterServiceProviderSelection(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-router",
		Port:        "8081",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{
			"openai": {
				Enabled: true,
				Config: map[string]interface{}{
					"api_key": "test-key",
				},
			},
		},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	// Test selectProvider method
	provider, err := service.selectProvider("gpt-4", domain.ProviderOpenAI)
	if err == nil {
		assert.Equal(t, domain.ProviderOpenAI, provider)
	} else {
		// Expected if no mock model registry is set up
		assert.Contains(t, err.Error(), "model")
	}
}

func TestRouterServiceGenerateCacheKey(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-router",
		Port:        "8081",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{
			"openai": {
				Enabled: true,
				Config: map[string]interface{}{
					"api_key": "test-key",
				},
			},
		},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	tenantID := domain.TenantID("test-tenant")
	req := &CompletionRequest{
		Model: "gpt-4",
		Messages: []domain.Message{
			{
				Role: domain.MessageRoleUser,
				Content: []domain.ContentPart{
					{
						Type: domain.ContentTypeText,
						Text: "Hello",
					},
				},
			},
		},
		Temperature: floatPtr(0.7),
		MaxTokens:   intPtr(100),
		User:        "test-user",
	}

	key1 := service.generateCacheKey(tenantID, req)
	assert.NotEmpty(t, key1)
	assert.Len(t, key1, 64) // SHA256 hex string

	// Same request should generate same key
	key2 := service.generateCacheKey(tenantID, req)
	assert.Equal(t, key1, key2)

	// Different tenant should generate different key
	key3 := service.generateCacheKey(domain.TenantID("different-tenant"), req)
	assert.NotEqual(t, key1, key3)

	// Different request should generate different key
	req.MaxTokens = intPtr(200)
	key4 := service.generateCacheKey(tenantID, req)
	assert.NotEqual(t, key1, key4)
}

func TestRouterServiceNoProvidersEnabled(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-router",
		Port:        "8081",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{
			"openai": {
				Enabled: false,
				Config: map[string]interface{}{
					"api_key": "test-key",
				},
			},
		},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no providers enabled")
	assert.Nil(t, service)
}

func TestGetConfigHelpers(t *testing.T) {
	config := map[string]interface{}{
		"string_key": "test-value",
		"map_key": map[string]interface{}{
			"nested": "value",
		},
	}

	// Test getStringFromConfig
	stringVal := getStringFromConfig(config, "string_key")
	assert.Equal(t, "test-value", stringVal)

	missingStringVal := getStringFromConfig(config, "missing_key")
	assert.Empty(t, missingStringVal)

	// Test getMapFromConfig
	mapVal := getMapFromConfig(config, "map_key")
	assert.Equal(t, map[string]string{"nested": "value"}, mapVal)

	missingMapVal := getMapFromConfig(config, "missing_key")
	assert.Empty(t, missingMapVal)

	// Test getStringFromMap
	mapConfig := map[string]interface{}{
		"key": "value",
	}
	stringFromMap := getStringFromMap(mapConfig, "key")
	assert.Equal(t, "value", stringFromMap)

	missingFromMap := getStringFromMap(mapConfig, "missing")
	assert.Empty(t, missingFromMap)
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func floatPtr(f float64) *float64 {
	return &f
}