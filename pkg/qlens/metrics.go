package qlens

import (
	"fmt"
	"sync"
	"time"
)

// MetricsCollector collects and tracks QLens metrics
type MetricsCollector struct {
	mu             sync.RWMutex
	requestCounts  map[string]int64
	errorCounts    map[string]map[string]int64 // operation -> error_type -> count
	responseTimes  map[string][]time.Duration
	tokenUsage     map[string]int64
	costs          map[string]float64
	cacheHits      map[string]int64
	cacheMisses    map[string]int64
	startTime      time.Time
}

// Metrics represents the current metrics snapshot
type Metrics struct {
	RequestCounts    map[string]int64            `json:"request_counts"`
	ErrorCounts      map[string]map[string]int64 `json:"error_counts"`
	AvgResponseTimes map[string]time.Duration    `json:"avg_response_times"`
	TokenUsage       map[string]int64            `json:"token_usage"`
	TotalCost        map[string]float64          `json:"total_cost"`
	CacheHitRates    map[string]float64          `json:"cache_hit_rates"`
	Uptime           time.Duration               `json:"uptime"`
	RequestRate      map[string]float64          `json:"request_rate"` // requests per second
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		requestCounts:  make(map[string]int64),
		errorCounts:    make(map[string]map[string]int64),
		responseTimes:  make(map[string][]time.Duration),
		tokenUsage:     make(map[string]int64),
		costs:          make(map[string]float64),
		cacheHits:      make(map[string]int64),
		cacheMisses:    make(map[string]int64),
		startTime:      time.Now(),
	}
}

// IncrementRequestCount increments the request count for an operation
func (m *MetricsCollector) IncrementRequestCount(operation string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.requestCounts[operation]++
}

// IncrementErrorCount increments the error count for an operation and error type
func (m *MetricsCollector) IncrementErrorCount(operation, errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.errorCounts[operation] == nil {
		m.errorCounts[operation] = make(map[string]int64)
	}
	m.errorCounts[operation][errorType]++
}

// RecordResponseTime records a response time for an operation
func (m *MetricsCollector) RecordResponseTime(operation string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Keep only the last 1000 response times to prevent memory leak
	times := m.responseTimes[operation]
	if len(times) >= 1000 {
		times = times[1:]
	}
	m.responseTimes[operation] = append(times, duration)
}

// RecordTokenUsage records token usage for an operation
func (m *MetricsCollector) RecordTokenUsage(operation string, tokens int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.tokenUsage[operation] += int64(tokens)
}

// RecordCost records cost for an operation
func (m *MetricsCollector) RecordCost(operation string, cost float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.costs[operation] += cost
}

// IncrementCacheHits increments cache hits for an operation
func (m *MetricsCollector) IncrementCacheHits(operation string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.cacheHits[operation]++
}

// IncrementCacheMisses increments cache misses for an operation
func (m *MetricsCollector) IncrementCacheMisses(operation string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.cacheMisses[operation]++
}

// GetMetrics returns a snapshot of current metrics
func (m *MetricsCollector) GetMetrics() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metrics := &Metrics{
		RequestCounts:    make(map[string]int64),
		ErrorCounts:      make(map[string]map[string]int64),
		AvgResponseTimes: make(map[string]time.Duration),
		TokenUsage:       make(map[string]int64),
		TotalCost:        make(map[string]float64),
		CacheHitRates:    make(map[string]float64),
		RequestRate:      make(map[string]float64),
		Uptime:           time.Since(m.startTime),
	}
	
	// Copy request counts
	for op, count := range m.requestCounts {
		metrics.RequestCounts[op] = count
	}
	
	// Copy error counts
	for op, errorMap := range m.errorCounts {
		metrics.ErrorCounts[op] = make(map[string]int64)
		for errorType, count := range errorMap {
			metrics.ErrorCounts[op][errorType] = count
		}
	}
	
	// Calculate average response times
	for op, times := range m.responseTimes {
		if len(times) > 0 {
			var total time.Duration
			for _, t := range times {
				total += t
			}
			metrics.AvgResponseTimes[op] = total / time.Duration(len(times))
		}
	}
	
	// Copy token usage
	for op, usage := range m.tokenUsage {
		metrics.TokenUsage[op] = usage
	}
	
	// Copy costs
	for op, cost := range m.costs {
		metrics.TotalCost[op] = cost
	}
	
	// Calculate cache hit rates
	for op := range m.cacheHits {
		hits := m.cacheHits[op]
		misses := m.cacheMisses[op]
		total := hits + misses
		if total > 0 {
			metrics.CacheHitRates[op] = float64(hits) / float64(total)
		}
	}
	
	// Calculate request rates (requests per second)
	uptimeSeconds := metrics.Uptime.Seconds()
	if uptimeSeconds > 0 {
		for op, count := range m.requestCounts {
			metrics.RequestRate[op] = float64(count) / uptimeSeconds
		}
	}
	
	return metrics
}

// Reset clears all metrics
func (m *MetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.requestCounts = make(map[string]int64)
	m.errorCounts = make(map[string]map[string]int64)
	m.responseTimes = make(map[string][]time.Duration)
	m.tokenUsage = make(map[string]int64)
	m.costs = make(map[string]float64)
	m.cacheHits = make(map[string]int64)
	m.cacheMisses = make(map[string]int64)
	m.startTime = time.Now()
}

