#!/bin/bash

# QLens Local Access Setup Script
# Sets up unified access to QLens services using Istio and MetalLB
set -e

# Configuration
NAMESPACE="${1:-qlens-staging}"
METALLB_VERSION="${2:-v0.14.8}"
ISTIO_VERSION="${3:-1.24.0}"
GATEWAY_IP=""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
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

print_header() {
    echo -e "${PURPLE}🚀 $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl not found. Please install kubectl."
        exit 1
    fi
    
    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        print_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi
    
    # Check if kustomize is available
    if ! command -v kustomize &> /dev/null; then
        print_warning "kustomize not found. Using kubectl apply -k instead."
    fi
    
    print_success "Prerequisites check passed"
}

# Install MetalLB for LoadBalancer support
install_metallb() {
    print_header "Installing MetalLB"
    
    # Check if MetalLB is already installed
    if kubectl get namespace metallb-system &> /dev/null; then
        print_warning "MetalLB namespace already exists. Checking installation..."
        if kubectl get deployment controller -n metallb-system &> /dev/null; then
            print_success "MetalLB is already installed"
            return 0
        fi
    fi
    
    print_status "Deploying MetalLB..."
    
    # Apply MetalLB manifests
    if command -v kustomize &> /dev/null; then
        kustomize build deployments/metallb | kubectl apply -f -
    else
        kubectl apply -k deployments/metallb
    fi
    
    # Wait for MetalLB to be ready
    print_status "Waiting for MetalLB to be ready..."
    kubectl wait --namespace metallb-system \
        --for=condition=ready pod \
        --selector=app=metallb \
        --timeout=300s
    
    print_success "MetalLB installed and ready"
}

# Install Istio
install_istio() {
    print_header "Installing Istio"
    
    # Check if istioctl is installed
    if ! command -v istioctl &> /dev/null; then
        print_status "Installing istioctl..."
        curl -L https://istio.io/downloadIstio | ISTIO_VERSION=$ISTIO_VERSION TARGET_ARCH=x86_64 sh -
        sudo mv istio-$ISTIO_VERSION/bin/istioctl /usr/local/bin/
        sudo chmod +x /usr/local/bin/istioctl
        rm -rf istio-$ISTIO_VERSION
        print_success "istioctl installed"
    fi
    
    # Check if Istio is already installed
    if kubectl get namespace istio-system &> /dev/null; then
        print_warning "Istio system namespace already exists"
        if kubectl get deployment istiod -n istio-system &> /dev/null; then
            print_success "Istio control plane already installed"
            return 0
        fi
    fi
    
    # Install Istio
    print_status "Installing Istio control plane..."
    istioctl install --set values.pilot.env.EXTERNAL_ISTIOD=false \
                     --set values.global.proxy.resources.requests.cpu=10m \
                     --set values.global.proxy.resources.requests.memory=40Mi \
                     --set values.global.proxy.resources.limits.cpu=100m \
                     --set values.global.proxy.resources.limits.memory=128Mi \
                     --set values.pilot.resources.requests.cpu=100m \
                     --set values.pilot.resources.requests.memory=128Mi \
                     --set values.gateways.istio-ingressgateway.type=LoadBalancer \
                     -y
    
    # Wait for Istio to be ready
    print_status "Waiting for Istio control plane to be ready..."
    kubectl wait --namespace istio-system \
        --for=condition=ready pod \
        --selector=app=istiod \
        --timeout=300s
    
    print_success "Istio control plane installed and ready"
}

# Install observability tools
install_observability() {
    print_header "Installing Observability Tools"
    
    # Install Kiali
    print_status "Installing Kiali..."
    kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.24/samples/addons/kiali.yaml
    
    # Install Grafana
    print_status "Installing Grafana..."
    kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.24/samples/addons/grafana.yaml
    
    # Install Prometheus
    print_status "Installing Prometheus..."
    kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.24/samples/addons/prometheus.yaml
    
    # Apply our custom Jaeger
    print_status "Installing Jaeger..."
    if command -v kustomize &> /dev/null; then
        kustomize build deployments/istio/local | kubectl apply -f -
    else
        kubectl apply -k deployments/istio/local
    fi
    
    # Wait for services to be ready
    print_status "Waiting for observability tools to be ready..."
    kubectl rollout status deployment/kiali -n istio-system --timeout=300s
    kubectl rollout status deployment/grafana -n istio-system --timeout=300s
    kubectl rollout status deployment/prometheus -n istio-system --timeout=300s
    kubectl rollout status deployment/jaeger -n istio-system --timeout=300s
    
    print_success "Observability tools installed and ready"
}

# Deploy QLens services
deploy_qlens() {
    print_header "Deploying QLens Services"
    
    # Enable Istio injection for the namespace
    print_status "Enabling Istio injection for namespace $NAMESPACE..."
    kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
    
    # Deploy QLens using Helm
    print_status "Deploying QLens services with Helm..."
    helm upgrade --install qlens charts/qlens \
        --namespace $NAMESPACE \
        --values charts/qlens/values-staging.yaml \
        --wait --timeout=600s
    
    print_success "QLens services deployed"
}

