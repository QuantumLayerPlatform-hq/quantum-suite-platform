# QLens - LLM Gateway Service

QLens is a production-ready LLM Gateway Service that provides a unified API for multiple Large Language Model providers, with built-in cost controls, monitoring, and enterprise-grade features.

## 🌟 Features

### Core Capabilities
- **Multi-Provider Support**: Azure OpenAI, AWS Bedrock Claude 3, with extensible provider architecture
- **Unified API**: Single OpenAI-compatible API for all providers
- **Intelligent Routing**: Cost-based routing and load balancing across providers
- **Response Caching**: Redis and in-memory caching for improved performance
- **Cost Controls**: Real-time cost tracking with daily limits and alerts
- **Rate Limiting**: Configurable rate limits per tenant and user
- **Streaming Support**: Server-sent events for real-time responses

### Enterprise Features
- **Multi-Tenancy**: Full tenant isolation with separate limits and monitoring
- **Authentication**: JWT-based authentication with configurable providers
- **Monitoring**: Prometheus metrics with Grafana dashboards
- **Health Checks**: Comprehensive health monitoring with circuit breakers
- **Security**: TLS encryption, secret management, and network policies
- **Scalability**: Horizontal pod autoscaling and cloud-native architecture

### Operational Excellence
- **Cloud Native**: Kubernetes-first with Helm charts
- **CI/CD Pipeline**: GitHub Actions with automated testing and deployment
- **Observability**: Structured logging, distributed tracing, and alerting
- **Documentation**: Comprehensive runbooks and operational guides
- **Testing**: 80%+ code coverage with unit, integration, and benchmark tests

## 🏗️ Architecture

QLens follows a microservices architecture with three main components:

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Gateway   │───▶│   Router    │───▶│   Cache     │
│             │    │             │    │             │
│ • Auth      │    │ • Routing   │    │ • Redis     │
│ • Rate Limit│    │ • Providers │    │ • Memory    │
│ • Validation│    │ • Load Bal. │    │ • TTL Mgmt  │
└─────────────┘    └─────────────┘    └─────────────┘
       │                   │                   │
       └───────────────────┼───────────────────┘
                          │
                   ┌─────────────┐
                   │ Monitoring  │
                   │             │
                   │ • Metrics   │
                   │ • Alerts    │
                   │ • Dashboards│
                   └─────────────┘
```

### Service Responsibilities

#### Gateway Service
- Public API endpoint (OpenAI-compatible)
- Authentication and authorization
- Request validation and sanitization
- Rate limiting enforcement
- CORS and security headers
- Request/response logging

#### Router Service
- Provider selection and routing
- Load balancing across providers
- Circuit breaker pattern
- Cost optimization logic
- Health monitoring
- Provider abstraction

#### Cache Service
- Response caching (Redis/Memory)
- Cache invalidation strategies
- TTL management
- Cache warming
- Statistics and metrics

## 🚀 Quick Start

### Prerequisites
- Kubernetes cluster (1.24+)
- Helm 3.x
- Azure OpenAI and/or AWS Bedrock credentials

### Staging Deployment

1. **Set up credentials**
   ```bash
   export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
   export AZURE_OPENAI_API_KEY="your-api-key"
   export AWS_REGION="us-east-1"
   export AWS_ACCESS_KEY_ID="your-access-key"
   export AWS_SECRET_ACCESS_KEY="your-secret-key"
   ```

2. **Deploy to staging**
   ```bash
   ./scripts/deploy-staging.sh
   ```

3. **Test the API**
   ```bash
   curl -X POST http://qlens-staging.local/v1/completions \
     -H "Authorization: Bearer your-token" \
     -H "X-Tenant-ID: test-tenant" \
     -H "Content-Type: application/json" \
     -d '{
       "model": "gpt-35-turbo",
       "messages": [{"role": "user", "content": [{"type": "text", "text": "Hello!"}]}]
     }'
   ```

### Production Deployment

```bash
./scripts/deploy-production.sh v1.0.0
```

## 📖 API Documentation

### Authentication
All requests require an `Authorization` header with a Bearer token and a `X-Tenant-ID` header.

### Endpoints

#### Create Completion
```http
POST /v1/completions
Content-Type: application/json
Authorization: Bearer <token>
X-Tenant-ID: <tenant-id>

{
  "model": "gpt-35-turbo",
  "messages": [
    {"role": "user", "content": [{"type": "text", "text": "Hello, world!"}]}
  ],
  "max_tokens": 100,
  "temperature": 0.7,
  "stream": false
}
```

#### List Models
```http
GET /v1/models
Authorization: Bearer <token>
X-Tenant-ID: <tenant-id>
```

#### Create Embeddings
```http
POST /v1/embeddings
Content-Type: application/json
Authorization: Bearer <token>
X-Tenant-ID: <tenant-id>

