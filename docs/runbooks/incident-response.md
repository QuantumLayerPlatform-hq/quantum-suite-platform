# QLens Incident Response Runbook

This runbook provides procedures for responding to incidents in the QLens LLM Gateway Service.

## Incident Severity Levels

### Severity 1 (P1) - Critical
- Complete service outage
- Data loss or security breach
- Revenue impact > $10k/hour
- **Response Time**: 5 minutes
- **Resolution Target**: 1 hour

### Severity 2 (P2) - High  
- Significant degraded performance
- One provider completely down
- Cost controls not functioning
- **Response Time**: 15 minutes
- **Resolution Target**: 4 hours

### Severity 3 (P3) - Medium
- Minor performance degradation
- Non-critical feature unavailable
- Alert threshold breached
- **Response Time**: 1 hour
- **Resolution Target**: 24 hours

### Severity 4 (P4) - Low
- Documentation issues
- Minor cosmetic problems
- Enhancement requests
- **Response Time**: Next business day
- **Resolution Target**: 1 week

## Alert Response Procedures

### QLensServiceDown
**Alert**: `up{job=~"qlens-.*"} == 0`

**Immediate Actions**:
1. Check service status
   ```bash
   kubectl get pods -n qlens-production -l app.kubernetes.io/name=qlens
   kubectl get svc -n qlens-production
   ```

2. Identify failed component
   ```bash
   kubectl describe pod <failed-pod> -n qlens-production
   kubectl logs <failed-pod> -n qlens-production --tail=100
   ```

3. Restart service if needed
   ```bash
   kubectl rollout restart deployment qlens-gateway -n qlens-production
   ```

**Escalation**: If restart doesn't work within 5 minutes, page Platform Team.

### QLensHighErrorRate
**Alert**: `(sum(rate(qlens_requests_total{status=~"5.*"}[5m])) / sum(rate(qlens_requests_total[5m]))) > 0.05`

**Investigation Steps**:
1. Check error patterns
   ```bash
   # View recent error logs
   kubectl logs -l app.kubernetes.io/name=qlens -n qlens-production | grep "ERROR"
   
   # Check error rate by provider
   curl -G 'http://prometheus:9090/api/v1/query' \
     --data-urlencode 'query=rate(qlens_provider_errors_total[5m]) by (provider)'
   ```

2. Identify root cause
   - Provider API issues
   - Authentication failures
   - Rate limiting
   - Resource constraints

3. Take corrective action
   - Disable problematic provider
   - Increase resource limits
   - Reset connections

### ProviderUnhealthy
**Alert**: `qlens_provider_health_status == 0`

**Response Actions**:
1. Check provider status
   ```bash
   # View provider health metrics
   kubectl exec deployment/qlens-router -n qlens-production -- \
     curl -s localhost:8081/internal/v1/models
   ```

2. Test provider directly
   ```bash
   # Test Azure OpenAI
   curl -X POST "https://your-resource.openai.azure.com/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-15-preview" \
     -H "api-key: $AZURE_OPENAI_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{"messages":[{"role":"user","content":"test"}],"max_tokens":5}'
   ```

3. Verify credentials and configuration
   ```bash
   kubectl get secret qlens-secrets -n qlens-production -o yaml
   ```

### HighDailyCost
**Alert**: `sum(increase(qlens_cost_usd_total[1d])) > 500`

**Immediate Actions**:
1. Check current usage
   ```bash
   # View cost breakdown
   curl -G 'http://prometheus:9090/api/v1/query' \
     --data-urlencode 'query=sum(rate(qlens_cost_usd_total[1h])) by (tenant_id, provider)'
   ```

2. Identify high-usage tenants
   ```bash
   # Check top consumers
   curl -G 'http://prometheus:9090/api/v1/query' \
     --data-urlencode 'query=topk(10, sum(increase(qlens_cost_usd_total[1h])) by (tenant_id))'
   ```

3. Apply emergency limits if needed
   ```bash
   # Reduce cost limits temporarily
   helm upgrade qlens charts/qlens \
     --namespace qlens-production \
     --set costControls.dailyLimits.total=300 \
     --reuse-values
   ```

### QLensHighLatency
**Alert**: `histogram_quantile(0.95, sum(rate(qlens_request_duration_seconds_bucket[5m])) by (le)) > 30`

**Investigation Steps**:
1. Identify latency source
   ```bash
   # Check provider latency
   curl -G 'http://prometheus:9090/api/v1/query' \
     --data-urlencode 'query=avg(rate(qlens_provider_latency_seconds_sum[5m]) / rate(qlens_provider_latency_seconds_count[5m])) by (provider)'
   ```

2. Check resource utilization
   ```bash
   kubectl top pods -n qlens-production
   kubectl top nodes
   ```

3. Scale up if needed
   ```bash
   kubectl scale deployment qlens-gateway --replicas=5 -n qlens-production
   ```

## Common Troubleshooting Procedures

### Service Discovery Issues

**Symptoms**: Services can't communicate with each other

**Investigation**:
```bash
# Check service endpoints
kubectl get endpoints -n qlens-production

# Test internal connectivity
kubectl exec deployment/qlens-gateway -n qlens-production -- \
  nslookup qlens-router.qlens-production.svc.cluster.local

# Test HTTP connectivity
kubectl exec deployment/qlens-gateway -n qlens-production -- \
  curl -f http://qlens-router:8081/health
```

**Resolution**:
- Verify service selectors match pod labels
- Check NetworkPolicy rules
- Restart CoreDNS if DNS resolution fails

### Authentication Failures

**Symptoms**: 401/403 errors from providers

