# Quantum Suite Kubernetes Deployment Guide

This guide provides comprehensive instructions for deploying the Quantum Suite platform to a Kubernetes cluster.

## Prerequisites

- Kubernetes cluster (1.25+) with at least 8GB RAM and 4 CPU cores
- kubectl configured and connected to your cluster
- Docker registry access for pushing images
- Optional: Helm 3.x for enhanced deployments

## Quick Start

### 1. Clean Up Existing Cluster

```bash
# Review and clean up your existing cluster
./scripts/cleanup-cluster.sh
```

**⚠️ Warning:** This will remove all non-system resources from your cluster. Make sure you have backups of any important data.

### 2. Deploy Quantum Suite

```bash
# Deploy the complete platform
./scripts/deploy-quantum-suite.sh
```

This will deploy:
- 4 namespaces (quantum-system, quantum-data, quantum-services, quantum-monitoring)
- PostgreSQL with pgvector extension
- Redis for caching
- NATS for messaging
- Qdrant and Weaviate vector databases
- Prometheus, Grafana, and Jaeger for monitoring
- All 6 Quantum Suite services

### 3. Build and Deploy Custom Images (Optional)

```bash
# Build and push your custom images
export DOCKER_REGISTRY="your-registry.com"
export DOCKER_NAMESPACE="your-org"
./scripts/build-and-push-images.sh

# Update deployments to use your images
sed -i 's|nginx:alpine # placeholder-for-|your-registry.com/your-org/|g' deployments/kubernetes/03-quantum-services.yaml
kubectl apply -f deployments/kubernetes/03-quantum-services.yaml
```

## Detailed Deployment Steps

### Step 1: Cluster Preparation

1. **Verify Cluster Resources:**
   ```bash
   kubectl get nodes
   kubectl top nodes  # Requires metrics server
   ```

2. **Check Storage Classes:**
   ```bash
   kubectl get storageclass
   ```

3. **Run Cleanup (if needed):**
   ```bash
   ./scripts/cleanup-cluster.sh
   ```

### Step 2: Database Deployment

The deployment script will create:

1. **PostgreSQL StatefulSet:**
   - Primary database with pgvector extension
   - 20GB persistent storage
   - Automated backups

2. **Redis StatefulSet:**
   - Cache and session storage
   - 10GB persistent storage
   - Memory optimization configured

### Step 3: Vector Databases

1. **Qdrant:**
   - High-performance vector search
   - 20GB persistent storage
   - Optimized for embeddings

2. **Weaviate:**
   - Semantic search capabilities
   - 20GB persistent storage
   - Multi-modal support

### Step 4: Messaging and Monitoring

1. **NATS JetStream:**
   - Event streaming
   - 3-node cluster for HA
   - 5GB storage per node

2. **Monitoring Stack:**
   - Prometheus for metrics
   - Grafana for dashboards
   - Jaeger for distributed tracing

### Step 5: Application Services

1. **API Gateway:**
   - Load balancer service
   - 3 replicas for HA
   - Auto-scaling enabled

2. **Quantum Suite Services:**
   - QAgent (AI code generation)
   - QTest (intelligent testing)
   - QSecure (security operations)
   - QSRE (site reliability)
   - QInfra (infrastructure management)

## Configuration

### Environment Variables

Key configuration is stored in ConfigMaps and Secrets:

```bash
# View current configuration
kubectl get configmap quantum-config -n quantum-system -o yaml

# View secrets (base64 encoded)
kubectl get secret quantum-secrets -n quantum-system -o yaml
```

### API Keys Configuration

Update the secrets with your API keys:

```bash
# OpenAI API Key
kubectl patch secret quantum-secrets -n quantum-system -p '{"data":{"OPENAI_API_KEY":"'$(echo -n 'your-openai-key' | base64)'"}}'

# Anthropic API Key  
kubectl patch secret quantum-secrets -n quantum-system -p '{"data":{"ANTHROPIC_API_KEY":"'$(echo -n 'your-anthropic-key' | base64)'"}}'
```

### Database Migration

Migrations are automatically run during deployment. To manually run migrations:

```bash
# Create migration job
kubectl create job manual-migration --from=cronjob/quantum-migrations -n quantum-data

# Check migration status
kubectl logs job/manual-migration -n quantum-data
```

## Access and Monitoring

### Service Access

1. **Production (LoadBalancer):**
   ```bash
   # Get external IPs
   kubectl get svc -n quantum-services quantum-gateway
   kubectl get svc -n quantum-monitoring grafana jaeger-query
   ```

2. **Development (Port Forwarding):**
   ```bash
   # Use the management script
   ./scripts/manage-cluster.sh port-forward
   
   # Or manually:
   kubectl port-forward -n quantum-services svc/quantum-gateway 8080:8080
   kubectl port-forward -n quantum-monitoring svc/grafana 3000:3000
   kubectl port-forward -n quantum-monitoring svc/jaeger-query 16686:16686
   ```

### Monitoring Dashboards

1. **Grafana:** http://localhost:3000 or http://EXTERNAL-IP:3000
   - Username: `admin`
   - Password: `quantum-admin`

2. **Jaeger:** http://localhost:16686 or http://EXTERNAL-IP:16686
   - Distributed tracing interface

3. **Prometheus:** http://localhost:9090
   - Metrics and alerting

## Management Operations

### Cluster Status

```bash
# Overall status
./scripts/manage-cluster.sh status

# Service logs
./scripts/manage-cluster.sh logs qagent

# All service logs
./scripts/manage-cluster.sh logs
```

### Scaling Services

```bash
# Scale QAgent to 5 replicas
./scripts/manage-cluster.sh scale qagent 5

# Scale all services
kubectl scale deployment --all --replicas=3 -n quantum-services
```

### Backup and Restore

```bash
# Create backup
./scripts/manage-cluster.sh backup

# Restore from backup (manual process)
kubectl apply -f ./backups/quantum-backup-YYYYMMDD-HHMMSS/
```

### Service Restart

```bash
# Restart all services
./scripts/manage-cluster.sh restart

# Restart specific service
kubectl rollout restart deployment/qagent -n quantum-services
```

## Troubleshooting

### Common Issues

1. **Pods Stuck in Pending:**
   ```bash
   kubectl describe pod POD_NAME -n NAMESPACE
   # Check for resource constraints or PVC issues
   ```

2. **Database Connection Issues:**
   ```bash
   # Test database connectivity
   kubectl exec -it -n quantum-data statefulset/postgresql -- psql -U quantum -d quantum -c "SELECT version();"
   ```

3. **Vector Database Issues:**
   ```bash
   # Check Qdrant health
   kubectl exec -it -n quantum-data statefulset/qdrant -- curl -s http://localhost:6333/healthz
   
   # Check Weaviate health
   kubectl exec -it -n quantum-data statefulset/weaviate -- curl -s http://localhost:8080/v1/.well-known/ready
   ```

4. **Service Mesh Issues:**
   ```bash
   # Check Istio sidecars (if using Istio)
   kubectl get pods -n quantum-services -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[*].name}{"\n"}{end}'
   ```

### Cleanup Failed Resources

```bash
# Clean up failed pods and jobs
./scripts/manage-cluster.sh cleanup
```

### Log Analysis

```bash
# Get logs from specific service
kubectl logs -n quantum-services deployment/qagent --tail=100 --follow

# Get previous container logs (after restart)
kubectl logs -n quantum-services deployment/qagent --previous

# Get logs from all containers in a pod
kubectl logs -n quantum-services POD_NAME --all-containers=true
```

## Performance Tuning

### Resource Requests and Limits

Adjust resources based on your cluster capacity:

```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "200m"
  limits:
    memory: "2Gi" 
    cpu: "1000m"
```

### Database Tuning

1. **PostgreSQL:**
   ```bash
   # Access PostgreSQL config
   kubectl exec -it -n quantum-data statefulset/postgresql -- psql -U quantum -d quantum
   
   # Common tuning parameters:
   # shared_buffers = 256MB
   # effective_cache_size = 1GB
   # maintenance_work_mem = 64MB
   ```

