# QLens Usage Analytics & Cost Management

## Overview

QLens provides comprehensive usage analytics and cost management for AI/LLM workloads across multiple providers (AWS Bedrock, Azure OpenAI) with real-time tracking, budgeting, and alerting.

## Features

- **Real-time Cost Tracking** - Per-request cost calculation and tracking
- **Multi-tenant Billing** - Usage isolation and tracking per tenant
- **Service-level Allocation** - Cost attribution to consuming services
- **Budget Management** - Daily/monthly limits with compliance checking
- **Usage Analytics** - Detailed breakdowns by model, provider, tenant
- **Alert System** - Proactive notifications at budget thresholds

## API Endpoints

### Usage Statistics

Access usage analytics through the QLens Gateway at `/v1/usage` with different scopes:

#### Tenant Usage
```bash
curl "http://192.168.1.240/v1/usage?scope=tenant&period=daily" \
  -H "X-API-Key: your-api-key" \
  -H "X-Tenant-ID: your-tenant-id"
```

**Response:**
```json
{
  "tenant_id": "your-tenant-id",
  "daily_cost": 5.67,
  "monthly_cost": 156.78,
  "request_count": 234,
  "model_usage": {
    "claude-3-sonnet": {
      "request_count": 45,
      "tokens_used": 12450,
      "cost": 2.34,
      "avg_latency_ms": 850.5
    },
    "gpt-4": {
      "request_count": 30,
      "tokens_used": 8900,
      "cost": 3.33,
      "avg_latency_ms": 1200.2
    }
  },
  "budget_limit": 50.0,
  "last_updated": "2025-09-07T17:15:00Z"
}
```

#### Cost Summary
```bash
curl "http://192.168.1.240/v1/usage?scope=summary" \
  -H "X-API-Key: your-api-key" \
  -H "X-Tenant-ID: your-tenant-id"
```

**Response:**
```json
{
  "daily_cost": 12.45,
  "request_count": 1247,
  "active_tenants": 42,
  "active_services": 8,
  "budget_utilization_percent": 62.3,
  "status": "healthy",
  "last_updated": "2025-09-07T17:15:00Z"
}
```

#### Global Usage (Admin)
```bash
curl "http://192.168.1.240/v1/usage?scope=global" \
  -H "X-API-Key: admin-api-key" \
  -H "X-Tenant-ID: admin-tenant"
```

**Response:**
```json
{
  "total_cost_today": 12.45,
  "request_count": 1247,
  "active_tenants": 42,
  "active_services": 8,
  "budget_utilization_percent": 62.3,
  "last_updated": "2025-09-07T17:15:00Z"
}
```

## Cost Tracking

### How It Works

1. **Request Interception** - Every LLM request is intercepted by the router
2. **Cost Calculation** - Real-time cost calculation based on model pricing
3. **Budget Compliance** - Pre-request budget validation prevents overruns
4. **Usage Recording** - Post-request tracking updates analytics

### Pricing Models

| Provider | Model | Input Cost (per 1K tokens) | Output Cost (per 1K tokens) |
|----------|-------|----------------------------|------------------------------|
| AWS Bedrock | Claude 3.7 Sonnet | $0.003 | $0.015 |
| AWS Bedrock | Claude 3 Sonnet | $0.003 | $0.015 |
| AWS Bedrock | Claude 3 Haiku | $0.00025 | $0.00125 |
| Azure OpenAI | GPT-4 | $0.03 | $0.06 |
| Azure OpenAI | GPT-4o | $0.005 | $0.015 |
| Azure OpenAI | GPT-5 | $0.01 | $0.03 |

### Budget Configuration

Default budget limits (configurable):
- **Global Daily Limit**: $1,000
- **Global Monthly Limit**: $20,000
- **Tenant Daily Limit**: $50
- **Tenant Monthly Limit**: $1,000

## Architecture

### Components

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Gateway   │───▶│   Router    │───▶│ Providers   │
│             │    │             │    │ (AWS/Azure) │
└─────────────┘    └─────────────┘    └─────────────┘
       │                  │
       │                  ▼
       │            ┌─────────────┐
       │            │ Cost Service│
       │            │             │
       └───────────▶└─────────────┘
          Usage APIs
