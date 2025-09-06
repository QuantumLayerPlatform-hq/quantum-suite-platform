# Contributing to Quantum Suite

Thank you for your interest in contributing to Quantum Suite! We welcome contributions from the community and are pleased to have them.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Process](#contributing-process)
- [Pull Request Guidelines](#pull-request-guidelines)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)

## Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally
3. **Create a feature branch** from `develop`
4. **Make your changes**
5. **Test your changes**
6. **Submit a pull request**

## Development Setup

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Kubernetes (optional, for full testing)
- PostgreSQL 15+
- Redis 7+

### Setup Instructions

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/quantum-suite-platform.git
cd quantum-suite-platform

# Set up development environment
make dev-setup

# Start development services
make dev-up

# Run tests
make test

# Build the project
make build
```

## Contributing Process

### 1. Choose an Issue

- Look for issues labeled `good first issue` for beginners
- Check the project roadmap for priority items
- Create an issue for new features or bugs

### 2. Create a Branch

```bash
# Create and switch to a new branch
git checkout -b feature/your-feature-name

# Or for bug fixes
git checkout -b fix/issue-description
```

### 3. Make Changes

- Write clean, maintainable code
- Follow our coding standards
- Add tests for new functionality
- Update documentation as needed

### 4. Test Your Changes

```bash
# Run all tests
make test

# Run linting
make lint

# Run security scan
make security-scan

# Test locally
make dev-up
```

## Pull Request Guidelines

### Before Submitting

- [ ] Code follows project style guidelines
- [ ] Tests pass locally
- [ ] Documentation is updated
- [ ] Commit messages are clear and descriptive
- [ ] Branch is up to date with `develop`

### PR Template

```markdown
## Description
Brief description of changes made.

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests pass
- [ ] Manual testing completed

## Documentation
- [ ] README updated
- [ ] API documentation updated
- [ ] Architecture docs updated

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Tests added for new functionality
- [ ] All tests pass
```

## Coding Standards

### Go Guidelines

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use meaningful variable and function names
- Add comments for exported functions and types

### Code Structure

```go
// Package comment
package main

import (
    // Standard library
    "context"
    "fmt"
    
    // Third party
    "github.com/gin-gonic/gin"
    
    // Internal
    "github.com/QuantumLayerPlatform-hq/quantum-suite-platform/pkg/shared"
)
```

### Error Handling

```go
// Good error handling
func processData(data string) (*Result, error) {
    if data == "" {
        return nil, errors.New("data cannot be empty")
    }
    
    result, err := someOperation(data)
    if err != nil {
        return nil, fmt.Errorf("failed to process data: %w", err)
    }
    
    return result, nil
}
```

## Testing Guidelines

### Test Structure

```go
func TestFunctionName(t *testing.T) {
    // Arrange
    input := "test input"
    expected := "expected output"
    
    // Act
    result, err := FunctionName(input)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### Test Categories

- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test component interactions
- **E2E Tests**: Test complete user workflows
- **Performance Tests**: Test system performance and scalability

### Testing Best Practices

- Use table-driven tests for multiple scenarios
- Test both success and error cases
- Use meaningful test names
- Keep tests independent and deterministic
- Mock external dependencies

## Documentation

### Code Documentation

- Document all exported functions and types
- Include usage examples for complex functions
- Keep comments up to date with code changes

### API Documentation

- Use OpenAPI 3.0 for REST API documentation
- Include request/response examples
- Document error responses

### Architecture Documentation

- Update architecture diagrams when making structural changes
- Document design decisions and trade-offs
- Keep deployment documentation current

## Security Guidelines

- Never commit secrets or API keys
- Use environment variables for configuration
- Follow security best practices
- Report security issues privately

## Performance Considerations

- Profile code for performance bottlenecks
- Use appropriate data structures and algorithms
- Consider memory usage and garbage collection
- Load test significant changes

## Release Process

1. **Feature Complete**: All features for the release are implemented
2. **Testing**: Comprehensive testing across all environments
3. **Documentation**: All documentation is updated
4. **Release Notes**: Detailed changelog is prepared
5. **Deployment**: Staged rollout to production

## Getting Help

- **Documentation**: Check our [documentation](https://docs.quantum-suite.io)
- **Discussions**: Join our [GitHub Discussions](https://github.com/orgs/QuantumLayerPlatform-hq/discussions)
- **Discord**: Join our [community Discord](https://discord.gg/quantum-suite)
- **Issues**: Create an issue for bugs or feature requests

## Recognition

Contributors will be recognized in:
- Repository contributors list
- Release notes
- Community highlights

Thank you for contributing to Quantum Suite! 🚀