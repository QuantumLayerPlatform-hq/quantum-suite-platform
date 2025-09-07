package types

import (
	"context"
	"fmt"
	"time"

	"github.com/quantum-suite/platform/internal/domain"
)

// Request and Response types that align with our domain models

// CompletionRequest represents a request for text completion
type CompletionRequest struct {
	// Core fields
	Model            string                     `json:"model,omitempty"`
	Provider         domain.Provider            `json:"provider,omitempty"`
	Messages         []domain.Message           `json:"messages"`
	MaxTokens        *int                       `json:"max_tokens,omitempty"`
	Temperature      *float64                   `json:"temperature,omitempty"`
	TopP             *float64                   `json:"top_p,omitempty"`
	Stream           bool                       `json:"stream,omitempty"`
	Stop             []string                   `json:"stop,omitempty"`
	PresencePenalty  *float64                   `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64                   `json:"frequency_penalty,omitempty"`
	User             string                     `json:"user,omitempty"`

	// Quantum Suite specific fields
	TenantID    domain.TenantID            `json:"tenant_id"`
	UserID      domain.UserID              `json:"user_id"`
	RequestID   string                     `json:"request_id,omitempty"`
	Metadata    map[string]interface{}     `json:"metadata,omitempty"`

	// Caching options
	CacheEnabled bool          `json:"cache_enabled,omitempty"`
	CacheTTL     time.Duration `json:"cache_ttl,omitempty"`

	// Rate limiting
	Priority domain.Priority `json:"priority,omitempty"`
}

// CompletionResponse represents a completion response
type CompletionResponse struct {
	ID       string                  `json:"id"`
	Object   string                  `json:"object"`
	Created  int64                   `json:"created"`
	Model    string                  `json:"model"`
	Provider domain.Provider         `json:"provider"`
	Choices  []domain.Choice         `json:"choices"`
	Usage    domain.Usage            `json:"usage"`
	Metadata map[string]interface{}  `json:"metadata,omitempty"`

	// Performance metrics
	ResponseTime time.Duration `json:"response_time"`
	CacheHit     bool          `json:"cache_hit"`
	RequestID    string        `json:"request_id,omitempty"`
}

// StreamResponse represents a streaming completion response chunk
type StreamResponse struct {
	ID       string                 `json:"id"`
	Object   string                 `json:"object"`
	Created  int64                  `json:"created"`
	Model    string                 `json:"model"`
	Provider domain.Provider        `json:"provider"`
	Choices  []StreamChoice         `json:"choices"`
	Done     bool                   `json:"done"`
	Error    *StreamError           `json:"error,omitempty"`

	// Performance metrics
	RequestID string `json:"request_id,omitempty"`
}

// StreamChoice represents a choice in a streaming response
type StreamChoice struct {
	Index        int              `json:"index"`
	Delta        StreamDelta      `json:"delta"`
	FinishReason *domain.FinishReason `json:"finish_reason,omitempty"`
}

// StreamDelta represents the incremental content in a stream
type StreamDelta struct {
	Role      *domain.MessageRole `json:"role,omitempty"`
	Content   *string             `json:"content,omitempty"`
	ToolCalls []domain.ToolCall   `json:"tool_calls,omitempty"`
}

// StreamError represents an error in streaming
type StreamError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// EmbeddingRequest represents a request for embeddings
type EmbeddingRequest struct {
	// Core fields
	Model          string             `json:"model,omitempty"`
	Provider       domain.Provider    `json:"provider,omitempty"`
	Input          []string           `json:"input"`
	EncodingFormat string             `json:"encoding_format,omitempty"`
	Dimensions     *int               `json:"dimensions,omitempty"`
	User           string             `json:"user,omitempty"`

	// Quantum Suite specific fields
	TenantID  domain.TenantID        `json:"tenant_id"`
	UserID    domain.UserID          `json:"user_id"`
	RequestID string                 `json:"request_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`

	// Processing options
	BatchSize int               `json:"batch_size,omitempty"`
	Priority  domain.Priority   `json:"priority,omitempty"`
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Object   string                    `json:"object"`
	Data     []domain.Embedding        `json:"data"`
	Model    string                    `json:"model"`
	Provider domain.Provider           `json:"provider"`
	Usage    domain.EmbeddingUsage     `json:"usage"`

	// Performance metrics
	ResponseTime time.Duration `json:"response_time"`
	RequestID    string        `json:"request_id,omitempty"`
}

// Model represents a model available through a provider
type Model struct {
	ID            string               `json:"id"`
	Provider      domain.Provider      `json:"provider"`
	Name          string               `json:"name"`
	Description   string               `json:"description"`
	Capabilities  []domain.Capability  `json:"capabilities"`
	ContextLength int                  `json:"context_length"`
	Pricing       domain.ModelPricing  `json:"pricing"`
	Status        domain.ModelStatus   `json:"status"`

	// Provider specific metadata
	ProviderData map[string]interface{} `json:"provider_data,omitempty"`
}

// ModelsResponse represents a list of models
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// ListModelsOptions represents options for listing models
type ListModelsOptions struct {
	Provider   domain.Provider     `json:"provider,omitempty"`
	Capability domain.Capability   `json:"capability,omitempty"`
	Status     domain.ModelStatus  `json:"status,omitempty"`
}

