#!/bin/bash

# Quantum Suite Cluster Management Script
# This script provides common management operations for the Quantum Suite cluster

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE_SYSTEM="quantum-system"
NAMESPACE_DATA="quantum-data"
NAMESPACE_SERVICES="quantum-services"
NAMESPACE_MONITORING="quantum-monitoring"

show_usage() {
    echo -e "${BLUE}Quantum Suite Cluster Management${NC}"
    echo -e "Usage: $0 [COMMAND]"
    echo -e ""
    echo -e "${BLUE}Commands:${NC}"
    echo -e "  status       - Show overall cluster status"
    echo -e "  logs         - Show logs from all services"
    echo -e "  restart      - Restart all Quantum Suite services"
    echo -e "  scale        - Scale services up or down"
    echo -e "  backup       - Create a backup of persistent data"
    echo -e "  restore      - Restore from backup"
    echo -e "  update       - Update service images"
    echo -e "  port-forward - Set up port forwarding for development"
    echo -e "  cleanup      - Clean up failed resources"
    echo -e "  dashboard    - Open monitoring dashboards"
    echo -e "  secrets      - Manage secrets and configuration"
    echo -e "  help         - Show this help message"
    echo -e ""
    echo -e "${BLUE}Examples:${NC}"
    echo -e "  $0 status"
    echo -e "  $0 logs qagent"
    echo -e "  $0 scale qagent 5"
    echo -e "  $0 port-forward"
}

