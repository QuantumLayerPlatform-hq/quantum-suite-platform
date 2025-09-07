package qlens

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/qlens-types"
)

// Cache interface is defined in interfaces.go


// CacheEntry represents a cached entry
type CacheEntry struct {
	Key          string                 `json:"key"`
	Data         interface{}            `json:"data"`
	CreatedAt    time.Time              `json:"created_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	TTL          time.Duration          `json:"ttl"`
	AccessCount  int64                  `json:"access_count"`
	LastAccessed time.Time              `json:"last_accessed"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// InMemoryCache implements the Cache interface using in-memory storage
type InMemoryCache struct {
	mu          sync.RWMutex
	entries     map[string]*CacheEntry
	maxSize     int
	stats       types.CacheStats
	stopCleanup chan struct{}
	cleanupOnce sync.Once
}

// RedisCache implements the Cache interface using Redis
type RedisCache struct {
	client   RedisClient
	keyPrefix string
	stats     types.CacheStats
	mu        sync.RWMutex
}

// RedisClient interface for Redis operations
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	FlushAll(ctx context.Context) error
	Close() error
}

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache(maxSize int) *InMemoryCache {
	cache := &InMemoryCache{
		entries:     make(map[string]*CacheEntry),
		maxSize:     maxSize,
		stopCleanup: make(chan struct{}),
	}
	
	// Start cleanup goroutine
	cache.startCleanup()
	
	return cache
}

// NewRedisCache creates a new Redis-backed cache
func NewRedisCache(client RedisClient, keyPrefix string) *RedisCache {
	return &RedisCache{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// InMemoryCache implementation

func (c *InMemoryCache) Get(ctx context.Context, key string) (*types.CompletionResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.entries[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}
	
	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		c.mu.RUnlock()
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		c.mu.RLock()
		c.stats.Misses++
		return nil, false
	}
	
	// Update access stats
	entry.AccessCount++
	entry.LastAccessed = time.Now()
	c.stats.Hits++
	
	if response, ok := entry.Data.(*types.CompletionResponse); ok {
		// Mark as cache hit
		responseCopy := *response
		responseCopy.CacheHit = true
		return &responseCopy, true
	}
	
	c.stats.Misses++
	return nil, false
}

func (c *InMemoryCache) Set(ctx context.Context, key string, response *types.CompletionResponse, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check size limit and evict if necessary
	if len(c.entries) >= c.maxSize {
		c.evictLRU()
	}
	
	now := time.Now()
	entry := &CacheEntry{
		Key:          key,
		Data:         response,
		CreatedAt:    now,
		ExpiresAt:    now.Add(ttl),
		TTL:          ttl,
		AccessCount:  0,
		LastAccessed: now,
		Metadata: map[string]interface{}{
			"type": "completion",
			"tokens_saved": response.Usage.TotalTokens,
			"cost_saved":   response.Usage.CostUSD,
		},
	}
	
	c.entries[key] = entry
	return nil
}

func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.entries, key)
	return nil
}

func (c *InMemoryCache) GetEmbedding(ctx context.Context, key string) (*types.EmbeddingResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.entries[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}
	
	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		c.mu.RUnlock()
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		c.mu.RLock()
		c.stats.Misses++
		return nil, false
	}
	
	// Update access stats
	entry.AccessCount++
	entry.LastAccessed = time.Now()
	c.stats.Hits++
	
	if response, ok := entry.Data.(*types.EmbeddingResponse); ok {
		return response, true
	}
	
	c.stats.Misses++
	return nil, false
}

func (c *InMemoryCache) SetEmbedding(ctx context.Context, key string, response *types.EmbeddingResponse, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check size limit and evict if necessary
	if len(c.entries) >= c.maxSize {
		c.evictLRU()
	}
	
	now := time.Now()
	entry := &CacheEntry{
		Key:          key,
		Data:         response,
		CreatedAt:    now,
		ExpiresAt:    now.Add(ttl),
		TTL:          ttl,
		AccessCount:  0,
		LastAccessed: now,
		Metadata: map[string]interface{}{
			"type": "embedding",
			"tokens_saved": response.Usage.TotalTokens,
			"cost_saved":   response.Usage.CostUSD,
		},
	}
	
	c.entries[key] = entry
	return nil
}

func (c *InMemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.entries = make(map[string]*CacheEntry)
	return nil
}


