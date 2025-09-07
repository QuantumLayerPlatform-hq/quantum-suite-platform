package providers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/shared/errors"
	"github.com/quantum-suite/platform/pkg/shared/logger"
)

type AzureOpenAIClient struct {
	endpoint   string
	apiKey     string
	apiVersion string
	httpClient *http.Client
	logger     logger.Logger
	models     []domain.Model
}

type AzureOpenAIConfig struct {
	Endpoint    string            `json:"endpoint"`
	APIKey      string            `json:"api_key"`
	APIVersion  string            `json:"api_version"`
	Deployments map[string]string `json:"deployments"`
}

type azureOpenAIRequest struct {
	Model            string                 `json:"model,omitempty"`
	Messages         []azureOpenAIMessage   `json:"messages"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	Temperature      *float64               `json:"temperature,omitempty"`
	TopP             *float64               `json:"top_p,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	PresencePenalty  *float64               `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64               `json:"frequency_penalty,omitempty"`
	User             string                 `json:"user,omitempty"`
	Stream           bool                   `json:"stream"`
}

type azureOpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type azureOpenAIResponse struct {
	ID      string                `json:"id"`
	Object  string                `json:"object"`
	Created int64                 `json:"created"`
	Model   string                `json:"model"`
	Choices []azureOpenAIChoice   `json:"choices"`
	Usage   azureOpenAIUsage      `json:"usage"`
	Error   *azureOpenAIError     `json:"error,omitempty"`
}

type azureOpenAIChoice struct {
	Index        int                  `json:"index"`
	Message      azureOpenAIMessage   `json:"message"`
	Delta        *azureOpenAIMessage  `json:"delta,omitempty"`
	FinishReason string               `json:"finish_reason"`
}

type azureOpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type azureOpenAIError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type azureOpenAIEmbeddingRequest struct {
	Input          []string `json:"input"`
	Model          string   `json:"model"`
	EncodingFormat string   `json:"encoding_format,omitempty"`
	Dimensions     *int     `json:"dimensions,omitempty"`
	User           string   `json:"user,omitempty"`
}

type azureOpenAIEmbeddingResponse struct {
	Object string                     `json:"object"`
	Data   []azureOpenAIEmbeddingData `json:"data"`
	Model  string                     `json:"model"`
	Usage  azureOpenAIUsage           `json:"usage"`
	Error  *azureOpenAIError          `json:"error,omitempty"`
}

type azureOpenAIEmbeddingData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

const (
	azureOpenAIDefaultAPIVersion = "2024-02-15-preview"
	azureOpenAIMaxRetries        = 3
	azureOpenAITimeout           = 30 * time.Second
)

var azureOpenAIModelPricing = map[string]domain.ModelPricing{
	"gpt-4": {
		InputTokenCost:  0.00003,
		OutputTokenCost: 0.00006,
		Unit:           "token",
	},
	"gpt-4-32k": {
		InputTokenCost:  0.00006,
		OutputTokenCost: 0.00012,
		Unit:           "token",
	},
	"gpt-35-turbo": {
		InputTokenCost:  0.0000015,
		OutputTokenCost: 0.000002,
		Unit:           "token",
	},
	"gpt-35-turbo-16k": {
		InputTokenCost:  0.000003,
		OutputTokenCost: 0.000004,
		Unit:           "token",
	},
	"text-embedding-ada-002": {
		InputTokenCost:  0.0000001,
		OutputTokenCost: 0,
		Unit:           "token",
	},
	"gpt-4o": {
		InputTokenCost:  0.000005,
		OutputTokenCost: 0.000015,
		Unit:           "token",
	},
	"gpt-4o-mini": {
		InputTokenCost:  0.00000015,
		OutputTokenCost: 0.0000006,
		Unit:           "token",
	},
	"gpt-5": {
		InputTokenCost:  0.00001,  // Premium pricing for GPT-5
		OutputTokenCost: 0.00003,
		Unit:           "token",
	},
	"gpt-5-mini": {
		InputTokenCost:  0.000005,
		OutputTokenCost: 0.000015,
		Unit:           "token",
	},
}

