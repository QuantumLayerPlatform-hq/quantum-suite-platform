# QLens Local Unified Access

This guide explains how to access QLens services running on local Kubernetes without port-forwarding, using Istio Gateway and MetalLB for a production-like experience.

## Architecture Overview

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Internet  │───▶│ MetalLB LB  │───▶│Istio Gateway│
│             │    │192.168.1.x  │    │             │
└─────────────┘    └─────────────┘    └─────────────┘
                                            │
                    ┌───────────────────────┼───────────────────────┐
                    │                       │                       │
              ┌─────────────┐        ┌─────────────┐        ┌─────────────┐
              │ QLens API   │        │ Swagger UI  │        │Observability│
              │             │        │             │        │             │
              │ • Gateway   │        │ • OpenAPI   │        │ • Grafana   │
              │ • Router    │        │ • Docs      │        │ • Kiali     │
              │ • Cache     │        │ • Testing   │        │ • Jaeger    │
              └─────────────┘        └─────────────┘        └─────────────┘
```

## Quick Start

### 1. One-Command Setup
```bash
make setup-local-access
```

This command will:
- Install MetalLB for LoadBalancer support
- Install Istio service mesh
- Deploy observability tools (Kiali, Grafana, Jaeger, Prometheus)
- Deploy QLens services with Istio integration
- Configure unified gateway access

### 2. Get Access Information
```bash
make get-access-info
```

## Manual Setup (Step by Step)

If you prefer to set up components individually:

### 1. Install MetalLB
```bash
make install-metallb
```

### 2. Install Istio
```bash
make install-istio
```

### 3. Setup Observability Tools
```bash
make setup-observability
```

### 4. Deploy QLens
```bash
make deploy-staging
```

## Access Patterns

After setup, you'll have access to services via:

### Using nip.io (Automatic DNS)
```bash
# Get the LoadBalancer IP
GATEWAY_IP=$(kubectl get svc istio-ingressgateway -n istio-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Access services
curl http://qlens.$GATEWAY_IP.nip.io/health
open http://swagger.$GATEWAY_IP.nip.io
open http://grafana.$GATEWAY_IP.nip.io
open http://kiali.$GATEWAY_IP.nip.io
```

### Using /etc/hosts (Local DNS)
Add to `/etc/hosts`:
```
192.168.1.240 qlens.local swagger.local grafana.local kiali.local jaeger.local
```

Then access:
```bash
curl http://qlens.local/health
open http://swagger.local
open http://grafana.local
open http://kiali.local
```

## Service URLs

Once setup is complete, you can access:

| Service | URL | Description |
|---------|-----|-------------|
| **QLens API** | `http://qlens.local` or `http://qlens.<IP>.nip.io` | Main LLM Gateway API |
| **Swagger UI** | `http://swagger.local` or `http://swagger.<IP>.nip.io` | Interactive API documentation |
| **Grafana** | `http://grafana.local` or `http://grafana.<IP>.nip.io` | Metrics dashboards |
| **Kiali** | `http://kiali.local` or `http://kiali.<IP>.nip.io` | Service mesh console |
| **Jaeger** | `http://jaeger.local` or `http://jaeger.<IP>.nip.io` | Distributed tracing |

## API Examples

### Health Check
```bash
curl http://qlens.local/health
```

### List Available Models
```bash
curl -H "Authorization: Bearer your-token" \
     -H "X-Tenant-ID: test-tenant" \
     http://qlens.local/v1/models
```

### Create Chat Completion
```bash
curl -X POST http://qlens.local/v1/chat/completions \
  -H "Authorization: Bearer your-token" \
  -H "X-Tenant-ID: test-tenant" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

## Configuration

### MetalLB IP Pool
The MetalLB configuration uses IP range `192.168.1.240-192.168.1.250`. To change this:

1. Edit `deployments/metallb/ipaddresspool.yaml`
2. Update the addresses range to match your network
3. Reapply: `kubectl apply -k deployments/metallb`

### Istio Gateway
The gateway configuration is in `deployments/istio/local/gateway.yaml`. It includes:
- HTTP/HTTPS listeners
- Virtual services for routing
- Automatic redirects
- Security headers

## Troubleshooting

### Check LoadBalancer Status
```bash
kubectl get svc istio-ingressgateway -n istio-system
```

### Check Gateway Configuration
```bash
kubectl get gateway,virtualservice -n istio-system
```

### Check Pod Status
```bash
kubectl get pods -n qlens-staging
kubectl get pods -n istio-system
```

### View Service Mesh Status
```bash
istioctl proxy-status
istioctl analyze
```

### Check Logs
```bash
# QLens Gateway logs
kubectl logs -f deployment/qlens-gateway -n qlens-staging

# Istio ingress gateway logs
kubectl logs -f deployment/istio-ingressgateway -n istio-system
```

### DNS Resolution Issues
If using .local domains doesn't work:

1. Use nip.io instead: `http://qlens.<IP>.nip.io`
2. Check if systemd-resolved is blocking .local domains
3. Use IP directly: `http://<GATEWAY-IP>`

## Benefits of This Setup

### 🚀 **No Port Forwarding Required**
- Direct access via LoadBalancer IP
- Production-like access patterns
- No need for multiple kubectl port-forward commands

### 🔍 **Unified Observability**
- All monitoring tools accessible via web UI
- Service mesh visualization with Kiali
- Distributed tracing with Jaeger
- Metrics dashboards with Grafana

### 🛡️ **Security Features**
- mTLS between services (Istio)
- Rate limiting and circuit breakers
- Request/response transformation
- Security headers automatically added

### 🎯 **Developer Experience**
- Easy-to-remember URLs
- Automatic service discovery
- Load balancing across replicas
- Consistent with production setup

## Cleanup

To clean up the installation:

```bash
# Remove QLens services
make k8s-delete-staging

# Remove Istio (optional)
istioctl uninstall --purge -y
kubectl delete namespace istio-system

# Remove MetalLB (optional)
kubectl delete -k deployments/metallb
```

## Next Steps

1. **Configure SSL/TLS**: Add certificates for HTTPS access
2. **Set up Authentication**: Configure JWT validation
3. **Add Rate Limiting**: Implement per-tenant rate limits
4. **Monitor Performance**: Set up alerts and dashboards
5. **Scale Services**: Configure HPA for auto-scaling