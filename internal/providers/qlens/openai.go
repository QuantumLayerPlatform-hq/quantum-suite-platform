package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/qlens-types"
)

// OpenAIClient implements the ProviderClient interface for OpenAI
type OpenAIClient struct {
	config     types.ProviderConfig
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(config types.ProviderConfig) *OpenAIClient {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &OpenAIClient{
		config:  config,
		baseURL: baseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Provider returns the provider type
func (c *OpenAIClient) Provider() domain.Provider {
	return domain.ProviderOpenAI
}

// Name returns the provider name
func (c *OpenAIClient) Name() string {
	return "OpenAI"
}

// CreateCompletion creates a completion using OpenAI API
func (c *OpenAIClient) CreateCompletion(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error) {
	start := time.Now()

	// Convert request to OpenAI format
	openAIReq := c.convertCompletionRequest(req)

	// Make API request
	respData, err := c.makeRequest(ctx, "POST", "/chat/completions", openAIReq)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API request failed: %w", err)
	}

	// Parse OpenAI response
	var openAIResp OpenAIChatCompletionResponse
	if err := json.Unmarshal(respData, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	// Convert to QLens response
	response := c.convertCompletionResponse(&openAIResp, req.RequestID, time.Since(start))

	return response, nil
}

// CreateCompletionStream creates a streaming completion using OpenAI API
func (c *OpenAIClient) CreateCompletionStream(ctx context.Context, req *types.CompletionRequest) (<-chan types.StreamResponse, error) {
	// Convert request to OpenAI format with streaming enabled
	openAIReq := c.convertCompletionRequest(req)
	openAIReq.Stream = true

	// Create HTTP request
	reqBody, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")

	// Make request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %s", string(body))
	}

	// Create stream channel
	streamChan := make(chan types.StreamResponse)

	go c.handleStream(ctx, resp.Body, streamChan, req.RequestID)

	return streamChan, nil
}

// CreateEmbeddings creates embeddings using OpenAI API
func (c *OpenAIClient) CreateEmbeddings(ctx context.Context, req *types.EmbeddingRequest) (*types.EmbeddingResponse, error) {
	start := time.Now()

	// Convert request to OpenAI format
	openAIReq := c.convertEmbeddingRequest(req)

	// Make API request
	respData, err := c.makeRequest(ctx, "POST", "/embeddings", openAIReq)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API request failed: %w", err)
	}

	// Parse OpenAI response
	var openAIResp OpenAIEmbeddingResponse
	if err := json.Unmarshal(respData, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI embedding response: %w", err)
	}

	// Convert to QLens response
	response := c.convertEmbeddingResponse(&openAIResp, req.RequestID, time.Since(start))

	return response, nil
}

// ListModels lists available models from OpenAI
func (c *OpenAIClient) ListModels(ctx context.Context) ([]types.Model, error) {
	respData, err := c.makeRequest(ctx, "GET", "/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list OpenAI models: %w", err)
	}

	var openAIResp OpenAIModelsResponse
	if err := json.Unmarshal(respData, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	models := make([]types.Model, 0, len(openAIResp.Data))
	for _, openAIModel := range openAIResp.Data {
		model := c.convertModel(&openAIModel)
		models = append(models, model)
	}

	return models, nil
}

// GetModel gets a specific model from OpenAI
func (c *OpenAIClient) GetModel(ctx context.Context, modelID string) (*types.Model, error) {
	respData, err := c.makeRequest(ctx, "GET", "/models/"+modelID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenAI model: %w", err)
	}

	var openAIModel OpenAIModel
	if err := json.Unmarshal(respData, &openAIModel); err != nil {
		return nil, fmt.Errorf("failed to parse model response: %w", err)
	}

	model := c.convertModel(&openAIModel)
	return &model, nil
}

// HealthCheck performs a health check against OpenAI API
func (c *OpenAIClient) HealthCheck(ctx context.Context) error {
	_, err := c.makeRequest(ctx, "GET", "/models", nil)
	return err
}

// Configure updates the client configuration
func (c *OpenAIClient) Configure(config types.ProviderConfig) error {
	c.config = config
	c.apiKey = config.APIKey

	if config.BaseURL != "" {
		c.baseURL = config.BaseURL
	}

	if config.Timeout > 0 {
		c.httpClient.Timeout = config.Timeout
	}

	return nil
}

// GetConfig returns the current configuration
func (c *OpenAIClient) GetConfig() types.ProviderConfig {
	return c.config
}

// Close cleans up resources
func (c *OpenAIClient) Close() error {
	// Close HTTP client if needed
	c.httpClient.CloseIdleConnections()
	return nil
}

// Helper methods

func (c *OpenAIClient) makeRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var openAIErr OpenAIError
		if err := json.Unmarshal(respBody, &openAIErr); err == nil {
			return nil, &types.QLensError{
				Type:     types.ErrorTypeProviderError,
				Message:  openAIErr.Error.Message,
				Code:     openAIErr.Error.Code,
				Provider: domain.ProviderOpenAI,
			}
		}
		return nil, fmt.Errorf("OpenAI API error: %s", string(respBody))
	}

	return respBody, nil
}