show_status() {
    echo -e "${BLUE}🔍 Quantum Suite Cluster Status${NC}\n"
    
    # Cluster info
    echo -e "${BLUE}Cluster Information:${NC}"
    kubectl cluster-info | head -3
    echo ""
    
    # Node status
    echo -e "${BLUE}Node Status:${NC}"
    kubectl get nodes -o wide
    echo ""
    
    # Namespace overview
    echo -e "${BLUE}Namespace Overview:${NC}"
    for ns in $NAMESPACE_SYSTEM $NAMESPACE_DATA $NAMESPACE_SERVICES $NAMESPACE_MONITORING; do
        pod_count=$(kubectl get pods -n $ns --no-headers 2>/dev/null | wc -l)
        running_count=$(kubectl get pods -n $ns --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
        echo -e "  ${GREEN}$ns${NC}: $running_count/$pod_count pods running"
    done
    echo ""
    
    # Service status
    echo -e "${BLUE}Service Status:${NC}"
    kubectl get pods -n $NAMESPACE_SERVICES -o custom-columns=NAME:.metadata.name,STATUS:.status.phase,READY:.status.containerStatuses[0].ready,RESTARTS:.status.containerStatuses[0].restartCount
    echo ""
    
    # Resource usage
    echo -e "${BLUE}Resource Usage:${NC}"
    kubectl top nodes 2>/dev/null || echo "Metrics server not available"
    echo ""
    
    # Storage usage
    echo -e "${BLUE}Storage Status:${NC}"
    kubectl get pv,pvc --all-namespaces
}

show_logs() {
    local service=${1:-"all"}
    local lines=${2:-50}
    
    if [ "$service" = "all" ]; then
        echo -e "${BLUE}📋 Showing logs from all Quantum Suite services (last $lines lines)${NC}\n"
        for svc in quantum-gateway qagent qtest qsecure qsre qinfra; do
            echo -e "${YELLOW}=== $svc ===${NC}"
            kubectl logs -n $NAMESPACE_SERVICES deployment/$svc --tail=$lines 2>/dev/null || echo "No logs available"
            echo ""
        done
    else
        echo -e "${BLUE}📋 Showing logs from $service (last $lines lines)${NC}"
        kubectl logs -n $NAMESPACE_SERVICES deployment/$service --tail=$lines --follow
    fi
}

restart_services() {
    echo -e "${BLUE}🔄 Restarting Quantum Suite services${NC}"
    
    for svc in quantum-gateway qagent qtest qsecure qsre qinfra; do
        echo -e "${YELLOW}♻️  Restarting $svc${NC}"
        kubectl rollout restart deployment/$svc -n $NAMESPACE_SERVICES
    done
    
    echo -e "${GREEN}✅ All services restarted${NC}"
    echo -e "${BLUE}⏳ Waiting for rollout to complete...${NC}"
    
    for svc in quantum-gateway qagent qtest qsecure qsre qinfra; do
        kubectl rollout status deployment/$svc -n $NAMESPACE_SERVICES --timeout=300s
    done
    
    echo -e "${GREEN}✅ All services are running${NC}"
}

scale_service() {
    local service=$1
    local replicas=${2:-2}
    
    if [ -z "$service" ]; then
        echo -e "${RED}❌ Please specify a service to scale${NC}"
        echo -e "Available services: quantum-gateway, qagent, qtest, qsecure, qsre, qinfra"
        return 1
    fi
    
    echo -e "${BLUE}📈 Scaling $service to $replicas replicas${NC}"
    kubectl scale deployment/$service --replicas=$replicas -n $NAMESPACE_SERVICES
    
    echo -e "${BLUE}⏳ Waiting for scaling to complete...${NC}"
    kubectl rollout status deployment/$service -n $NAMESPACE_SERVICES --timeout=300s
    
    echo -e "${GREEN}✅ $service scaled to $replicas replicas${NC}"
}

backup_data() {
    local backup_name="quantum-backup-$(date +%Y%m%d-%H%M%S)"
    local backup_dir="./backups/$backup_name"
    
    echo -e "${BLUE}💾 Creating backup: $backup_name${NC}"
    mkdir -p "$backup_dir"
    
    # Backup configurations
    echo -e "${YELLOW}📋 Backing up configurations...${NC}"
    kubectl get configmap,secret --all-namespaces -o yaml > "$backup_dir/configs.yaml"
    
    # Backup persistent volume claims
    echo -e "${YELLOW}💽 Backing up PVC information...${NC}"
    kubectl get pvc --all-namespaces -o yaml > "$backup_dir/pvcs.yaml"
    
    # Database backup (this would typically use pg_dump)
    echo -e "${YELLOW}🗄️  Creating database backup...${NC}"
    kubectl exec -n $NAMESPACE_DATA statefulset/postgresql -- pg_dumpall -U quantum > "$backup_dir/database.sql" 2>/dev/null || echo "Database backup skipped"
    
    # Vector database snapshots
    echo -e "${YELLOW}🔍 Creating vector database backups...${NC}"
    kubectl exec -n $NAMESPACE_DATA statefulset/qdrant -- tar -czf /tmp/qdrant-backup.tar.gz /qdrant/storage 2>/dev/null || echo "Qdrant backup skipped"
    
    # Backup metadata
    cat > "$backup_dir/metadata.yaml" << EOF
backup_name: $backup_name
created_at: $(date)
cluster: $(kubectl config current-context)
quantum_suite_version: latest
namespaces:
  - $NAMESPACE_SYSTEM
  - $NAMESPACE_DATA  
  - $NAMESPACE_SERVICES
  - $NAMESPACE_MONITORING
EOF
    
    echo -e "${GREEN}✅ Backup created: $backup_dir${NC}"
    echo -e "${BLUE}📦 Backup contents:${NC}"
    ls -la "$backup_dir"
}

setup_port_forwarding() {
    echo -e "${BLUE}🔗 Setting up port forwarding for development${NC}"
    
    # Create port forwarding script
    cat > port-forward.sh << 'EOF'
#!/bin/bash
echo "🚀 Starting Quantum Suite port forwarding..."

# Array of services and their ports
declare -A services=(
    ["quantum-gateway"]="8080:8080"
    ["qagent"]="8110:8110"
    ["qtest"]="8120:8120"
    ["qsecure"]="8130:8130"
    ["qsre"]="8140:8140"
    ["qinfra"]="8150:8150"
    ["grafana"]="3000:3000"
    ["jaeger-query"]="16686:16686"
    ["prometheus"]="9090:9090"
)

# Start port forwarding for each service
for service in "${!services[@]}"; do
    port_map="${services[$service]}"
    
    if [[ "$service" == "grafana" || "$service" == "jaeger-query" || "$service" == "prometheus" ]]; then
        namespace="quantum-monitoring"
    else
        namespace="quantum-services"
    fi
    
    echo "🔗 Port forwarding $service ($port_map)"
    kubectl port-forward -n $namespace svc/$service $port_map &
done

echo "✅ Port forwarding active. Press Ctrl+C to stop all."
wait
EOF

    chmod +x port-forward.sh
    
    echo -e "${GREEN}✅ Port forwarding script created: ./port-forward.sh${NC}"
    echo -e "${BLUE}🔧 Run: ./port-forward.sh to start port forwarding${NC}"
    echo -e ""
    echo -e "${BLUE}Access URLs (after running port-forward.sh):${NC}"
    echo -e "  Gateway: http://localhost:8080"
    echo -e "  QAgent: http://localhost:8110"
    echo -e "  QTest: http://localhost:8120"
    echo -e "  QSecure: http://localhost:8130"
    echo -e "  QSRE: http://localhost:8140"
    echo -e "  QInfra: http://localhost:8150"
    echo -e "  Grafana: http://localhost:3000"
    echo -e "  Jaeger: http://localhost:16686"
    echo -e "  Prometheus: http://localhost:9090"
}

cleanup_failed_resources() {
    echo -e "${BLUE}🧹 Cleaning up failed resources${NC}"
    
    # Delete failed pods
    kubectl get pods --all-namespaces --field-selector=status.phase=Failed -o json | \
        jq -r '.items[] | "\(.metadata.namespace) \(.metadata.name)"' | \
        while read namespace pod; do
            echo -e "${YELLOW}🗑️  Deleting failed pod: $namespace/$pod${NC}"
            kubectl delete pod "$pod" -n "$namespace"
        done
    
    # Delete evicted pods
    kubectl get pods --all-namespaces --field-selector=status.phase=Failed | grep Evicted | \
        awk '{print $1, $2}' | \
        while read namespace pod; do
            echo -e "${YELLOW}🗑️  Deleting evicted pod: $namespace/$pod${NC}"
            kubectl delete pod "$pod" -n "$namespace"
        done
    
    # Clean up completed jobs older than 1 day
    kubectl get jobs --all-namespaces -o json | \
        jq -r '.items[] | select(.status.conditions[]?.type == "Complete") | "\(.metadata.namespace) \(.metadata.name) \(.status.completionTime)"' | \
        while read namespace job completion_time; do
            if [ -n "$completion_time" ] && [ "$(date -d "$completion_time" +%s)" -lt "$(date -d '1 day ago' +%s)" ]; then
                echo -e "${YELLOW}🗑️  Deleting old completed job: $namespace/$job${NC}"
                kubectl delete job "$job" -n "$namespace"
            fi
        done
    
    echo -e "${GREEN}✅ Cleanup completed${NC}"
}

open_dashboards() {
    echo -e "${BLUE}📊 Opening monitoring dashboards${NC}"
    
    # Get LoadBalancer IPs or use port forwarding
    GRAFANA_IP=$(kubectl get svc grafana -n $NAMESPACE_MONITORING -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
    JAEGER_IP=$(kubectl get svc jaeger-query -n $NAMESPACE_MONITORING -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
    
    if [ -n "$GRAFANA_IP" ]; then
        echo -e "${GREEN}🌐 Grafana: http://$GRAFANA_IP:3000${NC}"
    else
        echo -e "${YELLOW}⚠️  Grafana LoadBalancer IP not available. Use port forwarding:${NC}"
        echo -e "  kubectl port-forward -n $NAMESPACE_MONITORING svc/grafana 3000:3000"
    fi
    
    if [ -n "$JAEGER_IP" ]; then
        echo -e "${GREEN}🌐 Jaeger: http://$JAEGER_IP:16686${NC}"
    else
        echo -e "${YELLOW}⚠️  Jaeger LoadBalancer IP not available. Use port forwarding:${NC}"
        echo -e "  kubectl port-forward -n $NAMESPACE_MONITORING svc/jaeger-query 16686:16686"
    fi
}

manage_secrets() {
    echo -e "${BLUE}🔐 Quantum Suite Secrets Management${NC}"
    echo -e ""
    echo -e "${BLUE}Current secrets:${NC}"
    kubectl get secret -n $NAMESPACE_SYSTEM quantum-secrets -o yaml | grep -E '^  [A-Z_]+:' | sed 's/:.*//' | sed 's/^  /  /'
    echo -e ""
    echo -e "${BLUE}To update a secret:${NC}"
    echo -e "  kubectl patch secret quantum-secrets -n $NAMESPACE_SYSTEM -p '{\"data\":{\"SECRET_NAME\":\"BASE64_VALUE\"}}'"
    echo -e ""
    echo -e "${BLUE}To encode a value:${NC}"
    echo -e "  echo -n 'your-secret-value' | base64"
}

# Main command handling
case "${1:-help}" in
    status)
        show_status
        ;;
    logs)
        show_logs "$2" "$3"
        ;;
    restart)
        restart_services
        ;;
    scale)
        scale_service "$2" "$3"
        ;;
    backup)
        backup_data
        ;;
    port-forward)
        setup_port_forwarding
        ;;
    cleanup)
        cleanup_failed_resources
        ;;
    dashboard)
        open_dashboards
        ;;
    secrets)
        manage_secrets
        ;;
    help|*)
        show_usage
        ;;
esac