```

### Data Flow

1. **Request** → Gateway receives LLM request
2. **Validation** → Router checks budget compliance
3. **Routing** → Request sent to appropriate provider
4. **Response** → LLM response with usage metadata
5. **Tracking** → Cost service records usage and cost
6. **Analytics** → Real-time updates to usage statistics

## Budget Alerts

### Alert Types

- **Daily Budget**: 80% of daily limit reached
- **Monthly Budget**: 80% of monthly limit reached  
- **Tenant Budget**: Per-tenant limit approaching
- **Cost Spike**: Unusual cost increase detected

### Alert Thresholds

- **Warning**: 75% of budget consumed
- **Critical**: 90% of budget consumed
- **Blocked**: 100% of budget consumed (requests rejected)

## Implementation Details

### Cost Service

Located in `internal/services/cost/service.go`:
- Thread-safe in-memory tracking
- Real-time budget compliance checking
- Configurable alert thresholds
- Daily/monthly counter resets

### Router Integration

Cost tracking integrated at router level:
- Pre-request budget validation
- Post-request cost recording
- Provider-specific pricing
- Request metadata enrichment

### Gateway Endpoints

Usage analytics exposed via gateway:
- Tenant-scoped access control
- Multiple query scopes (tenant/global/summary)
- Real-time data from cost service
- RESTful API design

## Monitoring

### Health Checks

Budget utilization status:
- **Healthy**: < 75% budget used
- **Warning**: 75-90% budget used  
- **Critical**: > 90% budget used

### Metrics

Key metrics tracked:
- Requests per second
- Cost per request
- Budget utilization percentage
- Provider success rates
- Average response latency

## Security

### Access Control

- **Tenant Isolation**: Users can only access their tenant's data
- **API Authentication**: Required for all analytics endpoints
- **Admin Access**: Global statistics require admin permissions

### Data Privacy

- No request content stored
- Only metadata and usage statistics tracked
- GDPR compliant data handling

## Deployment

QLens v1.0.9+ includes full usage analytics:
- **Gateway**: Handles external API requests
- **Router**: Manages cost tracking and provider routing  
- **Kubernetes**: Production deployment with Istio service mesh
- **Access**: Unified endpoint at `http://192.168.1.240`

## Configuration

Environment variables for cost management:
- `GLOBAL_DAILY_LIMIT`: Global daily budget limit
- `TENANT_DAILY_LIMIT`: Default tenant daily limit
- `ALERT_WEBHOOK_URL`: Webhook for budget alerts
- `COST_TRACKING_ENABLED`: Enable/disable cost tracking

## Examples

### Check Current Usage
```bash
# Get your tenant's usage for today
curl "http://192.168.1.240/v1/usage?scope=tenant" \
  -H "X-API-Key: $QLENS_API_KEY" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Monitor Budget Health
```bash
# Get cost summary with health status
curl "http://192.168.1.240/v1/usage?scope=summary" \
  -H "X-API-Key: $QLENS_API_KEY" \
  -H "X-Tenant-ID: $TENANT_ID" | jq '.status'
```

### Generate LLM Request with Cost Tracking
```bash
# Make LLM request - cost will be automatically tracked
curl "http://192.168.1.240/v1/completions" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $QLENS_API_KEY" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "model": "claude-3-sonnet",
    "messages": [{"role": "user", "content": "Hello!"}],
    "max_tokens": 100
  }'
```

## Troubleshooting

### Common Issues

1. **Budget Exceeded**: Request rejected with quota error
   - Check current usage: `curl .../usage?scope=tenant`
   - Contact admin to increase limits

2. **No Usage Data**: Analytics show zero usage
   - Ensure requests are going through QLens gateway
   - Check authentication headers

3. **High Costs**: Unexpected budget consumption
   - Review model usage breakdown
   - Check for high-token requests
   - Monitor request frequency

### Support

For issues with usage analytics:
- Check logs: `kubectl logs deployment/qlens-router -n staging`
- Health status: `curl http://192.168.1.240/health`
- GitHub Issues: https://github.com/quantumlayerplatform-hq/quantum-suite-platform/issues