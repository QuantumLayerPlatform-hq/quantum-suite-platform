package cost

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/shared/errors"
	"github.com/quantum-suite/platform/pkg/shared/logger"
)

// CostService manages cost tracking, budgeting, and alerts
type CostService struct {
	logger          logger.Logger
	mu              sync.RWMutex
	
	// In-memory tracking (would be backed by database in production)
	tenantUsage     map[domain.TenantID]*TenantCostTracker
	serviceUsage    map[string]*ServiceCostTracker
	dailyTotals     map[string]*DailyCostSummary
	
	// Configuration
	budgetLimits    *BudgetConfiguration
	alertThresholds []AlertThreshold
	
	// Real-time tracking
	requestCount    int64
	totalCostToday  float64
	lastReset       time.Time
}

// TenantCostTracker tracks costs per tenant
type TenantCostTracker struct {
	TenantID        domain.TenantID      `json:"tenant_id"`
	DailyCost       float64              `json:"daily_cost"`
	MonthlyCost     float64              `json:"monthly_cost"`
	RequestCount    int64                `json:"request_count"`
	ModelUsage      map[string]*ModelUsage `json:"model_usage"`
	ProviderUsage   map[string]*ProviderUsage `json:"provider_usage"`
	LastUpdated     time.Time            `json:"last_updated"`
	BudgetLimit     float64              `json:"budget_limit"`
	AlertsEnabled   bool                 `json:"alerts_enabled"`
}

// ServiceCostTracker tracks costs per consuming service
type ServiceCostTracker struct {
	ServiceName     string               `json:"service_name"`
	DailyCost       float64              `json:"daily_cost"`
	MonthlyCost     float64              `json:"monthly_cost"`
	RequestCount    int64                `json:"request_count"`
	ModelBreakdown  map[string]float64   `json:"model_breakdown"`
	LastUpdated     time.Time            `json:"last_updated"`
}

// ModelUsage tracks usage per model
type ModelUsage struct {
	ModelID         string    `json:"model_id"`
	RequestCount    int64     `json:"request_count"`
	TokensUsed      int64     `json:"tokens_used"`
	Cost            float64   `json:"cost"`
	AvgLatency      float64   `json:"avg_latency_ms"`
	LastUsed        time.Time `json:"last_used"`
}

// ProviderUsage tracks usage per provider
type ProviderUsage struct {
	Provider        domain.Provider `json:"provider"`
	RequestCount    int64           `json:"request_count"`
	Cost            float64         `json:"cost"`
	SuccessRate     float64         `json:"success_rate"`
	AvgLatency      float64         `json:"avg_latency_ms"`
}

// DailyCostSummary provides daily cost aggregation
type DailyCostSummary struct {
	Date            string                 `json:"date"`
	TotalCost       float64                `json:"total_cost"`
	RequestCount    int64                  `json:"request_count"`
	TopModels       []*ModelUsage          `json:"top_models"`
	TopTenants      []*TenantCostTracker   `json:"top_tenants"`
	ProviderSplit   map[string]float64     `json:"provider_split"`
}

// BudgetConfiguration defines cost limits
type BudgetConfiguration struct {
	GlobalDailyLimit    float64            `json:"global_daily_limit"`
	GlobalMonthlyLimit  float64            `json:"global_monthly_limit"`
	TenantDailyLimit    float64            `json:"tenant_daily_limit"`
	TenantMonthlyLimit  float64            `json:"tenant_monthly_limit"`
	ServiceLimits       map[string]float64 `json:"service_limits"`
}

// AlertThreshold defines when to send cost alerts
type AlertThreshold struct {
	Type        AlertType `json:"type"`
	Threshold   float64   `json:"threshold"`
	Enabled     bool      `json:"enabled"`
	Recipients  []string  `json:"recipients"`
}

type AlertType string

const (
	AlertTypeDailyBudget   AlertType = "daily_budget"
	AlertTypeMonthlyBudget AlertType = "monthly_budget"
	AlertTypeTenantBudget  AlertType = "tenant_budget"
	AlertTypeServiceBudget AlertType = "service_budget"
	AlertTypeSpike         AlertType = "cost_spike"
)