# Get and display access information
setup_access() {
    print_header "Setting Up Access Information"
    
    # Get the LoadBalancer IP
    print_status "Getting Istio Gateway LoadBalancer IP..."
    
    # Wait for LoadBalancer IP to be assigned
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        GATEWAY_IP=$(kubectl get service istio-ingressgateway -n istio-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
        
        if [ -n "$GATEWAY_IP" ] && [ "$GATEWAY_IP" != "null" ]; then
            break
        fi
        
        print_status "Waiting for LoadBalancer IP assignment... (attempt $attempt/$max_attempts)"
        sleep 10
        ((attempt++))
    done
    
    if [ -z "$GATEWAY_IP" ] || [ "$GATEWAY_IP" == "null" ]; then
        print_warning "LoadBalancer IP not assigned yet. You can check later with:"
        echo "kubectl get service istio-ingressgateway -n istio-system"
        GATEWAY_IP="<PENDING>"
    else
        print_success "LoadBalancer IP assigned: $GATEWAY_IP"
    fi
    
    # Setup /etc/hosts entries (optional)
    setup_local_dns
}

# Setup local DNS entries
setup_local_dns() {
    print_status "Setting up local DNS..."
    
    if [ "$GATEWAY_IP" != "<PENDING>" ]; then
        # Create hosts file entries
        cat << EOF > /tmp/qlens-hosts
# QLens Local Access - Added by setup-local-access.sh
$GATEWAY_IP qlens.local
$GATEWAY_IP swagger.local
$GATEWAY_IP grafana.local
$GATEWAY_IP kiali.local
$GATEWAY_IP jaeger.local
EOF
        
        print_warning "To set up local DNS, add these entries to your /etc/hosts file:"
        cat /tmp/qlens-hosts
        echo ""
        print_status "Or run: sudo cat /tmp/qlens-hosts >> /etc/hosts"
        
        # Provide nip.io alternatives
        print_status "Alternative URLs using nip.io (automatic DNS):"
        echo "  QLens API:    http://qlens.$GATEWAY_IP.nip.io"
        echo "  Swagger UI:   http://swagger.$GATEWAY_IP.nip.io"
        echo "  Grafana:      http://grafana.$GATEWAY_IP.nip.io"
        echo "  Kiali:        http://kiali.$GATEWAY_IP.nip.io"
        echo "  Jaeger:       http://jaeger.$GATEWAY_IP.nip.io"
    fi
}

# Display final access information
display_access_info() {
    print_header "🎉 Setup Complete!"
    
    echo ""
    echo "📋 QLens Local Access Summary:"
    echo "  • Namespace: $NAMESPACE"
    echo "  • LoadBalancer IP: $GATEWAY_IP"
    echo "  • Istio Version: $(istioctl version --short 2>/dev/null || echo $ISTIO_VERSION)"
    echo ""
    
    if [ "$GATEWAY_IP" != "<PENDING>" ]; then
        echo "🌐 Service URLs:"
        echo "  • QLens API:      http://qlens.local (or http://qlens.$GATEWAY_IP.nip.io)"
        echo "  • Swagger UI:     http://swagger.local (or http://swagger.$GATEWAY_IP.nip.io)"
        echo "  • Grafana:        http://grafana.local (or http://grafana.$GATEWAY_IP.nip.io)"
        echo "  • Kiali:          http://kiali.local (or http://kiali.$GATEWAY_IP.nip.io)"
        echo "  • Jaeger:         http://jaeger.local (or http://jaeger.$GATEWAY_IP.nip.io)"
        echo ""
        
        echo "🔧 Test Commands:"
        echo "  curl http://qlens.$GATEWAY_IP.nip.io/health"
        echo "  curl http://swagger.$GATEWAY_IP.nip.io"
        echo ""
    fi
    
    echo "📊 Management Commands:"
    echo "  • Check status:   kubectl get pods -n $NAMESPACE"
    echo "  • View logs:      kubectl logs -f deployment/qlens-gateway -n $NAMESPACE"
    echo "  • Istio proxy:    istioctl proxy-status"
    echo "  • Kiali console:  istioctl dashboard kiali"
    echo ""
    
    echo "🛠️  Troubleshooting:"
    echo "  • Gateway status: kubectl get gateway,virtualservice -n istio-system"
    echo "  • Service mesh:   kubectl get pods -n $NAMESPACE -o wide"
    echo "  • LoadBalancer:   kubectl get svc istio-ingressgateway -n istio-system"
    echo ""
    
    print_success "QLens is now accessible through unified gateway endpoints!"
}

# Cleanup function
cleanup() {
    if [ -f /tmp/qlens-hosts ]; then
        rm -f /tmp/qlens-hosts
    fi
}

# Main execution
main() {
    print_header "QLens Local Unified Access Setup"
    echo "Setting up MetalLB + Istio + QLens for local Kubernetes"
    echo ""
    
    # Set trap for cleanup
    trap cleanup EXIT
    
    check_prerequisites
    install_metallb
    install_istio
    install_observability
    deploy_qlens
    setup_access
    display_access_info
    
    print_success "Setup completed successfully! 🎉"
}

# Handle script arguments
case ${1:-} in
    help|--help|-h)
        echo "QLens Local Access Setup Script"
        echo ""
        echo "Usage: $0 [namespace] [metallb-version] [istio-version]"
        echo ""
        echo "Arguments:"
        echo "  namespace        Kubernetes namespace for QLens (default: qlens-staging)"
        echo "  metallb-version  MetalLB version to install (default: v0.14.8)"
        echo "  istio-version    Istio version to install (default: 1.24.0)"
        echo ""
        echo "Examples:"
        echo "  $0                           # Use defaults"
        echo "  $0 qlens-staging            # Custom namespace"
        echo "  $0 qlens-staging v0.14.8 1.24.0  # All custom"
        exit 0
        ;;
esac

# Run main function
main