func NewAzureOpenAIClient(config AzureOpenAIConfig, logger logger.Logger) (*AzureOpenAIClient, error) {
	if config.Endpoint == "" {
		config.Endpoint = os.Getenv("AZURE_OPENAI_ENDPOINT")
	}
	if config.APIKey == "" {
		config.APIKey = os.Getenv("AZURE_OPENAI_API_KEY")
	}
	if config.APIVersion == "" {
		config.APIVersion = os.Getenv("AZURE_OPENAI_API_VERSION")
		if config.APIVersion == "" {
			config.APIVersion = azureOpenAIDefaultAPIVersion
		}
	}

	if config.Endpoint == "" || config.APIKey == "" {
		return nil, errors.ConfigurationError("azure openai endpoint and api key are required")
	}

	// Create production-grade HTTP client with connection pooling and DNS caching
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second, // Connection timeout
			KeepAlive: 30 * time.Second, // Keep-alive for connection reuse
			DualStack: true,             // IPv4/IPv6 dual stack
		}).DialContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		},
		MaxIdleConns:        100,              // Max idle connections
		MaxIdleConnsPerHost: 10,               // Max idle connections per host
		IdleConnTimeout:     90 * time.Second, // Idle connection timeout
		TLSHandshakeTimeout: 10 * time.Second, // TLS handshake timeout
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second, // Response header timeout
		DisableKeepAlives:     false,            // Enable keep-alives
		DisableCompression:    false,            // Enable compression
		ForceAttemptHTTP2:     true,             // Prefer HTTP/2
	}

	client := &AzureOpenAIClient{
		endpoint:   strings.TrimRight(config.Endpoint, "/"),
		apiKey:     config.APIKey,
		apiVersion: config.APIVersion,
		httpClient: &http.Client{
			Timeout:   azureOpenAITimeout,
			Transport: transport,
		},
		logger: logger,
		models: generateModelList(config.Deployments),
	}

	return client, nil
}

