# Quantum Suite Platform Makefile
# Version: 1.0.0
# Description: Build, test, and deployment automation for Quantum Suite

.PHONY: help build test lint fmt clean dev-up dev-down deploy docs docker-build docker-push

# =============================================================================
# CONFIGURATION
# =============================================================================

# Project information
PROJECT_NAME := quantum-suite-platform
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go configuration
GO_VERSION := 1.21
GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin
GO_FILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")

# Docker configuration
DOCKER_REGISTRY := ghcr.io/quantum-suite
DOCKER_IMAGE_PREFIX := $(DOCKER_REGISTRY)/$(PROJECT_NAME)

# Build flags
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)

# Modules
MODULES := qagent qtest qsecure qsre qinfra llm-gateway vector-service mcp-hub

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
	@echo "$(BLUE)Quantum Suite Platform$(NC)"
	@echo "Version: $(VERSION)"
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

dev-up: ## Start development environment with Docker Compose
	@echo "$(BLUE)Starting development environment...$(NC)"
	@docker-compose -f deployments/docker/docker-compose.dev.yml up -d
	@echo "$(GREEN)Development environment started!$(NC)"
	@echo "$(YELLOW)Services available at:$(NC)"
	@echo "  - API Gateway: http://localhost:8000"
	@echo "  - Grafana: http://localhost:3000 (admin/quantum123)"
	@echo "  - Jaeger: http://localhost:16686"
	@echo "  - Prometheus: http://localhost:9090"

dev-down: ## Stop development environment
	@echo "$(BLUE)Stopping development environment...$(NC)"
	@docker-compose -f deployments/docker/docker-compose.dev.yml down
	@echo "$(GREEN)Development environment stopped!$(NC)"

dev-logs: ## View development environment logs
	@docker-compose -f deployments/docker/docker-compose.dev.yml logs -f

dev-clean: ## Clean development environment
	@echo "$(BLUE)Cleaning development environment...$(NC)"
	@docker-compose -f deployments/docker/docker-compose.dev.yml down -v --remove-orphans
	@docker system prune -f
	@echo "$(GREEN)Development environment cleaned!$(NC)"

# =============================================================================
# BUILD
# =============================================================================

build: ## Build all binaries
	@echo "$(BLUE)Building all modules...$(NC)"
	@for module in $(MODULES); do \
		echo "$(YELLOW)Building $$module...$(NC)"; \
		go build -ldflags "$(LDFLAGS)" -o bin/$$module ./cmd/$$module; \
	done
	@echo "$(GREEN)All modules built successfully!$(NC)"

build-linux: ## Build Linux binaries for production
	@echo "$(BLUE)Building Linux binaries...$(NC)"
	@for module in $(MODULES); do \
		echo "$(YELLOW)Building $$module for Linux...$(NC)"; \
		GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/linux/$$module ./cmd/$$module; \
	done
	@echo "$(GREEN)Linux binaries built successfully!$(NC)"

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

docker-build: ## Build Docker images for all services
	@echo "$(BLUE)Building Docker images...$(NC)"
	@for module in $(MODULES); do \
		echo "$(YELLOW)Building Docker image for $$module...$(NC)"; \
		docker build -t $(DOCKER_IMAGE_PREFIX)/$$module:$(VERSION) -f deployments/docker/Dockerfile.$$module .; \
		docker tag $(DOCKER_IMAGE_PREFIX)/$$module:$(VERSION) $(DOCKER_IMAGE_PREFIX)/$$module:latest; \
	done
	@echo "$(GREEN)Docker images built successfully!$(NC)"

docker-push: docker-build ## Push Docker images to registry
	@echo "$(BLUE)Pushing Docker images...$(NC)"
	@for module in $(MODULES); do \
		echo "$(YELLOW)Pushing $$module:$(VERSION)...$(NC)"; \
		docker push $(DOCKER_IMAGE_PREFIX)/$$module:$(VERSION); \
		docker push $(DOCKER_IMAGE_PREFIX)/$$module:latest; \
	done
	@echo "$(GREEN)Docker images pushed successfully!$(NC)"

docker-clean: ## Clean Docker artifacts
	@echo "$(BLUE)Cleaning Docker artifacts...$(NC)"
	@docker system prune -f
	@docker volume prune -f
	@echo "$(GREEN)Docker artifacts cleaned!$(NC)"

