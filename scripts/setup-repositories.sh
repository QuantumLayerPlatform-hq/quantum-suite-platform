#!/bin/bash

# Quantum Suite Repository Setup Script
# This script creates all repositories in the QuantumLayerPlatform-hq organization
# and pushes the initial codebase

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
ORG_NAME="QuantumLayerPlatform-hq"
GITHUB_USER="satish"  # Update this to your GitHub username

echo -e "${BLUE}🚀 Setting up Quantum Suite repositories in ${ORG_NAME}${NC}"

# Check if GitHub CLI is authenticated
if ! gh auth status >/dev/null 2>&1; then
    echo -e "${RED}❌ GitHub CLI is not authenticated. Please run 'gh auth login' first.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ GitHub CLI is authenticated${NC}"

# Function to create repository and push code
create_and_push_repo() {
    local repo_name=$1
    local description=$2
    local directory=$3
    
    echo -e "${YELLOW}📦 Creating repository: ${repo_name}${NC}"
    
    # Create repository if it doesn't exist
    if ! gh repo view "${ORG_NAME}/${repo_name}" >/dev/null 2>&1; then
        gh repo create "${ORG_NAME}/${repo_name}" --public --description "${description}"
        echo -e "${GREEN}✅ Repository ${repo_name} created${NC}"
    else
        echo -e "${YELLOW}⚠️  Repository ${repo_name} already exists${NC}"
    fi
    
    # Create local directory structure
    if [ ! -d "$directory" ]; then
        mkdir -p "$directory"
    fi
    
    cd "$directory"
    
    # Initialize git if not already done
    if [ ! -d ".git" ]; then
        git init
        git remote add origin "https://github.com/${ORG_NAME}/${repo_name}.git"
    fi
    
    # Add files and commit
    git add .
    if git diff --staged --quiet; then
        echo -e "${YELLOW}⚠️  No changes to commit in ${repo_name}${NC}"
    else
        git commit -m "Initial commit: Setup ${repo_name} with comprehensive architecture and documentation"
        git branch -M main
        git push -u origin main
        echo -e "${GREEN}✅ Code pushed to ${repo_name}${NC}"
    fi
    
    cd - > /dev/null
}

# Create main platform repository
echo -e "${BLUE}🏗️  Setting up main platform repository${NC}"
create_and_push_repo "quantum-suite-platform" "Core platform infrastructure and shared services" "."

# Create temporary directories for other repositories
TEMP_DIR=$(mktemp -d)
echo -e "${BLUE}📁 Working in temporary directory: ${TEMP_DIR}${NC}"

# Create QAgent repository
echo -e "${BLUE}🤖 Setting up QAgent repository${NC}"
QAGENT_DIR="${TEMP_DIR}/qagent"
mkdir -p "${QAGENT_DIR}"

# Copy QAgent specific files
cp -r cmd/qagent "${QAGENT_DIR}/cmd/" 2>/dev/null || mkdir -p "${QAGENT_DIR}/cmd/qagent"
cp migrations/002_qagent_schema.sql "${QAGENT_DIR}/qagent_schema.sql" 2>/dev/null || true

# Create QAgent README
cat > "${QAGENT_DIR}/README.md" << 'EOF'
# QAgent - AI-Powered Code Generation

QAgent is the AI-powered code generation module of the Quantum Suite platform. It provides intelligent code generation from natural language descriptions with self-criticism loops and meta-prompt optimization.

## Features

- 🧠 **Advanced NLP**: Sophisticated intent recognition and context understanding
- 📝 **Meta-Prompt Engineering**: Dynamic prompt optimization and chaining
- 🔄 **Self-Criticism Loop**: Automated code quality improvement
- 🌐 **Multi-Language Support**: Python, Go, JavaScript, TypeScript, and more
- 🎯 **Context-Aware**: Leverages project context and existing codebase
- 🔍 **Tree-Sitter Validation**: Real-time syntax and semantic validation

## Quick Start

```bash
# Build QAgent
make build

# Run QAgent service
./bin/qagent

# Generate code via API
curl -X POST http://localhost:8110/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Create a REST API for user management",
    "language": "go",
    "framework": "gin"
  }'
```

## Documentation

- [Architecture Overview](docs/architecture.md)
- [API Reference](docs/api.md)
- [Development Guide](docs/development.md)

## License

MIT License - see [LICENSE](LICENSE) for details.
EOF

# Create QAgent go.mod
cat > "${QAGENT_DIR}/go.mod" << 'EOF'
module github.com/QuantumLayerPlatform-hq/qagent

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/QuantumLayerPlatform-hq/quantum-suite-platform v0.1.0
)
EOF

create_and_push_repo "qagent" "AI-powered code generation module" "${QAGENT_DIR}"