{
  "model": "text-embedding-ada-002",
  "input": ["Hello, world!"]
}
```

### Supported Models

#### Azure OpenAI
- `gpt-4` - GPT-4 (8K context)
- `gpt-35-turbo` - GPT-3.5 Turbo (4K context)
- `text-embedding-ada-002` - Text Embeddings

#### AWS Bedrock
- `claude-3-sonnet` - Claude 3 Sonnet (200K context)
- `claude-3-haiku` - Claude 3 Haiku (200K context)

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ENVIRONMENT` | Deployment environment | `development` |
| `LOG_LEVEL` | Logging level | `info` |
| `PORT` | Service port | `8080` |
| `AZURE_OPENAI_ENDPOINT` | Azure OpenAI endpoint | - |
| `AZURE_OPENAI_API_KEY` | Azure OpenAI API key | - |
| `AWS_REGION` | AWS region | `us-east-1` |
| `AWS_ACCESS_KEY_ID` | AWS access key | - |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key | - |

### Helm Configuration

Key configuration options in `values.yaml`:

```yaml
# Cost controls
costControls:
  enabled: true
  dailyLimits:
    total: 1000      # Total daily limit in USD
    perTenant: 200   # Per-tenant daily limit
    perUser: 50      # Per-user daily limit

# Rate limiting
rateLimit:
  enabled: true
  global:
    requestsPerMinute: 10000
    tokensPerHour: 1000000

# Caching
cache:
  enabled: true
  type: redis      # redis or memory
  ttl: 3600       # Cache TTL in seconds
  maxSize: 10000  # Max cache entries
```

## 📊 Monitoring

### Key Metrics

- **Request Metrics**: Rate, latency, errors by service and provider
- **Provider Metrics**: Health, latency, error rates per provider
- **Cost Metrics**: Spend tracking by tenant, user, model, and provider
- **Cache Metrics**: Hit/miss rates, evictions, latency
- **System Metrics**: CPU, memory, connections

### Grafana Dashboard

Access the pre-built dashboard at `https://grafana.quantumlayer.ai/d/qlens-dashboard`

Key panels include:
- Request rate and error rate
- Response time percentiles
- Provider health and performance
- Cost tracking and trends
- Cache performance
- System resource usage

### Alerting

Prometheus alerts are configured for:
- Service downtime
- High error rates
- Cost threshold breaches
- Provider failures
- Performance degradation

## 🔐 Security

### Authentication & Authorization
- JWT token validation
- Configurable token issuers and audiences
- Tenant-based access control
- API key rotation support

### Network Security
- TLS encryption for all external traffic
- Kubernetes Network Policies
- Service mesh support (Istio)
- Ingress controller integration

### Secrets Management
- Kubernetes Secrets for staging
- Azure Key Vault integration for production
- External Secrets Operator support
- Secret rotation procedures

## 🧪 Testing

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests
INTEGRATION_TEST=true go test -tags=integration ./tests/integration/...

# Benchmark tests
go test -bench=. ./tests/benchmarks/...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Coverage
Current test coverage: **85%+**

- Unit tests for all core business logic
- Integration tests for API endpoints
- Provider integration tests
- Performance benchmark tests
- Chaos engineering tests

## 🚦 CI/CD Pipeline

GitHub Actions workflow includes:

1. **Code Quality**
   - Linting (golint, staticcheck)
   - Security scanning (gosec)
   - Dependency vulnerability checks

2. **Testing**
   - Unit tests with coverage reporting
   - Integration tests
   - Benchmark tests

3. **Build & Deploy**
   - Multi-arch Docker builds (amd64, arm64)
   - Push to GitHub Container Registry
   - Helm chart validation
   - Automated deployments

4. **Security**
   - Container image scanning
   - SBOM generation
   - Secret scanning

## 📚 Documentation

### Operational Documentation
- [Deployment Guide](./docs/runbooks/deployment-guide.md)
- [Incident Response](./docs/runbooks/incident-response.md)
- [Monitoring & Alerting](./docs/monitoring/README.md)
- [Security Guidelines](./docs/security/README.md)

### Development Documentation
- [API Reference](./docs/api/README.md)
- [Provider Integration](./docs/development/providers.md)
- [Contributing Guidelines](./CONTRIBUTING.md)
- [Architecture Decision Records](./docs/adr/)

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](./CONTRIBUTING.md) for details.

### Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/quantumlayerplatform/qlens.git
   cd qlens
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Run tests**
   ```bash
   make test
   ```

4. **Run locally**
   ```bash
   make run
   ```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.

## 🆘 Support

### Getting Help
- 📖 [Documentation](https://docs.quantumlayer.ai/qlens)
- 💬 [GitHub Discussions](https://github.com/quantumlayerplatform/qlens/discussions)
- 🐛 [Issue Tracker](https://github.com/quantumlayerplatform/qlens/issues)
- 📧 Email: support@quantumlayer.ai

### Enterprise Support
For enterprise customers:
- 24/7 technical support
- SLA guarantees
- Professional services
- Custom integrations

Contact: enterprise@quantumlayer.ai

## 🗺️ Roadmap

### Current Version (v1.0)
- ✅ Multi-provider support (Azure OpenAI, AWS Bedrock)
- ✅ Cost controls and monitoring
- ✅ Kubernetes deployment
- ✅ Comprehensive testing

### Next Release (v1.1)
- 🔄 Additional providers (OpenAI, Anthropic Direct)
- 🔄 Advanced caching strategies
- 🔄 Enhanced cost optimization
- 🔄 Multi-region deployments

### Future Releases
- 📋 Fine-tuning pipeline integration
- 📋 Advanced analytics and insights
- 📋 Workflow orchestration
- 📋 Edge deployment support

---

Made with ❤️ by the Quantum Layer Platform Team