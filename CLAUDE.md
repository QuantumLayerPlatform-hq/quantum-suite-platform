# Claude Code Session Context

**Purpose:** Maintain context and momentum across Claude Code sessions for the QLens project.

## 🏗️ Project: QLens LLM Gateway Service

**Repository:** `/home/satish/quantumlayerplatform`  
**Current Version:** 1.0.0  
**Architecture:** Microservices (Gateway, Router, Cache) with Istio Service Mesh  
**Environments:** Staging (Local K8s), Production (Azure K8s)  

## 📋 Session Continuation Protocol

### **Start of New Session**

1. **Read Project Status**
   ```bash
   cat PROGRESS.md | head -50    # Review current status
   git log --oneline -10         # Check recent commits
   make version                  # Check current version
   ```

2. **Environment Check**
   ```bash
   kubectl get nodes             # Verify K8s access
   kubectl get ns | grep qlens   # Check QLens namespaces
   make get-access-info          # Check service access
   ```

3. **Quick Health Check**
   ```bash
   make dev-status               # Check running services
   make test                     # Verify tests pass
   make lint                     # Check code quality
   ```

### **During Session**

- Update `PROGRESS.md` for major milestones
- Use TodoWrite tool to track current tasks
- Document decisions in relevant files
- Keep git commits small and descriptive

### **End of Session**

1. **Update Progress**
   ```bash
   # Update PROGRESS.md with completed items
   # Note any blockers or next priorities
   # Document session outcomes
   ```

2. **Clean Commit**
   ```bash
   git add .
   git commit -m "session: [brief description of work]"
   ```

3. **Leave Context Notes**
   - Update this file with current context
   - Note any unfinished work
   - Document next session priorities

## 🎯 Current Context (Session of 2025-09-06)

### **What We Just Completed:**
1. ✅ **Swagger/OpenAPI Integration**: Full API documentation with interactive UI
2. ✅ **Semantic Versioning System**: Automated version management across all artifacts
3. ✅ **Unified Local Access**: MetalLB + Istio setup eliminates port-forwarding
4. ✅ **Service Mesh Integration**: Complete Istio configuration with observability
5. ✅ **Project Tracking System**: This progress tracking framework

### **Current State:**
- **Status**: 🟡 Compilation issues blocking development
- **Priority**: Fix Go compilation errors in domain/events and shared/env packages
- **Infrastructure**: MetalLB + Istio + QLens fully configured
- **Access**: Unified gateway working (pending compilation fix)

### **Immediate Blockers:**
```bash
# These compilation errors need fixing:
# 1. internal/domain/events.go:491 - GoldenImageCreated interface issue
# 2. internal/domain/qlens.go:6 - unused import github.com/google/uuid
# 3. pkg/shared/env/detector.go:63 - Port redeclared
# 4. pkg/shared/env/detector.go:156 - type mismatch int vs string
# 5. pkg/shared/logger/logger.go:230 - withField vs WithField method
```

### **Next Session Priorities:**
1. 🔥 **P0**: Fix all compilation errors
2. 🎯 **P1**: Test complete system end-to-end
3. 🚀 **P1**: Deploy to staging with unified access
4. 🔒 **P2**: Implement authentication system
5. ⚡ **P2**: Add rate limiting

## 🛠️ Key Commands & Patterns

### **Development Workflow**
```bash
# Standard development cycle
make dev-up                   # Start everything locally
make get-access-info          # Get service URLs
make dev-logs                 # View logs
make dev-down                 # Clean shutdown
```

### **Testing & Quality**
```bash
make test                     # Run all tests
make test-coverage            # Generate coverage report
make lint                     # Code quality check
make security-scan            # Security analysis
```

### **Version Management**
```bash
make version                  # Show current version
make version-patch            # Increment patch (1.0.0 → 1.0.1)
make version-minor            # Increment minor (1.0.0 → 1.1.0)
make release-patch            # Full patch release process
```

### **Deployment**
```bash
make deploy-staging           # Deploy to local K8s
make deploy-production        # Deploy to Azure K8s
make rollback-staging         # Rollback if needed
```

## 🏗️ Architecture Decisions Made

1. **Microservices Pattern**: Gateway → Router → Cache
2. **Service Mesh**: Istio for traffic management, security, observability  
3. **Local Development**: MetalLB LoadBalancer + Istio Gateway (no port-forwarding)
4. **Versioning**: Semantic versioning with automated script
5. **Documentation**: Swagger/OpenAPI with interactive UI
6. **Infrastructure**: Kubernetes-native with Helm charts

## 📁 Important Files

### **Core Application**
- `cmd/gateway/main.go` - Main gateway service entry point
- `internal/services/gateway/handlers.go` - Swagger-annotated handlers
- `internal/services/gateway/models.go` - OpenAPI response models
- `internal/domain/` - Core domain models and logic

### **Infrastructure**
- `charts/qlens/` - Helm charts for deployment
- `deployments/metallb/` - LoadBalancer configuration
- `deployments/istio/local/` - Service mesh setup
- `scripts/setup-local-access.sh` - Unified setup automation

### **Configuration**
- `Makefile` - All automation commands
- `scripts/version.sh` - Semantic versioning management
- `go.mod` - Dependencies including Swagger tools

### **Documentation**
- `PROGRESS.md` - Project progress tracking
- `docs/LOCAL_ACCESS.md` - Local development guide
- `docs/swagger.json` - Generated OpenAPI specification

## 🔍 Debugging Tips

### **Common Issues**
1. **Compilation Errors**: Check import statements and type definitions
2. **Kubernetes Issues**: Verify namespace and service configurations
3. **Service Mesh Issues**: Check Istio gateway and virtual service configs
4. **LoadBalancer Issues**: Verify MetalLB IP pool and assignments

### **Quick Diagnostic Commands**
```bash
# Check service health
kubectl get pods -A | grep -E "(qlens|istio|metallb)"

# Check Istio configuration
kubectl get gateway,virtualservice -n istio-system

# Check LoadBalancer status
kubectl get svc istio-ingressgateway -n istio-system

# View recent logs
kubectl logs -f deployment/qlens-gateway -n qlens-staging --tail=50
```

## 📈 Success Metrics

- **Code Quality**: 80%+ test coverage, linting passes
- **Documentation**: All APIs documented with Swagger
- **Developer Experience**: One-command setup, no port-forwarding
- **Production Readiness**: Full observability, service mesh, automated versioning

---

**Last Updated**: 2025-09-06 by Claude Code session  
**Next Session**: Focus on fixing compilation errors and end-to-end testing