# =============================================================================
# KUBERNETES DEPLOYMENT
# =============================================================================

k8s-namespace: ## Create Kubernetes namespace
	@echo "$(BLUE)Creating Kubernetes namespace...$(NC)"
	@kubectl apply -f deployments/kubernetes/base/namespace.yaml
	@echo "$(GREEN)Namespace created!$(NC)"

k8s-deploy-dev: ## Deploy to development environment
	@echo "$(BLUE)Deploying to development environment...$(NC)"
	@kubectl apply -k deployments/kubernetes/overlays/development
	@echo "$(GREEN)Deployment to development completed!$(NC)"

k8s-deploy-staging: ## Deploy to staging environment
	@echo "$(BLUE)Deploying to staging environment...$(NC)"
	@kubectl apply -k deployments/kubernetes/overlays/staging
	@echo "$(GREEN)Deployment to staging completed!$(NC)"

k8s-deploy-prod: ## Deploy to production environment
	@echo "$(BLUE)Deploying to production environment...$(NC)"
	@kubectl apply -k deployments/kubernetes/overlays/production
	@echo "$(GREEN)Deployment to production completed!$(NC)"

k8s-status: ## Show Kubernetes deployment status
	@kubectl get all -n quantum-suite

k8s-logs: ## Show Kubernetes logs
	@kubectl logs -f -l app.kubernetes.io/name=quantum-suite -n quantum-suite

k8s-delete: ## Delete Kubernetes deployment
	@echo "$(RED)WARNING: This will delete the entire deployment!$(NC)"
	@read -p "Are you sure? (y/N): " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		kubectl delete namespace quantum-suite; \
		echo "$(GREEN)Deployment deleted!$(NC)"; \
	else \
		echo "$(YELLOW)Deletion cancelled$(NC)"; \
	fi

# =============================================================================
# MONITORING
# =============================================================================

metrics: ## Show application metrics
	@curl -s http://localhost:8100/metrics | grep -E "(qagent|qtest|qsecure|qsre|qinfra)_" | head -20

health-check: ## Check service health
	@echo "$(BLUE)Checking service health...$(NC)"
	@for port in 8100 8101 8102 8110 8111 8112 8113 8114; do \
		echo -n "Service on port $$port: "; \
		curl -s -f http://localhost:$$port/health > /dev/null && echo "$(GREEN)✓ Healthy$(NC)" || echo "$(RED)✗ Unhealthy$(NC)"; \
	done

monitor: ## Open monitoring dashboards
	@echo "$(BLUE)Opening monitoring dashboards...$(NC)"
	@echo "Grafana: http://localhost:3000"
	@echo "Prometheus: http://localhost:9090"
	@echo "Jaeger: http://localhost:16686"
	@command -v open >/dev/null 2>&1 && open http://localhost:3000 || echo "Open dashboards manually"

# =============================================================================
# DOCUMENTATION
# =============================================================================

docs: ## Generate documentation
	@echo "$(BLUE)Generating documentation...$(NC)"
	@command -v swag >/dev/null 2>&1 || { echo "$(RED)swag is not installed$(NC)"; exit 1; }
	@swag init -g cmd/api-gateway/main.go -o docs/swagger
	@echo "$(GREEN)Documentation generated!$(NC)"

docs-serve: ## Serve documentation locally
	@echo "$(BLUE)Starting documentation server...$(NC)"
	@cd docs && python3 -m http.server 8080 || python -m SimpleHTTPServer 8080
	@echo "Documentation available at http://localhost:8080"

# =============================================================================
# UTILITIES
# =============================================================================

install-tools: ## Install development tools
	@echo "$(BLUE)Installing development tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@go install github.com/pressly/goose/v3/cmd/goose@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "$(GREEN)Development tools installed!$(NC)"

version: ## Show version information
	@echo "$(BLUE)Quantum Suite Platform$(NC)"
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
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

release: ## Create a new release
	@echo "$(BLUE)Creating release $(VERSION)...$(NC)"
	@$(MAKE) ci-build
	@$(MAKE) docker-build
	@$(MAKE) docker-push
	@echo "$(GREEN)Release $(VERSION) created successfully!$(NC)"

# =============================================================================
# DEFAULT TARGET
# =============================================================================

.DEFAULT_GOAL := help