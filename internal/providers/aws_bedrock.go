package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/google/uuid"
	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/shared/errors"
	"github.com/quantum-suite/platform/pkg/shared/logger"
)

type AWSBedrockClient struct {
	client *bedrockruntime.Client
	region string
	logger logger.Logger
	models []domain.Model
}

type AWSBedrockConfig struct {
	Region          string                    `json:"region"`
	AccessKeyID     string                    `json:"access_key_id"`
	SecretAccessKey string                    `json:"secret_access_key"`
	SessionToken    string                    `json:"session_token"`
	Models          []BedrockModelConfig      `json:"models"`
}

type BedrockModelConfig struct {
	ID      string `json:"id"`
	ModelID string `json:"model_id"`
	Name    string `json:"name"`
}

type claudeRequest struct {
	AnthropicVersion string          `json:"anthropic_version"`
	MaxTokens        int             `json:"max_tokens"`
	Temperature      *float64        `json:"temperature,omitempty"`
	TopP             *float64        `json:"top_p,omitempty"`
	Messages         []claudeMessage `json:"messages"`
	System           string          `json:"system,omitempty"`
	Stop             []string        `json:"stop_sequences,omitempty"`
	Stream           bool            `json:"stream,omitempty"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	ID           string              `json:"id"`
	Type         string              `json:"type"`
	Role         string              `json:"role"`
	Content      []claudeContent     `json:"content"`
	Model        string              `json:"model"`
	StopReason   string              `json:"stop_reason"`
	StopSequence string              `json:"stop_sequence,omitempty"`
	Usage        claudeUsage         `json:"usage"`
	Error        *claudeError        `json:"error,omitempty"`
}

type claudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type claudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type claudeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type claudeStreamResponse struct {
	Type         string          `json:"type"`
	Index        int             `json:"index,omitempty"`
	Delta        *claudeContent  `json:"delta,omitempty"`
	Usage        *claudeUsage    `json:"usage,omitempty"`
	StopReason   string          `json:"stop_reason,omitempty"`
	StopSequence string          `json:"stop_sequence,omitempty"`
}

const (
	claudeAnthropicVersion = "bedrock-2023-05-31"
	bedrockDefaultRegion   = "us-east-1"
	bedrockTimeout         = 60 * time.Second
)

var bedrockModelPricing = map[string]domain.ModelPricing{
	"anthropic.claude-3-7-sonnet-20250219-v1:0": {
		InputTokenCost:  0.000003,  // Same as Claude 3 Sonnet
		OutputTokenCost: 0.000015,
		Unit:           "token",
	},
	"anthropic.claude-3-sonnet-20240229-v1:0": {
		InputTokenCost:  0.000003,
		OutputTokenCost: 0.000015,
		Unit:           "token",
	},
	"anthropic.claude-3-haiku-20240307-v1:0": {
		InputTokenCost:  0.00000025,
		OutputTokenCost: 0.00000125,
		Unit:           "token",
	},
	"anthropic.claude-3-opus-20240229-v1:0": {
		InputTokenCost:  0.000015,
		OutputTokenCost: 0.000075,
		Unit:           "token",
	},
}

func NewAWSBedrockClient(bedrockConfig AWSBedrockConfig, logger logger.Logger) (*AWSBedrockClient, error) {
	if bedrockConfig.Region == "" {
		bedrockConfig.Region = os.Getenv("AWS_REGION")
		if bedrockConfig.Region == "" {
			bedrockConfig.Region = bedrockDefaultRegion
		}
	}

	if bedrockConfig.AccessKeyID == "" {
		bedrockConfig.AccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	}
	if bedrockConfig.SecretAccessKey == "" {
		bedrockConfig.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}
	if bedrockConfig.SessionToken == "" {
		bedrockConfig.SessionToken = os.Getenv("AWS_SESSION_TOKEN")
	}

	// Configure AWS SDK with production-grade settings
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(bedrockConfig.Region),
		config.WithRetryMaxAttempts(3),
		config.WithRetryMode(aws.RetryModeAdaptive),
	)
	if err != nil {
		return nil, errors.ConfigurationError("failed to load aws config: " + err.Error())
	}

	if bedrockConfig.AccessKeyID != "" && bedrockConfig.SecretAccessKey != "" {
		cfg.Credentials = credentials.NewStaticCredentialsProvider(
			bedrockConfig.AccessKeyID,
			bedrockConfig.SecretAccessKey,
			bedrockConfig.SessionToken,
		)
	}

	client := bedrockruntime.NewFromConfig(cfg)

	return &AWSBedrockClient{
		client: client,
		region: bedrockConfig.Region,
		logger: logger,
		models: generateBedrockModelList(bedrockConfig.Models),
	}, nil
}

func generateBedrockModelList(modelConfigs []BedrockModelConfig) []domain.Model {
	models := []domain.Model{}

	for _, modelConfig := range modelConfigs {
		pricing, exists := bedrockModelPricing[modelConfig.ModelID]
		if !exists {
			pricing = domain.ModelPricing{
				InputTokenCost:  0.000003,
				OutputTokenCost: 0.000015,
				Unit:           "token",
			}
		}

		capabilities := []domain.Capability{domain.CapabilityCompletion}
		contextLength := 200000

		if strings.Contains(modelConfig.ModelID, "claude-3") {
			capabilities = append(capabilities, domain.CapabilityVision)
		}

		model := domain.Model{
			ModelID:       modelConfig.ID,
			Provider:      domain.ProviderAWSBedrock,
			Name:          modelConfig.Name,
			Description:   fmt.Sprintf("AWS Bedrock %s", modelConfig.ModelID),
			Capabilities:  capabilities,
			ContextLength: contextLength,
			Pricing:       pricing,
			Status:        domain.ModelStatusAvailable,
			IsActive:      true,
		}
		model.BaseEntity = domain.NewBaseEntity()

		models = append(models, model)
	}

	return models
}

func (c *AWSBedrockClient) CreateCompletion(ctx context.Context, req *domain.CompletionRequest) (*domain.CompletionResponse, error) {
	modelID := c.findModelID(req.Model)
	if modelID == "" {
		return nil, errors.ValidationError("model not found", "model")
	}

	claudeReq := c.convertCompletionRequest(req)
	
	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, errors.InternalError("failed to marshal request", err)
	}

	input := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        body,
	}

	result, err := c.client.InvokeModel(ctx, input)
	if err != nil {
		return nil, c.handleAWSError(err)
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(result.Body, &claudeResp); err != nil {
		return nil, errors.ProviderError("bedrock", "failed to parse response", err)
	}

	if claudeResp.Error != nil {
		return nil, errors.ProviderError("bedrock", claudeResp.Error.Message, nil)
	}

	return c.convertCompletionResponse(&claudeResp, req.Model), nil
}

func (c *AWSBedrockClient) CreateCompletionStream(ctx context.Context, req *domain.CompletionRequest) (<-chan *domain.StreamResponse, error) {
	modelID := c.findModelID(req.Model)
	if modelID == "" {
		return nil, errors.ValidationError("model not found", "model")
	}

	claudeReq := c.convertCompletionRequest(req)
	claudeReq.Stream = true
	
	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, errors.InternalError("failed to marshal request", err)
	}

	input := &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        body,
	}

	result, err := c.client.InvokeModelWithResponseStream(ctx, input)
	if err != nil {
		return nil, c.handleAWSError(err)
	}

	return c.processStreamResponse(result, req.Model), nil
}

func (c *AWSBedrockClient) CreateEmbeddings(ctx context.Context, req *domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	return nil, errors.InternalError("embeddings not supported by bedrock claude models", nil)
}

func (c *AWSBedrockClient) ListModels(ctx context.Context) ([]domain.Model, error) {
	return c.models, nil
}

func (c *AWSBedrockClient) HealthCheck(ctx context.Context) error {
	if len(c.models) == 0 {
		return fmt.Errorf("no models configured")
	}

	modelID := c.findModelID(c.models[0].ModelID)
	if modelID == "" {
		return fmt.Errorf("invalid model configuration")
	}

	testReq := &claudeRequest{
		AnthropicVersion: claudeAnthropicVersion,
		MaxTokens:        1,
		Messages: []claudeMessage{
			{
				Role:    "user",
				Content: "test",
			},
		},
	}

	body, err := json.Marshal(testReq)
	if err != nil {
		return err
	}

	input := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        body,
	}

	// Implement retry with exponential backoff for AWS Bedrock health check
	maxRetries := 3
	baseDelay := 200 * time.Millisecond
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 200ms, 400ms, 800ms
			delay := baseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Create a new timeout context for each attempt
		attemptCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		
		result, err := c.client.InvokeModel(attemptCtx, input)
		cancel()
		
		if err != nil {
			c.logger.Debug("AWS Bedrock health check attempt failed",
				logger.F("attempt", attempt+1),
				logger.F("model", modelID),
				logger.F("error", err),
			)
			
			// Don't retry on certain permanent errors, but handle throttling specially
			errStr := err.Error()
			if strings.Contains(errStr, "ValidationException") ||
			   strings.Contains(errStr, "InvalidParameterException") ||
			   strings.Contains(errStr, "ResourceNotFoundException") {
				return fmt.Errorf("bedrock health check failed (non-retryable): %w", err)
			}
			
			// For throttling, increase delay significantly
			if strings.Contains(errStr, "ThrottlingException") || strings.Contains(errStr, "429") {
				c.logger.Warn("AWS Bedrock throttling detected, using extended backoff",
					logger.F("attempt", attempt+1),
				)
				// Skip immediate retry on throttling - wait for next health check cycle
				if attempt < maxRetries-1 {
					select {
					case <-time.After(30 * time.Second):
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			
			if attempt == maxRetries-1 {
				return fmt.Errorf("bedrock health check failed after %d attempts: %w", maxRetries, err)
			}
			continue // Retry on network errors, throttling, etc.
		}

		// Success - validate the response
		if result != nil && len(result.Body) > 0 {
			if attempt > 0 {
				c.logger.Info("AWS Bedrock health check succeeded on retry",
					logger.F("attempt", attempt+1),
					logger.F("model", modelID),
				)
			}
			return nil
		}
		
		// Unexpected empty response, but don't fail immediately
		c.logger.Warn("AWS Bedrock returned empty response on health check",
			logger.F("attempt", attempt+1),
			logger.F("model", modelID),
		)
	}

	return fmt.Errorf("bedrock health check failed after %d attempts", maxRetries)
}

func (c *AWSBedrockClient) convertCompletionRequest(req *domain.CompletionRequest) *claudeRequest {
	messages := []claudeMessage{}
	systemMessage := ""

	for _, msg := range req.Messages {
		content := ""
		for _, part := range msg.Content {
			if part.Type == domain.ContentTypeText {
				content += part.Text
			}
		}

		if msg.Role == domain.MessageRoleSystem {
			systemMessage = content
		} else {
			role := string(msg.Role)
			if role == "assistant" {
				role = "assistant"
			} else {
				role = "user"
			}
			
			messages = append(messages, claudeMessage{
				Role:    role,
				Content: content,
			})
		}
	}

	maxTokens := 4096
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}

	claudeReq := &claudeRequest{
		AnthropicVersion: claudeAnthropicVersion,
		MaxTokens:        maxTokens,
		Messages:         messages,
		Temperature:      req.Temperature,
		TopP:             req.TopP,
		Stop:             req.Stop,
	}

	if systemMessage != "" {
		claudeReq.System = systemMessage
	}

	return claudeReq
}

func (c *AWSBedrockClient) convertCompletionResponse(claudeResp *claudeResponse, modelID string) *domain.CompletionResponse {
	content := ""
	if len(claudeResp.Content) > 0 {
		content = claudeResp.Content[0].Text
	}

	message := domain.Message{
		Role: domain.MessageRoleAssistant,
		Content: []domain.ContentPart{
			{
				Type: domain.ContentTypeText,
				Text: content,
			},
		},
	}

	choice := domain.Choice{
		Index:        0,
		Message:      message,
		FinishReason: c.convertFinishReason(claudeResp.StopReason),
	}

	usage := domain.Usage{
		PromptTokens:     claudeResp.Usage.InputTokens,
		CompletionTokens: claudeResp.Usage.OutputTokens,
		TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		CostUSD:          c.calculateCost(c.findModelID(modelID), claudeResp.Usage),
	}

	return &domain.CompletionResponse{
		ID:       claudeResp.ID,
		Object:   "chat.completion",
		Created:  time.Now().Unix(),
		Model:    modelID,
		Provider: domain.ProviderAWSBedrock,
		Choices:  []domain.Choice{choice},
		Usage:    usage,
	}
}

func (c *AWSBedrockClient) processStreamResponse(stream *bedrockruntime.InvokeModelWithResponseStreamOutput, modelID string) <-chan *domain.StreamResponse {
	ch := make(chan *domain.StreamResponse)

	go func() {
		defer close(ch)
		
		for event := range stream.GetStream().Events() {
			switch v := event.(type) {
			case *bedrocktypes.ResponseStreamMemberChunk:
				var streamResp claudeStreamResponse
				if err := json.Unmarshal(v.Value.Bytes, &streamResp); err != nil {
					ch <- &domain.StreamResponse{
						Error: errors.ProviderError("bedrock", "failed to parse stream response", err),
					}
					return
				}

				if streamResp.Type == "content_block_delta" && streamResp.Delta != nil {
					message := domain.Message{
						Role: domain.MessageRoleAssistant,
						Content: []domain.ContentPart{
							{
								Type: domain.ContentTypeText,
								Text: streamResp.Delta.Text,
							},
						},
					}

					choice := domain.Choice{
						Index:   streamResp.Index,
						Message: message,
					}

					ch <- &domain.StreamResponse{
						ID:       uuid.New().String(),
						Object:   "chat.completion.chunk",
						Created:  time.Now().Unix(),
						Model:    modelID,
						Provider: domain.ProviderAWSBedrock,
						Choices:  []domain.Choice{choice},
					}
				} else if streamResp.Type == "message_stop" {
					ch <- &domain.StreamResponse{Done: true}
					return
				}

			default:
				// Handle error cases - stream ended or error occurred
				ch <- &domain.StreamResponse{
					Error: errors.ProviderError("bedrock", "stream processing error", nil),
				}
				return
			}
		}
	}()

	return ch
}

func (c *AWSBedrockClient) findModelID(localID string) string {
	for _, model := range c.models {
		if model.ModelID == localID {
			return strings.Replace(model.Description, "AWS Bedrock ", "", 1)
		}
	}
	return ""
}

func (c *AWSBedrockClient) convertFinishReason(stopReason string) domain.FinishReason {
	switch stopReason {
	case "end_turn":
		return domain.FinishReasonStop
	case "max_tokens":
		return domain.FinishReasonLength
	case "stop_sequence":
		return domain.FinishReasonStop
	default:
		return domain.FinishReasonStop
	}
}

func (c *AWSBedrockClient) calculateCost(modelID string, usage claudeUsage) float64 {
	pricing, exists := bedrockModelPricing[modelID]
	if !exists {
		return 0
	}

	inputCost := float64(usage.InputTokens) * pricing.InputTokenCost
	outputCost := float64(usage.OutputTokens) * pricing.OutputTokenCost

	return inputCost + outputCost
}

func (c *AWSBedrockClient) handleAWSError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	
	if strings.Contains(errStr, "throttling") || strings.Contains(errStr, "rate") {
		return errors.InternalError("aws bedrock rate limit exceeded", nil)
	}
	
	if strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "access denied") {
		return errors.AuthenticationError("aws bedrock authentication failed")
	}
	
	if strings.Contains(errStr, "validation") || strings.Contains(errStr, "invalid") {
		return errors.ValidationError("aws bedrock validation error", "request")
	}

	return errors.ProviderError("bedrock", "aws bedrock error: " + errStr, err)
}