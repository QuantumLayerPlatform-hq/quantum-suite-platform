# QLens Development Best Practices

This document outlines best practices for maintaining momentum and context across Claude Code sessions.

## 📋 Session Management Best Practices

### **Starting a New Session**

1. **Always run session-start script:**
   ```bash
   make session-start
   # or directly: ./scripts/session-start.sh
   ```

2. **Read context files in order:**
   - `PROGRESS.md` - Overall project status
   - `CLAUDE.md` - Session-specific context
   - Recent git log - Latest changes

3. **Verify environment:**
   - Kubernetes cluster access
   - Required tools (kubectl, helm, istioctl)
   - Service status

### **During Development Session**

1. **Track progress actively:**
   - Use TodoWrite tool for current tasks
   - Update PROGRESS.md for major milestones
   - Commit frequently with descriptive messages

2. **Document decisions:**
   - Architecture changes in relevant docs
   - Configuration changes in comments
   - API changes in swagger annotations

3. **Maintain build health:**
   - Run `make build` regularly
   - Fix compilation issues immediately
   - Keep test coverage high

### **Ending a Session**

1. **Always run session-end script:**
   ```bash
   make session-end
   # or directly: ./scripts/session-end.sh
   ```

2. **Update context:**
   - Document accomplishments
   - Note any blockers
   - Set next session priorities

3. **Clean commit:**
   - Commit all meaningful changes
   - Use descriptive commit messages
   - Include co-authorship attribution

## 🏗️ Development Workflow

### **Feature Development Cycle**

1. **Start with planning:**
   ```bash
   make session-start          # Get context
   make get-access-info        # Check current state
   make dev-status            # Verify services
   ```

2. **Implement changes:**
   ```bash
   make build                 # Verify compilation
   make test                  # Run tests
   make lint                  # Check code quality
   ```

3. **Test integration:**
   ```bash
   make dev-up                # Start services
   make docs                  # Update documentation
   # Test manually via Swagger UI
   ```

4. **Complete feature:**
   ```bash
   make version-patch         # Increment version
   make session-end           # Document and clean up
   ```

### **Bug Fix Workflow**

1. **Reproduce issue:**
   - Check logs: `make dev-logs`
   - Review service status: `make dev-status`
   - Test affected endpoints

2. **Fix and verify:**
   - Make minimal changes
   - Add/update tests
   - Verify fix works end-to-end

3. **Deploy and monitor:**
   - Deploy to staging
   - Monitor for regressions
   - Update documentation if needed

## 🎯 Momentum Maintenance

### **Cross-Session Context**

1. **PROGRESS.md Structure:**
   - Keep current status section updated
   - Document completed components clearly
   - Maintain accurate blockers list
   - Update metrics regularly

2. **CLAUDE.md Usage:**
   - Always update current context
   - Note immediate priorities
   - Document any discovered issues
   - Keep command references current

3. **Git Hygiene:**
   - Commit frequently (every 15-30 minutes)
   - Use consistent commit message format
   - Include co-authorship for Claude sessions
   - Tag releases with semantic versions

### **Knowledge Capture**

1. **Architecture Decisions:**
   - Document in dedicated ADR files
   - Update diagrams when changes occur
   - Explain rationale for future reference

2. **Configuration Changes:**
   - Comment complex configurations
   - Document environment differences
   - Maintain deployment guides

3. **API Evolution:**
   - Keep Swagger docs current
   - Version API changes appropriately
   - Maintain backward compatibility

## 🔧 Troubleshooting Guidelines

### **Common Session Start Issues**

1. **Compilation Errors:**
   ```bash
   go build ./...             # Check specific errors
   make lint                  # Verify code quality
   ```

2. **Service Access Issues:**
   ```bash
   kubectl get pods -A        # Check all services
   make get-access-info       # Verify LoadBalancer IP
   ```

3. **Environment Problems:**
   ```bash
   kubectl cluster-info       # Verify cluster access
   make install-tools         # Reinstall if needed
   ```

### **Recovery Strategies**

1. **Lost Context:**
   - Read PROGRESS.md completely
   - Review last 10 git commits
   - Run session-start script

2. **Broken Services:**
   - Check PROGRESS.md for last known good state
   - Review recent commits for breaking changes
   - Restart from clean state: `make dev-down && make dev-up`

3. **Unclear Priorities:**
   - Review CLAUDE.md next session priorities
   - Check PROGRESS.md backlog
   - Focus on compilation issues first

## 📊 Quality Metrics

### **Code Quality Targets**

- **Test Coverage:** > 80%
- **Build Success:** 100% (zero tolerance for compilation errors)
- **Documentation Coverage:** All public APIs documented
- **Linting:** Zero warnings/errors

### **Process Metrics**

- **Session Start Time:** < 5 minutes to full context
- **Build Time:** < 2 minutes for full rebuild  
- **Test Suite:** < 5 minutes for full test run
- **Documentation Generation:** < 30 seconds

### **Deployment Metrics**

- **Staging Deployment:** < 5 minutes end-to-end
- **Service Startup:** < 2 minutes all services ready
- **Health Check Response:** < 500ms
- **API Response Time:** p95 < 2 seconds

## 🚀 Advanced Practices

### **Performance Optimization**

1. **Development Loop:**
   - Keep services running between sessions
   - Use incremental builds when possible
   - Cache dependencies aggressively

2. **Resource Management:**
   - Monitor resource usage in development
   - Right-size containers for development
   - Use development-optimized configurations

### **Collaboration Patterns**

1. **Handoff Documentation:**
   - Always update CLAUDE.md before ending
   - Include specific next steps
   - Note any gotchas or discoveries

2. **Code Review Preparation:**
   - Maintain clean commit history
   - Document reasoning in commit messages
   - Include tests for all changes

3. **Issue Tracking:**
   - Use PROGRESS.md for high-level tracking
   - Create GitHub issues for complex bugs
   - Link commits to issues when relevant

## 🎯 Success Indicators

### **Good Session Start:**
- Context loaded in < 5 minutes
- All services accessible
- Clear understanding of priorities
- No blocking compilation errors

### **Productive Session:**
- Multiple meaningful commits
- Forward progress on priorities
- Documentation kept current
- Tests passing

### **Clean Session End:**
- All changes committed
- Context documented for next session
- Services left in known good state
- Clear priorities set for next time

---

**Remember:** The goal is sustainable development velocity across multiple sessions. Invest time in good practices to compound productivity over time.