func (c *OpenAIClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "QLens/1.0.0")
}

func (c *OpenAIClient) handleStream(ctx context.Context, body io.ReadCloser, streamChan chan<- types.StreamResponse, requestID string) {
	defer close(streamChan)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			streamChan <- types.StreamResponse{
				Done:      true,
				RequestID: requestID,
			}
			return
		}

		var chunk OpenAIChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			streamChan <- types.StreamResponse{
				Error: &types.StreamError{
					Type:    types.ErrorTypeProviderError,
					Message: fmt.Sprintf("Failed to parse stream chunk: %v", err),
				},
			}
			return
		}

		streamResp := c.convertStreamChunk(&chunk, requestID)
		streamChan <- streamResp
	}

	if err := scanner.Err(); err != nil {
		streamChan <- types.StreamResponse{
			Error: &types.StreamError{
				Type:    types.ErrorTypeProviderError,
				Message: fmt.Sprintf("Stream reading error: %v", err),
			},
		}
	}
}

// Conversion methods

func (c *OpenAIClient) convertCompletionRequest(req *types.CompletionRequest) *OpenAIChatCompletionRequest {
	openAIReq := &OpenAIChatCompletionRequest{
		Model:    req.Model,
		Messages: make([]OpenAIMessage, len(req.Messages)),
		Stream:   req.Stream,
	}

	// Set default model if not specified
	if openAIReq.Model == "" {
		openAIReq.Model = "gpt-3.5-turbo"
	}

	// Convert messages
	for i, msg := range req.Messages {
		openAIReq.Messages[i] = c.convertMessage(msg)
	}

	// Set optional parameters
	if req.MaxTokens != nil {
		openAIReq.MaxTokens = req.MaxTokens
	}
	if req.Temperature != nil {
		openAIReq.Temperature = req.Temperature
	}
	if req.TopP != nil {
		openAIReq.TopP = req.TopP
	}
	if req.Stop != nil {
		openAIReq.Stop = req.Stop
	}
	if req.PresencePenalty != nil {
		openAIReq.PresencePenalty = req.PresencePenalty
	}
	if req.FrequencyPenalty != nil {
		openAIReq.FrequencyPenalty = req.FrequencyPenalty
	}
	if req.User != "" {
		openAIReq.User = req.User
	}

	return openAIReq
}

func (c *OpenAIClient) convertMessage(msg domain.Message) OpenAIMessage {
	openAIMsg := OpenAIMessage{
		Role: string(msg.Role),
	}

	// Handle content based on type
	if len(msg.Content) == 1 && msg.Content[0].Type == domain.ContentTypeText {
		// Simple text content
		openAIMsg.Content = msg.Content[0].Text
	} else {
		// Multi-modal content
		content := make([]OpenAIContentPart, len(msg.Content))
		for i, part := range msg.Content {
			content[i] = c.convertContentPart(part)
		}
		openAIMsg.MultiContent = content
	}

	// Handle tool calls
	if len(msg.ToolCalls) > 0 {
		openAIMsg.ToolCalls = make([]OpenAIToolCall, len(msg.ToolCalls))
		for i, toolCall := range msg.ToolCalls {
			openAIMsg.ToolCalls[i] = OpenAIToolCall{
				ID:   toolCall.ID,
				Type: toolCall.Type,
				Function: OpenAIFunction{
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
				},
			}
		}
	}

	if msg.ToolCallID != "" {
		openAIMsg.ToolCallID = msg.ToolCallID
	}

	if msg.Name != "" {
		openAIMsg.Name = msg.Name
	}

	return openAIMsg
}