// NewCostService creates a new cost management service
func NewCostService(logger logger.Logger, config *BudgetConfiguration) *CostService {
	return &CostService{
		logger:          logger.WithField("service", "cost_service"),
		tenantUsage:     make(map[domain.TenantID]*TenantCostTracker),
		serviceUsage:    make(map[string]*ServiceCostTracker),
		dailyTotals:     make(map[string]*DailyCostSummary),
		budgetLimits:    config,
		alertThresholds: getDefaultAlertThresholds(),
		lastReset:       time.Now().Truncate(24 * time.Hour),
	}
}

// TrackRequest records cost and usage for a request
func (s *CostService) TrackRequest(ctx context.Context, req *CostTrackingRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	
	// Check if we need to reset daily counters
	if s.shouldResetDaily(now) {
		s.resetDailyCounters(now)
	}

	// Update global counters
	s.requestCount++
	s.totalCostToday += req.Cost

	// Track tenant usage
	if err := s.trackTenantUsage(req); err != nil {
		s.logger.Warn("Failed to track tenant usage", logger.F("error", err))
	}

	// Track service usage
	if err := s.trackServiceUsage(req); err != nil {
		s.logger.Warn("Failed to track service usage", logger.F("error", err))
	}

	// Check budget limits and send alerts
	s.checkBudgetLimits(req)

	// Log high-cost requests
	if req.Cost > 0.10 { // More than 10 cents
		s.logger.Warn("High-cost request detected",
			logger.F("tenant_id", req.TenantID),
			logger.F("service", req.ServiceName),
			logger.F("model", req.ModelID),
			logger.F("cost", req.Cost),
			logger.F("tokens", req.TokensUsed),
		)
	}

	return nil
}

// CostTrackingRequest contains all information needed for cost tracking
type CostTrackingRequest struct {
	TenantID      domain.TenantID   `json:"tenant_id"`
	ServiceName   string            `json:"service_name"`
	ModelID       string            `json:"model_id"`
	Provider      domain.Provider   `json:"provider"`
	Cost          float64           `json:"cost"`
	TokensUsed    int64             `json:"tokens_used"`
	LatencyMs     float64           `json:"latency_ms"`
	Success       bool              `json:"success"`
	RequestID     string            `json:"request_id"`
	Timestamp     time.Time         `json:"timestamp"`
}

// GetTenantUsage returns usage statistics for a tenant
func (s *CostService) GetTenantUsage(tenantID domain.TenantID, period string) (*TenantCostTracker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tracker, exists := s.tenantUsage[tenantID]
	if !exists {
		return nil, errors.ValidationError("tenant not found", "tenant_id")
	}

	return tracker, nil
}

// GetGlobalUsage returns system-wide usage statistics
func (s *CostService) GetGlobalUsage() *GlobalUsageStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &GlobalUsageStats{
		TotalCostToday:    s.totalCostToday,
		RequestCount:      s.requestCount,
		ActiveTenants:     len(s.tenantUsage),
		ActiveServices:    len(s.serviceUsage),
		BudgetUtilization: s.totalCostToday / s.budgetLimits.GlobalDailyLimit * 100,
		LastUpdated:       time.Now(),
	}
}

// GlobalUsageStats provides system-wide statistics
type GlobalUsageStats struct {
	TotalCostToday    float64   `json:"total_cost_today"`
	RequestCount      int64     `json:"request_count"`
	ActiveTenants     int       `json:"active_tenants"`
	ActiveServices    int       `json:"active_services"`
	BudgetUtilization float64   `json:"budget_utilization_percent"`
	LastUpdated       time.Time `json:"last_updated"`
}

// CheckBudgetCompliance checks if a request would exceed budget limits
func (s *CostService) CheckBudgetCompliance(tenantID domain.TenantID, estimatedCost float64) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check global budget
	if s.totalCostToday+estimatedCost > s.budgetLimits.GlobalDailyLimit {
		return errors.NewError(errors.ErrorTypeQuotaExceeded, "global daily budget limit exceeded").Build()
	}

	// Check tenant budget
	if tracker, exists := s.tenantUsage[tenantID]; exists {
		if tracker.DailyCost+estimatedCost > tracker.BudgetLimit {
			return errors.NewError(errors.ErrorTypeQuotaExceeded, fmt.Sprintf("tenant daily budget limit exceeded: $%.4f", tracker.BudgetLimit)).Build()
		}
	}

	return nil
}

// Helper methods
func (s *CostService) shouldResetDaily(now time.Time) bool {
	return now.Truncate(24*time.Hour).After(s.lastReset)
}

