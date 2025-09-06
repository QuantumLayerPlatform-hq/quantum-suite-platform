#!/bin/bash

# Deploy QLens to Staging Environment
# Usage: ./scripts/deploy-staging.sh [IMAGE_TAG]

set -euo pipefail

# Configuration
NAMESPACE="qlens-staging"
RELEASE_NAME="qlens"
CHART_PATH="charts/qlens"
VALUES_FILE="charts/qlens/values-staging.yaml"
REGISTRY="ghcr.io"
REPOSITORY="quantumlayerplatform/quantumlayerplatform"
IMAGE_TAG="${1:-main}"

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
    
    # Check kubectl connection
    if ! kubectl cluster-info &> /dev/null; then
        echo_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    echo_success "Prerequisites check passed"
}

# Create namespace if it doesn't exist
create_namespace() {
    echo_info "Creating namespace ${NAMESPACE} if it doesn't exist..."
    kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
    echo_success "Namespace ${NAMESPACE} is ready"
}

# Create secrets
create_secrets() {
    echo_info "Creating secrets..."
    
    # Check if secrets exist in environment variables
    if [[ -z "${AZURE_OPENAI_ENDPOINT:-}" ]] || [[ -z "${AZURE_OPENAI_API_KEY:-}" ]]; then
        echo_error "Azure OpenAI credentials not found in environment variables"
        echo "Please set AZURE_OPENAI_ENDPOINT and AZURE_OPENAI_API_KEY"
        exit 1
    fi
    
    if [[ -z "${AWS_REGION:-}" ]] || [[ -z "${AWS_ACCESS_KEY_ID:-}" ]] || [[ -z "${AWS_SECRET_ACCESS_KEY:-}" ]]; then
        echo_error "AWS Bedrock credentials not found in environment variables"
        echo "Please set AWS_REGION, AWS_ACCESS_KEY_ID, and AWS_SECRET_ACCESS_KEY"
        exit 1
    fi
    
    # Create Azure OpenAI secrets
    kubectl create secret generic qlens-secrets \
        --from-literal=azure-openai-endpoint="${AZURE_OPENAI_ENDPOINT}" \
        --from-literal=azure-openai-api-key="${AZURE_OPENAI_API_KEY}" \
        --namespace="${NAMESPACE}" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Create AWS Bedrock secrets
    kubectl create secret generic aws-bedrock-secrets \
        --from-literal=aws-region="${AWS_REGION}" \
        --from-literal=aws-access-key-id="${AWS_ACCESS_KEY_ID}" \
        --from-literal=aws-secret-access-key="${AWS_SECRET_ACCESS_KEY}" \
        --namespace="${NAMESPACE}" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    echo_success "Secrets created successfully"
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
    
    echo_success "Helm chart validation passed"
}

# Deploy with Helm
deploy() {
    echo_info "Deploying QLens to staging..."
    
    helm upgrade --install "${RELEASE_NAME}" "${CHART_PATH}" \
        --namespace "${NAMESPACE}" \
        --values "${VALUES_FILE}" \
        --set "image.tag=${IMAGE_TAG}" \
        --set "image.registry=${REGISTRY}/${REPOSITORY}" \
        --wait \
        --timeout=10m \
        --create-namespace
    
    echo_success "Deployment completed"
}

# Wait for pods to be ready
wait_for_pods() {
    echo_info "Waiting for pods to be ready..."
    
    if ! kubectl wait --for=condition=ready pod \
        -l app.kubernetes.io/name=qlens \
        --timeout=300s \
        --namespace="${NAMESPACE}"; then
        echo_error "Pods failed to become ready within timeout"
        kubectl get pods --namespace="${NAMESPACE}"
        exit 1
    fi
    
    echo_success "All pods are ready"
}

# Run health checks
run_health_checks() {
    echo_info "Running health checks..."
    
    # Get pod information
    kubectl get pods --namespace="${NAMESPACE}" -l app.kubernetes.io/name=qlens
    
    # Check gateway service
    echo_info "Testing gateway health endpoint..."
    kubectl port-forward "svc/${RELEASE_NAME}-gateway" 8080:8080 --namespace="${NAMESPACE}" &
    PORT_FORWARD_PID=$!
    
    # Wait for port forward to be ready
    sleep 5
    
    # Test health endpoint
    if curl -f -s http://localhost:8080/health > /dev/null; then
        echo_success "Gateway health check passed"
    else
        echo_error "Gateway health check failed"
        kill $PORT_FORWARD_PID 2>/dev/null || true
        exit 1
    fi
    
    # Clean up port forward
    kill $PORT_FORWARD_PID 2>/dev/null || true
    
    echo_success "All health checks passed"
}

# Show deployment status
show_status() {
    echo_info "Deployment Status:"
    echo "==================="
    
    helm status "${RELEASE_NAME}" --namespace="${NAMESPACE}"
    
    echo ""
    echo_info "Pod Status:"
    kubectl get pods --namespace="${NAMESPACE}" -l app.kubernetes.io/name=qlens
    
    echo ""
    echo_info "Service Status:"
    kubectl get services --namespace="${NAMESPACE}" -l app.kubernetes.io/name=qlens
    
    echo ""
    echo_info "Ingress Status:"
    kubectl get ingress --namespace="${NAMESPACE}" 2>/dev/null || echo "No ingress found"
    
    echo ""
    echo_success "QLens staging deployment completed successfully!"
    echo_info "Access the application at: http://qlens-staging.local (if ingress is configured)"
}

# Main execution
main() {
    echo_info "Starting QLens staging deployment..."
    echo_info "Image tag: ${IMAGE_TAG}"
    echo_info "Registry: ${REGISTRY}/${REPOSITORY}"
    
    check_prerequisites
    create_namespace
    create_secrets
    validate_chart
    deploy
    wait_for_pods
    run_health_checks
    show_status
}

# Run main function
main "$@"