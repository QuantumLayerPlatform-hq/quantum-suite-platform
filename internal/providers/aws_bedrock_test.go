package providers

import (
	"context"
	"testing"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/internal/services/router"
	"github.com/quantum-suite/platform/pkg/shared/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAWSBedrockClient(t *testing.T) {
	tests := []struct {
		name    string
		config  AWSBedrockConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: AWSBedrockConfig{
				Region:          "us-east-1",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Models: []BedrockModelConfig{
					{
						ID:      "claude-3-sonnet",
						ModelID: "anthropic.claude-3-sonnet-20240229-v1:0",
						Name:    "Claude 3 Sonnet",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty config with defaults",
			config: AWSBedrockConfig{
				Models: []BedrockModelConfig{
					{
						ID:      "claude-3-sonnet",
						ModelID: "anthropic.claude-3-sonnet-20240229-v1:0",
						Name:    "Claude 3 Sonnet",
					},
				},
			},
			wantErr: false, // Should work with default region and env credentials
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.NewNoop()
			client, err := NewAWSBedrockClient(tt.config, log)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				// Note: This may fail in CI without AWS credentials
				// but we're testing the client creation logic
				if err != nil {
					t.Skipf("AWS credentials not available: %v", err)
				}
				assert.NotNil(t, client)
			}
		})
	}
}

func TestGenerateBedrockModelList(t *testing.T) {
	modelConfigs := []BedrockModelConfig{
		{
			ID:      "claude-3-sonnet",
			ModelID: "anthropic.claude-3-sonnet-20240229-v1:0",
			Name:    "Claude 3 Sonnet",
		},
		{
			ID:      "claude-3-haiku",
			ModelID: "anthropic.claude-3-haiku-20240307-v1:0",
			Name:    "Claude 3 Haiku",
		},
	}

	models := generateBedrockModelList(modelConfigs)
	assert.Len(t, models, 2)

	// Check first model
	model1 := models[0]
	assert.Equal(t, "claude-3-sonnet", model1.ModelID)
	assert.Equal(t, domain.ProviderAWSBedrock, model1.Provider)
	assert.Equal(t, "Claude 3 Sonnet", model1.Name)
	assert.Contains(t, model1.Capabilities, domain.CapabilityCompletion)
	assert.Contains(t, model1.Capabilities, domain.CapabilityVision)
	assert.Equal(t, 200000, model1.ContextLength)
	assert.True(t, model1.IsActive)
	assert.Equal(t, domain.ModelStatusAvailable, model1.Status)

	// Check pricing
	assert.Greater(t, model1.Pricing.InputTokenCost, 0.0)
	assert.Greater(t, model1.Pricing.OutputTokenCost, 0.0)
}

