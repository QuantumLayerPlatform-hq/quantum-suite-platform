package qlens

import (
	"time"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/qlens-types"
)

// Interfaces are now in interfaces.go to avoid circular imports

// Types are now in types.go to avoid circular imports

// ClientOption represents a functional option for configuring the client
type ClientOption func(*types.ClientConfig)

// WithProvider adds a provider configuration
func WithProvider(provider domain.Provider, config types.ProviderConfig) ClientOption {
	return func(c *types.ClientConfig) {
		if c.Providers == nil {
			c.Providers = make(map[domain.Provider]types.ProviderConfig)
		}
		c.Providers[provider] = config
	}
}

// WithDefaultProvider sets the default provider
func WithDefaultProvider(provider domain.Provider) ClientOption {
	return func(c *types.ClientConfig) {
		c.DefaultProvider = provider
	}
}

// WithCaching enables caching with specified TTL
func WithCaching(enabled bool, ttl time.Duration) ClientOption {
	return func(c *types.ClientConfig) {
		c.CacheEnabled = enabled
		c.CacheDefaultTTL = ttl
	}
}

// WithTimeout sets the default timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *types.ClientConfig) {
		c.DefaultTimeout = timeout
	}
}

// WithRetries configures retry behavior
func WithRetries(maxRetries int, backoff time.Duration) ClientOption {
	return func(c *types.ClientConfig) {
		c.MaxRetries = maxRetries
		c.RetryBackoff = backoff
	}
}

// WithObservability enables metrics and tracing
func WithObservability(metrics, tracing bool) ClientOption {
	return func(c *types.ClientConfig) {
		c.MetricsEnabled = metrics
		c.TracingEnabled = tracing
	}
}

// Error types are now in types.go to avoid circular imports

// DefaultClientConfig returns a default configuration
func DefaultClientConfig() *types.ClientConfig {
	return &types.ClientConfig{
		Providers:         make(map[domain.Provider]types.ProviderConfig),
		AutoFailover:      true,
		LoadBalancing:     false,
		CacheEnabled:      true,
		CacheDefaultTTL:   15 * time.Minute,
		CacheMaxSize:      10000,
		MetricsEnabled:    true,
		TracingEnabled:    false,
		LogLevel:          "info",
		DefaultTimeout:    30 * time.Second,
		StreamTimeout:     5 * time.Minute,
		MaxRetries:        3,
		RetryBackoff:      time.Second,
		RetryableErrors:   []string{"timeout", "provider_unavailable", "rate_limit_exceeded"},
	}
}