func (c *OpenAIClient) convertContentPart(part domain.ContentPart) OpenAIContentPart {
	switch part.Type {
	case domain.ContentTypeText:
		return OpenAIContentPart{
			Type: "text",
			Text: part.Text,
		}
	case domain.ContentTypeImageURL:
		return OpenAIContentPart{
			Type: "image_url",
			ImageURL: &OpenAIImageURL{
				URL:    part.ImageURL.URL,
				Detail: part.ImageURL.Detail,
			},
		}
	default:
		return OpenAIContentPart{
			Type: "text",
			Text: part.Text,
		}
	}
}

func (c *OpenAIClient) convertCompletionResponse(resp *OpenAIChatCompletionResponse, requestID string, responseTime time.Duration) *types.CompletionResponse {
	choices := make([]domain.Choice, len(resp.Choices))
	for i, choice := range resp.Choices {
		choices[i] = domain.Choice{
			Index:        choice.Index,
			Message:      c.convertResponseMessage(choice.Message),
			FinishReason: domain.FinishReason(choice.FinishReason),
		}
	}

	// Calculate cost based on usage
	cost := c.calculateCost(resp.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	return &types.CompletionResponse{
		ID:       resp.ID,
		Object:   resp.Object,
		Created:  resp.Created,
		Model:    resp.Model,
		Provider: domain.ProviderOpenAI,
		Choices:  choices,
		Usage: domain.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
			CostUSD:          cost,
		},
		ResponseTime: responseTime,
		RequestID:    requestID,
	}
}

func (c *OpenAIClient) convertResponseMessage(msg OpenAIMessage) domain.Message {
	message := domain.Message{
		Role: domain.MessageRole(msg.Role),
	}

	// Convert content
	if msg.Content != "" {
		message.Content = []domain.ContentPart{
			{
				Type: domain.ContentTypeText,
				Text: msg.Content,
			},
		}
	}

	// Convert tool calls
	if len(msg.ToolCalls) > 0 {
		message.ToolCalls = make([]domain.ToolCall, len(msg.ToolCalls))
		for i, toolCall := range msg.ToolCalls {
			message.ToolCalls[i] = domain.ToolCall{
				ID:   toolCall.ID,
				Type: toolCall.Type,
				Function: domain.FunctionCall{
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
				},
			}
		}
	}

	return message
}

func (c *OpenAIClient) convertStreamChunk(chunk *OpenAIChatCompletionChunk, requestID string) types.StreamResponse {
	choices := make([]types.StreamChoice, len(chunk.Choices))
	for i, choice := range chunk.Choices {
		streamChoice := types.StreamChoice{
			Index: choice.Index,
			Delta: types.StreamDelta{},
		}

		if choice.Delta.Role != "" {
			role := domain.MessageRole(choice.Delta.Role)
			streamChoice.Delta.Role = &role
		}

		if choice.Delta.Content != "" {
			streamChoice.Delta.Content = &choice.Delta.Content
		}

		if choice.FinishReason != "" {
			reason := domain.FinishReason(choice.FinishReason)
			streamChoice.FinishReason = &reason
		}

		choices[i] = streamChoice
	}

	return types.StreamResponse{
		ID:        chunk.ID,
		Object:    chunk.Object,
		Created:   chunk.Created,
		Model:     chunk.Model,
		Provider:  domain.ProviderOpenAI,
		Choices:   choices,
		Done:      false,
		RequestID: requestID,
	}
}

func (c *OpenAIClient) convertEmbeddingRequest(req *types.EmbeddingRequest) *OpenAIEmbeddingRequest {
	openAIReq := &OpenAIEmbeddingRequest{
		Model: req.Model,
		Input: req.Input,
	}

	// Set default model if not specified
	if openAIReq.Model == "" {
		openAIReq.Model = "text-embedding-ada-002"
	}

	if req.EncodingFormat != "" {
		openAIReq.EncodingFormat = req.EncodingFormat
	}

	if req.Dimensions != nil {
		openAIReq.Dimensions = req.Dimensions
	}

	if req.User != "" {
		openAIReq.User = req.User
	}

	return openAIReq
}

