# QLens Deployment Guide

This guide covers the deployment of QLens LLM Gateway Service across staging and production environments.

## Prerequisites

- Kubernetes cluster access
- Helm 3.x installed
- kubectl configured
- Azure OpenAI and/or AWS Bedrock credentials
- GitHub Container Registry (GHCR) access

## Quick Start

### Staging Deployment

1. **Set Environment Variables**
   ```bash
   export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com"
   export AZURE_OPENAI_API_KEY="your-api-key"
   export AWS_REGION="us-east-1"
   export AWS_ACCESS_KEY_ID="your-access-key"
   export AWS_SECRET_ACCESS_KEY="your-secret-key"
   ```

2. **Deploy to Staging**
   ```bash
   ./scripts/deploy-staging.sh
   ```

### Production Deployment

1. **Ensure Production Context**
   ```bash
   kubectl config current-context
   # Should show production cluster
   ```

2. **Deploy Tagged Version**
   ```bash
   ./scripts/deploy-production.sh v1.0.0
   ```

## Architecture Overview

QLens consists of three main microservices:

- **Gateway**: Public API endpoint, handles authentication and validation
- **Router**: Routes requests to appropriate providers, handles load balancing
- **Cache**: Manages response caching for improved performance

## Environment Configuration

### Staging Environment
- **Namespace**: `qlens-staging`
- **Domain**: `qlens-staging.local`
- **Cost Limit**: $100/day
- **Replicas**: 1 per service
- **Cache**: Memory-based
- **Monitoring**: Basic metrics

### Production Environment
- **Namespace**: `qlens-production`  
- **Domain**: `qlens.quantumlayer.ai`
- **Cost Limit**: $1000/day
- **Replicas**: 3 gateway, 3 router, 2 cache
- **Cache**: Redis-based
- **Monitoring**: Full observability stack
- **Security**: TLS, authentication required

## Configuration Management

### Helm Values Structure

```yaml
# Core configuration
environment: staging|production
namespace: qlens-staging|qlens-production

# Provider configuration
providers:
  azureOpenAI:
    enabled: true
    deployments:
      gpt-4: "gpt-4"
      gpt-35-turbo: "gpt-35-turbo"
  awsBedrock:
    enabled: true
    models:
      - id: "claude-3-sonnet"
        modelId: "anthropic.claude-3-sonnet-20240229-v1:0"
        name: "Claude 3 Sonnet"

# Cost controls
costControls:
  enabled: true
  dailyLimits:
    total: 1000
    perTenant: 200
    perUser: 50
```

### Environment-Specific Overrides

Use `values-staging.yaml` and `values-production.yaml` for environment-specific configurations.

## Secrets Management

### Staging
Secrets are created directly via kubectl:

```bash
kubectl create secret generic qlens-secrets \
  --from-literal=azure-openai-endpoint="${AZURE_OPENAI_ENDPOINT}" \
  --from-literal=azure-openai-api-key="${AZURE_OPENAI_API_KEY}" \
  --namespace=qlens-staging
```

### Production
Use Azure Key Vault with External Secrets Operator:

```yaml
externalSecrets:
  enabled: true
  backend: azurekv
  refreshInterval: 3600
```

## Health Checks and Monitoring

### Health Endpoints

- **Gateway**: `GET /health` and `GET /health/ready`
- **Router**: `GET /health` and `GET /health/ready`  
- **Cache**: `GET /health` and `GET /health/ready`

### Key Metrics

- Request rate and latency
- Provider health and performance
- Cost tracking and limits
- Cache hit rates
- Resource utilization

### Alerting

Prometheus alerts are configured for:
- Service availability
- High error rates
- Cost thresholds
- Provider failures
- Performance degradation

## Scaling

### Horizontal Pod Autoscaler (HPA)

Production environment includes HPA configuration:

```yaml
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
```

### Manual Scaling

```bash
kubectl scale deployment qlens-gateway --replicas=5 -n qlens-production
```

