package gateway


// Health check models
type HealthResponse struct {
	Status       string             `json:"status" example:"healthy"`
	Version      string             `json:"version" example:"1.0.0"`
	Timestamp    string             `json:"timestamp" example:"2025-09-06T19:00:00Z"`
	Dependencies []DependencyHealth `json:"dependencies,omitempty"`
} // @name HealthResponse

type DependencyHealth struct {
	Name   string `json:"name" example:"router"`
	Status string `json:"status" example:"healthy"`
} // @name DependencyHealth

// Error response models
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
} // @name ErrorResponse

type ErrorDetail struct {
	Message string `json:"message" example:"Invalid request"`
	Type    string `json:"type" example:"invalid_request_error"`
	Code    string `json:"code,omitempty" example:"invalid_json"`
} // @name ErrorDetail

// Models endpoint
type ModelsResponse struct {
	Object string  `json:"object" example:"list"`
	Data   []Model `json:"data"`
} // @name ModelsResponse

type Model struct {
	ID      string `json:"id" example:"gpt-4"`
	Object  string `json:"object" example:"model"`
	Created int64  `json:"created" example:"1677610602"`
	OwnedBy string `json:"owned_by" example:"azure-openai"`
} // @name Model

// Chat completion models
type ChatCompletionRequest struct {
	Model            string    `json:"model" binding:"required" example:"gpt-4"`
	Messages         []Message `json:"messages" binding:"required"`
	MaxTokens        int       `json:"max_tokens,omitempty" example:"100"`
	Temperature      float64   `json:"temperature,omitempty" example:"0.7"`
	TopP             float64   `json:"top_p,omitempty" example:"1.0"`
	N                int       `json:"n,omitempty" example:"1"`
	Stop             []string  `json:"stop,omitempty"`
	PresencePenalty  float64   `json:"presence_penalty,omitempty" example:"0.0"`
	FrequencyPenalty float64   `json:"frequency_penalty,omitempty" example:"0.0"`
	Stream           bool      `json:"stream,omitempty" example:"false"`
	User             string    `json:"user,omitempty" example:"user123"`
} // @name ChatCompletionRequest

type Message struct {
	Role    string `json:"role" example:"user" enums:"system,user,assistant"`
	Content string `json:"content" example:"Hello, how are you?"`
	Name    string `json:"name,omitempty" example:"assistant"`
} // @name Message

type ChatCompletionResponse struct {
	ID      string   `json:"id" example:"chatcmpl-123"`
	Object  string   `json:"object" example:"chat.completion"`
	Created int64    `json:"created" example:"1677652288"`
	Model   string   `json:"model" example:"gpt-4"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
} // @name ChatCompletionResponse

type Choice struct {
	Index        int     `json:"index" example:"0"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason" example:"stop"`
} // @name Choice

// Legacy completion models
type CompletionRequest struct {
	Model            string   `json:"model" binding:"required" example:"gpt-35-turbo"`
	Prompt           string   `json:"prompt" binding:"required" example:"Once upon a time"`
	MaxTokens        int      `json:"max_tokens,omitempty" example:"100"`
	Temperature      float64  `json:"temperature,omitempty" example:"0.7"`
	TopP             float64  `json:"top_p,omitempty" example:"1.0"`
	N                int      `json:"n,omitempty" example:"1"`
	Stream           bool     `json:"stream,omitempty" example:"false"`
	Stop             []string `json:"stop,omitempty"`
	PresencePenalty  float64  `json:"presence_penalty,omitempty" example:"0.0"`
	FrequencyPenalty float64  `json:"frequency_penalty,omitempty" example:"0.0"`
	User             string   `json:"user,omitempty" example:"user123"`
} // @name CompletionRequest

type CompletionResponse struct {
	ID      string             `json:"id" example:"cmpl-123"`
	Object  string             `json:"object" example:"text_completion"`
	Created int64              `json:"created" example:"1677652288"`
	Model   string             `json:"model" example:"gpt-35-turbo"`
	Choices []CompletionChoice `json:"choices"`
	Usage   Usage              `json:"usage"`
} // @name CompletionResponse

type CompletionChoice struct {
	Text         string `json:"text" example:" This is a test response."`
	Index        int    `json:"index" example:"0"`
	FinishReason string `json:"finish_reason" example:"stop"`
} // @name CompletionChoice