func (c *OpenAIClient) convertEmbeddingResponse(resp *OpenAIEmbeddingResponse, requestID string, responseTime time.Duration) *types.EmbeddingResponse {
	embeddings := make([]domain.Embedding, len(resp.Data))
	for i, emb := range resp.Data {
		embeddings[i] = domain.Embedding{
			Object:    emb.Object,
			Embedding: emb.Embedding,
			Index:     emb.Index,
		}
	}

	// Calculate cost for embeddings
	cost := c.calculateEmbeddingCost(resp.Model, resp.Usage.TotalTokens)

	return &types.EmbeddingResponse{
		Object:   resp.Object,
		Data:     embeddings,
		Model:    resp.Model,
		Provider: domain.ProviderOpenAI,
		Usage: domain.EmbeddingUsage{
			PromptTokens: resp.Usage.PromptTokens,
			TotalTokens:  resp.Usage.TotalTokens,
			CostUSD:      cost,
		},
		ResponseTime: responseTime,
		RequestID:    requestID,
	}
}

func (c *OpenAIClient) convertModel(openAIModel *OpenAIModel) types.Model {
	// Determine capabilities based on model ID
	capabilities := c.getModelCapabilities(openAIModel.ID)

	// Get pricing information
	pricing := c.getModelPricing(openAIModel.ID)

	return types.Model{
		ID:            openAIModel.ID,
		Provider:      domain.ProviderOpenAI,
		Name:          openAIModel.ID, // OpenAI uses ID as display name
		Description:   fmt.Sprintf("OpenAI %s model", openAIModel.ID),
		Capabilities:  capabilities,
		ContextLength: c.getModelContextLength(openAIModel.ID),
		Pricing:       pricing,
		Status:        domain.ModelStatusAvailable,
		ProviderData: map[string]interface{}{
			"created":   openAIModel.Created,
			"owned_by":  openAIModel.OwnedBy,
			"root":      openAIModel.Root,
			"parent":    openAIModel.Parent,
			"object":    openAIModel.Object,
		},
	}
}

func (c *OpenAIClient) getModelCapabilities(modelID string) []domain.Capability {
	capabilities := []domain.Capability{}

	// Determine capabilities based on model name patterns
	if strings.Contains(modelID, "gpt") {
		capabilities = append(capabilities, domain.CapabilityCompletion)
		if strings.Contains(modelID, "gpt-4") {
			capabilities = append(capabilities, domain.CapabilityVision)
		}
		capabilities = append(capabilities, domain.CapabilityFunctionCalling)
	}

	if strings.Contains(modelID, "code") || strings.Contains(modelID, "codex") {
		capabilities = append(capabilities, domain.CapabilityCode)
		capabilities = append(capabilities, domain.CapabilityCompletion)
	}

	if strings.Contains(modelID, "embedding") {
		capabilities = append(capabilities, domain.CapabilityEmbedding)
	}

	// Default to completion if no specific capabilities detected
	if len(capabilities) == 0 {
		capabilities = append(capabilities, domain.CapabilityCompletion)
	}

	return capabilities
}

func (c *OpenAIClient) getModelPricing(modelID string) domain.ModelPricing {
	// Simplified pricing based on known OpenAI models (as of knowledge cutoff)
	// In production, this should be loaded from a configuration file or API
	pricingMap := map[string]domain.ModelPricing{
		"gpt-4": {
			InputTokenCost:  0.03 / 1000,  // $0.03 per 1k tokens
			OutputTokenCost: 0.06 / 1000,  // $0.06 per 1k tokens
			Unit:            "token",
		},
		"gpt-4-turbo": {
			InputTokenCost:  0.01 / 1000,  // $0.01 per 1k tokens
			OutputTokenCost: 0.03 / 1000,  // $0.03 per 1k tokens
			Unit:            "token",
		},
		"gpt-3.5-turbo": {
			InputTokenCost:  0.0015 / 1000, // $0.0015 per 1k tokens
			OutputTokenCost: 0.002 / 1000,  // $0.002 per 1k tokens
			Unit:            "token",
		},
		"text-embedding-ada-002": {
			InputTokenCost:  0.0001 / 1000, // $0.0001 per 1k tokens
			OutputTokenCost: 0,
			Unit:            "token",
		},
		"text-embedding-3-small": {
			InputTokenCost:  0.00002 / 1000, // $0.00002 per 1k tokens
			OutputTokenCost: 0,
			Unit:            "token",
		},
		"text-embedding-3-large": {
			InputTokenCost:  0.00013 / 1000, // $0.00013 per 1k tokens
			OutputTokenCost: 0,
			Unit:            "token",
		},
	}

	// Check for exact match first
	if pricing, exists := pricingMap[modelID]; exists {
		return pricing
	}

	// Check for partial matches
	for model, pricing := range pricingMap {
		if strings.Contains(modelID, model) {
			return pricing
		}
	}

	// Default pricing if model not found
	return domain.ModelPricing{
		InputTokenCost:  0.002 / 1000,
		OutputTokenCost: 0.002 / 1000,
		Unit:            "token",
	}
}

