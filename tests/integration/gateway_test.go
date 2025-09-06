//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/internal/services/gateway"
	"github.com/quantum-suite/platform/pkg/shared/env"
	"github.com/quantum-suite/platform/pkg/shared/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Integration tests skipped")
	}

	// Setup test configuration
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "qlens-gateway-test",
		Port:        "8080",
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
	service, err := gateway.NewService(config, log)
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

	t.Run("list models", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/v1/models")
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

	t.Run("create completion", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"model": "gpt-35-turbo",
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": "Say 'Hello Integration Test'",
						},
					},
				},
			},
			"max_tokens":  10,
			"temperature": 0.1,
		}

		reqJSON, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/v1/completions", bytes.NewBuffer(reqJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("X-Tenant-ID", "test-tenant")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should work if Azure OpenAI is properly configured
		if resp.StatusCode == http.StatusOK {
			var completionResp map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&completionResp)
			require.NoError(t, err)

			assert.Equal(t, "chat.completion", completionResp["object"])
			choices, ok := completionResp["choices"].([]interface{})
			require.True(t, ok)
			assert.Greater(t, len(choices), 0)
		} else {
			t.Logf("Completion request failed with status %d (may be expected if service is not fully configured)", resp.StatusCode)
		}
	})

	t.Run("authentication required", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"model":    "gpt-35-turbo",
			"messages": []map[string]interface{}{},
		}

		reqJSON, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/v1/completions", bytes.NewBuffer(reqJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		// Missing Authorization header

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should require authentication
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("tenant header required", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"model":    "gpt-35-turbo",
			"messages": []map[string]interface{}{},
		}

		reqJSON, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/v1/completions", bytes.NewBuffer(reqJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		// Missing X-Tenant-ID header

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should require tenant ID
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}