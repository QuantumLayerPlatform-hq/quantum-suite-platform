package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/quantum-suite/platform/internal/domain"
)

// Metrics contains all Prometheus metrics for QLens
type Metrics struct {
	// Request metrics
	RequestsTotal      *prometheus.CounterVec
	RequestDuration    *prometheus.HistogramVec
	RequestsInFlight   *prometheus.GaugeVec
	
	// Provider metrics
	ProviderRequests   *prometheus.CounterVec
	ProviderLatency    *prometheus.HistogramVec
	ProviderErrors     *prometheus.CounterVec
	ProviderHealth     *prometheus.GaugeVec
	
	// Token and cost metrics
	TokensProcessed    *prometheus.CounterVec
	CostTotal          *prometheus.CounterVec
	CostPerRequest     *prometheus.HistogramVec
	
	// Cache metrics
	CacheHits          *prometheus.CounterVec
	CacheMisses        *prometheus.CounterVec
	CacheSize          *prometheus.GaugeVec
	CacheLatency       *prometheus.HistogramVec
	
	// Rate limiting metrics
	RateLimitHits      *prometheus.CounterVec
	RateLimitRemaining *prometheus.GaugeVec
	
	// System metrics
	ActiveConnections  *prometheus.GaugeVec
	MemoryUsage        *prometheus.GaugeVec
	CPUUsage           *prometheus.GaugeVec
	
	// Business metrics
	DailyActiveUsers   *prometheus.GaugeVec
	DailyActiveModels  *prometheus.GaugeVec
	StreamingRequests  *prometheus.CounterVec
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	return &Metrics{
		// Request metrics
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "qlens_requests_total",
				Help: "Total number of requests processed",
			},
			[]string{"service", "endpoint", "method", "status", "tenant_id", "provider", "model"},
		),
		
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "qlens_request_duration_seconds",
				Help:    "Request duration in seconds",
				Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
			},
			[]string{"service", "endpoint", "method", "provider", "model"},
		),
		
		RequestsInFlight: promauto.NewGaugeVec(
			prometheus.GaugeVec{
				Name: "qlens_requests_in_flight",
				Help: "Current number of requests being processed",
			},
			[]string{"service", "provider"},
		),
		
		// Provider metrics
		ProviderRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "qlens_provider_requests_total",
				Help: "Total requests sent to each provider",
			},
			[]string{"provider", "model", "status"},
		),
		
		ProviderLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "qlens_provider_latency_seconds",
				Help:    "Provider response latency in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
			},
			[]string{"provider", "model"},
		),
		
		ProviderErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "qlens_provider_errors_total",
				Help: "Total errors from providers",
			},
			[]string{"provider", "model", "error_type"},
		),
		
		ProviderHealth: promauto.NewGaugeVec(
			prometheus.GaugeVec{
				Name: "qlens_provider_health_status",
				Help: "Provider health status (1=healthy, 0=unhealthy)",
			},
			[]string{"provider"},
		),
		
		// Token and cost metrics
		TokensProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "qlens_tokens_processed_total",
				Help: "Total tokens processed",
			},
			[]string{"provider", "model", "type", "tenant_id"},
		),
		
		CostTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "qlens_cost_usd_total",
				Help: "Total cost in USD",
			},
			[]string{"provider", "model", "tenant_id"},
		),
		
		CostPerRequest: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "qlens_cost_per_request_usd",
				Help:    "Cost per request in USD",
				Buckets: []float64{0.001, 0.01, 0.1, 1, 10, 100},
			},
			[]string{"provider", "model"},
		),
		
		// Cache metrics
		CacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "qlens_cache_hits_total",
				Help: "Total cache hits",
			},
			[]string{"cache_type", "tenant_id"},
		),
		
		CacheMisses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "qlens_cache_misses_total",
				Help: "Total cache misses",
			},
			[]string{"cache_type", "tenant_id"},
		),
		
		CacheSize: promauto.NewGaugeVec(
			prometheus.GaugeVec{
				Name: "qlens_cache_size_entries",
				Help: "Current number of entries in cache",
			},
			[]string{"cache_type"},
		),
		
		CacheLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "qlens_cache_operation_duration_seconds",
				Help:    "Cache operation duration in seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1},
			},
			[]string{"cache_type", "operation"},
		),
		
		// Rate limiting metrics
		RateLimitHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "qlens_rate_limit_hits_total",
				Help: "Total rate limit hits",
			},
			[]string{"limit_type", "tenant_id"},
		),
		
		RateLimitRemaining: promauto.NewGaugeVec(
			prometheus.GaugeVec{
				Name: "qlens_rate_limit_remaining",
				Help: "Remaining rate limit allowance",
			},
			[]string{"limit_type", "tenant_id"},
		),
		
		// System metrics
		ActiveConnections: promauto.NewGaugeVec(
			prometheus.GaugeVec{
				Name: "qlens_active_connections",
				Help: "Number of active connections",
			},
			[]string{"service"},
		),
		
		MemoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeVec{
				Name: "qlens_memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
			[]string{"service", "type"},
		),
		
		CPUUsage: promauto.NewGaugeVec(
			prometheus.GaugeVec{
				Name: "qlens_cpu_usage_percentage",
				Help: "CPU usage percentage",
			},
			[]string{"service"},
		),
		
		// Business metrics
		DailyActiveUsers: promauto.NewGaugeVec(
			prometheus.GaugeVec{
				Name: "qlens_daily_active_users",
				Help: "Number of daily active users",
			},
			[]string{"tenant_id"},
		),
		
		DailyActiveModels: promauto.NewGaugeVec(
			prometheus.GaugeVec{
				Name: "qlens_daily_active_models",
				Help: "Number of daily active models",
			},
			[]string{"provider"},
		),
		
		StreamingRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "qlens_streaming_requests_total",
				Help: "Total streaming requests",
			},
			[]string{"provider", "model", "status"},
		),
	}
}

