package domain

import (
	"time"

	"github.com/quantum-suite/platform/pkg/shared/errors"
)

// CompletionRequest represents a request for text completion
type CompletionRequest struct {
	TenantID         TenantID            `json:"tenant_id"`
	UserID           UserID              `json:"user_id"`
	Provider         Provider            `json:"provider,omitempty"`
	Model            string              `json:"model"`
	Messages         []Message           `json:"messages"`
	MaxTokens        *int                `json:"max_tokens,omitempty"`
	Temperature      *float64            `json:"temperature,omitempty"`
	TopP             *float64            `json:"top_p,omitempty"`
	Stream           bool                `json:"stream"`
	Stop             []string            `json:"stop,omitempty"`
	PresencePenalty  *float64            `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64            `json:"frequency_penalty,omitempty"`
	User             string              `json:"user,omitempty"`
	RequestID        string              `json:"request_id"`
	Priority         Priority            `json:"priority"`
	CacheEnabled     bool                `json:"cache_enabled"`
	CacheTTL         time.Duration       `json:"cache_ttl"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// CompletionResponse represents a completion response
type CompletionResponse struct {
	ID       string                  `json:"id"`
	Object   string                  `json:"object"`
	Created  int64                   `json:"created"`
	Model    string                  `json:"model"`
	Provider Provider                `json:"provider"`
	Choices  []Choice                `json:"choices"`
	Usage    Usage                   `json:"usage"`
	Metadata map[string]interface{}  `json:"metadata,omitempty"`
}

// StreamResponse represents a streaming response chunk
type StreamResponse struct {
	ID       string                  `json:"id,omitempty"`
	Object   string                  `json:"object,omitempty"`
	Created  int64                   `json:"created,omitempty"`
	Model    string                  `json:"model,omitempty"`
	Provider Provider                `json:"provider,omitempty"`
	Choices  []Choice                `json:"choices,omitempty"`
	Done     bool                    `json:"done,omitempty"`
	Error    *errors.QLensError      `json:"error,omitempty"`
}

// Note: EmbeddingRequest and EmbeddingResponse are already defined in qlens.go

// ListModelsOptions represents options for listing models
type ListModelsOptions struct {
	Provider   Provider   `json:"provider,omitempty"`
	Capability Capability `json:"capability,omitempty"`
}

// ModelsResponse represents a models list response
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status    string                             `json:"status"`
	Timestamp time.Time                          `json:"timestamp"`
	Services  map[string]ServiceHealth           `json:"services"`
	Providers map[string]ProviderHealth          `json:"providers"`
}

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	Status  string `json:"status"`
	Latency int64  `json:"latency_ms"`
}

// ProviderHealth represents the health status of a provider
type ProviderHealth struct {
	Status    string  `json:"status"`
	Latency   int64   `json:"latency_ms"`
	ErrorRate float64 `json:"error_rate"`
}