func generateModelList(deployments map[string]string) []domain.Model {
	models := []domain.Model{}

	for deploymentName, modelName := range deployments {
		pricing, exists := azureOpenAIModelPricing[modelName]
		if !exists {
			pricing = domain.ModelPricing{
				InputTokenCost:  0.000002,
				OutputTokenCost: 0.000002,
				Unit:           "token",
			}
		}

		capabilities := []domain.Capability{domain.CapabilityCompletion}
		contextLength := 4096

		if strings.Contains(modelName, "gpt-4") {
			capabilities = append(capabilities, domain.CapabilityVision, domain.CapabilityFunctionCalling)
			if strings.Contains(modelName, "32k") {
				contextLength = 32768
			} else {
				contextLength = 8192
			}
		} else if strings.Contains(modelName, "gpt-35-turbo") {
			capabilities = append(capabilities, domain.CapabilityFunctionCalling)
			if strings.Contains(modelName, "16k") {
				contextLength = 16384
			} else {
				contextLength = 4096
			}
		} else if strings.Contains(modelName, "embedding") {
			capabilities = []domain.Capability{domain.CapabilityEmbedding}
			contextLength = 8191
		}

		model := domain.Model{
			ModelID:       deploymentName,
			Provider:      domain.ProviderAzureOpenAI,
			Name:          fmt.Sprintf("Azure OpenAI %s (%s)", modelName, deploymentName),
			Description:   fmt.Sprintf("Azure OpenAI deployment %s running %s", deploymentName, modelName),
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

func (c *AzureOpenAIClient) CreateCompletion(ctx context.Context, req *domain.CompletionRequest) (*domain.CompletionResponse, error) {
	azureReq := c.convertCompletionRequest(req)
	
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		c.endpoint, req.Model, c.apiVersion)

	body, err := json.Marshal(azureReq)
	if err != nil {
		return nil, errors.InternalError("failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.InternalError("failed to create request", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.ProviderError("azure-openai", "azure openai request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.ProviderError("azure-openai", "failed to read response", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleHTTPError(resp.StatusCode, respBody)
	}

	var azureResp azureOpenAIResponse
	if err := json.Unmarshal(respBody, &azureResp); err != nil {
		return nil, errors.ProviderError("azure-openai", "failed to parse response", err)
	}

	if azureResp.Error != nil {
		return nil, errors.ProviderError("azure-openai", azureResp.Error.Message, nil)
	}

	return c.convertCompletionResponse(&azureResp, req.Model), nil
}

func (c *AzureOpenAIClient) CreateCompletionStream(ctx context.Context, req *domain.CompletionRequest) (<-chan *domain.StreamResponse, error) {
	azureReq := c.convertCompletionRequest(req)
	azureReq.Stream = true

	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		c.endpoint, req.Model, c.apiVersion)

	body, err := json.Marshal(azureReq)
	if err != nil {
		return nil, errors.InternalError("failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.InternalError("failed to create request", err)
	}

	c.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.ProviderError("azure-openai", "azure openai stream request failed", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, c.handleHTTPError(resp.StatusCode, respBody)
	}

	return c.processStreamResponse(resp, req.Model), nil
}

func (c *AzureOpenAIClient) CreateEmbeddings(ctx context.Context, req *domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	azureReq := azureOpenAIEmbeddingRequest{
		Input:          req.Input,
		Model:          req.Model,
		EncodingFormat: req.EncodingFormat,
		Dimensions:     req.Dimensions,
		User:           req.User,
	}

	url := fmt.Sprintf("%s/openai/deployments/%s/embeddings?api-version=%s",
		c.endpoint, req.Model, c.apiVersion)

	body, err := json.Marshal(azureReq)
	if err != nil {
		return nil, errors.InternalError("failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.InternalError("failed to create request", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.ProviderError("azure-openai", "azure openai embeddings request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.ProviderError("azure-openai", "failed to read response", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleHTTPError(resp.StatusCode, respBody)
	}

	var azureResp azureOpenAIEmbeddingResponse
	if err := json.Unmarshal(respBody, &azureResp); err != nil {
		return nil, errors.ProviderError("azure-openai", "failed to parse response", err)
	}

	if azureResp.Error != nil {
		return nil, errors.ProviderError("azure-openai", azureResp.Error.Message, nil)
	}

	return c.convertEmbeddingResponse(&azureResp), nil
}

func (c *AzureOpenAIClient) ListModels(ctx context.Context) ([]domain.Model, error) {
	return c.models, nil
}

func (c *AzureOpenAIClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/openai/models?api-version=%s", c.endpoint, c.apiVersion)
	
	// Implement retry with exponential backoff
	maxRetries := 3
	baseDelay := 100 * time.Millisecond
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 100ms, 200ms, 400ms
			delay := baseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		
		httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue // Retry on request creation failure
		}

		c.setHeaders(httpReq)

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			c.logger.Debug("Azure OpenAI health check attempt failed",
				logger.F("attempt", attempt+1),
				logger.F("error", err),
			)
			if attempt == maxRetries-1 {
				return fmt.Errorf("health check failed after %d attempts: %w", maxRetries, err)
			}
			continue // Retry on network error
		}
		
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			if attempt > 0 {
				c.logger.Info("Azure OpenAI health check succeeded on retry",
					logger.F("attempt", attempt+1),
				)
			}
			return nil
		}

		// Don't retry on HTTP errors (4xx, 5xx) - they're likely persistent
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return fmt.Errorf("health check failed after %d attempts", maxRetries)
}

func (c *AzureOpenAIClient) convertCompletionRequest(req *domain.CompletionRequest) *azureOpenAIRequest {
	messages := make([]azureOpenAIMessage, len(req.Messages))
	for i, msg := range req.Messages {
		content := ""
		for _, part := range msg.Content {
			if part.Type == domain.ContentTypeText {
				content += part.Text
			}
		}
		
		messages[i] = azureOpenAIMessage{
			Role:    string(msg.Role),
			Content: content,
		}
	}

	return &azureOpenAIRequest{
		Messages:         messages,
		MaxTokens:        req.MaxTokens,
		Temperature:      req.Temperature,
		TopP:             req.TopP,
		Stop:             req.Stop,
		PresencePenalty:  req.PresencePenalty,
		FrequencyPenalty: req.FrequencyPenalty,
		User:             req.User,
		Stream:           req.Stream,
	}
}

func (c *AzureOpenAIClient) convertCompletionResponse(azureResp *azureOpenAIResponse, modelID string) *domain.CompletionResponse {
	choices := make([]domain.Choice, len(azureResp.Choices))
	for i, choice := range azureResp.Choices {
		message := domain.Message{
			Role: domain.MessageRole(choice.Message.Role),
			Content: []domain.ContentPart{
				{
					Type: domain.ContentTypeText,
					Text: choice.Message.Content,
				},
			},
		}

		choices[i] = domain.Choice{
			Index:        choice.Index,
			Message:      message,
			FinishReason: domain.FinishReason(choice.FinishReason),
		}
	}

	usage := domain.Usage{
		PromptTokens:     azureResp.Usage.PromptTokens,
		CompletionTokens: azureResp.Usage.CompletionTokens,
		TotalTokens:      azureResp.Usage.TotalTokens,
		CostUSD:          c.calculateCost(modelID, azureResp.Usage),
	}

	return &domain.CompletionResponse{
		ID:       azureResp.ID,
		Object:   azureResp.Object,
		Created:  azureResp.Created,
		Model:    modelID,
		Provider: domain.ProviderAzureOpenAI,
		Choices:  choices,
		Usage:    usage,
	}
}

func (c *AzureOpenAIClient) convertEmbeddingResponse(azureResp *azureOpenAIEmbeddingResponse) *domain.EmbeddingResponse {
	data := make([]domain.Embedding, len(azureResp.Data))
	for i, item := range azureResp.Data {
		data[i] = domain.Embedding{
			Object:    item.Object,
			Index:     item.Index,
			Embedding: item.Embedding,
		}
	}

	usage := domain.EmbeddingUsage{
		PromptTokens: azureResp.Usage.PromptTokens,
		TotalTokens:  azureResp.Usage.TotalTokens,
		CostUSD:      c.calculateEmbeddingCost(azureResp.Model, azureResp.Usage),
	}

	return &domain.EmbeddingResponse{
		Object:   azureResp.Object,
		Data:     data,
		Model:    azureResp.Model,
		Provider: domain.ProviderAzureOpenAI,
		Usage:    usage,
	}
}

func (c *AzureOpenAIClient) processStreamResponse(resp *http.Response, modelID string) <-chan *domain.StreamResponse {
	ch := make(chan *domain.StreamResponse)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		
		for {
			var line string
			if err := decoder.Decode(&line); err != nil {
				if err != io.EOF {
					ch <- &domain.StreamResponse{
						Error: errors.ProviderError("azure-openai", "failed to read stream", err),
					}
				}
				return
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- &domain.StreamResponse{Done: true}
				return
			}

			var azureResp azureOpenAIResponse
			if err := json.Unmarshal([]byte(data), &azureResp); err != nil {
				continue
			}

			if azureResp.Error != nil {
				ch <- &domain.StreamResponse{
					Error: errors.ProviderError("azure-openai", azureResp.Error.Message, nil),
				}
				return
			}

			streamResp := c.convertStreamResponse(&azureResp, modelID)
			ch <- streamResp
		}
	}()

	return ch
}

func (c *AzureOpenAIClient) convertStreamResponse(azureResp *azureOpenAIResponse, modelID string) *domain.StreamResponse {
	choices := make([]domain.Choice, len(azureResp.Choices))
	for i, choice := range azureResp.Choices {
		content := ""
		if choice.Delta != nil {
			content = choice.Delta.Content
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

		choices[i] = domain.Choice{
			Index:        choice.Index,
			Message:      message,
			FinishReason: domain.FinishReason(choice.FinishReason),
		}
	}

	return &domain.StreamResponse{
		ID:       azureResp.ID,
		Object:   azureResp.Object,
		Created:  azureResp.Created,
		Model:    modelID,
		Provider: domain.ProviderAzureOpenAI,
		Choices:  choices,
	}
}

func (c *AzureOpenAIClient) calculateCost(modelID string, usage azureOpenAIUsage) float64 {
	pricing, exists := azureOpenAIModelPricing[modelID]
	if !exists {
		return 0
	}

	inputCost := float64(usage.PromptTokens) * pricing.InputTokenCost
	outputCost := float64(usage.CompletionTokens) * pricing.OutputTokenCost

	return inputCost + outputCost
}

func (c *AzureOpenAIClient) calculateEmbeddingCost(modelID string, usage azureOpenAIUsage) float64 {
	pricing, exists := azureOpenAIModelPricing[modelID]
	if !exists {
		return 0
	}

	return float64(usage.PromptTokens) * pricing.InputTokenCost
}

func (c *AzureOpenAIClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)
	req.Header.Set("User-Agent", "QLens/1.0")
}

func (c *AzureOpenAIClient) handleHTTPError(statusCode int, body []byte) error {
	var azureError azureOpenAIError
	if err := json.Unmarshal(body, &azureError); err == nil && azureError.Message != "" {
		switch statusCode {
		case http.StatusUnauthorized:
			return errors.AuthenticationError(azureError.Message)
		case http.StatusForbidden:
			return errors.AuthorizationError(azureError.Message)
		case http.StatusTooManyRequests:
			return errors.NewError(errors.ErrorTypeTooManyRequests, azureError.Message).WithRetryable(true).Build()
		case http.StatusBadRequest:
			return errors.ValidationError(azureError.Message, "request")
		default:
			return errors.ProviderError("azure-openai", azureError.Message, nil)
		}
	}

	return errors.ProviderError("azure-openai", fmt.Sprintf("azure openai api error: %d", statusCode), nil)
}