func (s *CostService) resetDailyCounters(now time.Time) {
	// Save yesterday's totals
	yesterday := s.lastReset.Format("2006-01-02")
	s.dailyTotals[yesterday] = &DailyCostSummary{
		Date:         yesterday,
		TotalCost:    s.totalCostToday,
		RequestCount: s.requestCount,
	}

	// Reset daily counters
	s.requestCount = 0
	s.totalCostToday = 0
	s.lastReset = now.Truncate(24 * time.Hour)

	// Reset tenant daily counters
	for _, tracker := range s.tenantUsage {
		tracker.DailyCost = 0
	}

	// Reset service daily counters
	for _, tracker := range s.serviceUsage {
		tracker.DailyCost = 0
	}

	s.logger.Info("Daily cost counters reset",
		logger.F("date", s.lastReset.Format("2006-01-02")),
	)
}

func (s *CostService) trackTenantUsage(req *CostTrackingRequest) error {
	tracker, exists := s.tenantUsage[req.TenantID]
	if !exists {
		tracker = &TenantCostTracker{
			TenantID:      req.TenantID,
			ModelUsage:    make(map[string]*ModelUsage),
			ProviderUsage: make(map[string]*ProviderUsage),
			BudgetLimit:   s.budgetLimits.TenantDailyLimit,
			AlertsEnabled: true,
		}
		s.tenantUsage[req.TenantID] = tracker
	}

	tracker.DailyCost += req.Cost
	tracker.MonthlyCost += req.Cost
	tracker.RequestCount++
	tracker.LastUpdated = req.Timestamp

	// Update model usage
	if modelUsage, exists := tracker.ModelUsage[req.ModelID]; exists {
		modelUsage.RequestCount++
		modelUsage.TokensUsed += req.TokensUsed
		modelUsage.Cost += req.Cost
		modelUsage.AvgLatency = (modelUsage.AvgLatency + req.LatencyMs) / 2
		modelUsage.LastUsed = req.Timestamp
	} else {
		tracker.ModelUsage[req.ModelID] = &ModelUsage{
			ModelID:      req.ModelID,
			RequestCount: 1,
			TokensUsed:   req.TokensUsed,
			Cost:         req.Cost,
			AvgLatency:   req.LatencyMs,
			LastUsed:     req.Timestamp,
		}
	}

	return nil
}

func (s *CostService) trackServiceUsage(req *CostTrackingRequest) error {
	if req.ServiceName == "" {
		return nil // Skip if no service name provided
	}

	tracker, exists := s.serviceUsage[req.ServiceName]
	if !exists {
		tracker = &ServiceCostTracker{
			ServiceName:    req.ServiceName,
			ModelBreakdown: make(map[string]float64),
		}
		s.serviceUsage[req.ServiceName] = tracker
	}

	tracker.DailyCost += req.Cost
	tracker.MonthlyCost += req.Cost
	tracker.RequestCount++
	tracker.LastUpdated = req.Timestamp
	tracker.ModelBreakdown[req.ModelID] += req.Cost

	return nil
}

func (s *CostService) checkBudgetLimits(req *CostTrackingRequest) {
	// Check if we've exceeded 80% of global budget
	if s.totalCostToday >= s.budgetLimits.GlobalDailyLimit*0.8 {
		s.logger.Warn("Approaching global daily budget limit",
			logger.F("current_cost", s.totalCostToday),
			logger.F("limit", s.budgetLimits.GlobalDailyLimit),
			logger.F("utilization", s.totalCostToday/s.budgetLimits.GlobalDailyLimit*100),
		)
	}

	// Check tenant budget
	if tracker, exists := s.tenantUsage[req.TenantID]; exists {
		if tracker.DailyCost >= tracker.BudgetLimit*0.8 {
			s.logger.Warn("Tenant approaching budget limit",
				logger.F("tenant_id", req.TenantID),
				logger.F("current_cost", tracker.DailyCost),
				logger.F("limit", tracker.BudgetLimit),
			)
		}
	}
}

func getDefaultAlertThresholds() []AlertThreshold {
	return []AlertThreshold{
		{
			Type:      AlertTypeDailyBudget,
			Threshold: 0.8, // 80% of daily budget
			Enabled:   true,
		},
		{
			Type:      AlertTypeSpike,
			Threshold: 5.0, // $5 spike in 1 hour
			Enabled:   true,
		},
	}
}