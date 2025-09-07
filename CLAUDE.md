# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project: QLens LLM Gateway Service

This is a Go-based microservices platform that provides LLM gateway functionality as part of the larger Quantum Suite Platform. The primary component is QLens, which consists of Gateway, Router, and Cache services designed to run on Kubernetes with Istio service mesh.

## Essential Commands

### Development Workflow
```bash
# Setup and start development environment
make dev-setup              # Install dependencies and verify environment
make dev-up                 # Start local K8s staging with MetalLB + Istio
make get-access-info        # Show service URLs and access information
make dev-status             # Check running services status
make dev-logs               # View development environment logs
make dev-down               # Stop development environment

# Run services locally (single service development)
make run-local              # Run gateway locally with Swagger UI
make docs-dev               # Generate docs and run gateway with UI
```

### Build and Test
```bash
# Building
make build                  # Build all QLens services (gateway, router, cache)
make build-linux            # Build Linux binaries for production
make clean                  # Clean build artifacts

# Testing
make test                   # Run all tests with coverage
make test-unit              # Run unit tests only  
make test-integration       # Run integration tests only
make test-e2e               # Run end-to-end tests
make test-coverage          # Generate HTML coverage report

# Code Quality
make lint                   # Run golangci-lint
make fmt                    # Format code and tidy modules
make security-scan          # Run gosec security scanning
```

### Documentation and API
```bash
make docs                   # Generate Swagger documentation
make docs-serve             # Serve documentation on localhost:8080
```

## Architecture

The system follows a microservices architecture with three core services:

- **Gateway Service** (`cmd/qlens-gateway/main.go`) - Main API gateway with Swagger documentation
- **Router Service** (`cmd/qlens-router/main.go`) - Request routing and load balancing  
- **Cache Service** (`cmd/qlens-cache/main.go`) - Redis-based caching layer

Key architectural patterns:
- **Microservices**: Gateway → Router → Cache flow
- **Service Mesh**: Istio for traffic management and observability
- **Local Development**: MetalLB LoadBalancer + Istio Gateway (eliminates port-forwarding)
- **Documentation**: Swagger/OpenAPI with interactive UI
- **Infrastructure**: Kubernetes-native with Helm charts

## Code Structure

### Core Application Code
- `cmd/` - Service entrypoints (main.go files)
- `internal/` - Private application code
  - `internal/domain/` - Domain models, entities, and events
  - `internal/services/` - Business logic for gateway, router, cache
  - `internal/providers/` - External LLM provider integrations (AWS Bedrock, Azure OpenAI)
- `pkg/` - Public/shared packages
  - `pkg/qlens/` - QLens client libraries and core functionality
  - `pkg/shared/` - Shared utilities (logger, errors, env)

### Infrastructure & Deployment
- `charts/qlens/` - Helm charts with staging/production values
- `deployments/` - MetalLB and Istio configurations
- `scripts/` - Build automation and version management
- `Makefile` - All automation commands

### Dependencies
The project uses:
- **Web Framework**: Gin (github.com/gin-gonic/gin)
- **Documentation**: Swag for Swagger generation
- **Observability**: Prometheus, Zap logging
- **Cloud**: AWS SDK v2, Azure integrations
- **Caching**: Redis (github.com/redis/go-redis/v9)

## Development Context

### Current Issues
The project has known compilation issues including:
- Import cycle between pkg/qlens and pkg/qlens/providers
- Missing imports and type mismatches in domain/events and shared packages
- Method name inconsistencies (withField vs WithField)

### Version Management
The project uses semantic versioning with automated scripts:
```bash
make version                # Show current version
make version-patch          # Increment patch version
make version-minor          # Increment minor version  
make release-patch          # Full patch release process
```

### Local Development Setup
The project uses MetalLB + Istio for unified local access without port-forwarding:
- Services accessible via LoadBalancer IPs with .nip.io domains
- Swagger UI available at http://swagger.{IP}.nip.io
- Observability tools (Grafana, Kiali) integrated

### Testing Strategy
- Unit tests for individual components
- Integration tests for service interactions
- E2E tests in `tests/e2e/` directory
- Performance tests in `tests/performance/`
- Coverage reports generated in HTML format

## Important Notes

- Always run `make lint` and `make test` before committing
- Use `make dev-up` for full local Kubernetes environment
- Check `make get-access-info` for current service URLs
- The project is designed for Kubernetes deployment but services can run locally
- Swagger documentation is automatically generated from code annotations