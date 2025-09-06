package domain

import (
	"time"
)

// QLens Domain Models and Entities

// Provider types
type Provider string

const (
	ProviderOpenAI       Provider = "openai"
	ProviderAzureOpenAI  Provider = "azure-openai"
	ProviderAnthropic    Provider = "anthropic"
	ProviderAWSBedrock   Provider = "aws-bedrock"
	ProviderLocal        Provider = "local"
	ProviderAuto         Provider = "auto"
)

// Model capabilities
type Capability string

const (
	CapabilityCompletion     Capability = "completion"
	CapabilityEmbedding      Capability = "embedding"
	CapabilityVision         Capability = "vision"
	CapabilityCode           Capability = "code"
	CapabilityFunctionCalling Capability = "function_calling"
)

// Content types for messages
type ContentType string

const (
	ContentTypeText     ContentType = "text"
	ContentTypeImageURL ContentType = "image_url"
)

// Message roles
type MessageRole string

const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleTool      MessageRole = "tool"
)

// Finish reasons
type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonToolCalls     FinishReason = "tool_calls"
	FinishReasonContentFilter FinishReason = "content_filter"
	FinishReasonFunctionCall  FinishReason = "function_call"
)

// Provider health status
type ProviderHealthStatus string

const (
	ProviderHealthHealthy   ProviderHealthStatus = "healthy"
	ProviderHealthDegraded  ProviderHealthStatus = "degraded"
	ProviderHealthUnhealthy ProviderHealthStatus = "unhealthy"
)

// Model status
type ModelStatus string

const (
	ModelStatusAvailable  ModelStatus = "available"
	ModelStatusDeprecated ModelStatus = "deprecated"
	ModelStatusLimited    ModelStatus = "limited"
)

// Template variable types
type VariableType string

const (
	VariableTypeString  VariableType = "string"
	VariableTypeNumber  VariableType = "number"
	VariableTypeBoolean VariableType = "boolean"
	VariableTypeArray   VariableType = "array"
	VariableTypeObject  VariableType = "object"
)

// Core Domain Entities

// LLMRequest represents a completion request aggregate
type LLMRequest struct {
	BaseAggregateRoot
	TenantID          TenantID                   `json:"tenant_id"`
	UserID            UserID                     `json:"user_id"`
	Provider          Provider                   `json:"provider"`
	Model             string                     `json:"model"`
	Messages          []Message                  `json:"messages"`
	MaxTokens         *int                       `json:"max_tokens,omitempty"`
	Temperature       *float64                   `json:"temperature,omitempty"`
	TopP              *float64                   `json:"top_p,omitempty"`
	Stream            bool                       `json:"stream"`
	Stop              []string                   `json:"stop,omitempty"`
	PresencePenalty   *float64                   `json:"presence_penalty,omitempty"`
	FrequencyPenalty  *float64                   `json:"frequency_penalty,omitempty"`
	User              string                     `json:"user,omitempty"`
	Metadata          map[string]interface{}     `json:"metadata,omitempty"`
	Status            RequestStatus              `json:"status"`
	SubmittedAt       time.Time                  `json:"submitted_at"`
	CompletedAt       *time.Time                 `json:"completed_at,omitempty"`
	Usage             *Usage                     `json:"usage,omitempty"`
	Response          *LLMResponse               `json:"response,omitempty"`
	Error             *RequestError              `json:"error,omitempty"`
}

type RequestStatus string

const (
	RequestStatusPending   RequestStatus = "pending"
	RequestStatusProcessing RequestStatus = "processing"
	RequestStatusCompleted  RequestStatus = "completed"
	RequestStatusFailed     RequestStatus = "failed"
	RequestStatusCancelled  RequestStatus = "cancelled"
)

// Message represents a single message in a conversation
type Message struct {
	Role       MessageRole   `json:"role"`
	Content    []ContentPart `json:"content"`
	Name       string        `json:"name,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall    `json:"tool_calls,omitempty"`
}

// ContentPart represents a part of message content
type ContentPart struct {
	Type     ContentType `json:"type"`
	Text     string      `json:"text,omitempty"`
	ImageURL *ImageURL   `json:"image_url,omitempty"`
}

// ImageURL represents an image URL in message content
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// ToolCall represents a function call in a message
type ToolCall struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// LLMResponse represents the response from an LLM
type LLMResponse struct {
	ID       string    `json:"id"`
	Object   string    `json:"object"`
	Created  int64     `json:"created"`
	Model    string    `json:"model"`
	Provider Provider  `json:"provider"`
	Choices  []Choice  `json:"choices"`
	Usage    Usage     `json:"usage"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int           `json:"index"`
	Message      Message       `json:"message"`
	FinishReason FinishReason  `json:"finish_reason"`
	LogProbs     interface{}   `json:"logprobs,omitempty"`
}

