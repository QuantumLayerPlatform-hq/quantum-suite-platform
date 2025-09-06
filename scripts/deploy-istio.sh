#!/bin/bash

# QLens Istio Deployment Script
set -e

ENVIRONMENT=${1:-"development"}
NAMESPACE=${2:-"quantum-system"}
ISTIO_VERSION=${3:-"1.20.0"}

echo "🚀 Deploying QLens with Istio Service Mesh"
echo "Environment: $ENVIRONMENT"
echo "Namespace: $NAMESPACE"
echo "Istio Version: $ISTIO_VERSION"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl not found. Please install kubectl."
        exit 1
    fi
    
    # Check if kustomize is installed
    if ! command -v kustomize &> /dev/null; then
        print_error "kustomize not found. Please install kustomize."
        exit 1
    fi
    
    # Check if istioctl is installed
    if ! command -v istioctl &> /dev/null; then
        print_warning "istioctl not found. Installing Istio $ISTIO_VERSION..."
        install_istio
    fi
    
    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        print_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Install Istio if not present
install_istio() {
    print_status "Installing Istio $ISTIO_VERSION..."
    
    # Download istioctl
    curl -L https://istio.io/downloadIstio | ISTIO_VERSION=$ISTIO_VERSION TARGET_ARCH=x86_64 sh -
    sudo mv istio-$ISTIO_VERSION/bin/istioctl /usr/local/bin/
    rm -rf istio-$ISTIO_VERSION
    
    print_success "Istio installed successfully"
}

# Install or upgrade Istio control plane
setup_istio_control_plane() {
    print_status "Setting up Istio control plane..."
    
    # Check if Istio is already installed
    if kubectl get namespace istio-system &> /dev/null; then
        print_warning "Istio control plane already exists. Checking for upgrade..."
        istioctl upgrade --set values.pilot.env.EXTERNAL_ISTIOD=false
    else
        print_status "Installing Istio control plane..."
        istioctl install --set values.pilot.env.EXTERNAL_ISTIOD=false -y
    fi
    
    # Enable Istio injection for the namespace based on environment
    if [[ "$ENVIRONMENT" != "development" ]]; then
        print_status "Enabling Istio injection for namespace $NAMESPACE..."
        kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
    else
        print_status "Disabling Istio injection for development environment..."
        kubectl label namespace $NAMESPACE istio-injection=disabled --overwrite
    fi
    
    print_success "Istio control plane setup complete"
}

# Deploy QLens services
deploy_qlens_services() {
    print_status "Deploying QLens services..."
    
    # Deploy base Kubernetes resources
    print_status "Applying base Kustomize configuration..."
    kustomize build deployments/kustomize/overlays/$ENVIRONMENT | kubectl apply -f -
    
    # Deploy Istio configurations for non-development environments
    if [[ "$ENVIRONMENT" != "development" ]]; then
        print_status "Applying Istio service mesh configurations..."
        
        # Apply base Istio configurations
        kustomize build deployments/istio/base | kubectl apply -f -
        
        # Apply environment-specific Istio configurations
        if [[ -d "deployments/istio/overlays/$ENVIRONMENT" ]]; then
            print_status "Applying $ENVIRONMENT Istio overlay..."
            kustomize build deployments/istio/overlays/$ENVIRONMENT | kubectl apply -f -
        fi
    else
        print_warning "Skipping Istio configurations for development environment"
    fi
    
    print_success "QLens services deployed successfully"
}

# Verify deployment
verify_deployment() {
    print_status "Verifying deployment..."
    
    # Wait for pods to be ready
    print_status "Waiting for pods to be ready..."
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/part-of=quantum-suite -n $NAMESPACE --timeout=300s
    
    # Check service status
    print_status "Checking service status..."
    kubectl get pods,svc -n $NAMESPACE
    
    # Verify Istio injection (if enabled)
    if [[ "$ENVIRONMENT" != "development" ]]; then
        print_status "Verifying Istio sidecar injection..."
        PODS_WITH_ISTIO=$(kubectl get pods -n $NAMESPACE -o jsonpath='{.items[*].spec.containers[*].name}' | grep -o istio-proxy | wc -w)
        TOTAL_PODS=$(kubectl get pods -n $NAMESPACE --no-headers | wc -l)
        
        if [[ $PODS_WITH_ISTIO -eq $TOTAL_PODS ]]; then
            print_success "All pods have Istio sidecars injected"
        else
            print_warning "$PODS_WITH_ISTIO out of $TOTAL_PODS pods have Istio sidecars"
        fi
    fi
    
    print_success "Deployment verification complete"
}

