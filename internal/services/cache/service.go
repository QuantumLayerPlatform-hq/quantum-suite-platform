package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/shared/env"
	"github.com/quantum-suite/platform/pkg/shared/errors"
	"github.com/quantum-suite/platform/pkg/shared/logger"
)

type Service struct {
	config *env.Config
	logger logger.Logger
	router *gin.Engine
	store  CacheStore
}

// CacheStore interface for different cache implementations
type CacheStore interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	Stats(ctx context.Context) (*CacheStats, error)
}

type CacheStats struct {
	Keys     int64   `json:"keys"`
	Hits     int64   `json:"hits"`
	Misses   int64   `json:"misses"`
	HitRate  float64 `json:"hit_rate"`
	Size     int64   `json:"size_bytes"`
	MaxSize  int64   `json:"max_size_bytes,omitempty"`
	Evicted  int64   `json:"evicted"`
}

type CacheRequest struct {
	Key   string        `json:"key" binding:"required"`
	Value interface{}   `json:"value" binding:"required"`
	TTL   time.Duration `json:"ttl,omitempty"`
}

type CacheResponse struct {
	Key      string      `json:"key"`
	Value    interface{} `json:"value,omitempty"`
	Found    bool        `json:"found"`
	Cached   bool        `json:"cached,omitempty"`
	TTL      time.Duration `json:"ttl,omitempty"`
}

type HealthResponse struct {
	Status    string     `json:"status"`
	Timestamp time.Time  `json:"timestamp"`
	Stats     CacheStats `json:"stats"`
}

func NewService(config *env.Config, log logger.Logger) (*Service, error) {
	service := &Service{
		config: config,
		logger: log.WithService("cache"),
	}

	// Initialize cache store
	if err := service.initializeStore(); err != nil {
		return nil, errors.InternalError("failed to initialize cache store", err)
	}

	// Setup router
	service.setupRouter()

	return service, nil
}

func (s *Service) initializeStore() error {
	switch s.config.CacheType {
	case "redis":
		// In production, would initialize Redis client
		s.logger.Info("Redis cache not yet implemented, using memory cache")
		s.store = NewMemoryStore(s.logger)
	case "memory", "":
		s.store = NewMemoryStore(s.logger)
		s.logger.Info("Using in-memory cache store")
	default:
		return fmt.Errorf("unsupported cache type: %s", s.config.CacheType)
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

	// Internal cache API
	api := s.router.Group("/internal/v1/cache")
	{
		api.GET("/:key", s.handleGet)
		api.POST("", s.handleSet)
		api.DELETE("/:key", s.handleDelete)
		api.DELETE("", s.handleClear)
		api.GET("/stats", s.handleStats)
	}
}

func (s *Service) Handler() http.Handler {
	return s.router
}

func (s *Service) Close() error {
	// Close cache store if it has cleanup
	return nil
}

// Handlers

func (s *Service) handleHealth(c *gin.Context) {
	ctx := c.Request.Context()
	
	stats, err := s.store.Stats(ctx)
	if err != nil {
		s.respondWithError(c, errors.InternalError("failed to get cache stats", err))
		return
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Stats:     *stats,
	}

	c.JSON(http.StatusOK, response)
}

func (s *Service) handleReadiness(c *gin.Context) {
	// Simple readiness check
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func (s *Service) handleGet(c *gin.Context) {
	ctx := c.Request.Context()
	key := c.Param("key")

	if key == "" {
		s.respondWithError(c, errors.ValidationError("key is required", "key"))
		return
	}

	// FIXED: Validate tenant isolation in cache key
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		s.respondWithError(c, errors.ValidationError("X-Tenant-ID header is required", "tenant_id"))
		return
	}

	// Create tenant-scoped cache key
	scopedKey := fmt.Sprintf("tenant:%s:%s", tenantID, key)

	value, found, err := s.store.Get(ctx, scopedKey)
	if err != nil {
		s.respondWithError(c, errors.InternalError("cache get failed", err))
		return
	}

	response := CacheResponse{
		Key:   key,
		Found: found,
	}

	if found && len(value) > 0 {
		var data interface{}
		if err := json.Unmarshal(value, &data); err != nil {
			s.logger.Error("Failed to unmarshal cached value",
				logger.F("key", scopedKey),
				logger.F("error", err))
		} else {
			response.Value = data
		}
	}

	s.logger.Debug("Cache get",
		logger.F("key", key),
		logger.F("scoped_key", scopedKey),
		logger.F("hit", found),
		logger.F("tenant_id", tenantID))

	c.JSON(http.StatusOK, response)
}

func (s *Service) handleSet(c *gin.Context) {
	ctx := c.Request.Context()

	var req CacheRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondWithError(c, errors.ValidationError("invalid request format", "body"))
		return
	}

	// FIXED: Validate tenant isolation
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		s.respondWithError(c, errors.ValidationError("X-Tenant-ID header is required", "tenant_id"))
		return
	}

	// Create tenant-scoped cache key
	scopedKey := fmt.Sprintf("tenant:%s:%s", tenantID, req.Key)

	// Marshal value to JSON
	value, err := json.Marshal(req.Value)
	if err != nil {
		s.respondWithError(c, errors.ValidationError("invalid value format", "value"))
		return
	}

	// Set default TTL if not provided
	ttl := req.TTL
	if ttl == 0 {
		ttl = 15 * time.Minute // Default 15 minutes
	}

	if err := s.store.Set(ctx, scopedKey, value, ttl); err != nil {
		s.respondWithError(c, errors.InternalError("cache set failed", err))
		return
	}

	s.logger.Debug("Cache set",
		logger.F("key", req.Key),
		logger.F("scoped_key", scopedKey),
		logger.F("ttl", ttl),
		logger.F("tenant_id", tenantID))

	response := CacheResponse{
		Key:    req.Key,
		Cached: true,
		TTL:    ttl,
	}

	c.JSON(http.StatusOK, response)
}