func (c *OpenAIClient) getModelContextLength(modelID string) int {
	contextLengths := map[string]int{
		"gpt-4":              8192,
		"gpt-4-turbo":        128000,
		"gpt-4-32k":          32768,
		"gpt-3.5-turbo":      4096,
		"gpt-3.5-turbo-16k":  16384,
		"text-davinci-003":   4097,
		"text-davinci-002":   4097,
		"code-davinci-002":   8001,
	}

	// Check for exact match first
	if length, exists := contextLengths[modelID]; exists {
		return length
	}

	// Check for partial matches
	for model, length := range contextLengths {
		if strings.Contains(modelID, model) {
			return length
		}
	}

	// Default context length
	return 4096
}

func (c *OpenAIClient) calculateCost(model string, promptTokens, completionTokens int) float64 {
	pricing := c.getModelPricing(model)
	return float64(promptTokens)*pricing.InputTokenCost + float64(completionTokens)*pricing.OutputTokenCost
}

func (c *OpenAIClient) calculateEmbeddingCost(model string, totalTokens int) float64 {
	pricing := c.getModelPricing(model)
	return float64(totalTokens) * pricing.InputTokenCost
}

// OpenAI API types

type OpenAIChatCompletionRequest struct {
	Model            string         `json:"model"`
	Messages         []OpenAIMessage `json:"messages"`
	MaxTokens        *int           `json:"max_tokens,omitempty"`
	Temperature      *float64       `json:"temperature,omitempty"`
	TopP             *float64       `json:"top_p,omitempty"`
	Stream           bool           `json:"stream,omitempty"`
	Stop             []string       `json:"stop,omitempty"`
	PresencePenalty  *float64       `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64       `json:"frequency_penalty,omitempty"`
	User             string         `json:"user,omitempty"`
}

type OpenAIMessage struct {
	Role         string              `json:"role"`
	Content      string              `json:"content,omitempty"`
	MultiContent []OpenAIContentPart `json:"content,omitempty"`
	Name         string              `json:"name,omitempty"`
	ToolCallID   string              `json:"tool_call_id,omitempty"`
	ToolCalls    []OpenAIToolCall    `json:"tool_calls,omitempty"`
}

type OpenAIContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *OpenAIImageURL `json:"image_url,omitempty"`
}

type OpenAIImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type OpenAIToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function OpenAIFunction `json:"function"`
}

type OpenAIFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type OpenAIChatCompletionResponse struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []OpenAIChoice       `json:"choices"`
	Usage   OpenAIUsage          `json:"usage"`
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenAIChatCompletionChunk struct {
	ID      string                     `json:"id"`
	Object  string                     `json:"object"`
	Created int64                      `json:"created"`
	Model   string                     `json:"model"`
	Choices []OpenAIStreamChoice       `json:"choices"`
}

type OpenAIStreamChoice struct {
	Index        int                `json:"index"`
	Delta        OpenAIStreamDelta  `json:"delta"`
	FinishReason string             `json:"finish_reason"`
}

type OpenAIStreamDelta struct {
	Role      string             `json:"role,omitempty"`
	Content   string             `json:"content,omitempty"`
	ToolCalls []OpenAIToolCall   `json:"tool_calls,omitempty"`
}

type OpenAIEmbeddingRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	EncodingFormat string   `json:"encoding_format,omitempty"`
	Dimensions     *int     `json:"dimensions,omitempty"`
	User           string   `json:"user,omitempty"`
}

type OpenAIEmbeddingResponse struct {
	Object string             `json:"object"`
	Data   []OpenAIEmbedding  `json:"data"`
	Model  string             `json:"model"`
	Usage  OpenAIEmbeddingUsage `json:"usage"`
}

type OpenAIEmbedding struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type OpenAIEmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type OpenAIModelsResponse struct {
	Object string        `json:"object"`
	Data   []OpenAIModel `json:"data"`
}

type OpenAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
	Root    string `json:"root,omitempty"`
	Parent  string `json:"parent,omitempty"`
}

type OpenAIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}