// Close shuts down the metrics collector
func (m *MetricsCollector) Close() {
	// For in-memory metrics, nothing to close
}

// GetPrometheusMetrics returns metrics in Prometheus format
func (m *MetricsCollector) GetPrometheusMetrics() string {
	metrics := m.GetMetrics()
	var result string
	
	// Request counts
	result += "# HELP qlens_requests_total Total number of requests\n"
	result += "# TYPE qlens_requests_total counter\n"
	for operation, count := range metrics.RequestCounts {
		result += fmt.Sprintf("qlens_requests_total{operation=\"%s\"} %d\n", operation, count)
	}
	
	// Error counts
	result += "# HELP qlens_errors_total Total number of errors\n"
	result += "# TYPE qlens_errors_total counter\n"
	for operation, errorMap := range metrics.ErrorCounts {
		for errorType, count := range errorMap {
			result += fmt.Sprintf("qlens_errors_total{operation=\"%s\",error_type=\"%s\"} %d\n", operation, errorType, count)
		}
	}
	
	// Response times
	result += "# HELP qlens_response_duration_seconds Average response duration in seconds\n"
	result += "# TYPE qlens_response_duration_seconds gauge\n"
	for operation, duration := range metrics.AvgResponseTimes {
		result += fmt.Sprintf("qlens_response_duration_seconds{operation=\"%s\"} %f\n", operation, duration.Seconds())
	}
	
	// Token usage
	result += "# HELP qlens_tokens_total Total number of tokens processed\n"
	result += "# TYPE qlens_tokens_total counter\n"
	for operation, tokens := range metrics.TokenUsage {
		result += fmt.Sprintf("qlens_tokens_total{operation=\"%s\"} %d\n", operation, tokens)
	}
	
	// Costs
	result += "# HELP qlens_cost_total Total cost in USD\n"
	result += "# TYPE qlens_cost_total counter\n"
	for operation, cost := range metrics.TotalCost {
		result += fmt.Sprintf("qlens_cost_total{operation=\"%s\"} %f\n", operation, cost)
	}
	
	// Cache hit rates
	result += "# HELP qlens_cache_hit_rate Cache hit rate\n"
	result += "# TYPE qlens_cache_hit_rate gauge\n"
	for operation, rate := range metrics.CacheHitRates {
		result += fmt.Sprintf("qlens_cache_hit_rate{operation=\"%s\"} %f\n", operation, rate)
	}
	
	// Request rates
	result += "# HELP qlens_request_rate Requests per second\n"
	result += "# TYPE qlens_request_rate gauge\n"
	for operation, rate := range metrics.RequestRate {
		result += fmt.Sprintf("qlens_request_rate{operation=\"%s\"} %f\n", operation, rate)
	}
	
	// Uptime
	result += "# HELP qlens_uptime_seconds Uptime in seconds\n"
	result += "# TYPE qlens_uptime_seconds gauge\n"
	result += fmt.Sprintf("qlens_uptime_seconds %f\n", metrics.Uptime.Seconds())
	
	return result
}

// GetHealthMetrics returns key health indicators
func (m *MetricsCollector) GetHealthMetrics() map[string]interface{} {
	metrics := m.GetMetrics()
	
	// Calculate overall error rate
	var totalRequests, totalErrors int64
	for _, count := range metrics.RequestCounts {
		totalRequests += count
	}
	for _, errorMap := range metrics.ErrorCounts {
		for _, count := range errorMap {
			totalErrors += count
		}
	}
	
	var errorRate float64
	if totalRequests > 0 {
		errorRate = float64(totalErrors) / float64(totalRequests)
	}
	
	// Calculate overall cache hit rate
	var totalCacheHits, totalCacheMisses int64
	for op := range m.cacheHits {
		totalCacheHits += m.cacheHits[op]
		totalCacheMisses += m.cacheMisses[op]
	}
	
	var overallCacheHitRate float64
	if totalCacheHits+totalCacheMisses > 0 {
		overallCacheHitRate = float64(totalCacheHits) / float64(totalCacheHits+totalCacheMisses)
	}
	
	// Calculate average response time across all operations
	var totalDuration time.Duration
	var totalResponses int
	for _, times := range m.responseTimes {
		for _, duration := range times {
			totalDuration += duration
			totalResponses++
		}
	}
	
	var avgResponseTime time.Duration
	if totalResponses > 0 {
		avgResponseTime = totalDuration / time.Duration(totalResponses)
	}
	
	// Calculate total cost
	var totalCost float64
	for _, cost := range metrics.TotalCost {
		totalCost += cost
	}
	
	// Calculate total tokens
	var totalTokens int64
	for _, tokens := range metrics.TokenUsage {
		totalTokens += tokens
	}
	
	return map[string]interface{}{
		"total_requests":        totalRequests,
		"total_errors":          totalErrors,
		"error_rate":            errorRate,
		"avg_response_time_ms":  avgResponseTime.Milliseconds(),
		"cache_hit_rate":        overallCacheHitRate,
		"total_cost_usd":        totalCost,
		"total_tokens":          totalTokens,
		"uptime_seconds":        metrics.Uptime.Seconds(),
	}
}