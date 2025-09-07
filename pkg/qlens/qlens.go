package qlens

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/quantum-suite/platform/internal/domain"
	qlensProvider "github.com/quantum-suite/platform/internal/providers/qlens"
	"github.com/quantum-suite/platform/pkg/qlens-types"
)

// QLens is the main client that implements the Client interface
type QLens struct {
	mu        sync.RWMutex
	config    *types.ClientConfig
	router    Router
	cache     Cache
	providers map[domain.Provider]types.ProviderClient
	metrics   *MetricsCollector
	startTime time.Time
}

// New creates a new QLens client with the given configuration
func New(opts ...ClientOption) (*QLens, error) {
	config := DefaultClientConfig()
	
	// Apply options
	for _, opt := range opts {
		opt(config)
	}
	
	client := &QLens{
		config:    config,
		providers: make(map[domain.Provider]types.ProviderClient),
		startTime: time.Now(),
	}
	
	// Initialize router
	client.router = NewDefaultRouter(config)
	
	// Initialize cache
	if config.CacheEnabled {
		client.cache = NewInMemoryCache(config.CacheMaxSize)
	}
	
	// Initialize metrics collector
	if config.MetricsEnabled {
		client.metrics = NewMetricsCollector()
	}
	
	// Initialize providers
	if err := client.initializeProviders(); err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}
	
	return client, nil
}

// CreateCompletion implements the Client interface
func (q *QLens) CreateCompletion(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error) {
	start := time.Now()
	
	// Set request ID if not provided
	if req.RequestID == "" {
		req.RequestID = generateRequestID()
	}
	
	// Record metrics
	if q.metrics != nil {
		q.metrics.IncrementRequestCount("completion")
		defer func() {
			q.metrics.RecordResponseTime("completion", time.Since(start))
		}()
	}
	
	// Apply caching middleware if enabled
	completionFunc := q.createCompletionFunc()
	if q.cache != nil {
		completionFunc = CacheMiddleware(q.cache, q.config)(completionFunc)
	}
	
	// Execute the completion
	response, err := completionFunc(ctx, req)
	if err != nil {
		if q.metrics != nil {
			q.metrics.IncrementErrorCount("completion", err.Error())
		}
		return nil, err
	}
	
	// Record success metrics
	if q.metrics != nil {
		q.metrics.RecordTokenUsage("completion", response.Usage.TotalTokens)
		q.metrics.RecordCost("completion", response.Usage.CostUSD)
		if response.CacheHit {
			q.metrics.IncrementCacheHits("completion")
		} else {
			q.metrics.IncrementCacheMisses("completion")
		}
	}
	
	return response, nil
}

// CreateCompletionStream implements the Client interface
func (q *QLens) CreateCompletionStream(ctx context.Context, req *types.CompletionRequest) (<-chan types.StreamResponse, error) {
	start := time.Now()
	
	// Set request ID if not provided
	if req.RequestID == "" {
		req.RequestID = generateRequestID()
	}
	
	// Record metrics
	if q.metrics != nil {
		q.metrics.IncrementRequestCount("completion_stream")
		defer func() {
			q.metrics.RecordResponseTime("completion_stream", time.Since(start))
		}()
	}
	
	// Select provider
	provider, err := q.router.SelectProvider(ctx, req)
	if err != nil {
		if q.metrics != nil {
			q.metrics.IncrementErrorCount("completion_stream", err.Error())
		}
		return nil, err
	}
	
	// Get provider client
	q.mu.RLock()
	providerClient, exists := q.providers[provider]
	q.mu.RUnlock()
	
	if !exists {
		err := fmt.Errorf("provider %s not available", provider)
		if q.metrics != nil {
			q.metrics.IncrementErrorCount("completion_stream", err.Error())
		}
		return nil, err
	}
	
	// Create stream
	return providerClient.CreateCompletionStream(ctx, req)
}

// CreateEmbeddings implements the Client interface
func (q *QLens) CreateEmbeddings(ctx context.Context, req *types.EmbeddingRequest) (*types.EmbeddingResponse, error) {
	start := time.Now()
	
	// Set request ID if not provided
	if req.RequestID == "" {
		req.RequestID = generateRequestID()
	}
	
	// Record metrics
	if q.metrics != nil {
		q.metrics.IncrementRequestCount("embedding")
		defer func() {
			q.metrics.RecordResponseTime("embedding", time.Since(start))
		}()
	}
	
	// Apply caching middleware if enabled
	embeddingFunc := q.createEmbeddingFunc()
	if q.cache != nil {
		embeddingFunc = EmbeddingCacheMiddleware(q.cache, q.config)(embeddingFunc)
	}
	
	// Execute the embedding request
	response, err := embeddingFunc(ctx, req)
	if err != nil {
		if q.metrics != nil {
			q.metrics.IncrementErrorCount("embedding", err.Error())
		}
		return nil, err
	}
	
	// Record success metrics
	if q.metrics != nil {
		q.metrics.RecordTokenUsage("embedding", response.Usage.TotalTokens)
		q.metrics.RecordCost("embedding", response.Usage.CostUSD)
	}
	
	return response, nil
}