// HealthResponse represents the health status of QLens
type HealthResponse struct {
	Status    string                        `json:"status"`
	Timestamp time.Time                     `json:"timestamp"`
	Version   string                        `json:"version"`
	Uptime    time.Duration                 `json:"uptime"`
	Providers map[domain.Provider]ProviderHealth `json:"providers"`
}

// ProviderHealth represents the health status of a provider
type ProviderHealth struct {
	Status        domain.ProviderHealthStatus `json:"status"`
	LatencyMS     float64                     `json:"latency_ms"`
	ErrorRate     float64                     `json:"error_rate"`
	LastCheck     time.Time                   `json:"last_check"`
	HealthMessage string                      `json:"health_message,omitempty"`
}

// ProviderConfig represents configuration for a provider
type ProviderConfig struct {
	Provider  domain.Provider        `json:"provider"`
	APIKey    string                 `json:"api_key,omitempty"`
	BaseURL   string                 `json:"base_url,omitempty"`
	Timeout   time.Duration          `json:"timeout,omitempty"`
	RateLimit domain.RateLimitConfig `json:"rate_limit"`
	Enabled   bool                   `json:"enabled"`
	Priority  int                    `json:"priority"`
	Config    map[string]interface{} `json:"config,omitempty"`
}

// ClientConfig represents configuration for the QLens client
type ClientConfig struct {
	// Provider configurations
	Providers map[domain.Provider]ProviderConfig `json:"providers"`

	// Default behavior
	DefaultProvider    domain.Provider `json:"default_provider,omitempty"`
	AutoFailover      bool            `json:"auto_failover"`
	LoadBalancing     bool            `json:"load_balancing"`

	// Caching
	CacheEnabled      bool          `json:"cache_enabled"`
	CacheDefaultTTL   time.Duration `json:"cache_default_ttl"`
	CacheMaxSize      int           `json:"cache_max_size"`

	// Rate limiting
	GlobalRateLimit   domain.RateLimitConfig `json:"global_rate_limit"`

	// Observability
	MetricsEnabled    bool   `json:"metrics_enabled"`
	TracingEnabled    bool   `json:"tracing_enabled"`
	LogLevel          string `json:"log_level"`

	// Timeouts
	DefaultTimeout    time.Duration `json:"default_timeout"`
	StreamTimeout     time.Duration `json:"stream_timeout"`

	// Retries
	MaxRetries        int           `json:"max_retries"`
	RetryBackoff      time.Duration `json:"retry_backoff"`
	RetryableErrors   []string      `json:"retryable_errors"`
}

// QLensError represents an error from QLens
type QLensError struct {
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Code      string                 `json:"code,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Provider  domain.Provider        `json:"provider,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
}

func (e *QLensError) Error() string {
	if e.Provider != "" {
		return fmt.Sprintf("QLens [%s]: %s", e.Provider, e.Message)
	}
	return fmt.Sprintf("QLens: %s", e.Message)
}

// Error type constants
const (
	ErrorTypeInvalidRequest     = "invalid_request"
	ErrorTypeAuthenticationError = "authentication_error"
	ErrorTypeAuthorizationError  = "authorization_error"
	ErrorTypeRateLimitExceeded   = "rate_limit_exceeded"
	ErrorTypeProviderError       = "provider_error"
	ErrorTypeInternalError       = "internal_error"
	ErrorTypeTimeout            = "timeout"
	ErrorTypeProviderUnavailable = "provider_unavailable"
	ErrorTypeCacheError         = "cache_error"
)

// Configuration types
type RouterConfig struct {
	Strategy       string                 `json:"strategy"`
	HealthCheck    bool                   `json:"health_check"`
	LoadBalancing  bool                   `json:"load_balancing"`
	FailoverDelay  time.Duration          `json:"failover_delay"`
	Config         map[string]interface{} `json:"config,omitempty"`
}

type CacheConfig struct {
	Type           string                 `json:"type"`
	TTL            time.Duration          `json:"ttl"`
	MaxSize        int                    `json:"max_size"`
	CleanupInterval time.Duration         `json:"cleanup_interval"`
	Config         map[string]interface{} `json:"config,omitempty"`
}

// ProviderClient represents an interface for individual LLM providers
type ProviderClient interface {
	// Provider identification
	Provider() domain.Provider
	Name() string

	// Core operations
	CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
	CreateCompletionStream(ctx context.Context, req *CompletionRequest) (<-chan StreamResponse, error)
	CreateEmbeddings(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)

	// Model operations
	ListModels(ctx context.Context) ([]Model, error)
	GetModel(ctx context.Context, modelID string) (*Model, error)

	// Health and configuration
	HealthCheck(ctx context.Context) error
	Configure(config ProviderConfig) error
	GetConfig() ProviderConfig

	// Close resources
	Close() error
}

// CacheStats represents cache statistics
type CacheStats struct {
	Size      int     `json:"size"`
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	HitRate   float64 `json:"hit_rate"`
	Evictions int64   `json:"evictions"`
}