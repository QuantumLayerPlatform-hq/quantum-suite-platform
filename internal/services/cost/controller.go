package cost

import (
	"context"
	"sync"
	"time"

	"github.com/quantum-suite/platform/internal/domain"
	"github.com/quantum-suite/platform/pkg/shared/logger"
)

// CostController manages cost controls and tracking
type CostController struct {
	logger         logger.Logger
	config         CostConfig
	usageStore     UsageStore
	alertManager   AlertManager
	mu             sync.RWMutex
	dailyUsage     map[domain.TenantID]*DailyUsage
	userUsage      map[UserKey]*DailyUsage
	globalUsage    *DailyUsage
	lastReset      time.Time
}

type CostConfig struct {
	Enabled         bool                       `json:"enabled"`
	DailyLimits     DailyLimits               `json:"daily_limits"`
	AlertThresholds []int                     `json:"alert_thresholds"`
	ProviderRanking []domain.Provider         `json:"provider_ranking"`
	CostOptimization CostOptimizationConfig   `json:"cost_optimization"`
}

type DailyLimits struct {
	Total     float64 `json:"total"`
	PerTenant float64 `json:"per_tenant"`
	PerUser   float64 `json:"per_user"`
}

type CostOptimizationConfig struct {
	Enabled             bool `json:"enabled"`
	RouteByComplexity   bool `json:"route_by_complexity"`
	PreferCheaperModels bool `json:"prefer_cheaper_models"`
}

type DailyUsage struct {
	Cost      float64                   `json:"cost"`
	Requests  int                      `json:"requests"`
	Tokens    int                      `json:"tokens"`
	ByModel   map[string]ModelUsage    `json:"by_model"`
	ByProvider map[domain.Provider]float64 `json:"by_provider"`
	Date      time.Time                `json:"date"`
}

type ModelUsage struct {
	Cost     float64 `json:"cost"`
	Requests int     `json:"requests"`
	Tokens   int     `json:"tokens"`
}

type UserKey struct {
	TenantID domain.TenantID
	UserID   domain.UserID
}

type UsageRecord struct {
	TenantID  domain.TenantID   `json:"tenant_id"`
	UserID    domain.UserID     `json:"user_id"`
	Provider  domain.Provider   `json:"provider"`
	Model     string            `json:"model"`
	Cost      float64           `json:"cost"`
	Requests  int               `json:"requests"`
	Tokens    int               `json:"tokens"`
	Timestamp time.Time         `json:"timestamp"`
}

type CostCheckResult struct {
	Allowed          bool     `json:"allowed"`
	Reason           string   `json:"reason,omitempty"`
	RemainingBudget  float64  `json:"remaining_budget"`
	RecommendedProvider *domain.Provider `json:"recommended_provider,omitempty"`
}

type UsageStore interface {
	RecordUsage(ctx context.Context, record UsageRecord) error
	GetDailyUsage(ctx context.Context, tenantID domain.TenantID, date time.Time) (*DailyUsage, error)
	GetUserDailyUsage(ctx context.Context, tenantID domain.TenantID, userID domain.UserID, date time.Time) (*DailyUsage, error)
	GetGlobalDailyUsage(ctx context.Context, date time.Time) (*DailyUsage, error)
}

type AlertManager interface {
	SendCostAlert(ctx context.Context, alert CostAlert) error
}

type CostAlert struct {
	Type       string            `json:"type"`
	TenantID   domain.TenantID   `json:"tenant_id"`
	UserID     domain.UserID     `json:"user_id,omitempty"`
	Threshold  int               `json:"threshold"`
	Current    float64           `json:"current"`
	Limit      float64           `json:"limit"`
	Timestamp  time.Time         `json:"timestamp"`
}

