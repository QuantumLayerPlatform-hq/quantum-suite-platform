#!/bin/bash

# Quantum Suite Docker Image Build and Push Script
# This script builds all Quantum Suite Docker images and pushes them to a registry

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration - Update these for your environment
REGISTRY_URL="${DOCKER_REGISTRY:-docker.io}"
REGISTRY_NAMESPACE="${DOCKER_NAMESPACE:-quantum-suite}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
BUILD_CONTEXT="."

echo -e "${BLUE}🐳 Quantum Suite Docker Image Builder${NC}"
echo -e "${BLUE}Registry: ${REGISTRY_URL}/${REGISTRY_NAMESPACE}${NC}"
echo -e "${BLUE}Tag: ${IMAGE_TAG}${NC}"

# Function to build and push image
build_and_push() {
    local service_name=$1
    local dockerfile_path=$2
    local build_context=${3:-$BUILD_CONTEXT}
    
    local image_name="${REGISTRY_URL}/${REGISTRY_NAMESPACE}/${service_name}:${IMAGE_TAG}"
    
    echo -e "${BLUE}🔨 Building ${service_name}...${NC}"
    
    if [ -f "$dockerfile_path" ]; then
        docker build -t "$image_name" -f "$dockerfile_path" "$build_context"
        echo -e "${GREEN}✅ Built ${image_name}${NC}"
        
        echo -e "${BLUE}📤 Pushing ${service_name}...${NC}"
        docker push "$image_name"
        echo -e "${GREEN}✅ Pushed ${image_name}${NC}"
    else
        echo -e "${YELLOW}⚠️  Dockerfile not found: ${dockerfile_path}${NC}"
        echo -e "${YELLOW}📝 Creating placeholder Dockerfile for ${service_name}${NC}"
        
        # Create a basic Dockerfile template
        mkdir -p "$(dirname "$dockerfile_path")"
        cat > "$dockerfile_path" << EOF
# Dockerfile for ${service_name}
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/${service_name}

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Command to run
CMD ["./main"]
EOF
        
        echo -e "${GREEN}✅ Created ${dockerfile_path}${NC}"
        echo -e "${YELLOW}📝 You'll need to customize this Dockerfile for ${service_name}${NC}"
        
        # Build with the new Dockerfile
        docker build -t "$image_name" -f "$dockerfile_path" "$build_context"
        echo -e "${GREEN}✅ Built ${image_name} (placeholder)${NC}"
        
        echo -e "${BLUE}📤 Pushing ${service_name}...${NC}"
        docker push "$image_name"
        echo -e "${GREEN}✅ Pushed ${image_name} (placeholder)${NC}"
    fi
}

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running or not accessible${NC}"
    exit 1
fi

# Check registry authentication
if ! docker pull hello-world >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Docker registry authentication might be required${NC}"
    echo -e "${YELLOW}Run: docker login ${REGISTRY_URL}${NC}"
fi

# Build base image with common dependencies
echo -e "${BLUE}🏗️  Building base image...${NC}"
cat > Dockerfile.base << 'EOF'
FROM golang:1.21-alpine AS base

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy common internal packages
COPY internal/ ./internal/

# This will be extended by service-specific Dockerfiles
EOF

docker build -t "${REGISTRY_URL}/${REGISTRY_NAMESPACE}/base:${IMAGE_TAG}" -f Dockerfile.base .
docker push "${REGISTRY_URL}/${REGISTRY_NAMESPACE}/base:${IMAGE_TAG}"
echo -e "${GREEN}✅ Base image built and pushed${NC}"

# Build all service images
echo -e "${BLUE}🚀 Building Quantum Suite services...${NC}"

# Gateway
build_and_push "gateway" "cmd/gateway/Dockerfile"

# QAgent
build_and_push "qagent" "cmd/qagent/Dockerfile"

# QTest
build_and_push "qtest" "cmd/qtest/Dockerfile"

# QSecure
build_and_push "qsecure" "cmd/qsecure/Dockerfile"

# QSRE
build_and_push "qsre" "cmd/qsre/Dockerfile"

# QInfra
build_and_push "qinfra" "cmd/qinfra/Dockerfile"

# Create image manifest
echo -e "${BLUE}📋 Creating image manifest...${NC}"
cat > image-manifest.yaml << EOF
# Quantum Suite Image Manifest
# Generated on: $(date)

images:
  base:
    image: ${REGISTRY_URL}/${REGISTRY_NAMESPACE}/base:${IMAGE_TAG}
    size: $(docker images --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}" | grep "${REGISTRY_NAMESPACE}/base:${IMAGE_TAG}" | awk '{print $2}' || echo "unknown")
    
  gateway:
    image: ${REGISTRY_URL}/${REGISTRY_NAMESPACE}/gateway:${IMAGE_TAG}
    size: $(docker images --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}" | grep "${REGISTRY_NAMESPACE}/gateway:${IMAGE_TAG}" | awk '{print $2}' || echo "unknown")
    
  qagent:
    image: ${REGISTRY_URL}/${REGISTRY_NAMESPACE}/qagent:${IMAGE_TAG}
    size: $(docker images --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}" | grep "${REGISTRY_NAMESPACE}/qagent:${IMAGE_TAG}" | awk '{print $2}' || echo "unknown")
    
  qtest:
    image: ${REGISTRY_URL}/${REGISTRY_NAMESPACE}/qtest:${IMAGE_TAG}
    size: $(docker images --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}" | grep "${REGISTRY_NAMESPACE}/qtest:${IMAGE_TAG}" | awk '{print $2}' || echo "unknown")
    
  qsecure:
    image: ${REGISTRY_URL}/${REGISTRY_NAMESPACE}/qsecure:${IMAGE_TAG}
    size: $(docker images --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}" | grep "${REGISTRY_NAMESPACE}/qsecure:${IMAGE_TAG}" | awk '{print $2}' || echo "unknown")
    
  qsre:
    image: ${REGISTRY_URL}/${REGISTRY_NAMESPACE}/qsre:${IMAGE_TAG}
    size: $(docker images --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}" | grep "${REGISTRY_NAMESPACE}/qsre:${IMAGE_TAG}" | awk '{print $2}' || echo "unknown")
    
  qinfra:
    image: ${REGISTRY_URL}/${REGISTRY_NAMESPACE}/qinfra:${IMAGE_TAG}
    size: $(docker images --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}" | grep "${REGISTRY_NAMESPACE}/qinfra:${IMAGE_TAG}" | awk '{print $2}' || echo "unknown")

registry: ${REGISTRY_URL}
namespace: ${REGISTRY_NAMESPACE}
tag: ${IMAGE_TAG}
built_at: $(date)
EOF

# Display summary
echo -e "\n${GREEN}🎉 All images built and pushed successfully!${NC}"
echo -e "\n${BLUE}📋 Image Summary:${NC}"
docker images | grep "${REGISTRY_NAMESPACE}" | head -10

echo -e "\n${BLUE}📋 Next Steps:${NC}"
echo -e "  1. Update Kubernetes manifests with new image references"
echo -e "  2. Deploy to Kubernetes cluster"
echo -e "  3. Verify all services are running correctly"

echo -e "\n${BLUE}🔧 Update deployment images:${NC}"
echo -e "  sed -i 's|quantum-suite/|${REGISTRY_URL}/${REGISTRY_NAMESPACE}/|g' deployments/kubernetes/03-quantum-services.yaml"

# Cleanup
rm -f Dockerfile.base

echo -e "\n${GREEN}✨ Image build complete!${NC}"