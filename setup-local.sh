#!/usr/bin/env bash
#
# setup-local.sh — Full local installation of argo-ephemeral-operator
#
# This script provisions a local minikube cluster, installs ArgoCD, builds all
# container images (operator, API server, UI), loads them into the cluster,
# and deploys every component so you can test end-to-end locally.
#
# The deployment steps mirror the Makefile targets (make install, make
# deploy-operator, make deploy-api, make deploy-ui) so the local environment
# matches production as closely as possible.
#
# Usage:
#   ./setup-local.sh              # Full install (default)
#   ./setup-local.sh --skip-build # Skip image build (reuse existing images)
#   ./setup-local.sh --teardown   # Destroy the local cluster
#
set -euo pipefail

# ─── Configuration ───────────────────────────────────────────────────────────

CLUSTER_NAME="argo-ephemeral-local"
ARGOCD_NAMESPACE="argocd"
ARGOCD_VERSION="v2.14.20"  # Must match the version in go.mod
OPERATOR_NAMESPACE="argo-ephemeral-operator-system"
MINIKUBE_CPUS=4
MINIKUBE_MEMORY=8192

# These must match the image references used in config/ deployment manifests
IMG_OPERATOR="localhost/argo-ephemeral-operator:latest"
IMG_API="localhost/argo-ephemeral-api:latest"    # matches config/api/deployment.yaml
IMG_UI="localhost/argo-ephemeral-ui:latest"       # matches config/ui/deployment.yaml

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

SKIP_BUILD=false
TEARDOWN=false
RELOAD_IMAGES=false

# ─── Colors ──────────────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ─── Helper functions ────────────────────────────────────────────────────────

info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }
step()    { echo -e "\n${CYAN}━━━ $* ━━━${NC}"; }

# ─── Parse arguments ────────────────────────────────────────────────────────

for arg in "$@"; do
  case "$arg" in
    --skip-build)     SKIP_BUILD=true ;;
    --teardown)       TEARDOWN=true ;;
    --reload-images)  RELOAD_IMAGES=true ;;
    -h|--help)
      echo "Usage: $0 [--skip-build] [--teardown] [--reload-images] [-h|--help]"
      echo ""
      echo "  --skip-build      Skip building container images (reuse existing)"
      echo "  --teardown        Destroy the local minikube cluster and exit"
      echo "  --reload-images   Rebuild images, reload into cluster, and restart deployments"
      echo "  -h, --help        Show this help message"
      exit 0
      ;;
    *) warn "Unknown argument: $arg" ;;
  esac
done

# ─── Teardown ────────────────────────────────────────────────────────────────

if [ "$TEARDOWN" = true ]; then
  step "Tearing down local environment"
  if minikube profile list 2>/dev/null | grep -q "$CLUSTER_NAME"; then
    minikube delete -p "$CLUSTER_NAME"
    success "Cluster '${CLUSTER_NAME}' deleted"
  else
    warn "Cluster '${CLUSTER_NAME}' does not exist"
  fi
  exit 0
fi

# ─── Reload images (skip everything else) ───────────────────────────────────

if [ "$RELOAD_IMAGES" = true ]; then
  step "Rebuilding and reloading images"

  cd "$SCRIPT_DIR"

  # Build
  info "Building operator image (${IMG_OPERATOR})..."
  docker build -t "$IMG_OPERATOR" -f Dockerfile .
  success "Operator image built"

  info "Building API server image (${IMG_API})..."
  docker build -t "$IMG_API" -f Dockerfile.api .
  success "API server image built"

  info "Building UI assets (npm ci + npm run build)..."
  (cd web && npm ci --legacy-peer-deps && npm run build)
  success "UI assets built"

  info "Building UI image (${IMG_UI})..."
  docker build -t "$IMG_UI" -f Dockerfile.ui .
  success "UI image built"

  # Load
  IMAGES_TMP_DIR=$(mktemp -d)
  trap "rm -rf ${IMAGES_TMP_DIR}" EXIT

  for img in "$IMG_OPERATOR" "$IMG_API" "$IMG_UI"; do
    tarball="${IMAGES_TMP_DIR}/$(echo "$img" | tr '/:' '__').tar"
    info "Saving and loading ${img}..."
    docker save -o "$tarball" "$img"
    minikube image load "$tarball" -p "$CLUSTER_NAME"
    success "Loaded ${img}"
  done

  # Restart deployments that exist to pick up new images
  info "Restarting deployments..."
  for deploy in \
    argo-ephemeral-operator-controller-manager \
    argo-ephemeral-api \
    argo-ephemeral-ui; do
    if kubectl get deployment "$deploy" -n "$OPERATOR_NAMESPACE" &>/dev/null; then
      kubectl rollout restart deployment/"$deploy" -n "$OPERATOR_NAMESPACE"
      kubectl rollout status deployment/"$deploy" -n "$OPERATOR_NAMESPACE" --timeout=120s
      success "${deploy} restarted"
    else
      warn "${deploy} not found, skipping (run full install first)"
    fi
  done
  exit 0
