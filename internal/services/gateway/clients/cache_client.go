package clients

import (
	"context"
	"sync"
	"time"

	"github.com/quantum-suite/platform/pkg/shared/logger"
)

// CacheEntry represents a cached item with expiration
type CacheEntry struct {
	Value     interface{}
	ExpiresAt time.Time
}

// IsExpired checks if the cache entry has expired
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// SimpleCacheClient implements CacheClient interface using in-memory storage
type SimpleCacheClient struct {
	cache  map[string]*CacheEntry
	mu     sync.RWMutex
	logger logger.Logger
}

// NewSimpleCacheClient creates a new in-memory cache client
func NewSimpleCacheClient(log logger.Logger) *SimpleCacheClient {
	client := &SimpleCacheClient{
		cache:  make(map[string]*CacheEntry),
		logger: log.WithField("component", "cache_client"),
	}
	
	// Start cleanup goroutine
	go client.cleanup()
	
	return client
}

// Get retrieves a value from cache
func (c *SimpleCacheClient) Get(ctx context.Context, key string) (interface{}, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.cache[key]
	if !exists {
		c.logger.Debug("Cache miss", logger.F("key", key))
		return nil, false, nil
	}
	
	if entry.IsExpired() {
		c.logger.Debug("Cache hit but expired", logger.F("key", key))
		return nil, false, nil
	}
	
	c.logger.Debug("Cache hit", logger.F("key", key))
	return entry.Value, true, nil
}

// Set stores a value in cache with TTL
func (c *SimpleCacheClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	expiresAt := time.Now().Add(ttl)
	c.cache[key] = &CacheEntry{
		Value:     value,
		ExpiresAt: expiresAt,
	}
	
	c.logger.Debug("Cache set",
		logger.F("key", key),
		logger.F("ttl", ttl),
		logger.F("expires_at", expiresAt))
	
	return nil
}

// Delete removes a value from cache
func (c *SimpleCacheClient) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.cache, key)
	c.logger.Debug("Cache delete", logger.F("key", key))
	
	return nil
}

// Clear removes all entries from cache
func (c *SimpleCacheClient) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache = make(map[string]*CacheEntry)
	c.logger.Debug("Cache cleared")
	
	return nil
}

// Stats returns cache statistics
func (c *SimpleCacheClient) Stats(ctx context.Context) map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	total := len(c.cache)
	expired := 0
	
	for _, entry := range c.cache {
		if entry.IsExpired() {
			expired++
		}
	}
	
	return map[string]interface{}{
		"total_entries":   total,
		"expired_entries": expired,
		"active_entries":  total - expired,
		"cache_type":      "in-memory",
	}
}

// cleanup removes expired entries periodically
func (c *SimpleCacheClient) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mu.Lock()
		
		removed := 0
		for key, entry := range c.cache {
			if entry.IsExpired() {
				delete(c.cache, key)
				removed++
			}
		}
		
		if removed > 0 {
			c.logger.Debug("Cache cleanup completed",
				logger.F("removed_entries", removed),
				logger.F("remaining_entries", len(c.cache)))
		}
		
		c.mu.Unlock()
	}
}