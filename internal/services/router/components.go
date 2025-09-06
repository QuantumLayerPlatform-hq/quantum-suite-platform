package router

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/shared/logger"
)

// LoadBalancer handles provider selection and load distribution
type LoadBalancer struct {
	logger   logger.Logger
	counters map[domain.Provider]*atomic.Uint64
	mu       sync.RWMutex
}

func NewLoadBalancer(log logger.Logger) *LoadBalancer {
	return &LoadBalancer{
		logger:   log.WithField("component", "load_balancer"),
		counters: make(map[domain.Provider]*atomic.Uint64),
	}
}

func (lb *LoadBalancer) SelectProvider(providers []domain.Provider) domain.Provider {
	if len(providers) == 1 {
		return providers[0]
	}

	// Use round-robin load balancing
	// FIXED: Thread-safe provider selection
	lb.mu.Lock()
	defer lb.mu.Unlock()

	var selectedProvider domain.Provider
	var minCount uint64 = ^uint64(0) // Max uint64

	// Find provider with minimum request count
	for _, provider := range providers {
		if _, exists := lb.counters[provider]; !exists {
			lb.counters[provider] = &atomic.Uint64{}
		}
		
		count := lb.counters[provider].Load()
		if count < minCount {
			minCount = count
			selectedProvider = provider
		}
	}

	// Increment counter for selected provider
	if counter, exists := lb.counters[selectedProvider]; exists {
		counter.Add(1)
	}

	lb.logger.Debug("Selected provider",
		logger.F("provider", selectedProvider),
		logger.F("request_count", minCount+1),
		logger.F("available_providers", len(providers)),
	)

	return selectedProvider
}

// CircuitBreaker prevents cascading failures by failing fast when providers are unhealthy
type CircuitBreaker struct {
	logger    logger.Logger
	states    map[domain.Provider]*CircuitState
	mu        sync.RWMutex
	threshold int           // Number of failures before opening circuit
	timeout   time.Duration // Time before attempting to reset
}

type CircuitState struct {
	State        CircuitStateType
	FailureCount int
	LastFailure  time.Time
	LastSuccess  time.Time
}

type CircuitStateType int

const (
	CircuitStateClosed CircuitStateType = iota
	CircuitStateOpen
	CircuitStateHalfOpen
)

func NewCircuitBreaker(log logger.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		logger:    log.WithField("component", "circuit_breaker"),
		states:    make(map[domain.Provider]*CircuitState),
		threshold: 5,              // 5 failures
		timeout:   30 * time.Second, // 30 seconds
	}
}

func (cb *CircuitBreaker) CanExecute(provider domain.Provider) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.getOrCreateState(provider)

	switch state.State {
	case CircuitStateClosed:
		return true
	case CircuitStateOpen:
		// Check if we should move to half-open
		if time.Since(state.LastFailure) > cb.timeout {
			state.State = CircuitStateHalfOpen
			cb.logger.Info("Circuit breaker moving to half-open",
				logger.F("provider", provider))
			return true
		}
		return false
	case CircuitStateHalfOpen:
		return true
	}

	return true
}

func (cb *CircuitBreaker) RecordSuccess(provider domain.Provider) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.getOrCreateState(provider)
	state.LastSuccess = time.Now()
	
	if state.State == CircuitStateHalfOpen {
		// Reset to closed on successful half-open attempt
		state.State = CircuitStateClosed
		state.FailureCount = 0
		cb.logger.Info("Circuit breaker reset to closed",
			logger.F("provider", provider))
	}
}

func (cb *CircuitBreaker) RecordFailure(provider domain.Provider) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.getOrCreateState(provider)
	state.FailureCount++
	state.LastFailure = time.Now()

	if state.FailureCount >= cb.threshold && state.State == CircuitStateClosed {
		state.State = CircuitStateOpen
		cb.logger.Warn("Circuit breaker opened due to failures",
			logger.F("provider", provider),
			logger.F("failure_count", state.FailureCount))
	}
}

func (cb *CircuitBreaker) getOrCreateState(provider domain.Provider) *CircuitState {
	if state, exists := cb.states[provider]; exists {
		return state
	}

	state := &CircuitState{
		State:       CircuitStateClosed,
		LastSuccess: time.Now(),
	}
	cb.states[provider] = state
	return state
}