// Usage represents token usage statistics
type Usage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	CostUSD          float64 `json:"cost_usd,omitempty"`
	CacheHit         bool    `json:"cache_hit,omitempty"`
}

// RequestError represents an error in processing a request
type RequestError struct {
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Code      string                 `json:"code,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// EmbeddingRequest represents an embedding request
type EmbeddingRequest struct {
	BaseAggregateRoot
	TenantID        TenantID    `json:"tenant_id"`
	UserID          UserID      `json:"user_id"`
	Provider        Provider    `json:"provider"`
	Model           string      `json:"model"`
	Input           []string    `json:"input"`
	EncodingFormat  string      `json:"encoding_format,omitempty"`
	Dimensions      *int        `json:"dimensions,omitempty"`
	User            string      `json:"user,omitempty"`
	Status          RequestStatus `json:"status"`
	SubmittedAt     time.Time   `json:"submitted_at"`
	CompletedAt     *time.Time  `json:"completed_at,omitempty"`
	Usage           *EmbeddingUsage `json:"usage,omitempty"`
	Response        *EmbeddingResponse `json:"response,omitempty"`
	Error           *RequestError `json:"error,omitempty"`
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Object   string      `json:"object"`
	Data     []Embedding `json:"data"`
	Model    string      `json:"model"`
	Provider Provider    `json:"provider"`
	Usage    EmbeddingUsage `json:"usage"`
}

// Embedding represents a single embedding
type Embedding struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

// EmbeddingUsage represents embedding token usage
type EmbeddingUsage struct {
	PromptTokens int     `json:"prompt_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	CostUSD      float64 `json:"cost_usd,omitempty"`
}

// Model represents an available LLM model
type Model struct {
	BaseEntity
	ModelID      string       `json:"id"`
	Provider     Provider     `json:"provider"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Capabilities []Capability `json:"capabilities"`
	ContextLength int         `json:"context_length"`
	Pricing      ModelPricing `json:"pricing"`
	Status       ModelStatus  `json:"status"`
	IsActive     bool         `json:"is_active"`
}

// ModelPricing represents model pricing information
type ModelPricing struct {
	InputTokenCost  float64 `json:"input_token_cost"`
	OutputTokenCost float64 `json:"output_token_cost"`
	Unit           string  `json:"unit"`
}

// PromptTemplate represents a reusable prompt template
type PromptTemplate struct {
	BaseAggregateRoot
	TenantID    TenantID            `json:"tenant_id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Category    string              `json:"category"`
	Tags        []string            `json:"tags"`
	Content     string              `json:"content"`
	Variables   []TemplateVariable  `json:"variables"`
	CreatedBy   UserID              `json:"created_by"`
	IsPublic    bool                `json:"is_public"`
	UsageCount  int                 `json:"usage_count"`
}

// TemplateVariable represents a variable in a prompt template
type TemplateVariable struct {
	Name         string      `json:"name"`
	Type         VariableType `json:"type"`
	Description  string      `json:"description"`
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
}

