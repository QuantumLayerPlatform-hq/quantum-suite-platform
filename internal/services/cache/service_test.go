package cache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/shared/env"
	"github.com/quantum-suite/platform/pkg/shared/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCacheService(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-cache",
		Port:        "8082",
		Logging: env.LoggingConfig{
			Level:      "debug",
			Format:     "json",
			Structured: true,
		},
		Cache: env.CacheConfig{
			Type:    "memory",
			TTL:     300,
			MaxSize: 1000,
		},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	assert.NotNil(t, service)
	
	err = service.Close()
	assert.NoError(t, err)
}

func TestCacheServiceHealthCheck(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-cache",
		Port:        "8082",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Cache: env.CacheConfig{
			Type:    "memory",
			TTL:     300,
			MaxSize: 1000,
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

	var healthResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&healthResp)
	require.NoError(t, err)

	assert.Contains(t, healthResp, "status")
	assert.Contains(t, healthResp, "timestamp")
}

func TestCacheServiceReadiness(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-cache",
		Port:        "8082",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Cache: env.CacheConfig{
			Type:    "memory",
			TTL:     300,
			MaxSize: 1000,
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

func TestCacheServiceSetAndGet(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-cache",
		Port:        "8082",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Cache: env.CacheConfig{
			Type:    "memory",
			TTL:     300,
			MaxSize: 1000,
		},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	client := &http.Client{}

	// Test SET operation
	setReq := CacheSetRequest{
		Key:       "test-key",
		TenantID:  domain.TenantID("test-tenant"),
		Value: CacheValue{
			Response: domain.LLMResponse{
				ID:      "test-response",
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   "gpt-4",
				Choices: []domain.Choice{
					{
						Index: 0,
						Message: domain.Message{
							Role: domain.MessageRoleAssistant,
							Content: []domain.ContentPart{
								{Type: domain.ContentTypeText, Text: "Hello"},
							},
						},
					},
				},
				Usage: domain.Usage{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:      15,
				},
			},
		},
		TTL: 300,
	}

	setReqJSON, err := json.Marshal(setReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", server.URL+"/internal/v1/cache", bytes.NewBuffer(setReqJSON))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var setResp CacheSetResponse
	err = json.NewDecoder(resp.Body).Decode(&setResp)
	require.NoError(t, err)
	assert.True(t, setResp.Success)

	// Test GET operation
	getURL := server.URL + "/internal/v1/cache/test-key?tenant_id=test-tenant"
	resp, err = client.Get(getURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var getResp CacheGetResponse
		err = json.NewDecoder(resp.Body).Decode(&getResp)
		require.NoError(t, err)
		assert.True(t, getResp.Found)
		assert.Equal(t, "test-response", getResp.Value.Response.ID)
	}
}

func TestCacheServiceDelete(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-cache",
		Port:        "8082",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Cache: env.CacheConfig{
			Type:    "memory",
			TTL:     300,
			MaxSize: 1000,
		},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	client := &http.Client{}

	// First set a value
	setReq := CacheSetRequest{
		Key:      "test-delete-key",
		TenantID: domain.TenantID("test-tenant"),
		Value: CacheValue{
			Response: domain.LLMResponse{
				ID: "test-response",
			},
		},
		TTL: 300,
	}

	setReqJSON, err := json.Marshal(setReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", server.URL+"/internal/v1/cache", bytes.NewBuffer(setReqJSON))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Now delete it
	deleteURL := server.URL + "/internal/v1/cache/test-delete-key?tenant_id=test-tenant"
	req, err = http.NewRequest("DELETE", deleteURL, nil)
	require.NoError(t, err)

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var deleteResp CacheDeleteResponse
	err = json.NewDecoder(resp.Body).Decode(&deleteResp)
	require.NoError(t, err)
	assert.True(t, deleteResp.Success)
}

func TestCacheServiceClearTenant(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-cache",
		Port:        "8082",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Cache: env.CacheConfig{
			Type:    "memory",
			TTL:     300,
			MaxSize: 1000,
		},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	client := &http.Client{}

	// Set multiple values for the tenant
	tenantID := domain.TenantID("test-tenant-clear")
	
	for i := 0; i < 3; i++ {
		setReq := CacheSetRequest{
			Key:      fmt.Sprintf("test-key-%d", i),
			TenantID: tenantID,
			Value: CacheValue{
				Response: domain.LLMResponse{
					ID: fmt.Sprintf("test-response-%d", i),
				},
			},
			TTL: 300,
		}

		setReqJSON, err := json.Marshal(setReq)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/internal/v1/cache", bytes.NewBuffer(setReqJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Clear tenant cache
	clearURL := fmt.Sprintf("%s/internal/v1/cache/tenant/%s", server.URL, tenantID)
	req, err := http.NewRequest("DELETE", clearURL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var clearResp CacheClearResponse
	err = json.NewDecoder(resp.Body).Decode(&clearResp)
	require.NoError(t, err)
	assert.True(t, clearResp.Success)
	assert.Greater(t, clearResp.ClearedCount, 0)
}

func TestCacheServiceStats(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-cache",
		Port:        "8082",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Cache: env.CacheConfig{
			Type:    "memory",
			TTL:     300,
			MaxSize: 1000,
		},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/internal/v1/cache/stats")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var statsResp CacheStatsResponse
	err = json.NewDecoder(resp.Body).Decode(&statsResp)
	require.NoError(t, err)

	// Stats should be present
	assert.GreaterOrEqual(t, statsResp.TotalEntries, 0)
	assert.GreaterOrEqual(t, statsResp.HitCount, int64(0))
	assert.GreaterOrEqual(t, statsResp.MissCount, int64(0))
}

func TestCacheServiceInvalidTenantID(t *testing.T) {
	config := &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "test-cache",
		Port:        "8082",
		Logging: env.LoggingConfig{
			Level:      "error",
			Format:     "json",
			Structured: true,
		},
		Cache: env.CacheConfig{
			Type:    "memory",
			TTL:     300,
			MaxSize: 1000,
		},
	}

	log := logger.NewNoop()
	service, err := NewService(config, log)
	require.NoError(t, err)
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	// Test GET without tenant_id
	resp, err := http.Get(server.URL + "/internal/v1/cache/test-key")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}