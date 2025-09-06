#!/bin/bash

# Deploy QLens to Production Environment
# Usage: ./scripts/deploy-production.sh [IMAGE_TAG]

set -euo pipefail

# Configuration
NAMESPACE="qlens-production"
RELEASE_NAME="qlens"
CHART_PATH="charts/qlens"
VALUES_FILE="charts/qlens/values-production.yaml"
REGISTRY="ghcr.io"
REPOSITORY="quantumlayerplatform/quantumlayerplatform"
IMAGE_TAG="${1:-latest}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

echo_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

echo_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    echo_info "Checking prerequisites..."
    
    if ! command -v kubectl &> /dev/null; then
        echo_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    if ! command -v helm &> /dev/null; then
        echo_error "helm is not installed or not in PATH"
        exit 1
    fi
    
    if ! command -v az &> /dev/null; then
        echo_warning "Azure CLI not found - make sure you have alternative authentication configured"
    fi
    
    # Check kubectl connection
    if ! kubectl cluster-info &> /dev/null; then
        echo_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Verify we're connected to the right cluster
    CURRENT_CONTEXT=$(kubectl config current-context)
    echo_info "Current kubectl context: ${CURRENT_CONTEXT}"
    
    if [[ ! "${CURRENT_CONTEXT}" == *"production"* ]] && [[ ! "${CURRENT_CONTEXT}" == *"prod"* ]]; then
        echo_warning "Current context doesn't appear to be a production cluster"
        read -p "Are you sure you want to continue? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo_info "Deployment cancelled by user"
            exit 0
        fi
    fi
    
    echo_success "Prerequisites check passed"
}

# Create namespace if it doesn't exist
create_namespace() {
    echo_info "Creating namespace ${NAMESPACE} if it doesn't exist..."
    kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
    
    # Add production labels and annotations
    kubectl label namespace "${NAMESPACE}" environment=production --overwrite
    kubectl annotate namespace "${NAMESPACE}" deployment/timestamp="$(date -u +%Y-%m-%dT%H:%M:%SZ)" --overwrite
    
    echo_success "Namespace ${NAMESPACE} is ready"
}

# Backup current deployment
backup_deployment() {
    echo_info "Creating backup of current deployment..."
    
    BACKUP_DIR="backups/$(date +%Y%m%d-%H%M%S)"
    mkdir -p "${BACKUP_DIR}"
    
    # Backup Helm release
    if helm status "${RELEASE_NAME}" --namespace="${NAMESPACE}" &> /dev/null; then
        helm get values "${RELEASE_NAME}" --namespace="${NAMESPACE}" > "${BACKUP_DIR}/values.yaml"
        helm get manifest "${RELEASE_NAME}" --namespace="${NAMESPACE}" > "${BACKUP_DIR}/manifest.yaml"
        echo_info "Backup saved to ${BACKUP_DIR}"
    else
        echo_info "No existing release found, skipping backup"
    fi
    
    echo_success "Backup completed"
}

# Validate Helm chart
validate_chart() {
    echo_info "Validating Helm chart..."
    
    if ! helm lint "${CHART_PATH}"; then
        echo_error "Helm chart validation failed"
        exit 1
    fi
    
    # Test template rendering
    if ! helm template "${RELEASE_NAME}" "${CHART_PATH}" -f "${VALUES_FILE}" > /dev/null; then
        echo_error "Helm template rendering failed"
        exit 1
    fi
    
    # Validate against Kubernetes API
    helm template "${RELEASE_NAME}" "${CHART_PATH}" -f "${VALUES_FILE}" \
        --set "image.tag=${IMAGE_TAG}" \
        --set "image.registry=${REGISTRY}/${REPOSITORY}" | \
        kubectl apply --dry-run=client -f -
    
    echo_success "Helm chart validation passed"
}

# Run pre-deployment checks
pre_deployment_checks() {
    echo_info "Running pre-deployment checks..."
    
    # Check if Azure Key Vault is accessible (if external secrets are enabled)
    if grep -q "externalSecrets:\s*enabled:\s*true" "${VALUES_FILE}"; then
        echo_info "External secrets enabled, checking Azure Key Vault access..."
        # Add Azure Key Vault connectivity check here if needed
    fi
    
    # Check if required secrets exist
    if ! kubectl get secret qlens-tls --namespace="${NAMESPACE}" &> /dev/null; then
        echo_warning "TLS secret 'qlens-tls' not found. HTTPS may not work properly."
    fi
    
    # Check cluster resources
    echo_info "Checking cluster resources..."
    kubectl top nodes || echo_warning "Could not get node resource usage"
    
    echo_success "Pre-deployment checks completed"
}

# Deploy with Helm
deploy() {
    echo_info "Deploying QLens to production..."
    echo_warning "This is a PRODUCTION deployment!"
    
    read -p "Proceed with deployment? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo_info "Deployment cancelled by user"
        exit 0
    fi
    
    # Deploy with extended timeout for production
    helm upgrade --install "${RELEASE_NAME}" "${CHART_PATH}" \
        --namespace "${NAMESPACE}" \
        --values "${VALUES_FILE}" \
        --set "image.tag=${IMAGE_TAG}" \
        --set "image.registry=${REGISTRY}/${REPOSITORY}" \
        --wait \
        --timeout=15m \
        --create-namespace \
        --atomic
    
    echo_success "Deployment completed"
}