fi

# ─── Prerequisite checks ────────────────────────────────────────────────────

step "Checking prerequisites"

check_cmd() {
  if ! command -v "$1" &>/dev/null; then
    error "'$1' is not installed. $2"
  fi
  success "$1 found: $(command -v "$1")"
}

check_cmd "minikube" "Install minikube: https://minikube.sigs.k8s.io/docs/start/"
check_cmd "kubectl"  "Install kubectl: https://kubernetes.io/docs/tasks/tools/"
check_cmd "go"       "Install Go 1.24+: https://go.dev/doc/install"
check_cmd "node"     "Install Node.js 20+: https://nodejs.org/"
check_cmd "npm"      "Install npm (comes with Node.js): https://nodejs.org/"

# A container or VM runtime is needed (docker, podman, or kvm2/libvirt)
if command -v docker &>/dev/null || command -v podman &>/dev/null || command -v virsh &>/dev/null; then
  success "Container/VM runtime available"
else
  error "No container or VM runtime found. Install Docker, Podman, or libvirt (KVM)."
fi

# ─── Create minikube cluster ────────────────────────────────────────────────

step "Setting up minikube cluster"

if minikube status -p "$CLUSTER_NAME" &>/dev/null; then
  info "Cluster '${CLUSTER_NAME}' already exists, reusing it"
else
  info "Creating minikube cluster '${CLUSTER_NAME}'..."
  info "minikube will auto-detect the best available driver for your system."

  minikube start \
    -p "$CLUSTER_NAME" \
    --cpus="$MINIKUBE_CPUS" \
    --memory="$MINIKUBE_MEMORY" \
    --addons=default-storageclass,storage-provisioner

  success "minikube cluster created"
fi

# Make sure kubectl context points to the local cluster
kubectl config use-context "$CLUSTER_NAME" &>/dev/null
success "kubectl context set to ${CLUSTER_NAME}"

# Wait for the cluster to be ready
info "Waiting for cluster nodes to be Ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=120s
success "All nodes are Ready"

# ─── Install ArgoCD ─────────────────────────────────────────────────────────

step "Installing ArgoCD"

if kubectl get namespace "$ARGOCD_NAMESPACE" &>/dev/null; then
  info "ArgoCD namespace already exists, checking installation..."
else
  kubectl create namespace "$ARGOCD_NAMESPACE"
  info "Created namespace '${ARGOCD_NAMESPACE}'"
fi

info "Applying ArgoCD manifests (${ARGOCD_VERSION})..."
# Server-side apply is required because some ArgoCD CRDs (e.g. applicationsets.argoproj.io)
# exceed the 262144-byte annotation limit imposed by client-side kubectl apply.
kubectl apply -n "$ARGOCD_NAMESPACE" --server-side=true --force-conflicts \
  -f "https://raw.githubusercontent.com/argoproj/argo-cd/${ARGOCD_VERSION}/manifests/install.yaml"

info "Waiting for ArgoCD server to be ready (this may take a few minutes)..."
kubectl rollout status deployment/argocd-server -n "$ARGOCD_NAMESPACE" --timeout=300s
kubectl rollout status deployment/argocd-repo-server -n "$ARGOCD_NAMESPACE" --timeout=300s
kubectl rollout status deployment/argocd-applicationset-controller -n "$ARGOCD_NAMESPACE" --timeout=300s

success "ArgoCD is running"