func (s *Service) handleDelete(c *gin.Context) {
	ctx := c.Request.Context()
	key := c.Param("key")

	if key == "" {
		s.respondWithError(c, errors.ValidationError("key is required", "key"))
		return
	}

	// FIXED: Validate tenant isolation
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		s.respondWithError(c, errors.ValidationError("X-Tenant-ID header is required", "tenant_id"))
		return
	}

	// Create tenant-scoped cache key
	scopedKey := fmt.Sprintf("tenant:%s:%s", tenantID, key)

	if err := s.store.Delete(ctx, scopedKey); err != nil {
		s.respondWithError(c, errors.InternalError("cache delete failed", err))
		return
	}

	s.logger.Debug("Cache delete",
		logger.F("key", key),
		logger.F("scoped_key", scopedKey),
		logger.F("tenant_id", tenantID))

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (s *Service) handleClear(c *gin.Context) {
	ctx := c.Request.Context()

	// In a real implementation, we'd only clear keys for the specific tenant
	// For now, this is an admin operation
	if err := s.store.Clear(ctx); err != nil {
		s.respondWithError(c, errors.InternalError("cache clear failed", err))
		return
	}

	s.logger.Info("Cache cleared")
	c.JSON(http.StatusOK, gin.H{"cleared": true})
}

func (s *Service) handleStats(c *gin.Context) {
	ctx := c.Request.Context()

	stats, err := s.store.Stats(ctx)
	if err != nil {
		s.respondWithError(c, errors.InternalError("failed to get cache stats", err))
		return
	}

	c.JSON(http.StatusOK, stats)
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

// MemoryStore implements CacheStore using in-memory storage
type MemoryStore struct {
	logger logger.Logger
	data   map[string]*cacheEntry
	stats  *cacheStats
	mu     sync.RWMutex
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
	createdAt time.Time
}

type cacheStats struct {
	hits    int64
	misses  int64
	evicted int64
	mu      sync.RWMutex
}

func NewMemoryStore(log logger.Logger) *MemoryStore {
	store := &MemoryStore{
		logger: log.WithField("component", "memory_store"),
		data:   make(map[string]*cacheEntry),
		stats:  &cacheStats{},
	}

	// Start cleanup goroutine
	go store.cleanupLoop()

	return store
}

func (m *MemoryStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	m.mu.RLock()
	entry, exists := m.data[key]
	m.mu.RUnlock()

	if !exists || (entry.expiresAt.Before(time.Now()) && !entry.expiresAt.IsZero()) {
		m.stats.mu.Lock()
		m.stats.misses++
		m.stats.mu.Unlock()
		
		// Clean up expired entry
		if exists {
			m.mu.Lock()
			delete(m.data, key)
			m.mu.Unlock()
		}
		
		return nil, false, nil
	}

	m.stats.mu.Lock()
	m.stats.hits++
	m.stats.mu.Unlock()

	// Return a copy to prevent modifications
	valueCopy := make([]byte, len(entry.value))
	copy(valueCopy, entry.value)
	
	return valueCopy, true, nil
}

func (m *MemoryStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	entry := &cacheEntry{
		value:     make([]byte, len(value)),
		createdAt: time.Now(),
	}
	
	// Copy value to prevent external modifications
	copy(entry.value, value)
	
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}

	m.mu.Lock()
	m.data[key] = entry
	m.mu.Unlock()

	return nil
}

func (m *MemoryStore) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	delete(m.data, key)
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) Clear(ctx context.Context) error {
	m.mu.Lock()
	m.data = make(map[string]*cacheEntry)
	m.mu.Unlock()
	
	m.stats.mu.Lock()
	m.stats.hits = 0
	m.stats.misses = 0
	m.stats.evicted = 0
	m.stats.mu.Unlock()
	
	return nil
}

func (m *MemoryStore) Stats(ctx context.Context) (*CacheStats, error) {
	m.mu.RLock()
	keyCount := int64(len(m.data))
	
	// Calculate total size
	var totalSize int64
	for _, entry := range m.data {
		totalSize += int64(len(entry.value))
	}
	m.mu.RUnlock()

	m.stats.mu.RLock()
	hits := m.stats.hits
	misses := m.stats.misses
	evicted := m.stats.evicted
	m.stats.mu.RUnlock()

	var hitRate float64
	if hits+misses > 0 {
		hitRate = float64(hits) / float64(hits+misses)
	}

	return &CacheStats{
		Keys:    keyCount,
		Hits:    hits,
		Misses:  misses,
		HitRate: hitRate,
		Size:    totalSize,
		Evicted: evicted,
	}, nil
}

func (m *MemoryStore) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanup()
	}
}

func (m *MemoryStore) cleanup() {
	now := time.Now()
	expiredKeys := []string{}

	m.mu.RLock()
	for key, entry := range m.data {
		if !entry.expiresAt.IsZero() && entry.expiresAt.Before(now) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	m.mu.RUnlock()

	if len(expiredKeys) > 0 {
		m.mu.Lock()
		for _, key := range expiredKeys {
			delete(m.data, key)
		}
		m.mu.Unlock()

		m.stats.mu.Lock()
		m.stats.evicted += int64(len(expiredKeys))
		m.stats.mu.Unlock()

		m.logger.Debug("Cleaned up expired cache entries",
			logger.F("count", len(expiredKeys)))
	}
}