## Rollback Procedures

### Helm Rollback

```bash
helm rollback qlens --namespace=qlens-production
```

### Emergency Rollback

If automated rollback fails:

1. Check deployment status
   ```bash
   kubectl get pods -n qlens-production
   kubectl describe deployment qlens-gateway -n qlens-production
   ```

2. Scale down new version
   ```bash
   kubectl scale deployment qlens-gateway --replicas=0 -n qlens-production
   ```

3. Scale up previous version (if available)
   ```bash
   kubectl scale deployment qlens-gateway-previous --replicas=3 -n qlens-production
   ```

## Troubleshooting

### Common Issues

1. **Pod Startup Failures**
   ```bash
   kubectl logs -l app.kubernetes.io/name=qlens -n qlens-production
   kubectl describe pod <pod-name> -n qlens-production
   ```

2. **Provider Authentication Errors**
   - Verify secrets are correctly mounted
   - Check API key validity
   - Confirm endpoint URLs

3. **High Latency**
   - Check provider health status
   - Review cache hit rates
   - Examine resource constraints

4. **Cost Limit Exceeded**
   - Review cost control configuration
   - Check tenant usage patterns
   - Adjust daily limits if necessary

### Debug Commands

```bash
# Check service status
kubectl get all -n qlens-production

# View logs
kubectl logs -f deployment/qlens-gateway -n qlens-production

# Port forward for local testing
kubectl port-forward svc/qlens-gateway 8080:8080 -n qlens-production

# Execute into pod
kubectl exec -it <pod-name> -n qlens-production -- /bin/sh
```

## Maintenance Windows

### Planned Maintenance

1. **Pre-maintenance**
   - Notify stakeholders
   - Scale up replicas for redundancy
   - Backup current configuration

2. **During Maintenance**
   - Rolling updates preferred
   - Monitor health checks
   - Keep one service instance running

3. **Post-maintenance**
   - Verify all services healthy
   - Run smoke tests
   - Scale back to normal levels

### Emergency Maintenance

- Use deployment scripts with `--wait` flag
- Monitor metrics dashboard closely
- Have rollback plan ready

## Security Considerations

### Network Policies

Implement Kubernetes Network Policies to restrict traffic:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: qlens-network-policy
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: qlens
  policyTypes:
  - Ingress
  - Egress
```

### RBAC

Service accounts have minimal required permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: qlens-role
rules:
- apiGroups: [""]
  resources: ["secrets", "configmaps"]
  verbs: ["get", "list"]
```

### Resource Limits

All services have resource limits configured:

```yaml
resources:
  limits:
    cpu: 2000m
    memory: 2Gi
  requests:
    cpu: 1000m
    memory: 1Gi
```

## Performance Tuning

### Gateway Service
- Adjust connection pool sizes
- Configure timeout values
- Enable request compression

### Router Service  
- Tune circuit breaker settings
- Optimize load balancing algorithms
- Configure retry policies

### Cache Service
- Size cache appropriately
- Set optimal TTL values
- Monitor hit/miss ratios

## Disaster Recovery

### Backup Strategy

1. **Configuration Backup**
   ```bash
   helm get values qlens -n qlens-production > backup/values-$(date +%Y%m%d).yaml
   ```

2. **State Backup**
   - Cache data (if persistent)
   - Usage statistics
   - Configuration secrets

### Recovery Procedures

1. **Service Recovery**
   ```bash
   # Redeploy from backup
   helm upgrade qlens charts/qlens \
     --namespace qlens-production \
     --values backup/values-20240101.yaml
   ```

2. **Data Recovery**
   - Restore Redis cache (if applicable)
   - Reimport configuration
   - Verify provider connections

## Contact Information

- **On-call Engineer**: See PagerDuty rotation
- **Platform Team**: platform@quantumlayer.ai
- **Documentation**: https://docs.quantumlayer.ai/qlens
- **Monitoring**: https://grafana.quantumlayer.ai/d/qlens-dashboard