# Retrieve ArgoCD admin password (same as documented in README)
ARGOCD_PASSWORD=$(kubectl get secret argocd-initial-admin-secret -n "$ARGOCD_NAMESPACE" \
  -o jsonpath='{.data.password}' 2>/dev/null | base64 -d 2>/dev/null || echo "")

if [ -z "$ARGOCD_PASSWORD" ]; then
  warn "Could not retrieve ArgoCD admin password automatically."
  warn "You may need to set it manually later."
  ARGOCD_PASSWORD="argocd"
fi

success "ArgoCD admin password retrieved"

# ─── Build container images ─────────────────────────────────────────────────
# Mirrors: make docker-build (Makefile)

if [ "$SKIP_BUILD" = true ]; then
  step "Skipping image build (--skip-build)"
else
  step "Building container images"

  cd "$SCRIPT_DIR"

  # make docker-build-operator
  info "Building operator image (${IMG_OPERATOR})..."
  docker build -t "$IMG_OPERATOR" -f Dockerfile .
  success "Operator image built"

  # make docker-build-api
  info "Building API server image (${IMG_API})..."
  docker build -t "$IMG_API" -f Dockerfile.api .
  success "API server image built"

  # make build-ui + make docker-build-ui
  info "Building UI assets (npm ci + npm run build)..."
  (cd web && npm ci --legacy-peer-deps && npm run build)
  success "UI assets built"

  info "Building UI image (${IMG_UI})..."
  docker build -t "$IMG_UI" -f Dockerfile.ui .
  success "UI image built"
fi

# ─── Load images into minikube ───────────────────────────────────────────────

step "Loading images into minikube cluster"

IMAGES_TMP_DIR=$(mktemp -d)
trap "rm -rf ${IMAGES_TMP_DIR}" EXIT

for img in "$IMG_OPERATOR" "$IMG_API" "$IMG_UI"; do
  tarball="${IMAGES_TMP_DIR}/$(echo "$img" | tr '/:' '__').tar"
  info "Saving ${img} to tarball..."
  docker save -o "$tarball" "$img"
  info "Loading ${img} into minikube..."
  minikube image load "$tarball" -p "$CLUSTER_NAME"
  success "Loaded ${img}"
done

# ─── Install CRDs ───────────────────────────────────────────────────────────
# Mirrors: make install

step "Installing CRDs"

kubectl apply -f "${SCRIPT_DIR}/config/crd/bases/"
success "CRDs applied"

kubectl wait --for=condition=Established crd/ephemeralapplications.ephemeral.argo.io --timeout=30s
success "CRD 'ephemeralapplications.ephemeral.argo.io' is established"

# ─── Create namespace and operator config secret ────────────────────────────
# The namespace comes from config/manager/namespace.yaml.
# The secret follows the template in config/samples/secret-example.yaml,
# auto-populated with the ArgoCD password retrieved earlier.

step "Creating operator namespace and config secret"

kubectl apply -f "${SCRIPT_DIR}/config/manager/namespace.yaml"
success "Namespace '${OPERATOR_NAMESPACE}' created"

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: argo-ephemeral-operator-config
  namespace: ${OPERATOR_NAMESPACE}
type: Opaque
stringData:
  argo-server: "argocd-server.${ARGOCD_NAMESPACE}.svc.cluster.local"
  argo-port: "443"
  argo-username: "admin"
  argo-password: "${ARGOCD_PASSWORD}"
  argo-namespace: "${ARGOCD_NAMESPACE}"
  argo-insecure: "true"
EOF
success "Operator config secret created with ArgoCD credentials"

# ─── Deploy Operator ────────────────────────────────────────────────────────
# Mirrors: make deploy-operator
#   kubectl apply -f config/manager/namespace.yaml       (already done above)
#   kubectl apply -f config/rbac/service_account.yaml
#   kubectl apply -f config/rbac/role.yaml
#   kubectl apply -f config/rbac/role_binding.yaml
#   kubectl apply -f config/manager/deployment.yaml
#
# The operator deployment references the image from quay.io. For local minikube
# we apply a temporary copy that points to the locally built image instead.

step "Deploying Operator (make deploy-operator)"