// HealthChecker monitors provider health
type HealthChecker struct {
	providers map[domain.Provider]ProviderClient
	logger    logger.Logger
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

func NewHealthChecker(providers map[domain.Provider]ProviderClient, log logger.Logger) *HealthChecker {
	return &HealthChecker{
		providers: providers,
		logger:    log.WithField("component", "health_checker"),
		stopCh:    make(chan struct{}),
	}
}

func (hc *HealthChecker) Start() {
	hc.wg.Add(1)
	go hc.healthCheckLoop()
}

func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
}

func (hc *HealthChecker) healthCheckLoop() {
	defer hc.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	// Initial health check
	hc.checkAllProviders()

	for {
		select {
		case <-ticker.C:
			hc.checkAllProviders()
		case <-hc.stopCh:
			return
		}
	}
}

func (hc *HealthChecker) checkAllProviders() {
	for provider, client := range hc.providers {
		hc.wg.Add(1)
		go func(p domain.Provider, c ProviderClient) {
			defer hc.wg.Done()
			hc.checkProviderHealth(p, c)
		}(provider, client)
	}
}

func (hc *HealthChecker) checkProviderHealth(provider domain.Provider, client ProviderClient) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := client.HealthCheck(ctx)
	latency := time.Since(start)

	if err != nil {
		hc.logger.Warn("Provider health check failed",
			logger.F("provider", provider),
			logger.F("error", err),
			logger.F("latency_ms", latency.Milliseconds()),
		)
		// In a real implementation, this would update the provider config
	} else {
		hc.logger.Debug("Provider health check passed",
			logger.F("provider", provider),
			logger.F("latency_ms", latency.Milliseconds()),
		)
	}
}

// Mock provider client for development
type mockProviderClient struct {
	provider domain.Provider
	logger   logger.Logger
}

func (m *mockProviderClient) CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	m.logger.Info("Mock provider handling completion",
		logger.F("tenant_id", req.TenantID),
		logger.F("model", req.Model),
		logger.F("messages", len(req.Messages)),
	)

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	return &CompletionResponse{
		ID:       "cmpl-" + req.RequestID,
		Object:   "chat.completion",
		Created:  time.Now().Unix(),
		Model:    req.Model,
		Provider: m.provider,
		Choices: []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					Role: domain.MessageRoleAssistant,
					Content: []domain.ContentPart{
						{
							Type: domain.ContentTypeText,
							Text: "This is a mock response from provider " + string(m.provider),
						},
					},
				},
				FinishReason: domain.FinishReasonStop,
			},
		},
		Usage: domain.Usage{
			PromptTokens:     10,
			CompletionTokens: 15,
			TotalTokens:      25,
			CostUSD:         0.0001,
			CacheHit:        false,
		},
	}, nil
}

func (m *mockProviderClient) CreateCompletionStream(ctx context.Context, req *CompletionRequest) (<-chan *StreamResponse, error) {
	m.logger.Info("Mock provider handling streaming completion",
		logger.F("tenant_id", req.TenantID),
		logger.F("model", req.Model),
	)

	ch := make(chan *StreamResponse, 5)

	go func() {
		defer close(ch)

		words := []string{"Hello", " from", " provider", " " + string(m.provider), "!"}
		
		for i, word := range words {
			select {
			case ch <- &StreamResponse{
				ID:       "cmpl-" + req.RequestID,
				Object:   "chat.completion.chunk",
				Created:  time.Now().Unix(),
				Model:    req.Model,
				Provider: m.provider,
				Choices: []domain.Choice{
					{
						Index: 0,
						Message: domain.Message{
							Role: domain.MessageRoleAssistant,
							Content: []domain.ContentPart{
								{
									Type: domain.ContentTypeText,
									Text: word,
								},
							},
						},
						FinishReason: func() domain.FinishReason {
							if i == len(words)-1 {
								return domain.FinishReasonStop
							}
							return ""
						}(),
					},
				},
			}:
			case <-ctx.Done():
				return
			}
			time.Sleep(100 * time.Millisecond)
		}

		ch <- &StreamResponse{Done: true}
	}()

	return ch, nil
}

