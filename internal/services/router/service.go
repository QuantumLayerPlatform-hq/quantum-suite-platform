package router

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/internal/providers"
	"github.com/quantum-suite/platform/pkg/shared/env"
	"github.com/quantum-suite/platform/pkg/shared/errors"
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
	CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
	CreateCompletionStream(ctx context.Context, req *CompletionRequest) (<-chan *StreamResponse, error)
	CreateEmbeddings(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)
	ListModels(ctx context.Context) ([]domain.Model, error)
	HealthCheck(ctx context.Context) error
}

// Request/Response types (same as gateway service)
type CompletionRequest struct {
	TenantID         domain.TenantID            `json:"tenant_id"`
	UserID           domain.UserID              `json:"user_id"`
	Provider         domain.Provider            `json:"provider,omitempty"`
	Model            string                     `json:"model"`
	Messages         []domain.Message           `json:"messages"`
	MaxTokens        *int                       `json:"max_tokens,omitempty"`
	Temperature      *float64                   `json:"temperature,omitempty"`
	TopP             *float64                   `json:"top_p,omitempty"`
	Stream           bool                       `json:"stream"`
	Stop             []string                   `json:"stop,omitempty"`
	PresencePenalty  *float64                   `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64                   `json:"frequency_penalty,omitempty"`
	User             string                     `json:"user,omitempty"`
	RequestID        string                     `json:"request_id"`
	Priority         domain.Priority            `json:"priority"`
	CacheEnabled     bool                       `json:"cache_enabled"`
	CacheTTL         time.Duration              `json:"cache_ttl"`
	Metadata         map[string]interface{}     `json:"metadata,omitempty"`
}

type CompletionResponse struct {
	ID       string                  `json:"id"`
	Object   string                  `json:"object"`
	Created  int64                   `json:"created"`
	Model    string                  `json:"model"`
	Provider domain.Provider         `json:"provider"`
	Choices  []domain.Choice         `json:"choices"`
	Usage    domain.Usage            `json:"usage"`
	Metadata map[string]interface{}  `json:"metadata,omitempty"`
}

type StreamResponse struct {
	ID       string                  `json:"id,omitempty"`
	Object   string                  `json:"object,omitempty"`
	Created  int64                   `json:"created,omitempty"`
	Model    string                  `json:"model,omitempty"`
	Provider domain.Provider         `json:"provider,omitempty"`
	Choices  []domain.Choice         `json:"choices,omitempty"`
	Done     bool                    `json:"done,omitempty"`
	Error    *errors.QLensError      `json:"error,omitempty"`
}

type EmbeddingRequest struct {
	TenantID       domain.TenantID  `json:"tenant_id"`
	UserID         domain.UserID    `json:"user_id"`
	Provider       domain.Provider  `json:"provider,omitempty"`
	Model          string           `json:"model"`
	Input          []string         `json:"input"`
	EncodingFormat string           `json:"encoding_format,omitempty"`
	Dimensions     *int             `json:"dimensions,omitempty"`
	User           string           `json:"user,omitempty"`
	RequestID      string           `json:"request_id"`
	Priority       domain.Priority  `json:"priority"`
}

type EmbeddingResponse struct {
	Object   string              `json:"object"`
	Data     []domain.Embedding  `json:"data"`
	Model    string              `json:"model"`
	Provider domain.Provider     `json:"provider"`
	Usage    domain.EmbeddingUsage `json:"usage"`
}

type ListModelsOptions struct {
	Provider   domain.Provider   `json:"provider,omitempty"`
	Capability domain.Capability `json:"capability,omitempty"`
}

type ModelsResponse struct {
	Object string         `json:"object"`
	Data   []domain.Model `json:"data"`
}

type HealthResponse struct {
	Status    string                             `json:"status"`
	Timestamp time.Time                          `json:"timestamp"`
	Services  map[string]ServiceHealth           `json:"services"`
	Providers map[string]ProviderHealth          `json:"providers"`
}

type ServiceHealth struct {
	Status  string `json:"status"`
	Latency int64  `json:"latency_ms"`
}

type ProviderHealth struct {
	Status    string  `json:"status"`
	Latency   int64   `json:"latency_ms"`
	ErrorRate float64 `json:"error_rate"`
}

func NewService(config *env.Config, log logger.Logger) (*Service, error) {
	service := &Service{
		config:          config,
		logger:          log.WithService("router"),
		providerClients: make(map[domain.Provider]ProviderClient),
		providerConfigs: make(map[domain.Provider]*domain.ProviderConfig),
		modelRegistry:   make(map[string]*domain.Model),
	}

	// Initialize components
	if err := service.initializeComponents(); err != nil {
		return nil, errors.InternalError("failed to initialize router components", err)
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
		config.Config = providerConfig.Config
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
		return errors.InternalError("no providers enabled", nil)
	}

	return nil
}

func (s *Service) createProviderClient(provider domain.Provider, config env.ProviderConfig) (ProviderClient, error) {
	switch provider {
	case domain.ProviderAzureOpenAI:
		azureConfig := providers.AzureOpenAIConfig{
			Endpoint:    getStringFromConfig(config.Config, "endpoint"),
			APIKey:      getStringFromConfig(config.Config, "api_key"),
			APIVersion:  getStringFromConfig(config.Config, "api_version"),
			Deployments: getMapFromConfig(config.Config, "deployments"),
		}
		return providers.NewAzureOpenAIClient(azureConfig, s.logger.WithProvider(string(provider)))
		
	case domain.ProviderAWSBedrock:
		models := []providers.BedrockModelConfig{}
		if modelsConfig, ok := config.Config["models"].([]interface{}); ok {
			for _, modelConfig := range modelsConfig {
				if modelMap, ok := modelConfig.(map[string]interface{}); ok {
					models = append(models, providers.BedrockModelConfig{
						ID:      getStringFromMap(modelMap, "id"),
						ModelID: getStringFromMap(modelMap, "model_id"),
						Name:    getStringFromMap(modelMap, "name"),
					})
				}
			}
		}
		
		bedrockConfig := providers.AWSBedrockConfig{
			Region:          getStringFromConfig(config.Config, "region"),
			AccessKeyID:     getStringFromConfig(config.Config, "access_key_id"),
			SecretAccessKey: getStringFromConfig(config.Config, "secret_access_key"),
			SessionToken:    getStringFromConfig(config.Config, "session_token"),
			Models:          models,
		}
		return providers.NewAWSBedrockClient(bedrockConfig, s.logger.WithProvider(string(provider)))
		
	default:
		// For other providers, return mock implementations for now
		return &mockProviderClient{
			provider: provider,
			logger:   s.logger.WithProvider(string(provider)),
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
	if s.config.Environment == env.EnvironmentProduction {
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

	var req CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondWithError(c, errors.ValidationError("invalid request", "body"))
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

	var req CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondWithError(c, errors.ValidationError("invalid request", "body"))
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

	var req EmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondWithError(c, errors.ValidationError("invalid request", "body"))
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
	opts := &ListModelsOptions{}

	if provider := c.Query("provider"); provider != "" {
		opts.Provider = domain.Provider(provider)
	}

	if capability := c.Query("capability"); capability != "" {
		opts.Capability = domain.Capability(capability)
	}

	models := s.listModels(opts)
	c.JSON(http.StatusOK, &ModelsResponse{
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
	for provider, config := range s.providerConfigs {
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

func (s *Service) routeCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
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
		return nil, errors.ProviderUnavailableError(string(provider))
	}

	// Route to provider with retry logic
	client := s.providerClients[provider]
	response, err := s.executeWithRetry(ctx, func() (*CompletionResponse, error) {
		return client.CreateCompletion(ctx, req)
	}, provider)

	if err != nil {
		s.circuitBreaker.RecordFailure(provider)
		return nil, err
	}

	s.circuitBreaker.RecordSuccess(provider)

	// Cache response if enabled
	if req.CacheEnabled && cacheKey != "" {
		// TODO: Cache the response
	}

	return response, nil
}

func (s *Service) routeCompletionStream(ctx context.Context, req *CompletionRequest, c *gin.Context) error {
	// Select provider
	provider, err := s.selectProvider(req.Model, req.Provider)
	if err != nil {
		return err
	}

	// Check circuit breaker
	if !s.circuitBreaker.CanExecute(provider) {
		return errors.ProviderUnavailableError(string(provider))
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

func (s *Service) routeEmbedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	// Select provider
	provider, err := s.selectProvider(req.Model, req.Provider)
	if err != nil {
		return nil, err
	}

	// Check circuit breaker
	if !s.circuitBreaker.CanExecute(provider) {
		return nil, errors.ProviderUnavailableError(string(provider))
	}

	// Route to provider with retry logic
	client := s.providerClients[provider]
	response, err := s.executeWithRetry(ctx, func() (*EmbeddingResponse, error) {
		return client.CreateEmbeddings(ctx, req)
	}, provider)

	if err != nil {
		s.circuitBreaker.RecordFailure(provider)
		return nil, err
	}

	s.circuitBreaker.RecordSuccess(provider)
	return response, nil
}

func (s *Service) selectProvider(modelID string, preferredProvider domain.Provider) (domain.Provider, error) {
	// If provider is specified, validate and use it
	if preferredProvider != "" {
		if _, exists := s.providerClients[preferredProvider]; !exists {
			return "", errors.ValidationError("invalid provider", "provider")
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
		return "", errors.ValidationError("no providers support the specified model", "model")
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

func (s *Service) listModels(opts *ListModelsOptions) []domain.Model {
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

func (s *Service) generateHealthResponse() *HealthResponse {
	response := &HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  make(map[string]ServiceHealth),
		Providers: make(map[string]ProviderHealth),
	}

	// Check provider health
	unhealthyCount := 0
	
	s.mu.RLock()
	for provider, config := range s.providerConfigs {
		health := ProviderHealth{
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

func (s *Service) generateCacheKey(tenantID domain.TenantID, req *CompletionRequest) string {
	// Create a hash of the request for caching
	// FIXED: Include tenant ID to prevent cross-tenant data leakage
	data := fmt.Sprintf("%s:%s:%v:%v:%v:%s", 
		tenantID, req.Model, req.Messages, req.Temperature, req.MaxTokens, req.User)
	
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *Service) executeWithRetry[T any](ctx context.Context, fn func() (T, error), provider domain.Provider) (T, error) {
	var result T
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
		if qlensErr, ok := lastErr.(*errors.QLensError); ok && !qlensErr.Retryable {
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
	var qlensErr *errors.QLensError
	if !errors.Is(err, &qlensErr) {
		qlensErr = errors.InternalError("unexpected error", err)
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