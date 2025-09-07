package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/shared/errors"
	"github.com/quantum-suite/platform/pkg/shared/logger"
)

// Usage analytics response types (defined here to avoid import cycles)
type GlobalUsageStats struct {
	TotalCostToday    float64 `json:"total_cost_today"`
	RequestCount      int64   `json:"request_count"`
	ActiveTenants     int     `json:"active_tenants"`
	ActiveServices    int     `json:"active_services"`
	BudgetUtilization float64 `json:"budget_utilization_percent"`
	LastUpdated       string  `json:"last_updated"`
}

type TenantUsageStats struct {
	TenantID        string                     `json:"tenant_id"`
	DailyCost       float64                    `json:"daily_cost"`
	MonthlyCost     float64                    `json:"monthly_cost"`
	RequestCount    int64                      `json:"request_count"`
	ModelUsage      map[string]ModelUsageStats `json:"model_usage"`
	BudgetLimit     float64                    `json:"budget_limit"`
	LastUpdated     string                     `json:"last_updated"`
}

type ModelUsageStats struct {
	RequestCount    int64   `json:"request_count"`
	TokensUsed      int64   `json:"tokens_used"`
	Cost            float64 `json:"cost"`
	AvgLatency      float64 `json:"avg_latency_ms"`
}

type CostSummaryStats struct {
	DailyCost                 float64 `json:"daily_cost"`
	RequestCount              int64   `json:"request_count"`
	ActiveTenants             int     `json:"active_tenants"`
	ActiveServices            int     `json:"active_services"`
	BudgetUtilizationPercent  float64 `json:"budget_utilization_percent"`
	Status                    string  `json:"status"`
	LastUpdated               string  `json:"last_updated"`
}

// HTTPRouterClient implements RouterClient interface using HTTP calls
type HTTPRouterClient struct {
	baseURL string
	client  *http.Client
	logger  logger.Logger
}

// NewHTTPRouterClient creates a new HTTP-based router client
func NewHTTPRouterClient(baseURL string, log logger.Logger) *HTTPRouterClient {
	return &HTTPRouterClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: log.WithField("component", "router_client"),
	}
}

// RouteCompletion sends completion request to router service
func (c *HTTPRouterClient) RouteCompletion(ctx context.Context, req *domain.CompletionRequest) (*domain.CompletionResponse, error) {
	url := fmt.Sprintf("%s/internal/v1/completions", c.baseURL)
	
	// Convert to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, errors.InternalError("failed to marshal request", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, errors.InternalError("failed to create request", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	c.logger.Debug("Sending completion request to router",
		logger.F("url", url),
		logger.F("model", req.Model),
		logger.F("provider", req.Provider))

	// Execute request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, errors.InternalError("failed to call router service", err)
	}
	defer resp.Body.Close()

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleHTTPError(resp)
	}

	// Parse response
	var completionResp domain.CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
		return nil, errors.InternalError("failed to decode response", err)
	}

	c.logger.Debug("Received completion response from router",
		logger.F("response_id", completionResp.ID),
		logger.F("provider", completionResp.Provider))

	return &completionResp, nil
}

// RouteCompletionStream sends streaming completion request to router service
func (c *HTTPRouterClient) RouteCompletionStream(ctx context.Context, req *domain.CompletionRequest) (<-chan *domain.StreamResponse, error) {
	// Set stream flag
	req.Stream = true
	
	url := fmt.Sprintf("%s/internal/v1/completions", c.baseURL)
	
	// Convert to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, errors.InternalError("failed to marshal request", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, errors.InternalError("failed to create request", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	c.logger.Debug("Sending streaming completion request to router",
		logger.F("url", url),
		logger.F("model", req.Model))

	// Execute request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, errors.InternalError("failed to call router service", err)
	}

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, c.handleHTTPError(resp)
	}

	// Create channel for streaming
	ch := make(chan *domain.StreamResponse, 10)
	
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		
		decoder := json.NewDecoder(resp.Body)
		for {
			var streamResp domain.StreamResponse
			if err := decoder.Decode(&streamResp); err != nil {
				ch <- &domain.StreamResponse{
					Error: errors.InternalError("stream decode error", err),
				}
				return
			}
			
			ch <- &streamResp
			
			if streamResp.Done {
				return
			}
		}
	}()

	return ch, nil
}

