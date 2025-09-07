package qlens

import (
	"context"
	"time"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/qlens-types"
)

// Client represents the main QLens client interface
type Client interface {
	// Completion methods
	CreateCompletion(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error)
	CreateCompletionStream(ctx context.Context, req *types.CompletionRequest) (<-chan types.StreamResponse, error)

	// Embedding methods
	CreateEmbeddings(ctx context.Context, req *types.EmbeddingRequest) (*types.EmbeddingResponse, error)

	// Model management
	ListModels(ctx context.Context, opts *types.ListModelsOptions) (*types.ModelsResponse, error)
	GetModel(ctx context.Context, modelID string, provider domain.Provider) (*types.Model, error)

	// Health and status
	HealthCheck(ctx context.Context) (*types.HealthResponse, error)

	// Close gracefully shuts down the client
	Close() error
}

// ProviderClient represents an interface for individual LLM providers
type ProviderClient interface {
	// Provider identification
	Provider() domain.Provider
	Name() string

	// Core operations
	CreateCompletion(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error)
	CreateCompletionStream(ctx context.Context, req *types.CompletionRequest) (<-chan types.StreamResponse, error)
	CreateEmbeddings(ctx context.Context, req *types.EmbeddingRequest) (*types.EmbeddingResponse, error)

	// Model operations
	ListModels(ctx context.Context) ([]types.Model, error)
	GetModel(ctx context.Context, modelID string) (*types.Model, error)

	// Health and configuration
	HealthCheck(ctx context.Context) error
	Configure(config types.ProviderConfig) error
	GetConfig() types.ProviderConfig

	// Close resources
	Close() error
}

// Router represents the interface for request routing
type Router interface {
	// SelectProvider chooses the best provider for a request
	SelectProvider(ctx context.Context, req *types.CompletionRequest) (domain.Provider, error)
	
	// SelectEmbeddingProvider chooses the best provider for embedding requests
	SelectEmbeddingProvider(ctx context.Context, req *types.EmbeddingRequest) (domain.Provider, error)
	
	// UpdateProviderHealth updates the health status of a provider
	UpdateProviderHealth(provider domain.Provider, health types.ProviderHealth)
	
	// GetProviderHealth returns the current health of a provider
	GetProviderHealth(provider domain.Provider) (types.ProviderHealth, bool)
	
	// GetAvailableProviders returns all healthy providers
	GetAvailableProviders() []domain.Provider
	
	// RegisterProvider registers a new provider
	RegisterProvider(provider domain.Provider, config types.ProviderConfig) error
	
	// UnregisterProvider removes a provider
	UnregisterProvider(provider domain.Provider) error
	
	// Health check for the router
	HealthCheck(ctx context.Context) error
	
	// Configure the router
	Configure(config types.RouterConfig) error
	
	// Close router resources
	Close() error
}

// Cache represents the interface for caching responses
type Cache interface {
	// Get retrieves a cached response
	Get(ctx context.Context, key string) (*types.CompletionResponse, bool)
	
	// Set stores a response in cache
	Set(ctx context.Context, key string, response *types.CompletionResponse, ttl time.Duration) error
	
	// Delete removes an entry from cache
	Delete(ctx context.Context, key string) error
	
	// GetEmbedding retrieves cached embeddings
	GetEmbedding(ctx context.Context, key string) (*types.EmbeddingResponse, bool)
	
	// SetEmbedding stores embeddings in cache
	SetEmbedding(ctx context.Context, key string, response *types.EmbeddingResponse, ttl time.Duration) error
	
	// Clear removes all cached entries
	Clear(ctx context.Context) error
	
	// Stats returns cache statistics
	Stats() types.CacheStats
	
	// Health check for the cache
	HealthCheck(ctx context.Context) error
	
	// Configure the cache
	Configure(config types.CacheConfig) error
	
	// Close cache resources  
	Close() error
}

// Configuration types are now in types.go