package router

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/internal/providers"
	"github.com/quantum-suite/platform/pkg/shared/env"
	shared_errors "github.com/quantum-suite/platform/pkg/shared/errors"
	"github.com/quantum-suite/platform/pkg/shared/logger"
)


type Service struct {
	config            *env.Config
	logger            logger.Logger
	router            *gin.Engine
	providerClients   map[domain.Provider]ProviderClient
	providerConfigs   map[domain.Provider]*domain.ProviderConfig
	modelRegistry     map[string]*domain.Model
	healthChecker     *HealthChecker
	loadBalancer      *LoadBalancer
	circuitBreaker    *CircuitBreaker
	mu                sync.RWMutex
}

// ProviderClient interface for LLM providers
type ProviderClient interface {
	CreateCompletion(ctx context.Context, req *domain.CompletionRequest) (*domain.CompletionResponse, error)
	CreateCompletionStream(ctx context.Context, req *domain.CompletionRequest) (<-chan *domain.StreamResponse, error)
	CreateEmbeddings(ctx context.Context, req *domain.EmbeddingRequest) (*domain.EmbeddingResponse, error)
	ListModels(ctx context.Context) ([]domain.Model, error)
	HealthCheck(ctx context.Context) error
}

// Request/Response types (same as gateway service)
// Use domain types instead of duplicating them here

func NewService(config *env.Config, log logger.Logger) (*Service, error) {
	service := &Service{
		config:          config,
		logger:          log.WithField("service", "router"),
		providerClients: make(map[domain.Provider]ProviderClient),
		providerConfigs: make(map[domain.Provider]*domain.ProviderConfig),
		modelRegistry:   make(map[string]*domain.Model),
	}

	// Initialize components
	if err := service.initializeComponents(); err != nil {
		return nil, shared_errors.InternalError("failed to initialize router components", err)
	}

	// Setup HTTP router
	service.setupRouter()

	return service, nil
}

func (s *Service) initializeComponents() error {
	// Initialize provider clients
	if err := s.initializeProviders(); err != nil {
		return err
	}

	// Initialize load balancer
	s.loadBalancer = NewLoadBalancer(s.logger)

	// Initialize circuit breaker
	s.circuitBreaker = NewCircuitBreaker(s.logger)

	// Initialize health checker
	s.healthChecker = NewHealthChecker(s.providerClients, s.logger)
	s.healthChecker.Start()

	// Load model registry
	if err := s.loadModelRegistry(); err != nil {
		return err
	}

	return nil
}

func (s *Service) initializeProviders() error {
	// Initialize providers based on configuration
	providers := s.config.Providers

	for providerName, providerConfig := range providers {
		provider := domain.Provider(providerName)
		
		// Create provider config
		config := domain.NewProviderConfig(provider, domain.TenantID("system"))
		config.Enabled = providerConfig.Enabled
		config.Config = map[string]interface{}{
			"api_key": providerConfig.APIKey,
			"base_url": providerConfig.BaseURL,
			"timeout": providerConfig.Timeout,
			"max_retries": providerConfig.MaxRetries,
		}
		s.providerConfigs[provider] = config

		if !providerConfig.Enabled {
			s.logger.Info("Provider disabled", logger.F("provider", provider))
			continue
		}

		// Create provider client
		client, err := s.createProviderClient(provider, providerConfig)
		if err != nil {
			s.logger.Error("Failed to create provider client", 
				logger.F("provider", provider),
				logger.F("error", err))
			continue
		}

		s.providerClients[provider] = client
		s.logger.Info("Provider initialized", logger.F("provider", provider))
	}

	if len(s.providerClients) == 0 {
		return shared_errors.InternalError("no providers enabled", nil)
	}

	return nil
}

