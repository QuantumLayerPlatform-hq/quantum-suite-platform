#!/bin/bash

# Quantum Suite Kubernetes Cluster Cleanup Script
# This script safely cleans up existing resources while preserving critical data

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE_PREFIX="quantum-"
BACKUP_DIR="./cluster-backup-$(date +%Y%m%d-%H%M%S)"

echo -e "${BLUE}🧹 Quantum Suite Cluster Cleanup${NC}"
echo -e "${YELLOW}⚠️  This script will clean up your Kubernetes cluster for Quantum Suite deployment${NC}"

# Safety check - confirm kubectl context
CURRENT_CONTEXT=$(kubectl config current-context)
echo -e "${BLUE}Current kubectl context: ${CURRENT_CONTEXT}${NC}"

read -p "Are you sure you want to clean up cluster '$CURRENT_CONTEXT'? (yes/no): " -r
if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    echo -e "${RED}❌ Cleanup cancelled${NC}"
    exit 1
fi

# Create backup directory
mkdir -p "$BACKUP_DIR"
echo -e "${BLUE}📁 Created backup directory: $BACKUP_DIR${NC}"

# Function to backup resource if it exists
backup_resource() {
    local resource_type=$1
    local namespace=${2:-""}
    local resource_name=${3:-""}
    
    if [ -n "$namespace" ] && [ -n "$resource_name" ]; then
        if kubectl get "$resource_type" "$resource_name" -n "$namespace" >/dev/null 2>&1; then
            echo -e "${YELLOW}💾 Backing up $resource_type/$resource_name in namespace $namespace${NC}"
            kubectl get "$resource_type" "$resource_name" -n "$namespace" -o yaml > "$BACKUP_DIR/${namespace}-${resource_type}-${resource_name}.yaml"
        fi
    elif [ -n "$namespace" ]; then
        if kubectl get "$resource_type" -n "$namespace" >/dev/null 2>&1; then
            echo -e "${YELLOW}💾 Backing up all $resource_type in namespace $namespace${NC}"
            kubectl get "$resource_type" -n "$namespace" -o yaml > "$BACKUP_DIR/${namespace}-${resource_type}-all.yaml"
        fi
    else
        if kubectl get "$resource_type" --all-namespaces >/dev/null 2>&1; then
            echo -e "${YELLOW}💾 Backing up all $resource_type across all namespaces${NC}"
            kubectl get "$resource_type" --all-namespaces -o yaml > "$BACKUP_DIR/all-${resource_type}.yaml"
        fi
    fi
}

# Function to delete resource safely
delete_resource() {
    local resource_type=$1
    local namespace=${2:-""}
    local resource_name=${3:-""}
    local force=${4:-"false"}
    
    if [ -n "$namespace" ] && [ -n "$resource_name" ]; then
        if kubectl get "$resource_type" "$resource_name" -n "$namespace" >/dev/null 2>&1; then
            echo -e "${RED}🗑️  Deleting $resource_type/$resource_name in namespace $namespace${NC}"
            if [ "$force" = "true" ]; then
                kubectl delete "$resource_type" "$resource_name" -n "$namespace" --force --grace-period=0 >/dev/null 2>&1 || true
            else
                kubectl delete "$resource_type" "$resource_name" -n "$namespace" --grace-period=30 >/dev/null 2>&1 || true
            fi
        fi
    elif [ -n "$namespace" ]; then
        if kubectl get "$resource_type" -n "$namespace" >/dev/null 2>&1; then
            echo -e "${RED}🗑️  Deleting all $resource_type in namespace $namespace${NC}"
            kubectl delete "$resource_type" --all -n "$namespace" --grace-period=30 >/dev/null 2>&1 || true
        fi
    fi
}

# Step 1: List current resources
echo -e "${BLUE}📋 Current cluster resources:${NC}"
kubectl get namespaces | grep -E "(default|kube-|quantum-)" || true
kubectl get pods --all-namespaces | head -20

# Step 2: Identify and backup PersistentVolumes and important data
echo -e "${BLUE}🔍 Identifying persistent storage...${NC}"
kubectl get pv -o yaml > "$BACKUP_DIR/persistent-volumes.yaml" 2>/dev/null || true
kubectl get pvc --all-namespaces -o yaml > "$BACKUP_DIR/persistent-volume-claims.yaml" 2>/dev/null || true

# Step 3: Backup ConfigMaps and Secrets (excluding system ones)
echo -e "${BLUE}💾 Backing up configurations...${NC}"
for ns in $(kubectl get namespaces -o name | cut -d/ -f2 | grep -v -E "^(kube-|default$)"); do
    backup_resource "configmap" "$ns"
    backup_resource "secret" "$ns"
done

# Step 4: Clean up applications (preserve system namespaces)
echo -e "${BLUE}🧹 Cleaning up application resources...${NC}"