// MetricsCollector provides methods for updating metrics
type MetricsCollector struct {
	metrics *Metrics
}

func NewMetricsCollector(metrics *Metrics) *MetricsCollector {
	return &MetricsCollector{
		metrics: metrics,
	}
}

// RecordRequest records a completed request
func (c *MetricsCollector) RecordRequest(req RequestMetrics) {
	labels := prometheus.Labels{
		"service":   req.Service,
		"endpoint":  req.Endpoint,
		"method":    req.Method,
		"status":    req.Status,
		"tenant_id": string(req.TenantID),
		"provider":  string(req.Provider),
		"model":     req.Model,
	}
	c.metrics.RequestsTotal.With(labels).Inc()
	
	durationLabels := prometheus.Labels{
		"service":  req.Service,
		"endpoint": req.Endpoint,
		"method":   req.Method,
		"provider": string(req.Provider),
		"model":    req.Model,
	}
	c.metrics.RequestDuration.With(durationLabels).Observe(req.Duration.Seconds())
}

// RecordProviderRequest records a provider request
func (c *MetricsCollector) RecordProviderRequest(req ProviderMetrics) {
	labels := prometheus.Labels{
		"provider": string(req.Provider),
		"model":    req.Model,
		"status":   req.Status,
	}
	c.metrics.ProviderRequests.With(labels).Inc()
	
	latencyLabels := prometheus.Labels{
		"provider": string(req.Provider),
		"model":    req.Model,
	}
	c.metrics.ProviderLatency.With(latencyLabels).Observe(req.Latency.Seconds())
	
	if req.Error != "" {
		errorLabels := prometheus.Labels{
			"provider":   string(req.Provider),
			"model":      req.Model,
			"error_type": req.Error,
		}
		c.metrics.ProviderErrors.With(errorLabels).Inc()
	}
}