kubectl apply -f "${SCRIPT_DIR}/config/rbac/service_account.yaml"
kubectl apply -f "${SCRIPT_DIR}/config/rbac/role.yaml"
kubectl apply -f "${SCRIPT_DIR}/config/rbac/role_binding.yaml"
success "Operator RBAC deployed"

# Generate a temporary deployment manifest with the local image reference
OPERATOR_IMG_FROM=$(grep 'image:' "${SCRIPT_DIR}/config/manager/deployment.yaml" | awk '{print $2}')
sed "s|${OPERATOR_IMG_FROM}|${IMG_OPERATOR}|g; s|imagePullPolicy: IfNotPresent|imagePullPolicy: Never|g" \
  "${SCRIPT_DIR}/config/manager/deployment.yaml" | kubectl apply -f -
success "Operator deployment applied (image: ${IMG_OPERATOR})"

kubectl rollout status deployment/argo-ephemeral-operator-controller-manager \
  -n "$OPERATOR_NAMESPACE" --timeout=120s
success "Operator is running"

# ─── Deploy API Server ──────────────────────────────────────────────────────
# Mirrors: make deploy-api
#   kubectl apply -f config/api/

step "Deploying API Server (make deploy-api)"

kubectl apply -f "${SCRIPT_DIR}/config/api/"
success "API server resources applied"

kubectl rollout status deployment/argo-ephemeral-api \
  -n "$OPERATOR_NAMESPACE" --timeout=120s
success "API server is running"

# ─── Deploy UI ──────────────────────────────────────────────────────────────
# Mirrors: make deploy-ui
#   kubectl apply -f config/ui/

step "Deploying UI (make deploy-ui)"

kubectl apply -f "${SCRIPT_DIR}/config/ui/"
success "UI resources applied"

kubectl rollout status deployment/argo-ephemeral-ui \
  -n "$OPERATOR_NAMESPACE" --timeout=120s
success "UI is running"

# ─── Verify deployment ──────────────────────────────────────────────────────

step "Verifying deployment"

echo ""
info "Pods in namespace '${OPERATOR_NAMESPACE}':"
kubectl get pods -n "$OPERATOR_NAMESPACE" -o wide
echo ""

info "Services in namespace '${OPERATOR_NAMESPACE}':"
kubectl get svc -n "$OPERATOR_NAMESPACE"
echo ""

info "ArgoCD pods:"
kubectl get pods -n "$ARGOCD_NAMESPACE" --field-selector=status.phase=Running -o name | head -5
echo ""

# ─── Summary ─────────────────────────────────────────────────────────────────

step "Installation complete!"

cat <<EOF

${GREEN}All components have been deployed successfully!${NC}

${CYAN}Cluster:${NC}  ${CLUSTER_NAME} (minikube)
${CYAN}ArgoCD:${NC}   namespace '${ARGOCD_NAMESPACE}'
${CYAN}Operator:${NC} namespace '${OPERATOR_NAMESPACE}'

${YELLOW}── Access the services ────────────────────────────────────────────${NC}

  ${CYAN}ArgoCD UI:${NC}
    kubectl port-forward svc/argocd-server -n ${ARGOCD_NAMESPACE} 9090:443 &
    Open: https://localhost:9090
    User: admin  |  Password: ${ARGOCD_PASSWORD}

  ${CYAN}Ephemeral Operator API:${NC}
    kubectl port-forward svc/argo-ephemeral-api-service -n ${OPERATOR_NAMESPACE} 8080:8080 &
    Open: http://localhost:8080/api/v1/ephemeral-apps

  ${CYAN}Ephemeral Operator UI:${NC}
    kubectl port-forward svc/argo-ephemeral-ui-service -n ${OPERATOR_NAMESPACE} 8888:80 &
    Open: http://localhost:8888

${YELLOW}── Quick start ────────────────────────────────────────────────────${NC}

  # Apply the sample EphemeralApplication:
  kubectl apply -f config/samples/ephemeral_v1alpha1_ephemeralapplication.yaml

  # Check status:
  kubectl get ephapp

  # View operator logs:
  kubectl logs -f deployment/argo-ephemeral-operator-controller-manager -n ${OPERATOR_NAMESPACE}

${YELLOW}── Teardown ───────────────────────────────────────────────────────${NC}

  ./setup-local.sh --teardown

EOF
