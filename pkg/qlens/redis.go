package qlens

import (
	"context"
	"fmt"
	"time"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/redis/go-redis/v9"
)

// RedisClientImpl implements the RedisClient interface using go-redis
type RedisClientImpl struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client
func NewRedisClient(opts *redis.Options) *RedisClientImpl {
	rdb := redis.NewClient(opts)
	return &RedisClientImpl{
		client: rdb,
	}
}

// NewRedisClientFromURL creates a new Redis client from URL
func NewRedisClientFromURL(url string) (*RedisClientImpl, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	
	return NewRedisClient(opts), nil
}

// Get implements the RedisClient interface
func (r *RedisClientImpl) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Set implements the RedisClient interface
func (r *RedisClientImpl) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Del implements the RedisClient interface
func (r *RedisClientImpl) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// FlushAll implements the RedisClient interface
func (r *RedisClientImpl) FlushAll(ctx context.Context) error {
	return r.client.FlushAll(ctx).Err()
}

// Close implements the RedisClient interface
func (r *RedisClientImpl) Close() error {
	return r.client.Close()
}

// Ping tests the Redis connection
func (r *RedisClientImpl) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// GetClient returns the underlying Redis client for advanced operations
func (r *RedisClientImpl) GetClient() *redis.Client {
	return r.client
}

// Helper function to create QLens client with Redis caching
func NewWithRedisCache(redisURL, openAIKey string, opts ...ClientOption) (*QLens, error) {
	// Create Redis client
	redisClient, err := NewRedisClientFromURL(redisURL)
	if err != nil {
		return nil, err
	}
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx); err != nil {
		redisClient.Close()
		return nil, err
	}
	
	// Create Redis cache
	cache := NewRedisCache(redisClient, "qlens")
	
	// Create QLens client with custom cache
	config := DefaultClientConfig()
	config.CacheEnabled = true
	
	// Apply options
	for _, opt := range opts {
		opt(config)
	}
	
	// Add OpenAI provider
	if openAIKey != "" {
		config.Providers[domain.ProviderOpenAI] = ProviderConfig{
			Provider: domain.ProviderOpenAI,
			APIKey:   openAIKey,
			Enabled:  true,
			Priority: 1,
			Timeout:  30 * time.Second,
		}
		config.DefaultProvider = domain.ProviderOpenAI
	}
	
	client := &QLens{
		config:    config,
		providers: make(map[domain.Provider]ProviderClient),
		cache:     cache, // Use Redis cache instead of in-memory
		startTime: time.Now(),
	}
	
	// Initialize router
	client.router = NewDefaultRouter(config)
	
	// Initialize metrics collector
	if config.MetricsEnabled {
		client.metrics = NewMetricsCollector()
	}
	
	// Initialize providers
	if err := client.initializeProviders(); err != nil {
		client.Close()
		return nil, err
	}
	
	return client, nil
}