// Embedding models
type EmbeddingRequest struct {
	Input          []string `json:"input" binding:"required" example:"The food was delicious and the waiter..."`
	Model          string   `json:"model" binding:"required" example:"text-embedding-ada-002"`
	EncodingFormat string   `json:"encoding_format,omitempty" example:"float"`
	User           string   `json:"user,omitempty" example:"user123"`
} // @name EmbeddingRequest

type EmbeddingResponse struct {
	Object string            `json:"object" example:"list"`
	Data   []EmbeddingObject `json:"data"`
	Model  string            `json:"model" example:"text-embedding-ada-002"`
	Usage  EmbeddingUsage    `json:"usage"`
} // @name EmbeddingResponse

type EmbeddingObject struct {
	Object    string    `json:"object" example:"embedding"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index" example:"0"`
} // @name EmbeddingObject

type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens" example:"8"`
	TotalTokens  int `json:"total_tokens" example:"8"`
} // @name EmbeddingUsage

// Common models
type Usage struct {
	PromptTokens     int `json:"prompt_tokens" example:"9"`
	CompletionTokens int `json:"completion_tokens,omitempty" example:"12"`
	TotalTokens      int `json:"total_tokens" example:"21"`
} // @name Usage

// Usage analytics response models
type GlobalUsageStats struct {
	TotalCostToday    float64 `json:"total_cost_today" example:"12.45"`
	RequestCount      int64   `json:"request_count" example:"1247"`
	ActiveTenants     int     `json:"active_tenants" example:"42"`
	ActiveServices    int     `json:"active_services" example:"8"`
	BudgetUtilization float64 `json:"budget_utilization_percent" example:"62.3"`
	LastUpdated       string  `json:"last_updated" example:"2025-09-07T17:15:00Z"`
} // @name GlobalUsageStats

type TenantUsageStats struct {
	TenantID        string                     `json:"tenant_id" example:"tenant-123"`
	DailyCost       float64                    `json:"daily_cost" example:"5.67"`
	MonthlyCost     float64                    `json:"monthly_cost" example:"156.78"`
	RequestCount    int64                      `json:"request_count" example:"234"`
	ModelUsage      map[string]ModelUsageStats `json:"model_usage"`
	BudgetLimit     float64                    `json:"budget_limit" example:"50.0"`
	LastUpdated     string                     `json:"last_updated" example:"2025-09-07T17:15:00Z"`
} // @name TenantUsageStats

type ModelUsageStats struct {
	RequestCount    int64   `json:"request_count" example:"45"`
	TokensUsed      int64   `json:"tokens_used" example:"12450"`
	Cost            float64 `json:"cost" example:"2.34"`
	AvgLatency      float64 `json:"avg_latency_ms" example:"850.5"`
} // @name ModelUsageStats

type CostSummaryStats struct {
	DailyCost                 float64 `json:"daily_cost" example:"12.45"`
	RequestCount              int64   `json:"request_count" example:"1247"`
	ActiveTenants             int     `json:"active_tenants" example:"42"`
	ActiveServices            int     `json:"active_services" example:"8"`
	BudgetUtilizationPercent  float64 `json:"budget_utilization_percent" example:"62.3"`
	Status                    string  `json:"status" example:"healthy" enums:"healthy,warning,critical"`
	LastUpdated               string  `json:"last_updated" example:"2025-09-07T17:15:00Z"`
} // @name CostSummaryStats

// Metrics models
type MetricsResponse struct {
	RequestCount int64  `json:"request_count" example:"100"`
	ErrorCount   int64  `json:"error_count" example:"5"`
	Uptime       string `json:"uptime" example:"24h30m"`
	Version      string `json:"version" example:"1.0.0"`
} // @name MetricsResponse

// Streaming response models
type StreamResponse struct {
	ID      string        `json:"id" example:"chatcmpl-123"`
	Object  string        `json:"object" example:"chat.completion.chunk"`
	Created int64         `json:"created" example:"1677652288"`
	Model   string        `json:"model" example:"gpt-4"`
	Choices []StreamChoice `json:"choices"`
} // @name StreamResponse

type StreamChoice struct {
	Index        int           `json:"index" example:"0"`
	Delta        StreamMessage `json:"delta"`
	FinishReason *string       `json:"finish_reason"`
} // @name StreamChoice

type StreamMessage struct {
	Role    string `json:"role,omitempty" example:"assistant"`
	Content string `json:"content,omitempty" example:"Hello"`
} // @name StreamMessage