func (s *Service) createProviderClient(provider domain.Provider, config env.ProviderConfig) (ProviderClient, error) {
	switch provider {
	case domain.ProviderAzureOpenAI:
		azureConfig := providers.AzureOpenAIConfig{
			Endpoint:    config.BaseURL,
			APIKey:      config.APIKey,
			APIVersion:  "",  // Will use default
			Deployments: make(map[string]string),  // Empty for now
		}
		return providers.NewAzureOpenAIClient(azureConfig, s.logger.WithField("provider", string(provider)))
		
	case domain.ProviderAWSBedrock:
		models := []providers.BedrockModelConfig{
			{
				ID:      "claude-3-sonnet",
				ModelID: "anthropic.claude-3-sonnet-20240229-v1:0",
				Name:    "Claude 3 Sonnet",
			},
		}
		
		bedrockConfig := providers.AWSBedrockConfig{
			Region:          "us-east-1",  // Default region
			AccessKeyID:     config.APIKey,  // Using APIKey field
			SecretAccessKey: "",  // Will be loaded from env
			SessionToken:    "",
			Models:          models,
		}
		return providers.NewAWSBedrockClient(bedrockConfig, s.logger.WithField("provider", string(provider)))
		
	default:
		// For other providers, return mock implementations for now
		return &mockProviderClient{
			provider: provider,
			logger:   s.logger.WithField("provider", string(provider)),
		}, nil
	}
}

func (s *Service) loadModelRegistry() error {
	// Load available models from all providers
	for provider, client := range s.providerClients {
		models, err := client.ListModels(context.Background())
		if err != nil {
			s.logger.Error("Failed to load models from provider",
				logger.F("provider", provider),
				logger.F("error", err))
			continue
		}

		for _, model := range models {
			s.modelRegistry[model.ModelID] = &model
		}

		s.logger.Info("Loaded models from provider",
			logger.F("provider", provider),
			logger.F("count", len(models)))
	}

	return nil
}

func (s *Service) setupRouter() {
	if s.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	s.router = gin.New()
	s.router.Use(gin.Recovery())

	// Health endpoints
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/health/ready", s.handleReadiness)

	// Internal API endpoints (called by gateway)
	api := s.router.Group("/internal/v1")
	{
		api.POST("/completions", s.handleRouteCompletion)
		api.POST("/completions/stream", s.handleRouteCompletionStream)
		api.POST("/embeddings", s.handleRouteEmbedding)
		api.GET("/models", s.handleListModels)
	}
}

func (s *Service) Handler() http.Handler {
	return s.router
}

func (s *Service) Close() error {
	// Stop health checker
	if s.healthChecker != nil {
		s.healthChecker.Stop()
	}

	// Close provider clients if they have cleanup
	// This would be implemented by actual provider clients

	return nil
}

// Route handlers

func (s *Service) handleRouteCompletion(c *gin.Context) {
	ctx := c.Request.Context()

	var req domain.CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondWithError(c, shared_errors.ValidationError("invalid request", "body"))
		return
	}

	// Select provider and route request
	response, err := s.routeCompletion(ctx, &req)
	if err != nil {
		s.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (s *Service) handleRouteCompletionStream(c *gin.Context) {
	ctx := c.Request.Context()

	var req domain.CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondWithError(c, shared_errors.ValidationError("invalid request", "body"))
		return
	}

	// Set streaming headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")

	// Route streaming request
	if err := s.routeCompletionStream(ctx, &req, c); err != nil {
		s.respondWithError(c, err)
		return
	}
}