// RouteEmbedding sends embedding request to router service
func (c *HTTPRouterClient) RouteEmbedding(ctx context.Context, req *domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	url := fmt.Sprintf("%s/internal/v1/embeddings", c.baseURL)
	
	// Convert to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, errors.InternalError("failed to marshal request", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, errors.InternalError("failed to create request", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	c.logger.Debug("Sending embedding request to router",
		logger.F("url", url),
		logger.F("model", req.Model))

	// Execute request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, errors.InternalError("failed to call router service", err)
	}
	defer resp.Body.Close()

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleHTTPError(resp)
	}

	// Parse response
	var embeddingResp domain.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, errors.InternalError("failed to decode response", err)
	}

	return &embeddingResp, nil
}

// ListModels gets available models from router service
func (c *HTTPRouterClient) ListModels(ctx context.Context, opts *domain.ListModelsOptions) (*domain.ModelsResponse, error) {
	url := fmt.Sprintf("%s/internal/v1/models", c.baseURL)
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.InternalError("failed to create request", err)
	}
	
	httpReq.Header.Set("Accept", "application/json")

	// Add query parameters if provided
	if opts != nil {
		q := httpReq.URL.Query()
		if opts.Provider != "" {
			q.Add("provider", string(opts.Provider))
		}
		if opts.Capability != "" {
			q.Add("capability", string(opts.Capability))
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	c.logger.Debug("Getting models from router", logger.F("url", url))

	// Execute request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, errors.InternalError("failed to call router service", err)
	}
	defer resp.Body.Close()

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleHTTPError(resp)
	}

	// Parse response
	var modelsResp domain.ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, errors.InternalError("failed to decode response", err)
	}

	return &modelsResp, nil
}

// HealthCheck checks router service health
func (c *HTTPRouterClient) HealthCheck(ctx context.Context) (*domain.HealthResponse, error) {
	url := fmt.Sprintf("%s/health", c.baseURL)
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.InternalError("failed to create request", err)
	}
	
	httpReq.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, errors.InternalError("failed to call router service", err)
	}
	defer resp.Body.Close()

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleHTTPError(resp)
	}

	// Parse response
	var healthResp domain.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return nil, errors.InternalError("failed to decode response", err)
	}

	return &healthResp, nil
}

// GetGlobalUsage retrieves global usage statistics from router
func (c *HTTPRouterClient) GetGlobalUsage(ctx context.Context) (*GlobalUsageStats, error) {
	url := fmt.Sprintf("%s/internal/v1/usage/global", c.baseURL)
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.InternalError("failed to create request", err)
	}
	
	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, errors.InternalError("router request failed", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleHTTPError(resp)
	}
	
	var stats GlobalUsageStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, errors.InternalError("failed to decode response", err)
	}
	
	return &stats, nil
}

// GetTenantUsage retrieves usage statistics for a specific tenant from router
func (c *HTTPRouterClient) GetTenantUsage(ctx context.Context, tenantID string, period string) (*TenantUsageStats, error) {
	url := fmt.Sprintf("%s/internal/v1/usage/tenant/%s?period=%s", c.baseURL, tenantID, period)
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.InternalError("failed to create request", err)
	}
	
	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, errors.InternalError("router request failed", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleHTTPError(resp)
	}
	
	var stats TenantUsageStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, errors.InternalError("failed to decode response", err)
	}
	
	return &stats, nil
}

// GetCostSummary retrieves cost summary statistics from router
func (c *HTTPRouterClient) GetCostSummary(ctx context.Context) (*CostSummaryStats, error) {
	url := fmt.Sprintf("%s/internal/v1/costs/summary", c.baseURL)
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.InternalError("failed to create request", err)
	}
	
	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, errors.InternalError("router request failed", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleHTTPError(resp)
	}
	
	var stats CostSummaryStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, errors.InternalError("failed to decode response", err)
	}
	
	return &stats, nil
}

// handleHTTPError converts HTTP errors to QLens errors
func (c *HTTPRouterClient) handleHTTPError(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusBadRequest:
		return errors.ValidationError("router service: bad request", "request")
	case http.StatusUnauthorized:
		return errors.AuthenticationError("router service: unauthorized")
	case http.StatusForbidden:
		return errors.AuthorizationError("router service: forbidden")
	case http.StatusTooManyRequests:
		return errors.InternalError("router service: rate limit exceeded", nil)
	case http.StatusInternalServerError:
		return errors.InternalError("router service: internal error", nil)
	case http.StatusServiceUnavailable:
		return errors.InternalError("router service: service unavailable", nil)
	default:
		return errors.InternalError(fmt.Sprintf("router service: HTTP %d", resp.StatusCode), nil)
	}
}