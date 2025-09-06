package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/internal/services/router"
	"github.com/quantum-suite/platform/pkg/shared/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAzureOpenAIClient(t *testing.T) {
	tests := []struct {
		name    string
		config  AzureOpenAIConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: AzureOpenAIConfig{
				Endpoint:   "https://test.openai.azure.com",
				APIKey:     "test-key",
				APIVersion: "2024-02-15-preview",
				Deployments: map[string]string{
					"gpt-4": "gpt-4",
				},
			},
			wantErr: false,
		},
		{
			name: "missing endpoint",
			config: AzureOpenAIConfig{
				APIKey:     "test-key",
				APIVersion: "2024-02-15-preview",
			},
			wantErr: true,
		},
		{
			name: "missing api key",
			config: AzureOpenAIConfig{
				Endpoint:   "https://test.openai.azure.com",
				APIVersion: "2024-02-15-preview",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.NewNoop()
			client, err := NewAzureOpenAIClient(tt.config, log)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestAzureOpenAIClient_CreateCompletion(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/chat/completions")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.NotEmpty(t, r.Header.Get("api-key"))

		// Mock successful response
		response := azureOpenAIResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-4",
			Choices: []azureOpenAIChoice{
				{
					Index: 0,
					Message: azureOpenAIMessage{
						Role:    "assistant",
						Content: "Hello! How can I help you?",
					},
					FinishReason: "stop",
				},
			},
			Usage: azureOpenAIUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := AzureOpenAIConfig{
		Endpoint:   server.URL,
		APIKey:     "test-key",
		APIVersion: "2024-02-15-preview",
		Deployments: map[string]string{
			"gpt-4": "gpt-4",
		},
	}

	log := logger.NewNoop()
	client, err := NewAzureOpenAIClient(config, log)
	require.NoError(t, err)

	req := &router.CompletionRequest{
		TenantID: domain.TenantID("test-tenant"),
		UserID:   domain.UserID("test-user"),
		Model:    "gpt-4",
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
		MaxTokens: intPtr(100),
	}

	response, err := client.CreateCompletion(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "test-id", response.ID)
	assert.Equal(t, domain.ProviderAzureOpenAI, response.Provider)
	assert.Len(t, response.Choices, 1)
	assert.Equal(t, "Hello! How can I help you?", response.Choices[0].Message.Content[0].Text)
}

func TestAzureOpenAIClient_CreateCompletionError(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]interface{}{
			"error": azureOpenAIError{
				Type:    "invalid_request_error",
				Code:    "invalid_model",
				Message: "The model 'invalid-model' does not exist",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := AzureOpenAIConfig{
		Endpoint:   server.URL,
		APIKey:     "test-key",
		APIVersion: "2024-02-15-preview",
		Deployments: map[string]string{
			"gpt-4": "gpt-4",
		},
	}

	log := logger.NewNoop()
	client, err := NewAzureOpenAIClient(config, log)
	require.NoError(t, err)

	req := &router.CompletionRequest{
		Model: "invalid-model",
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
	}

	response, err := client.CreateCompletion(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "invalid_model")
}

func TestAzureOpenAIClient_CreateEmbeddings(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/embeddings")

		response := azureOpenAIEmbeddingResponse{
			Object: "list",
			Data: []azureOpenAIEmbeddingData{
				{
					Object:    "embedding",
					Index:     0,
					Embedding: []float64{0.1, 0.2, 0.3},
				},
			},
			Model: "text-embedding-ada-002",
			Usage: azureOpenAIUsage{
				PromptTokens: 5,
				TotalTokens:  5,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := AzureOpenAIConfig{
		Endpoint:   server.URL,
		APIKey:     "test-key",
		APIVersion: "2024-02-15-preview",
		Deployments: map[string]string{
			"text-embedding-ada-002": "text-embedding-ada-002",
		},
	}

	log := logger.NewNoop()
	client, err := NewAzureOpenAIClient(config, log)
	require.NoError(t, err)

	req := &router.EmbeddingRequest{
		Model: "text-embedding-ada-002",
		Input: []string{"test input"},
	}

	response, err := client.CreateEmbeddings(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "list", response.Object)
	assert.Len(t, response.Data, 1)
	assert.Equal(t, []float64{0.1, 0.2, 0.3}, response.Data[0].Embedding)
}

func TestAzureOpenAIClient_ListModels(t *testing.T) {
	config := AzureOpenAIConfig{
		Endpoint:   "https://test.openai.azure.com",
		APIKey:     "test-key",
		APIVersion: "2024-02-15-preview",
		Deployments: map[string]string{
			"gpt-4":      "gpt-4",
			"gpt-35-turbo": "gpt-35-turbo",
		},
	}

	log := logger.NewNoop()
	client, err := NewAzureOpenAIClient(config, log)
	require.NoError(t, err)

	models, err := client.ListModels(context.Background())
	require.NoError(t, err)
	assert.Len(t, models, 2)

	// Check that models have correct provider
	for _, model := range models {
		assert.Equal(t, domain.ProviderAzureOpenAI, model.Provider)
		assert.True(t, model.IsActive)
		assert.Equal(t, domain.ModelStatusAvailable, model.Status)
	}
}

func TestAzureOpenAIClient_HealthCheck(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openai/deployments" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []interface{}{},
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := AzureOpenAIConfig{
		Endpoint:   server.URL,
		APIKey:     "test-key",
		APIVersion: "2024-02-15-preview",
	}

	log := logger.NewNoop()
	client, err := NewAzureOpenAIClient(config, log)
	require.NoError(t, err)

	err = client.HealthCheck(context.Background())
	assert.NoError(t, err)
}

func TestGenerateModelList(t *testing.T) {
	deployments := map[string]string{
		"gpt4-deployment":      "gpt-4",
		"gpt35-deployment":     "gpt-35-turbo",
		"embedding-deployment": "text-embedding-ada-002",
	}

	models := generateModelList(deployments)
	assert.Len(t, models, 3)

	// Find GPT-4 model
	var gpt4Model *domain.Model
	for _, model := range models {
		if model.ModelID == "gpt4-deployment" {
			gpt4Model = &model
			break
		}
	}

	require.NotNil(t, gpt4Model)
	assert.Equal(t, domain.ProviderAzureOpenAI, gpt4Model.Provider)
	assert.Contains(t, gpt4Model.Capabilities, domain.CapabilityCompletion)
	assert.Contains(t, gpt4Model.Capabilities, domain.CapabilityVision)
	assert.Contains(t, gpt4Model.Capabilities, domain.CapabilityFunctionCalling)
	assert.Equal(t, 8192, gpt4Model.ContextLength)
}

func TestConvertCompletionRequest(t *testing.T) {
	config := AzureOpenAIConfig{
		Endpoint:   "https://test.openai.azure.com",
		APIKey:     "test-key",
		APIVersion: "2024-02-15-preview",
	}

	log := logger.NewNoop()
	client, err := NewAzureOpenAIClient(config, log)
	require.NoError(t, err)

	req := &router.CompletionRequest{
		Model: "gpt-4",
		Messages: []domain.Message{
			{
				Role: domain.MessageRoleUser,
				Content: []domain.ContentPart{
					{
						Type: domain.ContentTypeText,
						Text: "Hello world",
					},
				},
			},
		},
		MaxTokens:   intPtr(100),
		Temperature: float64Ptr(0.7),
	}

	azureReq := client.convertCompletionRequest(req)
	assert.Len(t, azureReq.Messages, 1)
	assert.Equal(t, "user", azureReq.Messages[0].Role)
	assert.Equal(t, "Hello world", azureReq.Messages[0].Content)
	assert.Equal(t, 100, *azureReq.MaxTokens)
	assert.Equal(t, 0.7, *azureReq.Temperature)
}

// Helper functions for tests
func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}