func (m *mockProviderClient) CreateEmbeddings(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	m.logger.Info("Mock provider handling embedding",
		logger.F("tenant_id", req.TenantID),
		logger.F("model", req.Model),
		logger.F("input_count", len(req.Input)),
	)

	// Simulate processing time
	time.Sleep(50 * time.Millisecond)

	data := make([]domain.Embedding, len(req.Input))
	for i := range req.Input {
		// Create mock embedding vector
		embedding := make([]float64, 1536)
		for j := range embedding {
			embedding[j] = 0.001 * float64(i+j)
		}

		data[i] = domain.Embedding{
			Object:    "embedding",
			Embedding: embedding,
			Index:     i,
		}
	}

	return &EmbeddingResponse{
		Object:   "list",
		Data:     data,
		Model:    req.Model,
		Provider: m.provider,
		Usage: domain.EmbeddingUsage{
			PromptTokens: len(req.Input) * 8,
			TotalTokens:  len(req.Input) * 8,
			CostUSD:      float64(len(req.Input)) * 0.0001,
		},
	}, nil
}

func (m *mockProviderClient) ListModels(ctx context.Context) ([]domain.Model, error) {
	// Return mock models based on provider
	switch m.provider {
	case domain.ProviderOpenAI:
		return []domain.Model{
			{
				ModelID:      "gpt-3.5-turbo",
				Provider:     domain.ProviderOpenAI,
				Name:         "GPT-3.5 Turbo",
				Description:  "Most capable GPT-3.5 model and optimized for chat",
				Capabilities: []domain.Capability{domain.CapabilityCompletion},
				ContextLength: 4096,
				Pricing: domain.ModelPricing{
					InputTokenCost:  0.0015,
					OutputTokenCost: 0.002,
					Unit:           "1K tokens",
				},
				Status:   domain.ModelStatusAvailable,
				IsActive: true,
			},
			{
				ModelID:      "gpt-4",
				Provider:     domain.ProviderOpenAI,
				Name:         "GPT-4",
				Description:  "More capable than any GPT-3.5 model",
				Capabilities: []domain.Capability{domain.CapabilityCompletion, domain.CapabilityVision},
				ContextLength: 8192,
				Pricing: domain.ModelPricing{
					InputTokenCost:  0.03,
					OutputTokenCost: 0.06,
					Unit:           "1K tokens",
				},
				Status:   domain.ModelStatusAvailable,
				IsActive: true,
			},
			{
				ModelID:      "text-embedding-ada-002",
				Provider:     domain.ProviderOpenAI,
				Name:         "Text Embedding Ada 002",
				Description:  "Most capable embedding model for measuring relatedness of text",
				Capabilities: []domain.Capability{domain.CapabilityEmbedding},
				ContextLength: 8192,
				Pricing: domain.ModelPricing{
					InputTokenCost:  0.0001,
					OutputTokenCost: 0,
					Unit:           "1K tokens",
				},
				Status:   domain.ModelStatusAvailable,
				IsActive: true,
			},
		}, nil
	case domain.ProviderAnthropic:
		return []domain.Model{
			{
				ModelID:      "claude-3-sonnet-20240229",
				Provider:     domain.ProviderAnthropic,
				Name:         "Claude 3 Sonnet",
				Description:  "Balance of intelligence and speed for enterprise workloads",
				Capabilities: []domain.Capability{domain.CapabilityCompletion},
				ContextLength: 200000,
				Pricing: domain.ModelPricing{
					InputTokenCost:  0.003,
					OutputTokenCost: 0.015,
					Unit:           "1K tokens",
				},
				Status:   domain.ModelStatusAvailable,
				IsActive: true,
			},
		}, nil
	default:
		return []domain.Model{}, nil
	}
}

func (m *mockProviderClient) HealthCheck(ctx context.Context) error {
	// Simulate occasional failures for testing
	// In real implementation, this would ping the actual provider
	m.logger.Debug("Mock provider health check", logger.F("provider", m.provider))
	return nil
}