func TestBedrockConvertCompletionRequest(t *testing.T) {
	config := AWSBedrockConfig{
		Models: []BedrockModelConfig{
			{
				ID:      "claude-3-sonnet",
				ModelID: "anthropic.claude-3-sonnet-20240229-v1:0",
				Name:    "Claude 3 Sonnet",
			},
		},
	}

	log := logger.NewNoop()
	client, err := NewAWSBedrockClient(config, log)
	if err != nil {
		t.Skipf("AWS credentials not available: %v", err)
	}
	require.NoError(t, err)

	req := &router.CompletionRequest{
		Model: "claude-3-sonnet",
		Messages: []domain.Message{
			{
				Role: domain.MessageRoleSystem,
				Content: []domain.ContentPart{
					{
						Type: domain.ContentTypeText,
						Text: "You are a helpful assistant.",
					},
				},
			},
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

	claudeReq := client.convertCompletionRequest(req)
	assert.Equal(t, claudeAnthropicVersion, claudeReq.AnthropicVersion)
	assert.Equal(t, 100, claudeReq.MaxTokens)
	assert.Equal(t, 0.7, *claudeReq.Temperature)
	assert.Equal(t, "You are a helpful assistant.", claudeReq.System)
	assert.Len(t, claudeReq.Messages, 1)
	assert.Equal(t, "user", claudeReq.Messages[0].Role)
	assert.Equal(t, "Hello world", claudeReq.Messages[0].Content)
}

func TestBedrockFindModelID(t *testing.T) {
	config := AWSBedrockConfig{
		Models: []BedrockModelConfig{
			{
				ID:      "claude-3-sonnet",
				ModelID: "anthropic.claude-3-sonnet-20240229-v1:0",
				Name:    "Claude 3 Sonnet",
			},
		},
	}

	log := logger.NewNoop()
	client, err := NewAWSBedrockClient(config, log)
	if err != nil {
		t.Skipf("AWS credentials not available: %v", err)
	}
	require.NoError(t, err)

	// Test finding existing model
	modelID := client.findModelID("claude-3-sonnet")
	assert.Equal(t, "anthropic.claude-3-sonnet-20240229-v1:0", modelID)

	// Test finding non-existent model
	modelID = client.findModelID("non-existent")
	assert.Empty(t, modelID)
}

func TestBedrockConvertFinishReason(t *testing.T) {
	config := AWSBedrockConfig{
		Models: []BedrockModelConfig{
			{
				ID:      "claude-3-sonnet",
				ModelID: "anthropic.claude-3-sonnet-20240229-v1:0",
				Name:    "Claude 3 Sonnet",
			},
		},
	}

	log := logger.NewNoop()
	client, err := NewAWSBedrockClient(config, log)
	if err != nil {
		t.Skipf("AWS credentials not available: %v", err)
	}
	require.NoError(t, err)

	tests := []struct {
		claudeReason string
		expected     domain.FinishReason
	}{
		{"end_turn", domain.FinishReasonStop},
		{"max_tokens", domain.FinishReasonLength},
		{"stop_sequence", domain.FinishReasonStop},
		{"unknown", domain.FinishReasonStop},
	}

	for _, tt := range tests {
		t.Run(tt.claudeReason, func(t *testing.T) {
			result := client.convertFinishReason(tt.claudeReason)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBedrockCalculateCost(t *testing.T) {
	config := AWSBedrockConfig{
		Models: []BedrockModelConfig{
			{
				ID:      "claude-3-sonnet",
				ModelID: "anthropic.claude-3-sonnet-20240229-v1:0",
				Name:    "Claude 3 Sonnet",
			},
		},
	}

	log := logger.NewNoop()
	client, err := NewAWSBedrockClient(config, log)
	if err != nil {
		t.Skipf("AWS credentials not available: %v", err)
	}
	require.NoError(t, err)

	usage := claudeUsage{
		InputTokens:  1000,
		OutputTokens: 500,
	}

	cost := client.calculateCost("anthropic.claude-3-sonnet-20240229-v1:0", usage)
	
	// Should be greater than 0 for known models
	assert.Greater(t, cost, 0.0)
	
	// Test with unknown model
	costUnknown := client.calculateCost("unknown-model", usage)
	assert.Equal(t, 0.0, costUnknown)
}

func TestBedrockListModels(t *testing.T) {
	config := AWSBedrockConfig{
		Models: []BedrockModelConfig{
			{
				ID:      "claude-3-sonnet",
				ModelID: "anthropic.claude-3-sonnet-20240229-v1:0",
				Name:    "Claude 3 Sonnet",
			},
			{
				ID:      "claude-3-haiku",
				ModelID: "anthropic.claude-3-haiku-20240307-v1:0",
				Name:    "Claude 3 Haiku",
			},
		},
	}

	log := logger.NewNoop()
	client, err := NewAWSBedrockClient(config, log)
	if err != nil {
		t.Skipf("AWS credentials not available: %v", err)
	}
	require.NoError(t, err)

	models, err := client.ListModels(context.Background())
	require.NoError(t, err)
	assert.Len(t, models, 2)

	// Check that all models have correct provider
	for _, model := range models {
		assert.Equal(t, domain.ProviderAWSBedrock, model.Provider)
		assert.True(t, model.IsActive)
		assert.Equal(t, domain.ModelStatusAvailable, model.Status)
	}
}

func TestBedrockCreateEmbeddings(t *testing.T) {
	config := AWSBedrockConfig{
		Models: []BedrockModelConfig{
			{
				ID:      "claude-3-sonnet",
				ModelID: "anthropic.claude-3-sonnet-20240229-v1:0",
				Name:    "Claude 3 Sonnet",
			},
		},
	}

	log := logger.NewNoop()
	client, err := NewAWSBedrockClient(config, log)
	if err != nil {
		t.Skipf("AWS credentials not available: %v", err)
	}
	require.NoError(t, err)

	req := &router.EmbeddingRequest{
		Model: "claude-3-sonnet",
		Input: []string{"test input"},
	}

	// Should return not implemented error
	response, err := client.CreateEmbeddings(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "not supported")
}