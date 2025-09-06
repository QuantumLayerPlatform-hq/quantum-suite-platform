package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quantum-suite/platform/pkg/shared/env"
	"github.com/quantum-suite/platform/pkg/shared/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-gateway",
		Port:        "8080",
		Logging: env.LoggingConfig{
			Level:      "debug",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	assert.NotNil(t, service)
	
	err = service.Close()
	assert.NoError(t, err)
}

func TestServiceHealthCheck(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-gateway",
		Port:        "8080",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{},
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

	var healthResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&healthResp)
	require.NoError(t, err)

	assert.Contains(t, healthResp, "status")
	assert.Contains(t, healthResp, "timestamp")
	assert.Contains(t, healthResp, "service")
}

func TestServiceModelsEndpoint(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-gateway",
		Port:        "8080",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/v1/models")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var modelsResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&modelsResp)
	require.NoError(t, err)

	assert.Equal(t, "list", modelsResp["object"])
	assert.Contains(t, modelsResp, "data")
}

func TestServiceAuthenticationRequired(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-gateway",
		Port:        "8080",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	// Test completion endpoint without authentication
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

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestServiceTenantHeaderRequired(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-gateway",
		Port:        "8080",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	// Test completion endpoint without tenant ID
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

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestServiceInvalidJSON(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-gateway",
		Port:        "8080",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	// Test with invalid JSON
	req, err := http.NewRequest("POST", server.URL+"/v1/completions", bytes.NewBufferString("invalid json"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Tenant-ID", "test-tenant")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestServiceCORS(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-gateway",
		Port:        "8080",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	// Test CORS preflight request
	req, err := http.NewRequest("OPTIONS", server.URL+"/v1/completions", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "authorization")
}

func TestServiceMetrics(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-gateway",
		Port:        "8080",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	// Test metrics endpoint
	resp, err := http.Get(server.URL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
}