**Investigation**:
```bash
# Check secret values (be careful with sensitive data)
kubectl get secret qlens-secrets -n qlens-production -o jsonpath='{.data}' | base64 -d

# Verify environment variables
kubectl exec deployment/qlens-router -n qlens-production -- env | grep -i azure
kubectl exec deployment/qlens-router -n qlens-production -- env | grep -i aws
```

**Resolution**:
- Rotate API keys if compromised
- Update secret values
- Restart affected pods

### Cache Performance Issues

**Symptoms**: High latency, low hit rates

**Investigation**:
```bash
# Check cache metrics
curl -G 'http://prometheus:9090/api/v1/query' \
  --data-urlencode 'query=rate(qlens_cache_hits_total[5m]) / (rate(qlens_cache_hits_total[5m]) + rate(qlens_cache_misses_total[5m]))'

# Check cache size and evictions
kubectl exec deployment/qlens-cache -n qlens-production -- \
  curl localhost:8082/internal/v1/cache/stats
```

**Resolution**:
- Increase cache size limits
- Adjust TTL values
- Scale cache service
- Check Redis connectivity (production)

### Resource Exhaustion

**Symptoms**: OOMKilled pods, throttled CPU

**Investigation**:
```bash
# Check resource usage
kubectl describe nodes
kubectl top pods -n qlens-production --sort-by=cpu
kubectl top pods -n qlens-production --sort-by=memory

# Check resource limits
kubectl describe deployment qlens-gateway -n qlens-production
```

**Resolution**:
```bash
# Increase resource limits
helm upgrade qlens charts/qlens \
  --namespace qlens-production \
  --set resources.gateway.limits.memory=4Gi \
  --set resources.gateway.limits.cpu=4000m \
  --reuse-values

# Scale horizontally
kubectl scale deployment qlens-gateway --replicas=6 -n qlens-production
```

## Escalation Procedures

### Level 1: On-call Engineer
- Initial triage and basic troubleshooting
- Follow runbook procedures
- Gather initial data and logs

### Level 2: Platform Team
- Complex debugging and analysis
- Infrastructure changes
- Provider relationship management

### Level 3: Engineering Leadership
- Major architectural decisions
- Vendor escalation
- Business impact decisions

### Level 4: Executive Team
- Public communications
- Legal/compliance issues
- Major business decisions

## Communication Templates

### Initial Incident Report
```
Subject: [P1] QLens Service Outage - <Brief Description>

WHAT: Brief description of the issue
WHEN: Start time (UTC)
WHERE: Affected services/regions
WHY: Root cause (if known, otherwise "Under investigation")
IMPACT: User impact and business metrics
STATUS: Current status and ETA for resolution
NEXT UPDATE: When next update will be provided
```

### Status Update
```
Subject: [UPDATE] [P1] QLens Service Outage - <Brief Description>

UPDATE: Current status of the investigation/resolution
PROGRESS: What has been done since last update  
NEXT STEPS: What will be done next
ETA: Estimated time for resolution
WORKAROUND: Any available workarounds
NEXT UPDATE: When next update will be provided
```

### Resolution Notice
```
Subject: [RESOLVED] [P1] QLens Service Outage - <Brief Description>

RESOLUTION: Brief description of how issue was resolved
ROOT CAUSE: Final root cause analysis
IMPACT: Final impact assessment
PREVENTION: Steps being taken to prevent recurrence  
POST-MORTEM: Link to detailed post-mortem (if applicable)
```

## Post-Incident Procedures

### Immediate (Within 4 hours)
1. **Service Restoration Verification**
   - Confirm all metrics back to normal
   - Verify no residual issues
   - Update monitoring if needed

2. **Initial Timeline Creation**
   - Document key events and timestamps
   - Identify decision points
   - Note what worked and what didn't

### Short-term (Within 24 hours)
1. **Stakeholder Communication**
   - Send resolution notice
   - Brief affected customers
   - Update status page

2. **Data Collection**
   - Export relevant logs and metrics
   - Document configuration changes
   - Collect feedback from responders

### Long-term (Within 1 week)
1. **Post-Mortem Review**
   - Conduct blameless post-mortem
   - Identify improvement opportunities
   - Create action items with owners

2. **Process Improvement**
   - Update runbooks based on learnings
   - Improve monitoring and alerting
   - Implement preventive measures

## Emergency Contacts

### Primary Contacts
- **On-call Engineer**: See PagerDuty
- **Platform Team Lead**: +1-555-PLATFORM
- **Engineering Manager**: +1-555-ENGINEER

### Vendor Contacts
- **Azure Support**: Case portal + phone
- **AWS Support**: Case portal + phone
- **GitHub Support**: Support tickets

### Internal Escalation
- **Incident Commander**: See incident response team
- **Communications Lead**: communications@quantumlayer.ai
- **Executive On-call**: See leadership rotation

## Tools and Resources

### Monitoring and Alerting
- **Prometheus**: http://prometheus.quantumlayer.ai
- **Grafana**: http://grafana.quantumlayer.ai/d/qlens-dashboard
- **PagerDuty**: https://quantumlayer.pagerduty.com

### Documentation
- **Runbooks**: https://docs.quantumlayer.ai/qlens/runbooks
- **Architecture**: https://docs.quantumlayer.ai/qlens/architecture
- **API Docs**: https://docs.quantumlayer.ai/qlens/api

### Communication Channels
- **Slack**: #qlens-incidents
- **Teams**: QLens Platform Team
- **Status Page**: https://status.quantumlayer.ai

Remember: Stay calm, follow the procedures, communicate clearly, and don't hesitate to escalate when needed.