func (c *InMemoryCache) Close() error {
	c.cleanupOnce.Do(func() {
		close(c.stopCleanup)
	})
	return nil
}

// Configure implements the Cache interface for InMemoryCache
func (c *InMemoryCache) Configure(config types.CacheConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Update configuration as needed
	// For now, this is a placeholder
	return nil
}

// HealthCheck implements the Cache interface for InMemoryCache
func (c *InMemoryCache) HealthCheck(ctx context.Context) error {
	// Check if cache is operational
	return nil
}

// Stats implements the Cache interface for InMemoryCache
func (c *InMemoryCache) Stats() types.CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	stats := c.stats
	stats.Size = len(c.entries)
	if stats.Hits+stats.Misses > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Hits+stats.Misses)
	}
	
	return stats
}

func (c *InMemoryCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time = time.Now()
	
	for key, entry := range c.entries {
		if entry.LastAccessed.Before(oldestTime) {
			oldestTime = entry.LastAccessed
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.stats.Evictions++
	}
}

func (c *InMemoryCache) startCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				c.cleanup()
			case <-c.stopCleanup:
				return
			}
		}
	}()
}

func (c *InMemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
			c.stats.Evictions++
		}
	}
}

// RedisCache implementation

func (c *RedisCache) Get(ctx context.Context, key string) (*types.CompletionResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	fullKey := c.keyPrefix + ":" + key
	data, err := c.client.Get(ctx, fullKey)
	if err != nil || data == "" {
		c.stats.Misses++
		return nil, false
	}
	
	var response types.CompletionResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		c.stats.Misses++
		return nil, false
	}
	
	c.stats.Hits++
	response.CacheHit = true
	return &response, true
}

func (c *RedisCache) Set(ctx context.Context, key string, response *types.CompletionResponse, ttl time.Duration) error {
	fullKey := c.keyPrefix + ":" + key
	
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}
	
	return c.client.Set(ctx, fullKey, data, ttl)
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := c.keyPrefix + ":" + key
	return c.client.Del(ctx, fullKey)
}

func (c *RedisCache) GetEmbedding(ctx context.Context, key string) (*types.EmbeddingResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	fullKey := c.keyPrefix + ":emb:" + key
	data, err := c.client.Get(ctx, fullKey)
	if err != nil || data == "" {
		c.stats.Misses++
		return nil, false
	}
	
	var response types.EmbeddingResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		c.stats.Misses++
		return nil, false
	}
	
	c.stats.Hits++
	return &response, true
}

func (c *RedisCache) SetEmbedding(ctx context.Context, key string, response *types.EmbeddingResponse, ttl time.Duration) error {
	fullKey := c.keyPrefix + ":emb:" + key
	
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding response: %w", err)
	}
	
	return c.client.Set(ctx, fullKey, data, ttl)
}

func (c *RedisCache) Clear(ctx context.Context) error {
	return c.client.FlushAll(ctx)
}

func (c *RedisCache) Stats() types.CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	stats := c.stats
	if stats.Hits+stats.Misses > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Hits+stats.Misses)
	}
	
	return stats
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

// Configure implements the Cache interface for RedisCache
func (c *RedisCache) Configure(config types.CacheConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Update configuration as needed
	// For now, this is a placeholder
	return nil
}

// HealthCheck implements the Cache interface for RedisCache
func (c *RedisCache) HealthCheck(ctx context.Context) error {
	// Check Redis connection by attempting a simple operation
	_, err := c.client.Get(ctx, "health_check")
	// Ignore "key not found" errors as they indicate Redis is responsive
	if err != nil && err.Error() != "redis: nil" {
		return err
	}
	return nil
}

// Cache key generation

// GenerateCompletionCacheKey creates a cache key for completion requests
func GenerateCompletionCacheKey(req *types.CompletionRequest) string {
	// Create a normalized request for hashing
	normalizedReq := struct {
		Model            string              `json:"model"`
		Messages         []domain.Message    `json:"messages"`
		MaxTokens        *int                `json:"max_tokens,omitempty"`
		Temperature      *float64            `json:"temperature,omitempty"`
		TopP             *float64            `json:"top_p,omitempty"`
		Stop             []string            `json:"stop,omitempty"`
		PresencePenalty  *float64            `json:"presence_penalty,omitempty"`
		FrequencyPenalty *float64            `json:"frequency_penalty,omitempty"`
	}{
		Model:            req.Model,
		Messages:         req.Messages,
		MaxTokens:        req.MaxTokens,
		Temperature:      req.Temperature,
		TopP:             req.TopP,
		Stop:             req.Stop,
		PresencePenalty:  req.PresencePenalty,
		FrequencyPenalty: req.FrequencyPenalty,
	}
	
	data, _ := json.Marshal(normalizedReq)
	hash := md5.Sum(data)
	return "completion:" + hex.EncodeToString(hash[:])
}