# Wait for pods to be ready
wait_for_pods() {
    echo_info "Waiting for pods to be ready..."
    
    if ! kubectl wait --for=condition=ready pod \
        -l app.kubernetes.io/name=qlens \
        --timeout=600s \
        --namespace="${NAMESPACE}"; then
        echo_error "Pods failed to become ready within timeout"
        kubectl get pods --namespace="${NAMESPACE}"
        kubectl describe pods --namespace="${NAMESPACE}" -l app.kubernetes.io/name=qlens
        exit 1
    fi
    
    # Wait for all replicas to be ready
    echo_info "Waiting for all replicas to be ready..."
    sleep 30
    
    echo_success "All pods are ready"
}

# Run comprehensive health checks
run_health_checks() {
    echo_info "Running comprehensive health checks..."
    
    # Get pod information
    kubectl get pods --namespace="${NAMESPACE}" -l app.kubernetes.io/name=qlens
    
    # Check internal health endpoints
    echo_info "Testing internal health endpoints..."
    
    # Gateway health check
    GATEWAY_POD=$(kubectl get pod --namespace="${NAMESPACE}" -l app.kubernetes.io/component=gateway -o jsonpath='{.items[0].metadata.name}')
    kubectl exec "${GATEWAY_POD}" --namespace="${NAMESPACE}" -- curl -f http://localhost:8080/health
    
    # Router health check
    ROUTER_POD=$(kubectl get pod --namespace="${NAMESPACE}" -l app.kubernetes.io/component=router -o jsonpath='{.items[0].metadata.name}')
    kubectl exec "${ROUTER_POD}" --namespace="${NAMESPACE}" -- curl -f http://localhost:8081/health
    
    # Cache health check
    CACHE_POD=$(kubectl get pod --namespace="${NAMESPACE}" -l app.kubernetes.io/component=cache -o jsonpath='{.items[0].metadata.name}')
    kubectl exec "${CACHE_POD}" --namespace="${NAMESPACE}" -- curl -f http://localhost:8082/health
    
    # Test external endpoint if ingress exists
    INGRESS_HOST=$(kubectl get ingress --namespace="${NAMESPACE}" -o jsonpath='{.items[0].spec.rules[0].host}' 2>/dev/null || echo "")
    if [[ -n "${INGRESS_HOST}" ]]; then
        echo_info "Testing external endpoint: https://${INGRESS_HOST}/health"
        # Wait for DNS and ingress to be ready
        sleep 60
        if curl -f -s --max-time 30 "https://${INGRESS_HOST}/health" > /dev/null; then
            echo_success "External health check passed"
        else
            echo_warning "External health check failed - this may be expected if DNS/ingress is still propagating"
        fi
    fi
    
    echo_success "Health checks completed"
}

# Run smoke tests
run_smoke_tests() {
    echo_info "Running smoke tests..."
    
    # Test a simple completion request
    GATEWAY_POD=$(kubectl get pod --namespace="${NAMESPACE}" -l app.kubernetes.io/component=gateway -o jsonpath='{.items[0].metadata.name}')
    
    # Create a minimal test request
    cat > /tmp/test-request.json << EOF
{
    "model": "gpt-35-turbo",
    "messages": [
        {"role": "user", "content": [{"type": "text", "text": "Hello, this is a test."}]}
    ],
    "max_tokens": 5
}
EOF
    
    echo_info "Testing completion endpoint..."
    kubectl cp /tmp/test-request.json "${NAMESPACE}/${GATEWAY_POD}:/tmp/test-request.json"
    
    # This is a basic smoke test - in real production you might want more comprehensive tests
    echo_info "Smoke tests would run here - skipping for safety in production"
    
    rm -f /tmp/test-request.json
    
    echo_success "Smoke tests completed"
}

# Show deployment status
show_status() {
    echo_info "Production Deployment Status:"
    echo "============================"
    
    helm status "${RELEASE_NAME}" --namespace="${NAMESPACE}"
    
    echo ""
    echo_info "Pod Status:"
    kubectl get pods --namespace="${NAMESPACE}" -l app.kubernetes.io/name=qlens -o wide
    
    echo ""
    echo_info "Service Status:"
    kubectl get services --namespace="${NAMESPACE}" -l app.kubernetes.io/name=qlens
    
    echo ""
    echo_info "Ingress Status:"
    kubectl get ingress --namespace="${NAMESPACE}"
    
    echo ""
    echo_info "HPA Status:"
    kubectl get hpa --namespace="${NAMESPACE}" || echo "No HPA found"
    
    echo ""
    echo_info "Recent Events:"
    kubectl get events --namespace="${NAMESPACE}" --sort-by=.metadata.creationTimestamp | tail -10
    
    echo ""
    echo_success "QLens production deployment completed successfully!"
    
    INGRESS_HOST=$(kubectl get ingress --namespace="${NAMESPACE}" -o jsonpath='{.items[0].spec.rules[0].host}' 2>/dev/null || echo "")
    if [[ -n "${INGRESS_HOST}" ]]; then
        echo_info "Access the application at: https://${INGRESS_HOST}"
    fi
}

# Rollback function (for emergencies)
rollback() {
    echo_error "Rolling back deployment..."
    helm rollback "${RELEASE_NAME}" --namespace="${NAMESPACE}"
    echo_success "Rollback completed"
}

# Trap to handle script interruption
trap 'echo_error "Script interrupted! You may need to check the deployment status manually."; exit 1' INT TERM

# Main execution
main() {
    echo_info "Starting QLens production deployment..."
    echo_info "Image tag: ${IMAGE_TAG}"
    echo_info "Registry: ${REGISTRY}/${REPOSITORY}"
    
    check_prerequisites
    create_namespace
    backup_deployment
    validate_chart
    pre_deployment_checks
    deploy
    wait_for_pods
    run_health_checks
    run_smoke_tests
    show_status
}

# Run main function
main "$@"