# Create QTest repository
echo -e "${BLUE}🧪 Setting up QTest repository${NC}"
QTEST_DIR="${TEMP_DIR}/qtest"
mkdir -p "${QTEST_DIR}"

cat > "${QTEST_DIR}/README.md" << 'EOF'
# QTest - Intelligent Testing Framework

QTest provides intelligent test generation, coverage analysis, and quality assurance for the Quantum Suite platform.

## Features

- 🎯 **Automated Test Generation**: Unit, integration, and E2E test creation
- 📊 **Coverage Analysis**: Comprehensive code coverage reporting
- 🔄 **Mutation Testing**: Quality validation through mutation analysis
- ⚡ **Performance Testing**: Load and stress test generation
- 📈 **Analytics**: Detailed testing metrics and insights

## Quick Start

```bash
# Generate tests for a file
qtest generate --file main.go --type unit

# Analyze coverage
qtest coverage --project ./

# Run performance tests
qtest perf --target http://localhost:8080
```

## Documentation

See the [Quantum Suite Documentation](https://github.com/QuantumLayerPlatform-hq/quantum-docs) for detailed information.
EOF

cat > "${QTEST_DIR}/go.mod" << 'EOF'
module github.com/QuantumLayerPlatform-hq/qtest

go 1.21

require (
    github.com/QuantumLayerPlatform-hq/quantum-suite-platform v0.1.0
)
EOF

create_and_push_repo "qtest" "Intelligent testing and coverage analysis" "${QTEST_DIR}"

# Create QSecure repository
echo -e "${BLUE}🔒 Setting up QSecure repository${NC}"
QSECURE_DIR="${TEMP_DIR}/qsecure"
mkdir -p "${QSECURE_DIR}"

cat > "${QSECURE_DIR}/README.md" << 'EOF'
# QSecure - Security Operations Center

QSecure provides comprehensive security scanning, vulnerability management, and compliance automation for the Quantum Suite platform.

## Features

- 🔍 **Multi-Scanner Support**: SAST, DAST, SCA, and container scanning
- 🛡️ **Vulnerability Management**: Automated detection and remediation
- 📋 **Compliance Automation**: SOC2, ISO27001, HIPAA, PCI-DSS frameworks
- 🚨 **Real-time Monitoring**: Continuous security posture assessment
- 🔧 **Auto-Remediation**: Intelligent security issue resolution

## Quick Start

```bash
# Scan code repository
qsecure scan --type sast --target ./

# Check compliance
qsecure compliance --framework soc2

# Generate security report
qsecure report --format pdf --output security-report.pdf
```

## Documentation

See the [Quantum Suite Documentation](https://github.com/QuantumLayerPlatform-hq/quantum-docs) for detailed information.
EOF

create_and_push_repo "qsecure" "Security scanning and vulnerability management" "${QSECURE_DIR}"

# Create QSRE repository
echo -e "${BLUE}📊 Setting up QSRE repository${NC}"
QSRE_DIR="${TEMP_DIR}/qsre"
mkdir -p "${QSRE_DIR}"

cat > "${QSRE_DIR}/README.md" << 'EOF'
# QSRE - Site Reliability Engineering

QSRE provides intelligent monitoring, incident management, and reliability engineering for the Quantum Suite platform.

## Features

- 📊 **Smart Monitoring**: AI-powered anomaly detection
- 🚨 **Incident Management**: Automated response and escalation
- 🎭 **Chaos Engineering**: Resilience testing and validation
- 📈 **SLO Management**: Service level objective tracking
- 🤖 **Runbook Automation**: Intelligent operational procedures

## Quick Start

```bash
# Setup monitoring for a service
qsre monitor --service api-gateway --slo 99.9

# Run chaos experiment
qsre chaos --experiment pod-kill --target frontend

# Check SLO status
qsre slo --service all --period 30d
```

## Documentation

See the [Quantum Suite Documentation](https://github.com/QuantumLayerPlatform-hq/quantum-docs) for detailed information.
EOF

create_and_push_repo "qsre" "Site reliability engineering and monitoring" "${QSRE_DIR}"

# Create QInfra repository
echo -e "${BLUE}☁️ Setting up QInfra repository${NC}"
QINFRA_DIR="${TEMP_DIR}/qinfra"
mkdir -p "${QINFRA_DIR}"

cat > "${QINFRA_DIR}/README.md" << 'EOF'
# QInfra - Multi-Cloud Infrastructure Orchestration

QInfra provides intelligent infrastructure management, golden image creation, and disaster recovery automation across multiple cloud providers.

## Features

- ☁️ **Multi-Cloud Support**: AWS, Azure, GCP, and hybrid deployments
- 🖼️ **Golden Images**: Automated AMI/image creation and management
- 🔄 **Disaster Recovery**: Automated DR setup and testing
- 📏 **Compliance**: Built-in compliance policies and enforcement
- 💰 **Cost Optimization**: Intelligent resource management and cost control

## Quick Start

```bash
# Provision infrastructure
qinfra provision --provider aws --template webapp --region us-west-2

# Create golden image
qinfra image create --base ubuntu-22.04 --name web-server-v1

# Setup disaster recovery
qinfra dr setup --primary us-west-2 --backup us-east-1
```

## Documentation

See the [Quantum Suite Documentation](https://github.com/QuantumLayerPlatform-hq/quantum-docs) for detailed information.
EOF

create_and_push_repo "qinfra" "Multi-cloud infrastructure orchestration" "${QINFRA_DIR}"

# Create Documentation repository
echo -e "${BLUE}📚 Setting up Documentation repository${NC}"
DOCS_DIR="${TEMP_DIR}/quantum-docs"
mkdir -p "${DOCS_DIR}"

# Copy documentation files
cp -r docs/* "${DOCS_DIR}/" 2>/dev/null || mkdir -p "${DOCS_DIR}/placeholder"

cat > "${DOCS_DIR}/README.md" << 'EOF'
# Quantum Suite Documentation

Comprehensive documentation for the Quantum Suite DevSecOps platform.

## 📋 Table of Contents

- [Architecture Overview](architecture/system-overview.md)
- [Getting Started](getting-started/README.md)
- [API Reference](api/README.md)
- [Deployment Guide](deployment/README.md)
- [Monitoring Guide](monitoring/README.md)

## 🏗️ Architecture

- [System Overview](architecture/system-overview.md)
- [Component Architecture](architecture/components.md)
- [Data Flow](architecture/data-flow.md)
- [Security Architecture](architecture/security.md)

## 🚀 Quick Start

1. [Installation Guide](getting-started/installation.md)
2. [Configuration](getting-started/configuration.md)
3. [First Steps](getting-started/first-steps.md)

## 📊 Monitoring & Operations

- [Metrics and KPIs](monitoring/metrics-and-monitoring.md)
- [Alerting Setup](monitoring/alerting.md)
- [Incident Response](monitoring/incident-response.md)

## 🤝 Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
EOF

create_and_push_repo "quantum-docs" "Comprehensive platform documentation" "${DOCS_DIR}"

# Create CLI repository
echo -e "${BLUE}⚡ Setting up CLI repository${NC}"
CLI_DIR="${TEMP_DIR}/quantum-cli"
mkdir -p "${CLI_DIR}"

cat > "${CLI_DIR}/README.md" << 'EOF'
# Quantum CLI

Command-line interface for the Quantum Suite platform.

## Installation

```bash
# Install via Go
go install github.com/QuantumLayerPlatform-hq/quantum-cli@latest

# Install via curl (Linux/macOS)
curl -sSL https://cli.quantum-suite.io/install.sh | bash

# Install via Homebrew (macOS)
brew install quantum-suite/tap/quantum
```

## Quick Start

```bash
# Login to Quantum Suite
quantum auth login

# Generate code
quantum agent generate "Create a REST API for user management"

# Run tests
quantum test generate --file main.go

# Deploy infrastructure
quantum infra deploy --env production

# Check system status
quantum status
```

## Commands

- `quantum agent` - AI code generation commands
- `quantum test` - Testing and quality assurance
- `quantum secure` - Security scanning and compliance
- `quantum sre` - Monitoring and reliability
- `quantum infra` - Infrastructure management

## Documentation

See the [full documentation](https://docs.quantum-suite.io/cli) for detailed command reference.
EOF

create_and_push_repo "quantum-cli" "Command-line interface for Quantum Suite" "${CLI_DIR}"

# Cleanup
echo -e "${BLUE}🧹 Cleaning up temporary files${NC}"
rm -rf "${TEMP_DIR}"

echo -e "${GREEN}🎉 Successfully set up all Quantum Suite repositories in ${ORG_NAME}!${NC}"
echo -e "${BLUE}📋 Repository Summary:${NC}"
echo -e "  • quantum-suite-platform: Core platform and shared services"
echo -e "  • qagent: AI-powered code generation"
echo -e "  • qtest: Intelligent testing framework"
echo -e "  • qsecure: Security operations center"
echo -e "  • qsre: Site reliability engineering"
echo -e "  • qinfra: Multi-cloud infrastructure"
echo -e "  • quantum-docs: Comprehensive documentation"
echo -e "  • quantum-cli: Command-line interface"

echo -e "${YELLOW}🔗 Next steps:${NC}"
echo -e "  1. Visit https://github.com/${ORG_NAME} to see your repositories"
echo -e "  2. Configure branch protection rules"
echo -e "  3. Set up CI/CD workflows"
echo -e "  4. Add team members and permissions"
echo -e "  5. Create first release tags"

echo -e "${GREEN}✨ Quantum Suite is ready for development!${NC}"