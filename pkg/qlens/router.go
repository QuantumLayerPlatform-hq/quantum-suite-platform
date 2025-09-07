package qlens

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/qlens-types"
)

// Router interface is defined in interfaces.go

// DefaultRouter implements the Router interface with load balancing and failover
type DefaultRouter struct {
	mu              sync.RWMutex
	providers       map[domain.Provider]types.ProviderConfig
	providerHealth  map[domain.Provider]types.ProviderHealth
	defaultProvider domain.Provider
	loadBalancing   bool
	autoFailover    bool
	
	// Load balancing state
	roundRobinIndex map[string]int
	
	// Health checking
	healthCheckInterval time.Duration
	stopHealthCheck     chan struct{}
	healthCheckOnce     sync.Once
}

// NewDefaultRouter creates a new default router
func NewDefaultRouter(config *types.ClientConfig) *DefaultRouter {
	router := &DefaultRouter{
		providers:           make(map[domain.Provider]types.ProviderConfig),
		providerHealth:      make(map[domain.Provider]types.ProviderHealth),
		roundRobinIndex:     make(map[string]int),
		loadBalancing:       config.LoadBalancing,
		autoFailover:        config.AutoFailover,
		defaultProvider:     config.DefaultProvider,
		healthCheckInterval: 30 * time.Second,
		stopHealthCheck:     make(chan struct{}),
	}
	
	// Register providers from config
	for provider, providerConfig := range config.Providers {
		router.RegisterProvider(provider, providerConfig)
	}
	
	// Start health checking
	router.startHealthChecking()
	
	return router
}

// SelectProvider implements the Router interface
func (r *DefaultRouter) SelectProvider(ctx context.Context, req *types.CompletionRequest) (domain.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// If provider is explicitly specified and healthy, use it
	if req.Provider != "" && req.Provider != domain.ProviderAuto {
		if r.isProviderHealthy(req.Provider) {
			return req.Provider, nil
		}
		
		// If auto failover is disabled, return error
		if !r.autoFailover {
			return "", fmt.Errorf("provider %s is unhealthy and auto failover is disabled", req.Provider)
		}
	}
	
	// Get healthy providers
	healthyProviders := r.getHealthyProviders()
	if len(healthyProviders) == 0 {
		return "", errors.New("no healthy providers available")
	}
	
	// Apply model-specific filtering
	if req.Model != "" {
		healthyProviders = r.filterProvidersForModel(healthyProviders, req.Model)
		if len(healthyProviders) == 0 {
			return "", fmt.Errorf("no healthy providers support model %s", req.Model)
		}
	}
	
	// Apply priority-based filtering if priority is specified
	if req.Priority != "" {
		healthyProviders = r.filterProvidersByPriority(healthyProviders, req.Priority)
	}
	
	// Select provider based on strategy
	return r.selectProviderByStrategy(healthyProviders, "completion"), nil
}

// SelectEmbeddingProvider implements the Router interface
func (r *DefaultRouter) SelectEmbeddingProvider(ctx context.Context, req *types.EmbeddingRequest) (domain.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// If provider is explicitly specified and healthy, use it
	if req.Provider != "" && req.Provider != domain.ProviderAuto {
		if r.isProviderHealthy(req.Provider) {
			return req.Provider, nil
		}
		
		if !r.autoFailover {
			return "", fmt.Errorf("provider %s is unhealthy and auto failover is disabled", req.Provider)
		}
	}
	
	// Get healthy providers that support embeddings
	healthyProviders := r.getHealthyEmbeddingProviders()
	if len(healthyProviders) == 0 {
		return "", errors.New("no healthy embedding providers available")
	}
	
	// Apply model-specific filtering
	if req.Model != "" {
		healthyProviders = r.filterProvidersForModel(healthyProviders, req.Model)
		if len(healthyProviders) == 0 {
			return "", fmt.Errorf("no healthy providers support embedding model %s", req.Model)
		}
	}
	
	// Select provider based on strategy
	return r.selectProviderByStrategy(healthyProviders, "embedding"), nil
}

// UpdateProviderHealth implements the Router interface
func (r *DefaultRouter) UpdateProviderHealth(provider domain.Provider, health types.ProviderHealth) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.providerHealth[provider] = health
}

