# QLens LLM Gateway Service Makefile
# Semantic versioning and build automation

.PHONY: help version build test lint clean docker helm deploy

# =============================================================================
# CONFIGURATION
# =============================================================================

# Project information
PROJECT_NAME := qlens
VERSION := $(shell cat VERSION 2>/dev/null || echo "0.0.0")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME)

# Docker configuration
REGISTRY := ghcr.io/quantumlayerplatform-hq/quantum-suite-platform
DOCKER_PLATFORMS := linux/amd64,linux/arm64

# Directories
BUILD_DIR := build
DIST_DIR := dist
CHARTS_DIR := charts/qlens

# Go settings
GO := go
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# QLens Services
SERVICES := qlens-gateway qlens-router qlens-cache
MODULES := $(SERVICES)

# Docker image prefix
DOCKER_IMAGE_PREFIX := $(REGISTRY)

# Git information
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git branch --show-current 2>/dev/null || echo "unknown")

# Go files for formatting
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*' -not -path './.git/*')

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# =============================================================================
# HELP
# =============================================================================

help: ## Display this help screen
	@echo "$(BLUE)QLens LLM Gateway Service$(NC)"
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Git Branch: $(GIT_BRANCH)"
	@echo ""
	@echo "$(YELLOW)Available commands:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# =============================================================================
# DEVELOPMENT ENVIRONMENT
# =============================================================================

dev-setup: ## Setup development environment
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@command -v go >/dev/null 2>&1 || { echo "$(RED)Go is not installed$(NC)"; exit 1; }
	@echo "Go version: $$(go version)"
	@go mod download
	@go mod verify
	@echo "$(GREEN)Development environment ready!$(NC)"

dev-up: ## Start development environment (local k8s staging)
	@echo "$(BLUE)Starting QLens staging environment...$(NC)"
	@$(MAKE) setup-local-access
	@echo "$(GREEN)QLens staging environment started!$(NC)"

dev-down: ## Stop development environment (local k8s staging)
	@echo "$(BLUE)Stopping QLens staging environment...$(NC)"
	@$(MAKE) k8s-delete-staging
	@echo "$(GREEN)QLens staging environment stopped!$(NC)"

dev-logs: ## View development environment logs
	@$(MAKE) k8s-logs-staging

dev-status: ## Show development environment status
	@$(MAKE) k8s-status-staging

# =============================================================================
# LOCAL ACCESS SETUP
# =============================================================================

setup-local-access: ## Setup unified local access with MetalLB + Istio
	@echo "$(BLUE)Setting up unified local access...$(NC)"
	@./scripts/setup-local-access.sh qlens-staging
	@echo "$(GREEN)Local access setup completed!$(NC)"

install-metallb: ## Install MetalLB for local LoadBalancer support
	@echo "$(BLUE)Installing MetalLB...$(NC)"
	@kubectl apply -k deployments/metallb
	@echo "$(GREEN)MetalLB installed!$(NC)"

install-istio: ## Install Istio service mesh
	@echo "$(BLUE)Installing Istio...$(NC)"
	@command -v istioctl >/dev/null 2>&1 || { echo "$(RED)istioctl is not installed$(NC)"; exit 1; }
	@istioctl install --set values.pilot.env.EXTERNAL_ISTIOD=false -y
	@kubectl apply -k deployments/istio/local
	@echo "$(GREEN)Istio installed and configured!$(NC)"

setup-observability: ## Install Kiali, Grafana, Jaeger, Prometheus
	@echo "$(BLUE)Setting up observability tools...$(NC)"
	@kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.24/samples/addons/kiali.yaml
	@kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.24/samples/addons/grafana.yaml
	@kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.24/samples/addons/prometheus.yaml
	@echo "$(GREEN)Observability tools installed!$(NC)"

get-access-info: ## Show current access information
	@echo "$(BLUE)QLens Access Information$(NC)"
	@echo ""
	@GATEWAY_IP=$$(kubectl get service istio-ingressgateway -n istio-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending") \
	&& echo "LoadBalancer IP: $$GATEWAY_IP" \
	&& if [ "$$GATEWAY_IP" != "pending" ] && [ -n "$$GATEWAY_IP" ]; then \
		echo ""; \
		echo "$(YELLOW)Service URLs:$(NC)"; \
		echo "  QLens API:    http://qlens.$$GATEWAY_IP.nip.io"; \
		echo "  Swagger UI:   http://swagger.$$GATEWAY_IP.nip.io"; \
		echo "  Grafana:      http://grafana.$$GATEWAY_IP.nip.io"; \
		echo "  Kiali:        http://kiali.$$GATEWAY_IP.nip.io"; \
		echo "  Jaeger:       http://jaeger.$$GATEWAY_IP.nip.io"; \
		echo ""; \
		echo "$(YELLOW)Test Commands:$(NC)"; \
		echo "  curl http://qlens.$$GATEWAY_IP.nip.io/health"; \
		echo "  curl http://swagger.$$GATEWAY_IP.nip.io"; \
	else \
		echo "$(YELLOW)LoadBalancer IP is pending. Check with:$(NC)"; \
		echo "  kubectl get svc istio-ingressgateway -n istio-system"; \
	fi

# =============================================================================
# VERSION MANAGEMENT
# =============================================================================

version: ## Show current version information
	@./scripts/version.sh info

version-patch: ## Increment patch version (1.0.0 → 1.0.1)
	@./scripts/version.sh release patch

version-minor: ## Increment minor version (1.0.0 → 1.1.0)
	@./scripts/version.sh release minor

version-major: ## Increment major version (1.0.0 → 2.0.0)
	@./scripts/version.sh release major

version-prerelease: ## Set pre-release version (requires TYPE: alpha, beta, rc.1)
	@if [ -z "$(TYPE)" ]; then \
		echo "$(RED)Error: TYPE required (alpha, beta, rc.1)$(NC)"; \
		echo "Usage: make version-prerelease TYPE=alpha"; \
		exit 1; \
	fi
	@./scripts/version.sh prerelease $(TYPE)

version-set: ## Set specific version (requires VER: 1.2.3)
	@if [ -z "$(VER)" ]; then \
		echo "$(RED)Error: VER required$(NC)"; \
		echo "Usage: make version-set VER=1.2.3"; \
		exit 1; \
	fi
	@./scripts/version.sh set $(VER)

version-build-metadata: ## Generate version with build metadata
	@./scripts/version.sh build-metadata

# =============================================================================
# BUILD
# =============================================================================

build: ## Build all QLens services
	@echo "$(BLUE)Building QLens services...$(NC)"
	@mkdir -p bin
	@for service in $(SERVICES); do \
		echo "$(YELLOW)Building $$service...$(NC)"; \
		go build -ldflags "$(LDFLAGS)" -o bin/$$service ./cmd/$$service; \
	done
	@echo "$(GREEN)All QLens services built successfully!$(NC)"

build-linux: ## Build Linux binaries for production
	@echo "$(BLUE)Building QLens Linux binaries...$(NC)"
	@mkdir -p bin/linux
	@for service in $(SERVICES); do \
		echo "$(YELLOW)Building qlens-$$service for Linux...$(NC)"; \
		GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/linux/qlens-$$service ./cmd/$$service; \
	done
	@echo "$(GREEN)QLens Linux binaries built successfully!$(NC)"

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf bin/
	@rm -rf dist/
	@go clean -cache
	@echo "$(GREEN)Build artifacts cleaned!$(NC)"

# =============================================================================
# TESTING
# =============================================================================

test: ## Run all tests
	@echo "$(BLUE)Running tests...$(NC)"
	@go test -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)Tests completed!$(NC)"

test-unit: ## Run unit tests only
	@echo "$(BLUE)Running unit tests...$(NC)"
	@go test -race -short -coverprofile=coverage-unit.out ./...

test-integration: ## Run integration tests only
	@echo "$(BLUE)Running integration tests...$(NC)"
	@go test -race -run Integration -coverprofile=coverage-integration.out ./...

test-e2e: ## Run end-to-end tests
	@echo "$(BLUE)Running E2E tests...$(NC)"
	@cd tests/e2e && go test -v ./...

test-performance: ## Run performance tests
	@echo "$(BLUE)Running performance tests...$(NC)"
	@cd tests/performance && go test -v -bench=. -benchmem ./...

test-coverage: test ## Generate test coverage report
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

# =============================================================================
# CODE QUALITY
# =============================================================================

lint: ## Run linter
	@echo "$(BLUE)Running linter...$(NC)"
	@command -v golangci-lint >/dev/null 2>&1 || { echo "$(RED)golangci-lint is not installed$(NC)"; exit 1; }
	@golangci-lint run ./...
	@echo "$(GREEN)Linting completed!$(NC)"

fmt: ## Format code
	@echo "$(BLUE)Formatting code...$(NC)"
	@gofmt -w $(GO_FILES)
	@go mod tidy
	@echo "$(GREEN)Code formatted!$(NC)"

security-scan: ## Run security scanning
	@echo "$(BLUE)Running security scan...$(NC)"
	@command -v gosec >/dev/null 2>&1 || { echo "$(RED)gosec is not installed$(NC)"; exit 1; }
	@gosec ./...
	@echo "$(GREEN)Security scan completed!$(NC)"

# =============================================================================
# DATABASE
# =============================================================================

db-migrate: ## Run database migrations
	@echo "$(BLUE)Running database migrations...$(NC)"
	@command -v goose >/dev/null 2>&1 || { echo "$(RED)goose is not installed$(NC)"; exit 1; }
	@goose -dir migrations postgres "postgres://quantum_user:quantum_pass@localhost:5432/quantum_dev?sslmode=disable" up
	@echo "$(GREEN)Database migrations completed!$(NC)"

db-rollback: ## Rollback last database migration
	@echo "$(BLUE)Rolling back database migration...$(NC)"
	@goose -dir migrations postgres "postgres://quantum_user:quantum_pass@localhost:5432/quantum_dev?sslmode=disable" down
	@echo "$(GREEN)Database rollback completed!$(NC)"

db-status: ## Show database migration status
	@goose -dir migrations postgres "postgres://quantum_user:quantum_pass@localhost:5432/quantum_dev?sslmode=disable" status

db-reset: ## Reset database (WARNING: destructive)
	@echo "$(RED)WARNING: This will destroy all data!$(NC)"
	@read -p "Are you sure? (y/N): " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		goose -dir migrations postgres "postgres://quantum_user:quantum_pass@localhost:5432/quantum_dev?sslmode=disable" reset; \
		echo "$(GREEN)Database reset completed!$(NC)"; \
	else \
		echo "$(YELLOW)Database reset cancelled$(NC)"; \
	fi

# =============================================================================
# DOCKER
# =============================================================================

docker-build: ## Build Docker images for all QLens services
	@echo "$(BLUE)Building QLens Docker images...$(NC)"
	@for service in $(SERVICES); do \
		short_name=$${service#qlens-}; \
		echo "$(YELLOW)Building Docker image for $$service:$(VERSION)...$(NC)"; \
		docker build -t $(REGISTRY)/$$service:$(VERSION) -f Dockerfile.$$short_name .; \
		docker tag $(REGISTRY)/$$service:$(VERSION) $(REGISTRY)/$$service:latest; \
	done
	@echo "$(GREEN)QLens Docker images built successfully!$(NC)"

docker-push: docker-build ## Push Docker images to registry
	@echo "$(BLUE)Pushing QLens Docker images...$(NC)"
	@for service in $(SERVICES); do \
		echo "$(YELLOW)Pushing $$service:$(VERSION)...$(NC)"; \
		docker push $(REGISTRY)/$$service:$(VERSION); \
		docker push $(REGISTRY)/$$service:latest; \
	done
	@echo "$(GREEN)QLens Docker images pushed successfully!$(NC)"

docker-clean: ## Clean Docker artifacts
	@echo "$(BLUE)Cleaning Docker artifacts...$(NC)"
	@docker system prune -f
	@docker volume prune -f
	@echo "$(GREEN)Docker artifacts cleaned!$(NC)"

# =============================================================================
# HELM CHARTS
# =============================================================================

helm-lint: ## Lint Helm charts
	@echo "$(BLUE)Linting QLens Helm chart...$(NC)"
	@helm lint charts/qlens
	@echo "$(GREEN)Helm chart linting completed!$(NC)"

helm-template: ## Generate Kubernetes manifests from Helm chart
	@echo "$(BLUE)Generating Kubernetes manifests...$(NC)"
	@helm template qlens charts/qlens --values charts/qlens/values-staging.yaml
	@echo "$(GREEN)Kubernetes manifests generated!$(NC)"

helm-package: ## Package Helm chart with current version
	@echo "$(BLUE)Packaging QLens Helm chart v$(VERSION)...$(NC)"
	@helm package charts/qlens --version $(VERSION) --app-version $(VERSION)
	@echo "$(GREEN)Helm chart packaged: qlens-$(VERSION).tgz$(NC)"

helm-install-staging: ## Install QLens to staging environment
	@echo "$(BLUE)Installing QLens v$(VERSION) to staging...$(NC)"
	@helm upgrade --install qlens charts/qlens \
		--namespace qlens-staging \
		--create-namespace \
		--values charts/qlens/values-staging.yaml \
		--set image.tag=$(VERSION)
	@echo "$(GREEN)QLens v$(VERSION) installed to staging!$(NC)"

helm-install-production: ## Install QLens to production environment
	@echo "$(BLUE)Installing QLens v$(VERSION) to production...$(NC)"
	@helm upgrade --install qlens charts/qlens \
		--namespace qlens-production \
		--create-namespace \
		--values charts/qlens/values-production.yaml \
		--set image.tag=$(VERSION)
	@echo "$(GREEN)QLens v$(VERSION) installed to production!$(NC)"

# =============================================================================
# KUBERNETES DEPLOYMENT
# =============================================================================

k8s-status-staging: ## Show QLens staging deployment status
	@kubectl get all -n qlens-staging -l app.kubernetes.io/name=qlens

k8s-status-production: ## Show QLens production deployment status
	@kubectl get all -n qlens-production -l app.kubernetes.io/name=qlens

k8s-logs-staging: ## Show QLens staging logs
	@kubectl logs -f -l app.kubernetes.io/name=qlens -n qlens-staging

k8s-logs-production: ## Show QLens production logs
	@kubectl logs -f -l app.kubernetes.io/name=qlens -n qlens-production

k8s-delete-staging: ## Delete QLens staging deployment
	@echo "$(RED)WARNING: This will delete the staging deployment!$(NC)"
	@read -p "Are you sure? (y/N): " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		kubectl delete namespace qlens-staging; \
		echo "$(GREEN)Staging deployment deleted!$(NC)"; \
	else \
		echo "$(YELLOW)Deletion cancelled$(NC)"; \
	fi

k8s-delete-production: ## Delete QLens production deployment
	@echo "$(RED)WARNING: This will delete the production deployment!$(NC)"
	@read -p "Are you sure? (y/N): " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		kubectl delete namespace qlens-production; \
		echo "$(GREEN)Production deployment deleted!$(NC)"; \
	else \
		echo "$(YELLOW)Deletion cancelled$(NC)"; \
	fi

# =============================================================================
# MONITORING
# =============================================================================

metrics: ## Show QLens application metrics
	@curl -s http://localhost:8080/metrics | grep -E "qlens_" | head -20

health-check: ## Check QLens service health
	@echo "$(BLUE)Checking QLens service health...$(NC)"
	@echo -n "Gateway (8080): "; curl -s -f http://localhost:8080/health > /dev/null && echo "$(GREEN)✓ Healthy$(NC)" || echo "$(RED)✗ Unhealthy$(NC)"
	@echo -n "Router (8081): "; curl -s -f http://localhost:8081/health > /dev/null && echo "$(GREEN)✓ Healthy$(NC)" || echo "$(RED)✗ Unhealthy$(NC)"
	@echo -n "Cache (8082): "; curl -s -f http://localhost:8082/health > /dev/null && echo "$(GREEN)✓ Healthy$(NC)" || echo "$(RED)✗ Unhealthy$(NC)"

monitor: ## Open monitoring dashboards
	@echo "$(BLUE)Opening monitoring dashboards...$(NC)"
	@echo "Grafana: http://localhost:3000"
	@echo "Prometheus: http://localhost:9090"
	@echo "Jaeger: http://localhost:16686"
	@command -v open >/dev/null 2>&1 && open http://localhost:3000 || echo "Open dashboards manually"

# =============================================================================
# DOCUMENTATION
# =============================================================================

docs: ## Generate Swagger documentation
	@echo "$(BLUE)Generating QLens Swagger documentation...$(NC)"
	@command -v swag >/dev/null 2>&1 || { echo "$(RED)swag is not installed$(NC)"; exit 1; }
	@swag init -g cmd/gateway/main.go -o docs --parseDependency --parseInternal
	@echo "$(GREEN)Swagger documentation generated!$(NC)"

docs-serve: ## Serve documentation locally
	@echo "$(BLUE)Starting documentation server...$(NC)"
	@cd docs && python3 -m http.server 8080 || python -m SimpleHTTPServer 8080
	@echo "Documentation available at http://localhost:8080"

# =============================================================================
# SESSION MANAGEMENT
# =============================================================================

session-start: ## Start new development session with context
	@./scripts/session-start.sh

session-end: ## End development session and prepare for next
	@./scripts/session-end.sh

# =============================================================================
# UTILITIES
# =============================================================================

install-tools: ## Install development tools
	@echo "$(BLUE)Installing development tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securecodewarrior/gosec/cmd/gosec@latest
	@go install github.com/pressly/goose/v3/cmd/goose@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "$(GREEN)Development tools installed!$(NC)"

docs-dev: docs ## Generate docs and run gateway locally with Swagger UI
	@echo "$(BLUE)Starting QLens Gateway locally with Swagger UI...$(NC)"
	@echo "$(YELLOW)Swagger UI will be available at: http://localhost:8080/swagger/index.html$(NC)"
	@echo "$(YELLOW)API Documentation: http://localhost:8080/docs$(NC)"
	@echo "$(YELLOW)Health Check: http://localhost:8080/health$(NC)"
	@cd cmd/gateway && go run main.go

run-local: docs ## Run QLens Gateway locally for development
	@echo "$(BLUE)Running QLens Gateway locally...$(NC)"
	@export ENVIRONMENT=development && cd cmd/gateway && go run main.go

show-version: ## Show version information
	@echo "$(BLUE)QLens LLM Gateway Service$(NC)"
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Git Branch: $(GIT_BRANCH)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $$(go version | cut -d' ' -f3)"

generate: ## Run go generate
	@echo "$(BLUE)Running code generation...$(NC)"
	@go generate ./...
	@echo "$(GREEN)Code generation completed!$(NC)"

vendor: ## Download dependencies to vendor directory
	@echo "$(BLUE)Vendoring dependencies...$(NC)"
	@go mod vendor
	@echo "$(GREEN)Dependencies vendored!$(NC)"

# =============================================================================
# CI/CD
# =============================================================================

ci-test: ## Run CI tests
	@echo "$(BLUE)Running CI test suite...$(NC)"
	@go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -func=coverage.out

ci-build: ## CI build process
	@echo "$(BLUE)Running CI build...$(NC)"
	@$(MAKE) lint
	@$(MAKE) security-scan
	@$(MAKE) ci-test
	@$(MAKE) build-linux

ci-deploy: ## CI deployment process
	@echo "$(BLUE)Running CI deployment...$(NC)"
	@$(MAKE) docker-build
	@$(MAKE) docker-push

# =============================================================================
# RELEASE
# =============================================================================

pre-release-check: ## Perform pre-release checks
	@echo "$(BLUE)Running pre-release checks for v$(VERSION)...$(NC)"
	@echo "$(YELLOW)1. Checking Git status...$(NC)"
	@git diff-index --quiet HEAD -- || (echo "$(RED)Error: Uncommitted changes detected$(NC)" && exit 1)
	@echo "$(YELLOW)2. Running tests...$(NC)"
	@$(MAKE) test
	@echo "$(YELLOW)3. Running linter...$(NC)"
	@$(MAKE) lint
	@echo "$(YELLOW)4. Running security scan...$(NC)"
	@$(MAKE) security-scan
	@echo "$(GREEN)Pre-release checks passed!$(NC)"

release-patch: pre-release-check ## Create patch release (1.0.0 → 1.0.1)
	@echo "$(BLUE)Creating patch release...$(NC)"
	@$(MAKE) version-patch
	@$(MAKE) release-build-and-push

release-minor: pre-release-check ## Create minor release (1.0.0 → 1.1.0)
	@echo "$(BLUE)Creating minor release...$(NC)"
	@$(MAKE) version-minor
	@$(MAKE) release-build-and-push

release-major: pre-release-check ## Create major release (1.0.0 → 2.0.0)
	@echo "$(BLUE)Creating major release...$(NC)"
	@$(MAKE) version-major
	@$(MAKE) release-build-and-push

release-build-and-push: ## Build and push release artifacts
	@echo "$(BLUE)Building and pushing release v$(VERSION)...$(NC)"
	@$(MAKE) build-linux
	@$(MAKE) docker-build
	@$(MAKE) docker-push
	@$(MAKE) helm-package
	@echo "$(GREEN)Release v$(VERSION) artifacts created and pushed!$(NC)"
	@echo ""
	@echo "$(YELLOW)Next Steps:$(NC)"
	@echo "  1. Review and commit version changes: git add . && git commit -m 'chore: release v$(VERSION)'"
	@echo "  2. Push changes: git push origin $(GIT_BRANCH)"
	@echo "  3. Push tag: git push origin v$(VERSION)"
	@echo "  4. Create GitHub release"
	@echo "  5. Deploy to staging: make deploy-staging"
	@echo "  6. Deploy to production: make deploy-production"

deploy-staging: ## Deploy current version to staging
	@echo "$(BLUE)Deploying QLens v$(VERSION) to staging...$(NC)"
	@$(MAKE) helm-install-staging
	@echo "$(GREEN)QLens v$(VERSION) deployed to staging!$(NC)"

deploy-production: ## Deploy current version to production
	@echo "$(BLUE)Deploying QLens v$(VERSION) to production...$(NC)"
	@$(MAKE) helm-install-production
	@echo "$(GREEN)QLens v$(VERSION) deployed to production!$(NC)"

rollback-staging: ## Rollback staging to previous version
	@echo "$(BLUE)Rolling back staging deployment...$(NC)"
	@helm rollback qlens -n qlens-staging
	@echo "$(GREEN)Staging rollback completed!$(NC)"

rollback-production: ## Rollback production to previous version
	@echo "$(RED)WARNING: This will rollback production!$(NC)"
	@read -p "Are you sure? (y/N): " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		helm rollback qlens -n qlens-production; \
		echo "$(GREEN)Production rollback completed!$(NC)"; \
	else \
		echo "$(YELLOW)Rollback cancelled$(NC)"; \
	fi

# =============================================================================
# DEFAULT TARGET
# =============================================================================

.DEFAULT_GOAL := help