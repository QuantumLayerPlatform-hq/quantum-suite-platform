#!/bin/bash

# QLens Session Start Script
# Provides quick project context for new Claude Code sessions

set -e

# Color codes
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
PURPLE='\033[0;35m'
NC='\033[0m'

print_header() {
    echo -e "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${PURPLE}🚀 QLens Project Session Start${NC}"
    echo -e "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

print_section() {
    echo -e "\n${BLUE}📋 $1${NC}"
    echo "────────────────────────────────────────────────"
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

# Project Overview
show_project_overview() {
    print_section "Project Overview"
    
    if [ -f VERSION ]; then
        VERSION=$(cat VERSION)
        echo "Project: QLens LLM Gateway Service"
        echo "Version: $VERSION"
        echo "Repository: $(pwd)"
        echo "Architecture: Microservices with Istio Service Mesh"
    else
        print_warning "VERSION file not found"
    fi
}

# Check Git Status
check_git_status() {
    print_section "Git Status"
    
    if git rev-parse --git-dir > /dev/null 2>&1; then
        echo "Current branch: $(git branch --show-current)"
        echo "Last commit: $(git log -1 --oneline)"
        
        if [ -n "$(git status --porcelain)" ]; then
            print_warning "Working directory has uncommitted changes"
            git status --short | head -10
        else
            print_success "Working directory is clean"
        fi
    else
        print_error "Not a git repository"
    fi
}

# Show Recent Progress
show_recent_progress() {
    print_section "Recent Progress"
    
    if [ -f PROGRESS.md ]; then
        echo "Recent milestones:"
        grep -A 5 "Recent Milestones" PROGRESS.md | tail -5 | sed 's/^/  /'
        
        echo ""
        echo "Current blockers:"
        grep -A 10 "Blockers & Risks" PROGRESS.md | grep -E "^\|.*\|.*High.*\|" | head -3 | sed 's/^/  /'
    else
        print_warning "PROGRESS.md not found - consider creating it"
    fi
}

# Check Environment
check_environment() {
    print_section "Environment Check"
    
    # Check kubectl
    if command -v kubectl &> /dev/null; then
        CONTEXT=$(kubectl config current-context 2>/dev/null || echo "none")
        print_success "kubectl available (context: $CONTEXT)"
        
        # Check cluster access
        if kubectl cluster-info &> /dev/null; then
            NODES=$(kubectl get nodes --no-headers 2>/dev/null | wc -l)
            print_success "Cluster access confirmed ($NODES nodes)"
        else
            print_warning "Cluster access issue"
        fi
    else
        print_warning "kubectl not found"
    fi
    
    # Check Docker
    if command -v docker &> /dev/null; then
        if docker info &> /dev/null; then
            print_success "Docker available and running"
        else
            print_warning "Docker available but not running"
        fi
    else
        print_warning "Docker not found"
    fi
    
    # Check Helm
    if command -v helm &> /dev/null; then
        print_success "Helm available ($(helm version --short 2>/dev/null || echo 'unknown version'))"
    else
        print_warning "Helm not found"
    fi
    
    # Check istioctl
    if command -v istioctl &> /dev/null; then
        print_success "istioctl available ($(istioctl version --short 2>/dev/null || echo 'unknown version'))"
    else
        print_warning "istioctl not found"
    fi
}

# Show QLens Status
check_qlens_status() {
    print_section "QLens Service Status"
    
    if kubectl get ns qlens-staging &> /dev/null; then
        PODS=$(kubectl get pods -n qlens-staging --no-headers 2>/dev/null | wc -l)
        READY=$(kubectl get pods -n qlens-staging --no-headers 2>/dev/null | grep "Running" | wc -l)
        
        if [ "$PODS" -gt 0 ]; then
            print_success "QLens staging namespace exists ($READY/$PODS pods ready)"
            kubectl get pods -n qlens-staging 2>/dev/null | head -5
        else
            print_warning "QLens staging namespace exists but no pods found"
        fi
    else
        print_warning "QLens staging namespace not found"
    fi
    
    # Check Istio
    if kubectl get ns istio-system &> /dev/null; then
        ISTIO_PODS=$(kubectl get pods -n istio-system --no-headers 2>/dev/null | grep "Running" | wc -l)
        if [ "$ISTIO_PODS" -gt 0 ]; then
            print_success "Istio system running ($ISTIO_PODS pods)"
        else
            print_warning "Istio system namespace exists but pods not ready"
        fi
    else
        print_warning "Istio system not installed"
    fi
}

# Show Access Information
show_access_info() {
    print_section "Service Access"
    
    if kubectl get svc istio-ingressgateway -n istio-system &> /dev/null 2>&1; then
        GATEWAY_IP=$(kubectl get svc istio-ingressgateway -n istio-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
        
        if [ -n "$GATEWAY_IP" ] && [ "$GATEWAY_IP" != "null" ]; then
            print_success "LoadBalancer IP assigned: $GATEWAY_IP"
            echo ""
            echo "Service URLs:"
            echo "  QLens API:    http://qlens.$GATEWAY_IP.nip.io"
            echo "  Swagger UI:   http://swagger.$GATEWAY_IP.nip.io"
            echo "  Grafana:      http://grafana.$GATEWAY_IP.nip.io"
            echo "  Kiali:        http://kiali.$GATEWAY_IP.nip.io"
            echo ""
            echo "Quick test:"
            echo "  curl http://qlens.$GATEWAY_IP.nip.io/health"
        else
            print_warning "LoadBalancer IP pending"
            echo "Check with: kubectl get svc istio-ingressgateway -n istio-system"
        fi
    else
        print_warning "Istio ingress gateway not found"
    fi
}

# Show Available Commands
show_commands() {
    print_section "Key Commands"
    
    if [ -f Makefile ]; then
        echo "Development:"
        echo "  make dev-up              # Start local development"
        echo "  make get-access-info     # Show service URLs"
        echo "  make dev-status          # Check service status"
        echo "  make dev-logs            # View logs"
        echo ""
        echo "Build & Test:"
        echo "  make build               # Build all services"
        echo "  make test                # Run tests"
        echo "  make lint                # Run linter"
        echo "  make docs                # Generate docs"
        echo ""
        echo "Version & Deploy:"
        echo "  make version             # Show version"
        echo "  make version-patch       # Increment patch"
        echo "  make deploy-staging      # Deploy to staging"
    else
        print_warning "Makefile not found"
    fi
}

# Show Session Recommendations
show_recommendations() {
    print_section "Session Recommendations"
    
    # Check for compilation issues
    if ! go build ./... &> /dev/null; then
        print_error "Compilation issues detected - fix these first!"
        echo "Run: go build ./... to see errors"
    else
        print_success "Code compiles successfully"
    fi
    
    # Check if services are running
    if kubectl get pods -n qlens-staging --no-headers 2>/dev/null | grep -q "Running"; then
        print_success "Services are running - ready for development"
        echo "Suggested: Test end-to-end functionality"
    else
        print_warning "Services not running"
        echo "Suggested: Run 'make dev-up' to start services"
    fi
    
    # Check documentation
    if [ -f docs/swagger.json ]; then
        print_success "API documentation is current"
    else
        print_warning "API documentation may need regeneration"
        echo "Suggested: Run 'make docs'"
    fi
}

# Show Next Steps
show_next_steps() {
    print_section "Suggested Next Steps"
    
    if [ -f PROGRESS.md ]; then
        echo "Based on PROGRESS.md:"
        grep -A 10 "Next Session Priorities" PROGRESS.md 2>/dev/null | grep -E "^\s*[0-9]" | head -5 | sed 's/^/  /'
    fi
    
    echo ""
    echo "Session workflow:"
    echo "  1. Fix any compilation issues"
    echo "  2. Start services: make dev-up"
    echo "  3. Test functionality end-to-end"
    echo "  4. Work on next priority items"
    echo "  5. Update PROGRESS.md before ending session"
}

# Main execution
main() {
    print_header
    show_project_overview
    check_git_status
    show_recent_progress
    check_environment
    check_qlens_status
    show_access_info
    show_commands
    show_recommendations
    show_next_steps
    
    echo ""
    echo -e "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}🎯 Session context loaded successfully!${NC}"
    echo -e "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Run main function
main