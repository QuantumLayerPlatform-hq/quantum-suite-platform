package clients

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/quantum-suite/platform/pkg/shared/logger"
)

// PrometheusMetricsClient implements MetricsClient interface using Prometheus API
type PrometheusMetricsClient struct {
	client api.Client
	v1API  v1.API
	logger logger.Logger
}

// NewPrometheusMetricsClient creates a new Prometheus-based metrics client
func NewPrometheusMetricsClient(prometheusURL string, log logger.Logger) (*PrometheusMetricsClient, error) {
	client, err := api.NewClient(api.Config{
		Address: prometheusURL,
		RoundTripper: &http.Transport{
			IdleConnTimeout: 30 * time.Second,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus client: %w", err)
	}

	return &PrometheusMetricsClient{
		client: client,
		v1API:  v1.NewAPI(client),
		logger: log.WithField("component", "metrics_client"),
	}, nil
}

// RecordRequest records a request metric
func (m *PrometheusMetricsClient) RecordRequest(ctx context.Context, method, endpoint, status string, duration time.Duration) error {
	m.logger.Debug("Recording request metric",
		logger.F("method", method),
		logger.F("endpoint", endpoint),
		logger.F("status", status),
		logger.F("duration", duration))
	
	// In a real implementation, this would increment counters and record histograms
	// For now, we'll just log the metric
	return nil
}

// RecordProviderRequest records a provider-specific request metric
func (m *PrometheusMetricsClient) RecordProviderRequest(ctx context.Context, provider, model, status string, duration time.Duration, tokens int) error {
	m.logger.Debug("Recording provider request metric",
		logger.F("provider", provider),
		logger.F("model", model),
		logger.F("status", status),
		logger.F("duration", duration),
		logger.F("tokens", tokens))
	
	// In a real implementation, this would increment provider-specific counters
	return nil
}

// GetRequestCount gets total request count for a time range
func (m *PrometheusMetricsClient) GetRequestCount(ctx context.Context, since time.Time) (int64, error) {
	query := fmt.Sprintf("sum(increase(qlens_requests_total[%s]))", time.Since(since))
	
	result, warnings, err := m.v1API.Query(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to query prometheus: %w", err)
	}
	
	if len(warnings) > 0 {
		m.logger.Warn("Prometheus query warnings", logger.F("warnings", warnings))
	}
	
	switch v := result.(type) {
	case model.Vector:
		if len(v) == 0 {
			return 0, nil
		}
		return int64(v[0].Value), nil
	default:
		return 0, fmt.Errorf("unexpected result type: %T", result)
	}
}

// GetErrorCount gets total error count for a time range
func (m *PrometheusMetricsClient) GetErrorCount(ctx context.Context, since time.Time) (int64, error) {
	query := fmt.Sprintf("sum(increase(qlens_requests_total{status=~\"4..|5..\"}[%s]))", time.Since(since))
	
	result, warnings, err := m.v1API.Query(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to query prometheus: %w", err)
	}
	
	if len(warnings) > 0 {
		m.logger.Warn("Prometheus query warnings", logger.F("warnings", warnings))
	}
	
	switch v := result.(type) {
	case model.Vector:
		if len(v) == 0 {
			return 0, nil
		}
		return int64(v[0].Value), nil
	default:
		return 0, fmt.Errorf("unexpected result type: %T", result)
	}
}

// GetAverageLatency gets average latency for a time range
func (m *PrometheusMetricsClient) GetAverageLatency(ctx context.Context, since time.Time) (time.Duration, error) {
	query := fmt.Sprintf("rate(qlens_request_duration_seconds_sum[%s]) / rate(qlens_request_duration_seconds_count[%s])", time.Since(since), time.Since(since))
	
	result, warnings, err := m.v1API.Query(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to query prometheus: %w", err)
	}
	
	if len(warnings) > 0 {
		m.logger.Warn("Prometheus query warnings", logger.F("warnings", warnings))
	}
	
	switch v := result.(type) {
	case model.Vector:
		if len(v) == 0 {
			return 0, nil
		}
		seconds := float64(v[0].Value)
		return time.Duration(seconds * float64(time.Second)), nil
	default:
		return 0, fmt.Errorf("unexpected result type: %T", result)
	}
}

// GetProviderMetrics gets metrics for a specific provider
func (m *PrometheusMetricsClient) GetProviderMetrics(ctx context.Context, provider string, since time.Time) (map[string]interface{}, error) {
	duration := time.Since(since)
	
	// Get request count for provider
	requestQuery := fmt.Sprintf("sum(increase(qlens_provider_requests_total{provider=\"%s\"}[%s]))", provider, duration)
	requestResult, _, err := m.v1API.Query(ctx, requestQuery, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query provider requests: %w", err)
	}
	
	// Get error count for provider
	errorQuery := fmt.Sprintf("sum(increase(qlens_provider_requests_total{provider=\"%s\",status=~\"4..|5..\"}[%s]))", provider, duration)
	errorResult, _, err := m.v1API.Query(ctx, errorQuery, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query provider errors: %w", err)
	}
	
	// Get token usage for provider
	tokenQuery := fmt.Sprintf("sum(increase(qlens_provider_tokens_total{provider=\"%s\"}[%s]))", provider, duration)
	tokenResult, _, err := m.v1API.Query(ctx, tokenQuery, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query provider tokens: %w", err)
	}
	
	metrics := map[string]interface{}{
		"provider":     provider,
		"requests":     extractScalarValue(requestResult),
		"errors":       extractScalarValue(errorResult),
		"tokens":       extractScalarValue(tokenResult),
		"time_range":   duration.String(),
	}
	
	return metrics, nil
}

// extractScalarValue extracts a scalar value from Prometheus query result
func extractScalarValue(result model.Value) int64 {
	switch v := result.(type) {
	case model.Vector:
		if len(v) == 0 {
			return 0
		}
		return int64(v[0].Value)
	case *model.Scalar:
		return int64(v.Value)
	default:
		return 0
	}
}

// Health checks if the metrics client is healthy
func (m *PrometheusMetricsClient) Health(ctx context.Context) error {
	// Simple query to check if Prometheus is accessible
	_, _, err := m.v1API.Query(ctx, "up", time.Now())
	if err != nil {
		return fmt.Errorf("prometheus health check failed: %w", err)
	}
	return nil
}