// ListModels implements the Client interface
func (q *QLens) ListModels(ctx context.Context, opts *types.ListModelsOptions) (*types.ModelsResponse, error) {
	var allModels []types.Model
	
	q.mu.RLock()
	providers := make([]domain.Provider, 0, len(q.providers))
	for provider := range q.providers {
		// Filter by provider if specified
		if opts != nil && opts.Provider != "" && opts.Provider != provider {
			continue
		}
		providers = append(providers, provider)
	}
	q.mu.RUnlock()
	
	// Get models from each provider
	for _, provider := range providers {
		q.mu.RLock()
		providerClient, exists := q.providers[provider]
		q.mu.RUnlock()
		
		if !exists {
			continue
		}
		
		models, err := providerClient.ListModels(ctx)
		if err != nil {
			// Log error but continue with other providers
			continue
		}
		
		// Apply filters
		if opts != nil {
			models = q.filterModels(models, opts)
		}
		
		allModels = append(allModels, models...)
	}
	
	return &types.ModelsResponse{
		Object: "list",
		Data:   allModels,
	}, nil
}

// GetModel implements the Client interface
func (q *QLens) GetModel(ctx context.Context, modelID string, provider domain.Provider) (*types.Model, error) {
	q.mu.RLock()
	providerClient, exists := q.providers[provider]
	q.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("provider %s not available", provider)
	}
	
	return providerClient.GetModel(ctx, modelID)
}

// HealthCheck implements the Client interface
func (q *QLens) HealthCheck(ctx context.Context) (*types.HealthResponse, error) {
	response := &types.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0", // This should come from build info
		Uptime:    time.Since(q.startTime),
		Providers: make(map[domain.Provider]types.ProviderHealth),
	}
	
	// Check each provider's health
	q.mu.RLock()
	providers := make([]domain.Provider, 0, len(q.providers))
	for provider := range q.providers {
		providers = append(providers, provider)
	}
	q.mu.RUnlock()
	
	for _, provider := range providers {
		health, exists := q.router.GetProviderHealth(provider)
		if !exists {
			health = types.ProviderHealth{
				Status:        domain.ProviderHealthUnhealthy,
				LatencyMS:     0,
				ErrorRate:     1.0,
				LastCheck:     time.Now(),
				HealthMessage: "Provider not registered",
			}
		}
		response.Providers[provider] = health
		
		// Set overall status based on provider health
		if health.Status == domain.ProviderHealthUnhealthy && len(q.providers) == 1 {
			response.Status = "unhealthy"
		}
	}
	
	return response, nil
}