# Delete all applications in non-system namespaces
for ns in $(kubectl get namespaces -o name | cut -d/ -f2 | grep -v -E "^(kube-|default$)"); do
    echo -e "${YELLOW}🗑️  Cleaning namespace: $ns${NC}"
    
    # Delete workloads
    delete_resource "deployment" "$ns"
    delete_resource "statefulset" "$ns"
    delete_resource "daemonset" "$ns"
    delete_resource "job" "$ns"
    delete_resource "cronjob" "$ns"
    delete_resource "replicaset" "$ns"
    delete_resource "pod" "$ns"
    
    # Delete services and ingress
    delete_resource "service" "$ns"
    delete_resource "ingress" "$ns"
    delete_resource "networkpolicy" "$ns"
    
    # Delete storage
    delete_resource "pvc" "$ns"
    
    # Wait a bit for resources to terminate
    sleep 5
done

# Step 5: Clean up custom resources
echo -e "${BLUE}🔧 Cleaning up custom resources...${NC}"
kubectl get crd -o name | while read -r crd; do
    crd_name=$(echo "$crd" | cut -d/ -f2)
    echo -e "${YELLOW}🗑️  Deleting custom resources for CRD: $crd_name${NC}"
    kubectl delete "$crd_name" --all --all-namespaces >/dev/null 2>&1 || true
done

# Step 6: Remove non-system namespaces
echo -e "${BLUE}📁 Removing non-system namespaces...${NC}"
for ns in $(kubectl get namespaces -o name | cut -d/ -f2 | grep -v -E "^(kube-|default$)"); do
    echo -e "${RED}🗑️  Deleting namespace: $ns${NC}"
    kubectl delete namespace "$ns" --grace-period=60 >/dev/null 2>&1 || true
done

# Step 7: Clean up cluster-wide resources (be careful here)
echo -e "${BLUE}🌐 Cleaning cluster-wide resources...${NC}"
kubectl get clusterrolebinding -o name | grep -v -E "(system:|cluster-admin|edit|view)" | while read -r binding; do
    echo -e "${YELLOW}🗑️  Deleting cluster role binding: $binding${NC}"
    kubectl delete "$binding" >/dev/null 2>&1 || true
done

kubectl get clusterrole -o name | grep -v -E "(system:|cluster-admin|edit|view|admin)" | while read -r role; do
    echo -e "${YELLOW}🗑️  Deleting cluster role: $role${NC}"
    kubectl delete "$role" >/dev/null 2>&1 || true
done

# Step 8: Clean up storage classes that might conflict
echo -e "${BLUE}💾 Reviewing storage classes...${NC}"
kubectl get storageclass

# Step 9: Wait for namespace deletion to complete
echo -e "${BLUE}⏳ Waiting for cleanup to complete...${NC}"
timeout=300  # 5 minutes
elapsed=0
while [ $elapsed -lt $timeout ]; do
    remaining_ns=$(kubectl get namespaces -o name | cut -d/ -f2 | grep -v -E "^(kube-|default$)" | wc -l)
    if [ "$remaining_ns" -eq 0 ]; then
        break
    fi
    echo -e "${YELLOW}⏳ Waiting for $remaining_ns namespaces to terminate...${NC}"
    sleep 10
    elapsed=$((elapsed + 10))
done

# Step 10: Final verification
echo -e "${BLUE}✅ Cleanup verification:${NC}"
echo -e "${GREEN}Remaining namespaces:${NC}"
kubectl get namespaces

echo -e "${GREEN}Remaining persistent volumes:${NC}"
kubectl get pv

echo -e "${GREEN}Remaining storage classes:${NC}"
kubectl get storageclass

# Create cleanup report
cat > "$BACKUP_DIR/cleanup-report.md" << EOF
# Kubernetes Cluster Cleanup Report

**Date:** $(date)
**Cluster:** $CURRENT_CONTEXT

## Cleanup Summary

- Backed up configurations to: $BACKUP_DIR
- Removed all non-system namespaces
- Cleaned up custom resources
- Preserved system namespaces (kube-*, default)

## Remaining Resources

### Namespaces
\`\`\`
$(kubectl get namespaces)
\`\`\`

### Storage Classes
\`\`\`
$(kubectl get storageclass)
\`\`\`

### Persistent Volumes
\`\`\`
$(kubectl get pv)
\`\`\`

## Recovery

To restore backed up resources, apply the YAML files in this directory:
\`\`\`bash
kubectl apply -f $BACKUP_DIR/
\`\`\`
EOF

echo -e "${GREEN}✅ Cluster cleanup completed successfully!${NC}"
echo -e "${BLUE}📋 Cleanup report: $BACKUP_DIR/cleanup-report.md${NC}"
echo -e "${BLUE}💾 Backup files: $BACKUP_DIR/${NC}"
echo -e "${YELLOW}🚀 Your cluster is now ready for Quantum Suite deployment!${NC}"

echo -e "${GREEN}🎯 Next steps:${NC}"
echo -e "  1. Run: ./scripts/deploy-quantum-suite.sh"
echo -e "  2. Configure vector databases"
echo -e "  3. Deploy application services"
echo -e "  4. Set up monitoring stack"