// GenerateEmbeddingCacheKey creates a cache key for embedding requests
func GenerateEmbeddingCacheKey(req *types.EmbeddingRequest) string {
	// Create a normalized request for hashing
	normalizedReq := struct {
		Model          string   `json:"model"`
		Input          []string `json:"input"`
		EncodingFormat string   `json:"encoding_format,omitempty"`
		Dimensions     *int     `json:"dimensions,omitempty"`
	}{
		Model:          req.Model,
		Input:          req.Input,
		EncodingFormat: req.EncodingFormat,
		Dimensions:     req.Dimensions,
	}
	
	data, _ := json.Marshal(normalizedReq)
	hash := md5.Sum(data)
	return "embedding:" + hex.EncodeToString(hash[:])
}

// ShouldCache determines if a request should be cached based on configuration
func ShouldCache(req *types.CompletionRequest, config *types.ClientConfig) bool {
	if !config.CacheEnabled {
		return false
	}
	
	// Don't cache streaming requests
	if req.Stream {
		return false
	}
	
	// Don't cache requests with user-specific data
	if req.User != "" {
		return false
	}
	
	// Don't cache requests with randomness (high temperature)
	if req.Temperature != nil && *req.Temperature > 0.8 {
		return false
	}
	
	// Don't cache if explicitly disabled in request
	if !req.CacheEnabled {
		return false
	}
	
	return true
}

// ShouldCacheEmbedding determines if an embedding request should be cached
func ShouldCacheEmbedding(req *types.EmbeddingRequest, config *types.ClientConfig) bool {
	if !config.CacheEnabled {
		return false
	}
	
	// Don't cache requests with user-specific data
	if req.User != "" {
		return false
	}
	
	return true
}

// CacheMiddleware wraps operations with caching logic
func CacheMiddleware(cache Cache, config *types.ClientConfig) func(next CompletionFunc) CompletionFunc {
	return func(next CompletionFunc) CompletionFunc {
		return func(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error) {
			// Check if caching is enabled for this request
			if !ShouldCache(req, config) {
				return next(ctx, req)
			}
			
			// Generate cache key
			key := GenerateCompletionCacheKey(req)
			
			// Try to get from cache
			if cached, found := cache.Get(ctx, key); found {
				return cached, nil
			}
			
			// Call the next function
			response, err := next(ctx, req)
			if err != nil {
				return nil, err
			}
			
			// Cache the response
			ttl := config.CacheDefaultTTL
			if req.CacheTTL > 0 {
				ttl = req.CacheTTL
			}
			
			_ = cache.Set(ctx, key, response, ttl)
			
			return response, nil
		}
	}
}

// CompletionFunc represents a completion function signature
type CompletionFunc func(ctx context.Context, req *types.CompletionRequest) (*types.CompletionResponse, error)

// EmbeddingFunc represents an embedding function signature
type EmbeddingFunc func(ctx context.Context, req *types.EmbeddingRequest) (*types.EmbeddingResponse, error)

// EmbeddingCacheMiddleware wraps embedding operations with caching logic
func EmbeddingCacheMiddleware(cache Cache, config *types.ClientConfig) func(next EmbeddingFunc) EmbeddingFunc {
	return func(next EmbeddingFunc) EmbeddingFunc {
		return func(ctx context.Context, req *types.EmbeddingRequest) (*types.EmbeddingResponse, error) {
			// Check if caching is enabled for this request
			if !ShouldCacheEmbedding(req, config) {
				return next(ctx, req)
			}
			
			// Generate cache key
			key := GenerateEmbeddingCacheKey(req)
			
			// Try to get from cache
			if cached, found := cache.GetEmbedding(ctx, key); found {
				return cached, nil
			}
			
			// Call the next function
			response, err := next(ctx, req)
			if err != nil {
				return nil, err
			}
			
			// Cache the response
			ttl := config.CacheDefaultTTL
			_ = cache.SetEmbedding(ctx, key, response, ttl)
			
			return response, nil
		}
	}
}