// ProviderConfig represents configuration for an LLM provider
type ProviderConfig struct {
	BaseEntity
	Provider     Provider               `json:"provider"`
	TenantID     TenantID               `json:"tenant_id"`
	Enabled      bool                   `json:"enabled"`
	Priority     int                    `json:"priority"`
	Config       map[string]interface{} `json:"config"`
	RateLimit    RateLimitConfig        `json:"rate_limit"`
	LastHealthCheck time.Time           `json:"last_health_check"`
	HealthStatus ProviderHealthStatus   `json:"health_status"`
	Latency      float64               `json:"latency_ms"`
	ErrorRate    float64               `json:"error_rate"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
	RequestsPerDay    int `json:"requests_per_day"`
	TokensPerDay      int `json:"tokens_per_day"`
}

// UsageStatistics represents usage statistics for a tenant
type UsageStatistics struct {
	BaseEntity
	TenantID      TenantID               `json:"tenant_id"`
	Period        UsagePeriod            `json:"period"`
	TotalRequests int                    `json:"total_requests"`
	TotalTokens   int                    `json:"total_tokens"`
	TotalCost     float64                `json:"total_cost"`
	ByProvider    map[Provider]ProviderUsage `json:"by_provider"`
	ByModel       map[string]ModelUsage   `json:"by_model"`
}

// UsagePeriod represents a time period for usage statistics
type UsagePeriod struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// ProviderUsage represents usage statistics for a specific provider
type ProviderUsage struct {
	Requests           int     `json:"requests"`
	Tokens             int     `json:"tokens"`
	Cost               float64 `json:"cost"`
	AvgResponseTime    float64 `json:"avg_response_time"`
	ErrorRate          float64 `json:"error_rate"`
}

// ModelUsage represents usage statistics for a specific model
type ModelUsage struct {
	Requests              int     `json:"requests"`
	Tokens                int     `json:"tokens"`
	Cost                  float64 `json:"cost"`
	AvgTokensPerRequest   float64 `json:"avg_tokens_per_request"`
}

// CacheEntry represents a cached response
type CacheEntry struct {
	BaseEntity
	Key          string                 `json:"key"`
	TenantID     TenantID               `json:"tenant_id"`
	RequestHash  string                 `json:"request_hash"`
	Response     LLMResponse            `json:"response"`
	TTL          time.Duration          `json:"ttl"`
	ExpiresAt    time.Time              `json:"expires_at"`
	HitCount     int                    `json:"hit_count"`
	LastAccessed time.Time              `json:"last_accessed"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Factory methods for creating new entities

func NewLLMRequest(tenantID TenantID, userID UserID) *LLMRequest {
	return &LLMRequest{
		BaseAggregateRoot: NewBaseAggregateRoot(),
		TenantID:         tenantID,
		UserID:           userID,
		Status:           RequestStatusPending,
		SubmittedAt:      time.Now(),
		Messages:         make([]Message, 0),
		Metadata:         make(map[string]interface{}),
	}
}

func NewEmbeddingRequest(tenantID TenantID, userID UserID) *EmbeddingRequest {
	return &EmbeddingRequest{
		BaseAggregateRoot: NewBaseAggregateRoot(),
		TenantID:         tenantID,
		UserID:           userID,
		Status:           RequestStatusPending,
		SubmittedAt:      time.Now(),
		Input:            make([]string, 0),
	}
}

func NewPromptTemplate(tenantID TenantID, createdBy UserID, name, content string) *PromptTemplate {
	return &PromptTemplate{
		BaseAggregateRoot: NewBaseAggregateRoot(),
		TenantID:         tenantID,
		CreatedBy:        createdBy,
		Name:             name,
		Content:          content,
		Variables:        make([]TemplateVariable, 0),
		Tags:             make([]string, 0),
		IsPublic:         false,
		UsageCount:       0,
	}
}

func NewProviderConfig(provider Provider, tenantID TenantID) *ProviderConfig {
	return &ProviderConfig{
		BaseEntity:      NewBaseEntity(),
		Provider:        provider,
		TenantID:        tenantID,
		Enabled:         true,
		Priority:        1,
		Config:          make(map[string]interface{}),
		LastHealthCheck: time.Now(),
		HealthStatus:    ProviderHealthHealthy,
	}
}

func NewModel(modelID string, provider Provider, name string) *Model {
	return &Model{
		BaseEntity:    NewBaseEntity(),
		ModelID:       modelID,
		Provider:      provider,
		Name:          name,
		Capabilities:  make([]Capability, 0),
		Status:        ModelStatusAvailable,
		IsActive:      true,
	}
}

func NewCacheEntry(key string, tenantID TenantID, requestHash string, response LLMResponse, ttl time.Duration) *CacheEntry {
	return &CacheEntry{
		BaseEntity:   NewBaseEntity(),
		Key:          key,
		TenantID:     tenantID,
		RequestHash:  requestHash,
		Response:     response,
		TTL:          ttl,
		ExpiresAt:    time.Now().Add(ttl),
		HitCount:     0,
		LastAccessed: time.Now(),
		Metadata:     make(map[string]interface{}),
	}
}

// Helper methods

func (r *LLMRequest) SetCompleted(response LLMResponse, usage Usage) {
	now := time.Now()
	r.Status = RequestStatusCompleted
	r.CompletedAt = &now
	r.Response = &response
	r.Usage = &usage
	r.updatedAt = now
}

func (r *LLMRequest) SetFailed(err RequestError) {
	now := time.Now()
	r.Status = RequestStatusFailed
	r.CompletedAt = &now
	r.Error = &err
	r.updatedAt = now
}

func (r *EmbeddingRequest) SetCompleted(response EmbeddingResponse, usage EmbeddingUsage) {
	now := time.Now()
	r.Status = RequestStatusCompleted
	r.CompletedAt = &now
	r.Response = &response
	r.Usage = &usage
	r.updatedAt = now
}

func (r *EmbeddingRequest) SetFailed(err RequestError) {
	now := time.Now()
	r.Status = RequestStatusFailed
	r.CompletedAt = &now
	r.Error = &err
	r.updatedAt = now
}

func (t *PromptTemplate) IncrementUsage() {
	t.UsageCount++
	t.updatedAt = time.Now()
}

func (c *CacheEntry) Hit() {
	c.HitCount++
	c.LastAccessed = time.Now()
}

func (c *CacheEntry) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

func (pc *ProviderConfig) UpdateHealth(status ProviderHealthStatus, latency, errorRate float64) {
	pc.HealthStatus = status
	pc.Latency = latency
	pc.ErrorRate = errorRate
	pc.LastHealthCheck = time.Now()
	pc.updatedAt = time.Now()
}