// GetProviderHealth implements the Router interface
func (r *DefaultRouter) GetProviderHealth(provider domain.Provider) (types.ProviderHealth, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	health, exists := r.providerHealth[provider]
	return health, exists
}

// GetAvailableProviders implements the Router interface
func (r *DefaultRouter) GetAvailableProviders() []domain.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return r.getHealthyProviders()
}

// RegisterProvider implements the Router interface
func (r *DefaultRouter) RegisterProvider(provider domain.Provider, config types.ProviderConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if !config.Enabled {
		return fmt.Errorf("cannot register disabled provider %s", provider)
	}
	
	r.providers[provider] = config
	
	// Initialize health status
	r.providerHealth[provider] = types.ProviderHealth{
		Status:        domain.ProviderHealthHealthy,
		LatencyMS:     0,
		ErrorRate:     0,
		LastCheck:     time.Now(),
		HealthMessage: "Provider registered",
	}
	
	return nil
}

// UnregisterProvider implements the Router interface
func (r *DefaultRouter) UnregisterProvider(provider domain.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.providers, provider)
	delete(r.providerHealth, provider)
	
	return nil
}

// Helper methods

func (r *DefaultRouter) isProviderHealthy(provider domain.Provider) bool {
	health, exists := r.providerHealth[provider]
	if !exists {
		return false
	}
	
	config, exists := r.providers[provider]
	if !exists || !config.Enabled {
		return false
	}
	
	return health.Status == domain.ProviderHealthHealthy || 
		   health.Status == domain.ProviderHealthDegraded
}

func (r *DefaultRouter) getHealthyProviders() []domain.Provider {
	var healthy []domain.Provider
	
	for provider := range r.providers {
		if r.isProviderHealthy(provider) {
			healthy = append(healthy, provider)
		}
	}
	
	// Sort by priority (higher priority first)
	sort.Slice(healthy, func(i, j int) bool {
		return r.providers[healthy[i]].Priority > r.providers[healthy[j]].Priority
	})
	
	return healthy
}

func (r *DefaultRouter) getHealthyEmbeddingProviders() []domain.Provider {
	var healthy []domain.Provider
	
	for provider := range r.providers {
		if r.isProviderHealthy(provider) {
			// Check if provider supports embeddings
			if r.supportsEmbeddings(provider) {
				healthy = append(healthy, provider)
			}
		}
	}
	
	// Sort by priority
	sort.Slice(healthy, func(i, j int) bool {
		return r.providers[healthy[i]].Priority > r.providers[healthy[j]].Priority
	})
	
	return healthy
}

func (r *DefaultRouter) supportsEmbeddings(provider domain.Provider) bool {
	// For now, assume all providers except local support embeddings
	// This should be enhanced with actual capability checking
	switch provider {
	case domain.ProviderOpenAI, domain.ProviderAnthropic:
		return true
	case domain.ProviderLocal:
		return false
	default:
		return false
	}
}

func (r *DefaultRouter) filterProvidersForModel(providers []domain.Provider, model string) []domain.Provider {
	var filtered []domain.Provider
	
	for _, provider := range providers {
		if r.providerSupportsModel(provider, model) {
			filtered = append(filtered, provider)
		}
	}
	
	return filtered
}

func (r *DefaultRouter) providerSupportsModel(provider domain.Provider, model string) bool {
	// This is a simplified check - should be enhanced with actual model registry
	switch provider {
	case domain.ProviderOpenAI:
		return isOpenAIModel(model)
	case domain.ProviderAnthropic:
		return isAnthropicModel(model)
	case domain.ProviderLocal:
		return true // Local providers can potentially support any model
	default:
		return false
	}
}

func (r *DefaultRouter) filterProvidersByPriority(providers []domain.Provider, priority domain.Priority) []domain.Provider {
	// For high priority requests, prefer providers with lower latency
	if priority == domain.PriorityHigh || priority == domain.PriorityCritical {
		sort.Slice(providers, func(i, j int) bool {
			healthI := r.providerHealth[providers[i]]
			healthJ := r.providerHealth[providers[j]]
			return healthI.LatencyMS < healthJ.LatencyMS
		})
	}
	
	return providers
}