// RecordTokenUsage records token usage and cost
func (c *MetricsCollector) RecordTokenUsage(usage TokenUsageMetrics) {
	promptLabels := prometheus.Labels{
		"provider":  string(usage.Provider),
		"model":     usage.Model,
		"type":      "prompt",
		"tenant_id": string(usage.TenantID),
	}
	c.metrics.TokensProcessed.With(promptLabels).Add(float64(usage.PromptTokens))
	
	completionLabels := prometheus.Labels{
		"provider":  string(usage.Provider),
		"model":     usage.Model,
		"type":      "completion",
		"tenant_id": string(usage.TenantID),
	}
	c.metrics.TokensProcessed.With(completionLabels).Add(float64(usage.CompletionTokens))
	
	costLabels := prometheus.Labels{
		"provider":  string(usage.Provider),
		"model":     usage.Model,
		"tenant_id": string(usage.TenantID),
	}
	c.metrics.CostTotal.With(costLabels).Add(usage.CostUSD)
	
	costPerReqLabels := prometheus.Labels{
		"provider": string(usage.Provider),
		"model":    usage.Model,
	}
	c.metrics.CostPerRequest.With(costPerReqLabels).Observe(usage.CostUSD)
}

// RecordCacheOperation records cache metrics
func (c *MetricsCollector) RecordCacheOperation(cache CacheMetrics) {
	if cache.Hit {
		labels := prometheus.Labels{
			"cache_type": cache.Type,
			"tenant_id":  string(cache.TenantID),
		}
		c.metrics.CacheHits.With(labels).Inc()
	} else {
		labels := prometheus.Labels{
			"cache_type": cache.Type,
			"tenant_id":  string(cache.TenantID),
		}
		c.metrics.CacheMisses.With(labels).Inc()
	}
	
	latencyLabels := prometheus.Labels{
		"cache_type": cache.Type,
		"operation":  cache.Operation,
	}
	c.metrics.CacheLatency.With(latencyLabels).Observe(cache.Duration.Seconds())
}

// UpdateProviderHealth updates provider health status
func (c *MetricsCollector) UpdateProviderHealth(provider domain.Provider, healthy bool) {
	labels := prometheus.Labels{
		"provider": string(provider),
	}
	
	value := 0.0
	if healthy {
		value = 1.0
	}
	c.metrics.ProviderHealth.With(labels).Set(value)
}

// RecordRateLimitHit records rate limit events
func (c *MetricsCollector) RecordRateLimitHit(limitType string, tenantID domain.TenantID, remaining int) {
	hitLabels := prometheus.Labels{
		"limit_type": limitType,
		"tenant_id":  string(tenantID),
	}
	c.metrics.RateLimitHits.With(hitLabels).Inc()
	
	remainingLabels := prometheus.Labels{
		"limit_type": limitType,
		"tenant_id":  string(tenantID),
	}
	c.metrics.RateLimitRemaining.With(remainingLabels).Set(float64(remaining))
}

// UpdateSystemMetrics updates system-level metrics
func (c *MetricsCollector) UpdateSystemMetrics(system SystemMetrics) {
	connLabels := prometheus.Labels{
		"service": system.Service,
	}
	c.metrics.ActiveConnections.With(connLabels).Set(float64(system.ActiveConnections))
	
	memLabels := prometheus.Labels{
		"service": system.Service,
		"type":    "heap",
	}
	c.metrics.MemoryUsage.With(memLabels).Set(float64(system.MemoryUsage))
	
	cpuLabels := prometheus.Labels{
		"service": system.Service,
	}
	c.metrics.CPUUsage.With(cpuLabels).Set(system.CPUUsage)
}

// Metric data structures
type RequestMetrics struct {
	Service   string
	Endpoint  string
	Method    string
	Status    string
	TenantID  domain.TenantID
	Provider  domain.Provider
	Model     string
	Duration  time.Duration
}

type ProviderMetrics struct {
	Provider domain.Provider
	Model    string
	Status   string
	Latency  time.Duration
	Error    string
}

type TokenUsageMetrics struct {
	Provider         domain.Provider
	Model            string
	TenantID         domain.TenantID
	PromptTokens     int
	CompletionTokens int
	CostUSD          float64
}

type CacheMetrics struct {
	Type      string
	TenantID  domain.TenantID
	Operation string
	Hit       bool
	Duration  time.Duration
}

type SystemMetrics struct {
	Service           string
	ActiveConnections int
	MemoryUsage       uint64
	CPUUsage          float64
}