2. **Redis:**
   ```bash
   # Redis memory optimization is configured in the deployment
   # maxmemory: 1gb
   # maxmemory-policy: allkeys-lru
   ```

### Vector Database Optimization

1. **Qdrant Configuration:**
   - Adjust `lists` parameter in HNSW index for your data size
   - Monitor memory usage and adjust accordingly

2. **Weaviate Configuration:**
   - Configure vectorizer modules based on your needs
   - Adjust memory limits for your embedding dimensions

## Security Considerations

### Network Policies

Implement network policies to restrict pod-to-pod communication:

```bash
# Apply network policies (create as needed)
kubectl apply -f deployments/kubernetes/network-policies.yaml
```

### RBAC

Review and adjust RBAC permissions:

```bash
# View current RBAC
kubectl get clusterrole,clusterrolebinding | grep quantum
kubectl get role,rolebinding -n quantum-services
```

### Secrets Management

1. **Use external secret management:**
   - Consider using tools like External Secrets Operator
   - Integrate with cloud provider secret managers

2. **Rotate secrets regularly:**
   ```bash
   # Update database password
   kubectl patch secret quantum-secrets -n quantum-system -p '{"data":{"POSTGRES_PASSWORD":"NEW_BASE64_PASSWORD"}}'
   
   # Restart services to pick up new secrets
   ./scripts/manage-cluster.sh restart
   ```

## Upgrades and Updates

### Application Updates

```bash
# Build new images
./scripts/build-and-push-images.sh

# Rolling update
kubectl set image deployment/qagent qagent=your-registry.com/your-org/qagent:v2.0.0 -n quantum-services

# Monitor rollout
kubectl rollout status deployment/qagent -n quantum-services
```

### Database Upgrades

Database upgrades require careful planning:

1. **Backup before upgrade:**
   ```bash
   ./scripts/manage-cluster.sh backup
   ```

2. **Use blue-green deployment for databases:**
   - Deploy new version alongside old
   - Migrate data
   - Switch traffic
   - Remove old version

## Production Readiness

### High Availability

1. **Multi-node cluster with node affinity rules**
2. **Database replication and failover**
3. **Load balancer configuration**
4. **Cross-region deployment for disaster recovery**

### Monitoring and Alerting

1. **Set up alerting rules in Prometheus**
2. **Configure notification channels (Slack, email, PagerDuty)**
3. **Create custom Grafana dashboards**
4. **Set up log aggregation (ELK stack or similar)**

### Backup Strategy

1. **Automated database backups**
2. **Persistent volume snapshots**
3. **Configuration backup**
4. **Disaster recovery testing**

## Support and Maintenance

### Regular Tasks

1. **Monitor resource usage and adjust limits**
2. **Review and rotate secrets**
3. **Update base images and dependencies**
4. **Test backup and restore procedures**
5. **Review and update security policies**

### Health Checks

Create automated health checks:

```bash
# Create health check script
cat > health-check.sh << 'EOF'
#!/bin/bash
# Quantum Suite Health Check

echo "🔍 Running health checks..."

# Check all deployments are ready
kubectl get deployments -n quantum-services -o json | jq -r '.items[] | "\(.metadata.name): \(.status.readyReplicas)/\(.status.replicas)"'

# Check database connectivity
kubectl exec -n quantum-data statefulset/postgresql -- pg_isready -U quantum

# Check vector databases
kubectl exec -n quantum-data statefulset/qdrant -- curl -s http://localhost:6333/healthz
kubectl exec -n quantum-data statefulset/weaviate -- curl -s http://localhost:8080/v1/.well-known/ready

echo "✅ Health check completed"
EOF

chmod +x health-check.sh
```

### Getting Help

1. **Check deployment logs:** `./scripts/manage-cluster.sh logs`
2. **Review cluster status:** `./scripts/manage-cluster.sh status`  
3. **Check resource usage:** `kubectl top pods --all-namespaces`
4. **Review events:** `kubectl get events --all-namespaces --sort-by='.lastTimestamp'`

For additional support, check the documentation in the `quantum-docs` repository or create an issue in the appropriate GitHub repository.