// Close implements the Client interface
func (q *QLens) Close() error {
	var errors []error
	
	// Close providers
	q.mu.Lock()
	for _, provider := range q.providers {
		if err := provider.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	q.providers = make(map[domain.Provider]types.ProviderClient)
	q.mu.Unlock()
	
	// Close cache
	if q.cache != nil {
		if err := q.cache.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	
	// Close router
	if router, ok := q.router.(*DefaultRouter); ok {
		router.Stop()
	}
	
	// Close metrics
	if q.metrics != nil {
		q.metrics.Close()
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("multiple close errors: %v", errors)
	}
	
	return nil
}

// Helper methods

func (q *QLens) initializeProviders() error {
	for provider, config := range q.config.Providers {
		if !config.Enabled {
			continue
		}
		
		var providerClient types.ProviderClient
		
		switch provider {
		case domain.ProviderOpenAI:
			providerClient = qlensProvider.NewOpenAIClient(config)
		case domain.ProviderAnthropic:
			// TODO: Implement Anthropic client
			continue
		case domain.ProviderLocal:
			// TODO: Implement local client
			continue
		default:
			return fmt.Errorf("unsupported provider: %s", provider)
		}
		
		q.mu.Lock()
		q.providers[provider] = providerClient
		q.mu.Unlock()
		
		// Register with router
		if err := q.router.RegisterProvider(provider, config); err != nil {
			return fmt.Errorf("failed to register provider %s: %w", provider, err)
		}
	}
	
	if len(q.providers) == 0 {
		return fmt.Errorf("no providers configured")
	}
	
	return nil
}

func (q *QLens) createCompletionFunc() CompletionFunc {
	return func(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error) {
		// Select provider
		provider, err := q.router.SelectProvider(ctx, req)
		if err != nil {
			return nil, err
		}
		
		// Get provider client
		q.mu.RLock()
		providerClient, exists := q.providers[provider]
		q.mu.RUnlock()
		
		if !exists {
			return nil, fmt.Errorf("provider %s not available", provider)
		}
		
		// Make request with retry logic
		return q.executeWithRetry(ctx, func() (*types.CompletionResponse, error) {
			return providerClient.CreateCompletion(ctx, req)
		})
	}
}

func (q *QLens) createEmbeddingFunc() EmbeddingFunc {
	return func(ctx context.Context, req *types.EmbeddingRequest) (*types.EmbeddingResponse, error) {
		// Select provider
		provider, err := q.router.SelectEmbeddingProvider(ctx, req)
		if err != nil {
			return nil, err
		}
		
		// Get provider client
		q.mu.RLock()
		providerClient, exists := q.providers[provider]
		q.mu.RUnlock()
		
		if !exists {
			return nil, fmt.Errorf("provider %s not available", provider)
		}
		
		// Make request with retry logic
		return q.executeEmbeddingWithRetry(ctx, func() (*types.EmbeddingResponse, error) {
			return providerClient.CreateEmbeddings(ctx, req)
		})
	}
}

func (q *QLens) executeWithRetry(ctx context.Context, fn func() (*types.CompletionResponse, error)) (*types.CompletionResponse, error) {
	var lastErr error
	
	for attempt := 0; attempt <= q.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait with exponential backoff
			backoff := time.Duration(attempt) * q.config.RetryBackoff
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		
		resp, err := fn()
		if err == nil {
			return resp, nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		if !q.isRetryableError(err) {
			break
		}
	}
	
	return nil, fmt.Errorf("request failed after %d attempts: %w", q.config.MaxRetries+1, lastErr)
}

func (q *QLens) executeEmbeddingWithRetry(ctx context.Context, fn func() (*types.EmbeddingResponse, error)) (*types.EmbeddingResponse, error) {
	var lastErr error
	
	for attempt := 0; attempt <= q.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait with exponential backoff
			backoff := time.Duration(attempt) * q.config.RetryBackoff
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		
		resp, err := fn()
		if err == nil {
			return resp, nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		if !q.isRetryableError(err) {
			break
		}
	}
	
	return nil, fmt.Errorf("embedding request failed after %d attempts: %w", q.config.MaxRetries+1, lastErr)
}

func (q *QLens) isRetryableError(err error) bool {
	if qlensErr, ok := err.(*types.QLensError); ok {
		for _, retryableType := range q.config.RetryableErrors {
			if qlensErr.Type == retryableType {
				return true
			}
		}
	}
	
	return false
}

func (q *QLens) filterModels(models []types.Model, opts *types.ListModelsOptions) []types.Model {
	var filtered []types.Model
	
	for _, model := range models {
		// Filter by capability
		if opts.Capability != "" {
			hasCapability := false
			for _, cap := range model.Capabilities {
				if cap == opts.Capability {
					hasCapability = true
					break
				}
			}
			if !hasCapability {
				continue
			}
		}
		
		// Filter by status
		if opts.Status != "" && model.Status != opts.Status {
			continue
		}
		
		filtered = append(filtered, model)
	}
	
	return filtered
}

// Utility functions

func generateRequestID() string {
	// Generate a simple request ID - in production, use a more robust UUID library
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// Builder methods for easier client construction

// NewWithOpenAI creates a QLens client configured for OpenAI
func NewWithOpenAI(apiKey string, opts ...ClientOption) (*QLens, error) {
	options := append([]ClientOption{
		WithProvider(domain.ProviderOpenAI, types.ProviderConfig{
			Provider: domain.ProviderOpenAI,
			APIKey:   apiKey,
			Enabled:  true,
			Priority: 1,
			Timeout:  30 * time.Second,
		}),
		WithDefaultProvider(domain.ProviderOpenAI),
	}, opts...)
	
	return New(options...)
}

// NewWithAnthropic creates a QLens client configured for Anthropic
func NewWithAnthropic(apiKey string, opts ...ClientOption) (*QLens, error) {
	options := append([]ClientOption{
		WithProvider(domain.ProviderAnthropic, types.ProviderConfig{
			Provider: domain.ProviderAnthropic,
			APIKey:   apiKey,
			Enabled:  true,
			Priority: 1,
			Timeout:  30 * time.Second,
		}),
		WithDefaultProvider(domain.ProviderAnthropic),
	}, opts...)
	
	return New(options...)
}

// NewWithMultipleProviders creates a QLens client with multiple providers
func NewWithMultipleProviders(providerConfigs map[domain.Provider]types.ProviderConfig, opts ...ClientOption) (*QLens, error) {
	var options []ClientOption
	
	for provider, config := range providerConfigs {
		options = append(options, WithProvider(provider, config))
	}
	
	options = append(options, opts...)
	
	return New(options...)
}