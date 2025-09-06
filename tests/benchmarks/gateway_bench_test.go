package benchmarks

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quantum-suite/platform/internal/services/gateway"
	"github.com/quantum-suite/platform/pkg/shared/env"
	"github.com/quantum-suite/platform/pkg/shared/logger"
)

func BenchmarkGatewayHealthCheck(b *testing.B) {
	config := createTestConfig()
	log := logger.NewNoop()
	service, err := gateway.NewService(config, log)
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	client := &http.Client{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(server.URL + "/health")
			if err != nil {
				b.Errorf("Health check failed: %v", err)
				continue
			}
			resp.Body.Close()
			
			if resp.StatusCode != http.StatusOK {
				b.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

func BenchmarkGatewayListModels(b *testing.B) {
	config := createTestConfig()
	log := logger.NewNoop()
	service, err := gateway.NewService(config, log)
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	client := &http.Client{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(server.URL + "/v1/models")
			if err != nil {
				b.Errorf("List models failed: %v", err)
				continue
			}
			resp.Body.Close()
			
			if resp.StatusCode != http.StatusOK {
				b.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

func BenchmarkGatewayCompletionRequest(b *testing.B) {
	config := createTestConfig()
	log := logger.NewNoop()
	service, err := gateway.NewService(config, log)
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	client := &http.Client{}

	// Prepare request body
	reqBody := map[string]interface{}{
		"model": "gpt-35-turbo",
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "Hello",
					},
				},
			},
		},
		"max_tokens": 5,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		b.Fatalf("Failed to marshal request: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest("POST", server.URL+"/v1/completions", bytes.NewBuffer(reqJSON))
			if err != nil {
				b.Errorf("Failed to create request: %v", err)
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("X-Tenant-ID", "test-tenant")

			resp, err := client.Do(req)
			if err != nil {
				b.Errorf("Completion request failed: %v", err)
				continue
			}
			resp.Body.Close()

			// Note: This will likely fail without real credentials, but we're measuring
			// the overhead of request processing, validation, and routing
		}
	})
}

func BenchmarkGatewayAuthenticationValidation(b *testing.B) {
	config := createTestConfig()
	log := logger.NewNoop()
	service, err := gateway.NewService(config, log)
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	server := httptest.NewServer(service.Handler())
	defer server.Close()

	client := &http.Client{}

	reqBody := map[string]interface{}{
		"model":    "gpt-35-turbo",
		"messages": []map[string]interface{}{},
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		b.Fatalf("Failed to marshal request: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest("POST", server.URL+"/v1/completions", bytes.NewBuffer(reqJSON))
			if err != nil {
				b.Errorf("Failed to create request: %v", err)
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			// Missing auth headers - should fail fast

			resp, err := client.Do(req)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			resp.Body.Close()

			// Should return 401 quickly
			if resp.StatusCode != http.StatusUnauthorized {
				b.Errorf("Expected status 401, got %d", resp.StatusCode)
			}
		}
	})
}

func createTestConfig() *env.Config {
	return &env.Config{
		Environment: env.EnvironmentDevelopment,
		ServiceName: "qlens-gateway-bench",
		Port:        "8080",
		Logging: env.LoggingConfig{
			Level:      "error", // Reduce logging noise in benchmarks
			Format:     "json",
			Structured: true,
		},
		Providers: map[string]env.ProviderConfig{
			// Use mock provider for benchmarking to avoid external dependencies
			"openai": {
				Enabled: true,
				Config: map[string]interface{}{
					"api_key": "test-key",
				},
			},
		},
	}
}