func NewCostController(config CostConfig, store UsageStore, alertManager AlertManager, logger logger.Logger) *CostController {
	return &CostController{
		logger:       logger,
		config:       config,
		usageStore:   store,
		alertManager: alertManager,
		dailyUsage:   make(map[domain.TenantID]*DailyUsage),
		userUsage:    make(map[UserKey]*DailyUsage),
		globalUsage:  &DailyUsage{
			ByModel:    make(map[string]ModelUsage),
			ByProvider: make(map[domain.Provider]float64),
			Date:       time.Now().UTC().Truncate(24 * time.Hour),
		},
		lastReset: time.Now().UTC().Truncate(24 * time.Hour),
	}
}

func (c *CostController) CheckRequestAllowed(ctx context.Context, req CostCheckRequest) (*CostCheckResult, error) {
	if !c.config.Enabled {
		return &CostCheckResult{
			Allowed:         true,
			RemainingBudget: -1, // Unlimited
		}, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Reset daily counters if new day
	c.resetDailyCountersIfNeeded()

	// Estimate cost for the request
	estimatedCost := c.estimateRequestCost(req)

	// Check global limit
	if c.config.DailyLimits.Total > 0 {
		if c.globalUsage.Cost+estimatedCost > c.config.DailyLimits.Total {
			return &CostCheckResult{
				Allowed:         false,
				Reason:          "Global daily limit exceeded",
				RemainingBudget: c.config.DailyLimits.Total - c.globalUsage.Cost,
			}, nil
		}
	}

	// Check tenant limit
	if c.config.DailyLimits.PerTenant > 0 {
		tenantUsage := c.getTenantUsage(req.TenantID)
		if tenantUsage.Cost+estimatedCost > c.config.DailyLimits.PerTenant {
			return &CostCheckResult{
				Allowed:         false,
				Reason:          "Tenant daily limit exceeded",
				RemainingBudget: c.config.DailyLimits.PerTenant - tenantUsage.Cost,
			}, nil
		}
	}

	// Check user limit
	if c.config.DailyLimits.PerUser > 0 {
		userKey := UserKey{TenantID: req.TenantID, UserID: req.UserID}
		userUsage := c.getUserUsage(userKey)
		if userUsage.Cost+estimatedCost > c.config.DailyLimits.PerUser {
			return &CostCheckResult{
				Allowed:         false,
				Reason:          "User daily limit exceeded",
				RemainingBudget: c.config.DailyLimits.PerUser - userUsage.Cost,
			}, nil
		}
	}

	result := &CostCheckResult{
		Allowed:         true,
		RemainingBudget: c.calculateRemainingBudget(req.TenantID, req.UserID),
	}

	// Provide cost optimization recommendations
	if c.config.CostOptimization.Enabled {
		recommended := c.recommendProvider(req)
		if recommended != nil {
			result.RecommendedProvider = recommended
		}
	}

	return result, nil
}

func (c *CostController) RecordUsage(ctx context.Context, record UsageRecord) error {
	if !c.config.Enabled {
		return nil
	}

	// Store in persistent storage
	if err := c.usageStore.RecordUsage(ctx, record); err != nil {
		c.logger.Error("Failed to store usage record", logger.F("error", err))
		// Continue with in-memory tracking even if storage fails
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Reset daily counters if needed
	c.resetDailyCountersIfNeeded()

	// Update global usage
	c.updateUsageRecord(c.globalUsage, record)

	// Update tenant usage
	tenantUsage := c.getTenantUsage(record.TenantID)
	c.updateUsageRecord(tenantUsage, record)

	// Update user usage
	userKey := UserKey{TenantID: record.TenantID, UserID: record.UserID}
	userUsage := c.getUserUsage(userKey)
	c.updateUsageRecord(userUsage, record)

	// Check for alert thresholds
	go c.checkAlertThresholds(ctx, record)

	return nil
}

func (c *CostController) GetUsageStats(ctx context.Context, tenantID domain.TenantID, userID domain.UserID) (*UsageStats, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := &UsageStats{
		TenantID:  tenantID,
		UserID:    userID,
		Date:      time.Now().UTC().Truncate(24 * time.Hour),
	}

	// Get tenant usage
	if tenantUsage, exists := c.dailyUsage[tenantID]; exists {
		stats.TenantUsage = *tenantUsage
	}

	// Get user usage
	userKey := UserKey{TenantID: tenantID, UserID: userID}
	if userUsage, exists := c.userUsage[userKey]; exists {
		stats.UserUsage = *userUsage
	}

	// Get global usage
	stats.GlobalUsage = *c.globalUsage

	// Calculate remaining budgets
	stats.RemainingBudgets.Global = c.config.DailyLimits.Total - c.globalUsage.Cost
	stats.RemainingBudgets.Tenant = c.config.DailyLimits.PerTenant - stats.TenantUsage.Cost
	stats.RemainingBudgets.User = c.config.DailyLimits.PerUser - stats.UserUsage.Cost

	return stats, nil
}

type CostCheckRequest struct {
	TenantID           domain.TenantID `json:"tenant_id"`
	UserID             domain.UserID   `json:"user_id"`
	Model              string          `json:"model"`
	EstimatedTokens    int             `json:"estimated_tokens"`
	Provider           domain.Provider `json:"provider"`
}

type UsageStats struct {
	TenantID          domain.TenantID `json:"tenant_id"`
	UserID            domain.UserID   `json:"user_id"`
	Date              time.Time       `json:"date"`
	TenantUsage       DailyUsage      `json:"tenant_usage"`
	UserUsage         DailyUsage      `json:"user_usage"`
	GlobalUsage       DailyUsage      `json:"global_usage"`
	RemainingBudgets  struct {
		Global float64 `json:"global"`
		Tenant float64 `json:"tenant"`
		User   float64 `json:"user"`
	} `json:"remaining_budgets"`
}

func (c *CostController) resetDailyCountersIfNeeded() {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	if now.After(c.lastReset) {
		c.logger.Info("Resetting daily cost counters")
		
		c.dailyUsage = make(map[domain.TenantID]*DailyUsage)
		c.userUsage = make(map[UserKey]*DailyUsage)
		c.globalUsage = &DailyUsage{
			ByModel:    make(map[string]ModelUsage),
			ByProvider: make(map[domain.Provider]float64),
			Date:       now,
		}
		c.lastReset = now
	}
}

func (c *CostController) getTenantUsage(tenantID domain.TenantID) *DailyUsage {
	if usage, exists := c.dailyUsage[tenantID]; exists {
		return usage
	}
	
	usage := &DailyUsage{
		ByModel:    make(map[string]ModelUsage),
		ByProvider: make(map[domain.Provider]float64),
		Date:       time.Now().UTC().Truncate(24 * time.Hour),
	}
	c.dailyUsage[tenantID] = usage
	return usage
}

func (c *CostController) getUserUsage(userKey UserKey) *DailyUsage {
	if usage, exists := c.userUsage[userKey]; exists {
		return usage
	}
	
	usage := &DailyUsage{
		ByModel:    make(map[string]ModelUsage),
		ByProvider: make(map[domain.Provider]float64),
		Date:       time.Now().UTC().Truncate(24 * time.Hour),
	}
	c.userUsage[userKey] = usage
	return usage
}

func (c *CostController) updateUsageRecord(usage *DailyUsage, record UsageRecord) {
	usage.Cost += record.Cost
	usage.Requests += record.Requests
	usage.Tokens += record.Tokens
	
	// Update by model
	if modelUsage, exists := usage.ByModel[record.Model]; exists {
		modelUsage.Cost += record.Cost
		modelUsage.Requests += record.Requests
		modelUsage.Tokens += record.Tokens
		usage.ByModel[record.Model] = modelUsage
	} else {
		usage.ByModel[record.Model] = ModelUsage{
			Cost:     record.Cost,
			Requests: record.Requests,
			Tokens:   record.Tokens,
		}
	}
	
	// Update by provider
	usage.ByProvider[record.Provider] += record.Cost
}

func (c *CostController) estimateRequestCost(req CostCheckRequest) float64 {
	// This is a simplified estimation
	// In reality, you'd have more sophisticated cost models based on provider pricing
	
	baseCostPerToken := 0.000002 // Base cost estimation
	
	switch req.Provider {
	case domain.ProviderAzureOpenAI:
		if req.Model == "gpt-4" {
			baseCostPerToken = 0.00003
		} else if req.Model == "gpt-35-turbo" {
			baseCostPerToken = 0.0000015
		}
	case domain.ProviderAWSBedrock:
		baseCostPerToken = 0.000003 // Claude Sonnet pricing
	}
	
	return float64(req.EstimatedTokens) * baseCostPerToken
}

func (c *CostController) calculateRemainingBudget(tenantID domain.TenantID, userID domain.UserID) float64 {
	globalRemaining := c.config.DailyLimits.Total - c.globalUsage.Cost
	tenantRemaining := c.config.DailyLimits.PerTenant - c.getTenantUsage(tenantID).Cost
	userKey := UserKey{TenantID: tenantID, UserID: userID}
	userRemaining := c.config.DailyLimits.PerUser - c.getUserUsage(userKey).Cost
	
	// Return the most restrictive limit
	remaining := globalRemaining
	if tenantRemaining < remaining && c.config.DailyLimits.PerTenant > 0 {
		remaining = tenantRemaining
	}
	if userRemaining < remaining && c.config.DailyLimits.PerUser > 0 {
		remaining = userRemaining
	}
	
	return remaining
}

func (c *CostController) recommendProvider(req CostCheckRequest) *domain.Provider {
	if !c.config.CostOptimization.RouteByComplexity {
		return nil
	}
	
	// Simple heuristic: recommend cheaper provider for simpler requests
	if req.EstimatedTokens < 100 {
		// For simple requests, recommend cheaper providers
		for _, provider := range []domain.Provider{
			domain.ProviderAzureOpenAI, // gpt-35-turbo is cheaper than claude
			domain.ProviderAWSBedrock,
		} {
			if provider != req.Provider {
				return &provider
			}
		}
	}
	
	return nil
}

func (c *CostController) checkAlertThresholds(ctx context.Context, record UsageRecord) {
	if c.alertManager == nil {
		return
	}
	
	c.mu.RLock()
	tenantUsage := c.dailyUsage[record.TenantID]
	userKey := UserKey{TenantID: record.TenantID, UserID: record.UserID}
	userUsage := c.userUsage[userKey]
	c.mu.RUnlock()
	
	// Check tenant thresholds
	if tenantUsage != nil && c.config.DailyLimits.PerTenant > 0 {
		percentage := (tenantUsage.Cost / c.config.DailyLimits.PerTenant) * 100
		for _, threshold := range c.config.AlertThresholds {
			if percentage >= float64(threshold) && percentage < float64(threshold)+5 { // Within 5% to avoid spam
				alert := CostAlert{
					Type:      "tenant_threshold",
					TenantID:  record.TenantID,
					Threshold: threshold,
					Current:   tenantUsage.Cost,
					Limit:     c.config.DailyLimits.PerTenant,
					Timestamp: time.Now(),
				}
				c.alertManager.SendCostAlert(ctx, alert)
			}
		}
	}
	
	// Check user thresholds
	if userUsage != nil && c.config.DailyLimits.PerUser > 0 {
		percentage := (userUsage.Cost / c.config.DailyLimits.PerUser) * 100
		for _, threshold := range c.config.AlertThresholds {
			if percentage >= float64(threshold) && percentage < float64(threshold)+5 {
				alert := CostAlert{
					Type:      "user_threshold",
					TenantID:  record.TenantID,
					UserID:    record.UserID,
					Threshold: threshold,
					Current:   userUsage.Cost,
					Limit:     c.config.DailyLimits.PerUser,
					Timestamp: time.Now(),
				}
				c.alertManager.SendCostAlert(ctx, alert)
			}
		}
	}
}