# Test endpoints
test_endpoints() {
    print_status "Testing service endpoints..."
    
    # Port-forward to test locally
    print_status "Setting up port forwarding..."
    kubectl port-forward -n $NAMESPACE svc/qlens-gateway 8105:8105 &
    PORT_FORWARD_PID=$!
    
    # Wait a moment for port-forward to establish
    sleep 5
    
    # Test health endpoint
    if curl -s http://localhost:8105/health > /dev/null; then
        print_success "Gateway health endpoint is responding"
    else
        print_error "Gateway health endpoint is not responding"
    fi
    
    # Clean up port-forward
    kill $PORT_FORWARD_PID 2>/dev/null || true
    
    print_success "Endpoint testing complete"
}

# Generate access information
generate_access_info() {
    print_status "Generating access information..."
    
    echo ""
    echo "🎉 QLens Deployment Complete!"
    echo ""
    echo "📋 Deployment Summary:"
    echo "  • Environment: $ENVIRONMENT"
    echo "  • Namespace: $NAMESPACE"
    echo "  • Istio Enabled: $(if [[ "$ENVIRONMENT" != "development" ]]; then echo "Yes"; else echo "No"; fi)"
    echo ""
    
    if [[ "$ENVIRONMENT" != "development" ]]; then
        # Get Istio ingress gateway info
        INGRESS_HOST=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
        INGRESS_PORT=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="http2")].port}')
        SECURE_INGRESS_PORT=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="https")].port}')
        
        echo "🌐 External Access:"
        if [[ "$INGRESS_HOST" != "" ]]; then
            echo "  • HTTP:  http://$INGRESS_HOST:$INGRESS_PORT"
            echo "  • HTTPS: https://$INGRESS_HOST:$SECURE_INGRESS_PORT"
        else
            echo "  • LoadBalancer IP pending... Check with: kubectl get svc istio-ingressgateway -n istio-system"
        fi
        echo ""
        
        echo "🔍 Istio Dashboard:"
        echo "  • Kiali: istioctl dashboard kiali"
        echo "  • Jaeger: istioctl dashboard jaeger"
        echo "  • Grafana: istioctl dashboard grafana"
        echo ""
    fi
    
    echo "🔧 Local Testing:"
    echo "  kubectl port-forward -n $NAMESPACE svc/qlens-gateway 8105:8105"
    echo "  curl http://localhost:8105/health"
    echo ""
    
    echo "📊 Monitoring:"
    echo "  kubectl get pods -n $NAMESPACE"
    echo "  kubectl logs -f deployment/qlens-gateway -n $NAMESPACE"
    echo ""
    
    if [[ "$ENVIRONMENT" != "development" ]]; then
        echo "🛡️  Security Features Enabled:"
        echo "  • mTLS: Strict mode"
        echo "  • Authorization Policies: Enabled"
        echo "  • Rate Limiting: Enabled"
        echo "  • WAF Protection: Enabled"
        echo ""
    fi
}

# Main execution
main() {
    echo "🏁 Starting QLens Istio deployment..."
    
    check_prerequisites
    
    if [[ "$ENVIRONMENT" != "development" ]]; then
        setup_istio_control_plane
    fi
    
    deploy_qlens_services
    verify_deployment
    test_endpoints
    generate_access_info
    
    print_success "QLens deployment completed successfully! 🎉"
}

# Handle script arguments
case $ENVIRONMENT in
    development|dev)
        ENVIRONMENT="development"
        ;;
    staging|stage)
        ENVIRONMENT="staging"
        ;;
    production|prod)
        ENVIRONMENT="production"
        ;;
    *)
        print_error "Invalid environment: $ENVIRONMENT"
        echo "Usage: $0 [development|staging|production] [namespace] [istio-version]"
        echo ""
        echo "Examples:"
        echo "  $0 development"
        echo "  $0 staging quantum-system"
        echo "  $0 production quantum-system 1.20.0"
        exit 1
        ;;
esac

# Run main function
main