func (s *Service) handleRouteEmbedding(c *gin.Context) {
	ctx := c.Request.Context()

	var req domain.EmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondWithError(c, shared_errors.ValidationError("invalid request", "body"))
		return
	}

	// Route embedding request
	response, err := s.routeEmbedding(ctx, &req)
	if err != nil {
		s.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (s *Service) handleListModels(c *gin.Context) {
	opts := &domain.ListModelsOptions{}

	if provider := c.Query("provider"); provider != "" {
		opts.Provider = domain.Provider(provider)
	}

	if capability := c.Query("capability"); capability != "" {
		opts.Capability = domain.Capability(capability)
	}

	models := s.listModels(opts)
	c.JSON(http.StatusOK, &domain.ModelsResponse{
		Object: "list",
		Data:   models,
	})
}

func (s *Service) handleHealth(c *gin.Context) {
	health := s.generateHealthResponse()
	
	status := http.StatusOK
	if health.Status != "healthy" {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, health)
}

func (s *Service) handleReadiness(c *gin.Context) {
	// Check if we have at least one healthy provider
	hasHealthyProvider := false
	
	s.mu.RLock()
	for _, config := range s.providerConfigs {
		if config.Enabled && config.HealthStatus == domain.ProviderHealthHealthy {
			hasHealthyProvider = true
			break
		}
	}
	s.mu.RUnlock()

	if !hasHealthyProvider {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"reason": "no_healthy_providers",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// Core routing logic

func (s *Service) routeCompletion(ctx context.Context, req *domain.CompletionRequest) (*domain.CompletionResponse, error) {
	// Generate cache key if caching is enabled
	var cacheKey string
	if req.CacheEnabled {
		cacheKey = s.generateCacheKey(req.TenantID, req)
		// TODO: Check cache first
	}

	// Select provider
	provider, err := s.selectProvider(req.Model, req.Provider)
	if err != nil {
		return nil, err
	}

	// Check circuit breaker
	if !s.circuitBreaker.CanExecute(provider) {
		return nil, shared_errors.ProviderUnavailableError(string(provider))
	}

	// Route to provider with retry logic
	client := s.providerClients[provider]
	result, err := s.executeWithRetry(ctx, func() (interface{}, error) {
		return client.CreateCompletion(ctx, req)
	}, provider)
	
	if err != nil {
		return nil, err
	}
	
	response := result.(*domain.CompletionResponse)

	s.circuitBreaker.RecordSuccess(provider)

	// Cache response if enabled
	if req.CacheEnabled && cacheKey != "" {
		// TODO: Cache the response
	}

	return response, nil
}

func (s *Service) routeCompletionStream(ctx context.Context, req *domain.CompletionRequest, c *gin.Context) error {
	// Select provider
	provider, err := s.selectProvider(req.Model, req.Provider)
	if err != nil {
		return err
	}

	// Check circuit breaker
	if !s.circuitBreaker.CanExecute(provider) {
		return shared_errors.ProviderUnavailableError(string(provider))
	}

	// Route to provider
	client := s.providerClients[provider]
	streamChan, err := client.CreateCompletionStream(ctx, req)
	if err != nil {
		s.circuitBreaker.RecordFailure(provider)
		return err
	}

	// Stream responses
	for {
		select {
		case response, ok := <-streamChan:
			if !ok {
				s.circuitBreaker.RecordSuccess(provider)
				return nil
			}

			if response.Error != nil {
				s.circuitBreaker.RecordFailure(provider)
				errorData := map[string]interface{}{
					"error": response.Error.PublicError(),
				}
				data, _ := json.Marshal(errorData)
				c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data)))
				c.Writer.Flush()
				return nil
			}

			if response.Done {
				c.Writer.Write([]byte("data: [DONE]\n\n"))
				c.Writer.Flush()
				s.circuitBreaker.RecordSuccess(provider)
				return nil
			}

			data, _ := json.Marshal(response)
			c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", data)))
			c.Writer.Flush()

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Service) routeEmbedding(ctx context.Context, req *domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	// Select provider
	provider, err := s.selectProvider(req.Model, req.Provider)
	if err != nil {
		return nil, err
	}

	// Check circuit breaker
	if !s.circuitBreaker.CanExecute(provider) {
		return nil, shared_errors.ProviderUnavailableError(string(provider))
	}

	// Route to provider with retry logic
	client := s.providerClients[provider]
	result, err := s.executeWithRetry(ctx, func() (interface{}, error) {
		return client.CreateEmbeddings(ctx, req)
	}, provider)
	
	if err != nil {
		return nil, err
	}
	
	response := result.(*domain.EmbeddingResponse)

	s.circuitBreaker.RecordSuccess(provider)
	return response, nil
}

func (s *Service) selectProvider(modelID string, preferredProvider domain.Provider) (domain.Provider, error) {
	// If provider is specified, validate and use it
	if preferredProvider != "" {
		if _, exists := s.providerClients[preferredProvider]; !exists {
			return "", shared_errors.ValidationError("invalid provider", "provider")
		}
		return preferredProvider, nil
	}

	// Find providers that support the model
	supportedProviders := []domain.Provider{}
	
	s.mu.RLock()
	for provider, config := range s.providerConfigs {
		if !config.Enabled || config.HealthStatus != domain.ProviderHealthHealthy {
			continue
		}
		
		// Check if provider supports the model
		if s.providerSupportsModel(provider, modelID) {
			supportedProviders = append(supportedProviders, provider)
		}
	}
	s.mu.RUnlock()

	if len(supportedProviders) == 0 {
		return "", shared_errors.ValidationError("no providers support the specified model", "model")
	}

	// Use load balancer to select provider
	return s.loadBalancer.SelectProvider(supportedProviders), nil
}

func (s *Service) providerSupportsModel(provider domain.Provider, modelID string) bool {
	// Check if the provider supports this model
	// This would typically check against the model registry
	model, exists := s.modelRegistry[modelID]
	if !exists {
		return false
	}
	
	return model.Provider == provider
}

func (s *Service) listModels(opts *domain.ListModelsOptions) []domain.Model {
	models := []domain.Model{}
	
	for _, model := range s.modelRegistry {
		// Filter by provider
		if opts.Provider != "" && model.Provider != opts.Provider {
			continue
		}
		
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
		
		models = append(models, *model)
	}
	
	return models
}

func (s *Service) generateHealthResponse() *domain.HealthResponse {
	response := &domain.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  make(map[string]domain.ServiceHealth),
		Providers: make(map[string]domain.ProviderHealth),
	}

	// Check provider health
	unhealthyCount := 0
	
	s.mu.RLock()
	for provider, config := range s.providerConfigs {
		health := domain.ProviderHealth{
			Status:    string(config.HealthStatus),
			Latency:   int64(config.Latency),
			ErrorRate: config.ErrorRate,
		}
		
		response.Providers[string(provider)] = health
		
		if config.HealthStatus != domain.ProviderHealthHealthy {
			unhealthyCount++
		}
	}
	s.mu.RUnlock()

	// Set overall status
	if unhealthyCount == len(s.providerConfigs) {
		response.Status = "unhealthy"
	} else if unhealthyCount > 0 {
		response.Status = "degraded"
	}

	return response
}

func (s *Service) generateCacheKey(tenantID domain.TenantID, req *domain.CompletionRequest) string {
	// Create a hash of the request for caching
	// FIXED: Include tenant ID to prevent cross-tenant data leakage
	data := fmt.Sprintf("%s:%s:%v:%v:%v:%s", 
		tenantID, req.Model, req.Messages, req.Temperature, req.MaxTokens, req.User)
	
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *Service) executeWithRetry(ctx context.Context, fn func() (interface{}, error), provider domain.Provider) (interface{}, error) {
	var result interface{}
	var lastErr error

	maxRetries := 3
	backoff := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2 // Exponential backoff
			}
		}

		result, lastErr = fn()
		if lastErr == nil {
			return result, nil
		}

		// Check if error is retryable
		if qlensErr, ok := lastErr.(*shared_errors.QLensError); ok && !qlensErr.Retryable {
			break
		}

		s.logger.Warn("Request failed, retrying",
			logger.F("provider", provider),
			logger.F("attempt", attempt+1),
			logger.F("error", lastErr))
	}

	return result, lastErr
}

func (s *Service) respondWithError(c *gin.Context, err error) {
	var qlensErr *shared_errors.QLensError
	if !errors.As(err, &qlensErr) {
		qlensErr = shared_errors.InternalError("unexpected error", err)
	}

	status := qlensErr.HTTPStatusCode()
	publicErr := qlensErr.PublicError()

	c.JSON(status, gin.H{
		"error": gin.H{
			"type":       publicErr.Type,
			"code":       publicErr.Code,
			"message":    publicErr.Message,
			"details":    publicErr.Details,
			"timestamp":  publicErr.Timestamp,
			"request_id": publicErr.RequestID,
		},
	})
}

// Helper functions for configuration parsing

func getStringFromConfig(config map[string]interface{}, key string) string {
	if value, ok := config[key].(string); ok {
		return value
	}
	return ""
}

func getMapFromConfig(config map[string]interface{}, key string) map[string]string {
	result := make(map[string]string)
	if value, ok := config[key].(map[string]interface{}); ok {
		for k, v := range value {
			if str, ok := v.(string); ok {
				result[k] = str
			}
		}
	}
	return result
}

func getStringFromMap(config map[string]interface{}, key string) string {
	if value, ok := config[key].(string); ok {
		return value
	}
	return ""
}