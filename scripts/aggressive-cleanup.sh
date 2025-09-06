#!/usr/bin/env bash
set -euo pipefail

read -rp $'\n*** DANGER *** This will DELETE apps, CRDs, PVCs/PVs, webhooks, Helm releases, and (optionally) Longhorn data.\nType "YES" to continue: ' ANSW
[[ "${ANSW:-}" == "YES" ]] || { echo "Aborted."; exit 1; }

# 0) Context check
echo -e "\n==> kubectl context:"; kubectl config current-context
echo -e "==> Nodes:"; kubectl get nodes -o wide || true
echo -e "\nProceeding in 5 seconds... Ctrl+C to abort"; sleep 5

# 1) Uninstall ALL Helm releases (cluster-wide)
echo -e "\n==> Uninstalling Helm releases..."
helm list -A | awk 'NR>1 {print "helm uninstall -n "$2" "$1}' | bash || true

# 2) Scale everything down to stop thrashing
echo -e "\n==> Scaling all Deployments/StatefulSets to 0 (all namespaces)..."
kubectl get deploy -A --no-headers | awk '{print "kubectl -n "$1" scale deploy "$2" --replicas=0"}' | bash || true
kubectl get sts    -A --no-headers | awk '{print "kubectl -n "$1" scale sts "$2"    --replicas=0"}' | bash || true

# 3) Delete workload objects cluster-wide (namespaced)
echo -e "\n==> Deleting namespaced workloads..."
for kind in deploy sts ds rs po job cronjob svc ing cm secret sa pdb hpa gw httproute tcproute udproute; do
  kubectl -A delete $kind --all --ignore-not-found --timeout=30s || true
done

# 4) Delete PersistentVolumeClaims (this drops app data)
echo -e "\n==> Deleting all PVCs..."
kubectl -A delete pvc --all --ignore-not-found --timeout=30s || true

# 5) Delete cluster-scoped extras: PVs, StorageClasses, Webhooks, APIService, CRDs
echo -e "\n==> Deleting PVs and StorageClasses..."
kubectl delete pv --all --ignore-not-found --timeout=30s || true
kubectl delete sc --all --ignore-not-found --timeout=30s || true

echo -e "\n==> Deleting Webhook Configurations and API Services..."
kubectl delete mutatingwebhookconfigurations.admissionregistration.k8s.io --all --ignore-not-found || true
kubectl delete validatingwebhookconfigurations.admissionregistration.k8s.io --all --ignore-not-found || true
kubectl delete apiservices.apiregistration.k8s.io --all --ignore-not-found || true

echo -e "\n==> Deleting ALL CRDs (this removes operators like ArgoCD, cert-manager, Longhorn, Temporal, etc.)"
kubectl get crd -o name | xargs -r kubectl delete || true

# 6) Delete ALL non-core namespaces (handles stuck finalizers)
echo -e "\n==> Deleting non-core namespaces (finalizers will be cleared if needed)..."
PRESERVE='^(kube-system|kube-public|kube-node-lease|default)$'
for ns in $(kubectl get ns --no-headers | awk '{print $1}'); do
  if [[ ! $ns =~ $PRESERVE ]]; then
    echo "Deleting namespace: $ns"
    kubectl delete ns "$ns" --timeout=60s || true
    # If stuck: strip finalizers
    if kubectl get ns "$ns" -o json >/dev/null 2>&1; then
      kubectl get ns "$ns" -o json | jq '.spec.finalizers=[]' | \
        kubectl replace --raw /api/v1/namespaces/"$ns"/finalize -f - || true
    fi
  fi
done

# 7) Optional: Longhorn deep clean (comment this block if you want to keep Longhorn data)
echo -e "\n==> Attempting Longhorn uninstall and data purge (optional)..."
if kubectl get ns longhorn-system >/dev/null 2>&1; then
  # Remove Longhorn workloads and CRDs
  kubectl -n longhorn-system delete all,ds,deploy,sts,svc,cm,secret,sa,role,rolebinding,psp,ing --all --ignore-not-found || true
  kubectl get crd | awk '/longhorn.io/ {print $1}' | xargs -r kubectl delete crd || true
fi
echo -e "NOTE: You may also need to wipe Longhorn node data dirs (run on each node): sudo rm -rf /var/lib/longhorn /var/lib/rancher/longhorn || true"

# 8) CNI/cache cleanup (safe to run if using flannel/cilium/calico)
echo -e "\n==> Cleaning CNI caches on this node (non-destructive for cluster)..."
sudo rm -rf /var/lib/cni/* /var/run/flannel/* 2>/dev/null || true
sudo rm -rf /etc/cni/net.d/* 2>/dev/null || true

# 9) Garbage-collect any leftover pods in Failed/CrashLoop
echo -e "\n==> Final pod sweep..."
kubectl get pods -A --field-selector=status.phase=Failed -o name | xargs -r kubectl delete || true
kubectl get pods -A | awk '/CrashLoopBackOff/ {print $1" "$2}' | while read ns pod; do kubectl -n "$ns" delete pod "$pod" || true; done

echo -e "\n==> DONE. Current state:"
kubectl get ns
kubectl get pods -A

echo -e "\n🎉 Cluster cleanup completed! Your cluster is now clean and ready for Quantum Suite deployment."