func (r *DefaultRouter) selectProviderByStrategy(providers []domain.Provider, requestType string) domain.Provider {
	if len(providers) == 0 {
		return ""
	}
	
	// If there's a default provider in the list, prefer it
	if r.defaultProvider != "" {
		for _, provider := range providers {
			if provider == r.defaultProvider {
				return provider
			}
		}
	}
	
	// If load balancing is enabled, use round-robin
	if r.loadBalancing && len(providers) > 1 {
		return r.selectRoundRobin(providers, requestType)
	}
	
	// Otherwise, return the highest priority provider
	return providers[0]
}

func (r *DefaultRouter) selectRoundRobin(providers []domain.Provider, requestType string) domain.Provider {
	key := requestType
	index := r.roundRobinIndex[key]
	provider := providers[index%len(providers)]
	r.roundRobinIndex[key] = (index + 1) % len(providers)
	return provider
}

func (r *DefaultRouter) startHealthChecking() {
	r.healthCheckOnce.Do(func() {
		go r.healthCheckLoop()
	})
}

func (r *DefaultRouter) healthCheckLoop() {
	ticker := time.NewTicker(r.healthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			r.performHealthChecks()
		case <-r.stopHealthCheck:
			return
		}
	}
}

func (r *DefaultRouter) performHealthChecks() {
	r.mu.RLock()
	providers := make([]domain.Provider, 0, len(r.providers))
	for provider := range r.providers {
		providers = append(providers, provider)
	}
	r.mu.RUnlock()
	
	for _, provider := range providers {
		go r.checkProviderHealth(provider)
	}
}

func (r *DefaultRouter) checkProviderHealth(provider domain.Provider) {
	start := time.Now()
	
	// This is a placeholder for actual health checking
	// In a real implementation, this would ping the provider's health endpoint
	
	// Simulate health check
	healthy := r.simulateHealthCheck(provider)
	
	latency := time.Since(start).Seconds() * 1000 // Convert to milliseconds
	
	var status domain.ProviderHealthStatus
	var errorRate float64
	var message string
	
	if healthy {
		if latency > 2000 {
			status = domain.ProviderHealthDegraded
			message = "High latency detected"
		} else {
			status = domain.ProviderHealthHealthy
			message = "Provider is healthy"
		}
		errorRate = 0.0
	} else {
		status = domain.ProviderHealthUnhealthy
		errorRate = 1.0
		message = "Health check failed"
	}
	
	health := types.ProviderHealth{
		Status:        status,
		LatencyMS:     latency,
		ErrorRate:     errorRate,
		LastCheck:     time.Now(),
		HealthMessage: message,
	}
	
	r.UpdateProviderHealth(provider, health)
}

func (r *DefaultRouter) simulateHealthCheck(provider domain.Provider) bool {
	// Simulate 95% uptime for providers
	randomNum, _ := rand.Int(rand.Reader, big.NewInt(100))
	return randomNum.Int64() < 95
}

// Stop gracefully shuts down the router
func (r *DefaultRouter) Stop() {
	close(r.stopHealthCheck)
}

// Close implements the Router interface
func (r *DefaultRouter) Close() error {
	r.Stop()
	return nil
}

// Configure implements the Router interface
func (r *DefaultRouter) Configure(config types.RouterConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Update router configuration based on config
	// For now, this is a placeholder
	return nil
}

// HealthCheck implements the Router interface
func (r *DefaultRouter) HealthCheck(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Check if router has any healthy providers
	healthyProviders := r.getHealthyProviders()
	if len(healthyProviders) == 0 {
		return fmt.Errorf("no healthy providers available")
	}
	
	return nil
}

// Helper functions for model validation

func isOpenAIModel(model string) bool {
	openAIModels := []string{
		"gpt-4", "gpt-4-turbo", "gpt-4-turbo-preview",
		"gpt-3.5-turbo", "gpt-3.5-turbo-16k",
		"text-davinci-003", "text-davinci-002",
		"code-davinci-002", "text-embedding-ada-002",
		"text-embedding-3-small", "text-embedding-3-large",
	}
	
	for _, m := range openAIModels {
		if model == m || contains(model, m) {
			return true
		}
	}
	return false
}

func isAnthropicModel(model string) bool {
	anthropicModels := []string{
		"claude-3-opus", "claude-3-sonnet", "claude-3-haiku",
		"claude-2.1", "claude-2.0", "claude-instant",
	}
	
	for _, m := range anthropicModels {
		if model == m || contains(model, m) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}