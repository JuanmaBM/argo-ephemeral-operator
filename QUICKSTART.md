# Quick Start Guide

This guide will help you get the argo-ephemeral-operator running in your Kubernetes cluster in less than 10 minutes.

## Prerequisites

- Kubernetes cluster (minikube, kind, or any other cluster)
- kubectl configured
- ArgoCD installed in your cluster
- ArgoCD CLI (for generating tokens)

## Step 1: Install ArgoCD (if not already installed)

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ArgoCD to be ready
kubectl wait --for=condition=available --timeout=300s deployment/argocd-server -n argocd
```

## Step 2: Get ArgoCD Token

```bash
# Port forward ArgoCD server
kubectl port-forward svc/argocd-server -n argocd 8080:443 &

# Get initial admin password
ARGO_PASSWORD=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)

# Login with argocd CLI
argocd login localhost:8080 --username admin --password $ARGO_PASSWORD --insecure

# Generate a token
ARGO_TOKEN=$(argocd account generate-token --account admin)

echo "Your ArgoCD token: $ARGO_TOKEN"
```

## Step 3: Install the Operator

```bash
# Clone the repository (or use your local copy)
cd argo-ephemeral-operator

# Install the CRD
make install

# Create the operator namespace
kubectl apply -f config/manager/namespace.yaml

# Create the secret with ArgoCD credentials
kubectl create secret generic argo-ephemeral-operator-config \
  --from-literal=argo-server="argocd-server.argocd.svc.cluster.local" \
  --from-literal=argo-token="$ARGO_TOKEN" \
  --from-literal=argo-namespace="argocd" \
  --from-literal=argo-insecure="true" \
  -n argo-ephemeral-operator-system

# Deploy the operator
make deploy

# Verify the operator is running
kubectl get pods -n argo-ephemeral-operator-system
```

## Step 4: Create Your First Ephemeral Application

```bash
# Create an ephemeral application
cat <<EOF | kubectl apply -f -
apiVersion: ephemeral.argo.io/v1alpha1
kind: EphemeralApplication
metadata:
  name: my-first-ephemeral-app
  namespace: default
spec:
  repoURL: https://github.com/argoproj/argocd-example-apps.git
  path: guestbook
  targetRevision: HEAD
  expirationDate: "$(date -u -d '+1 hour' +%Y-%m-%dT%H:%M:%SZ)"
  namespacePrefix: demo
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
EOF
```

## Step 5: Watch It Work

```bash
# Watch the ephemeral application status
kubectl get ephapp my-first-ephemeral-app -w

# Once it's Active, check the namespace
kubectl get namespaces | grep demo

# Check the ArgoCD application
kubectl get applications -n argocd | grep ephemeral

# Check the resources in the ephemeral namespace
EPHEMERAL_NS=$(kubectl get ephapp my-first-ephemeral-app -o jsonpath='{.status.namespace}')
kubectl get all -n $EPHEMERAL_NS
```

## Step 6: Verify the Guestbook Application

```bash
# Port forward to access the guestbook
kubectl port-forward svc/guestbook-ui -n $EPHEMERAL_NS 8081:80

# Open in browser: http://localhost:8081
```

## Step 7: Cleanup (Optional)

```bash
# Delete the ephemeral application
kubectl delete ephapp my-first-ephemeral-app

# Watch the cleanup happen
kubectl get namespaces -w | grep demo
```

## Common Issues

### Operator Pod Not Starting

```bash
# Check logs
kubectl logs -n argo-ephemeral-operator-system deployment/argo-ephemeral-operator-controller-manager

# Common issues:
# - Missing secret
# - Invalid ArgoCD token
# - Network connectivity to ArgoCD
```

### EphemeralApplication Stuck in "Pending"

```bash
# Check operator logs
kubectl logs -n argo-ephemeral-operator-system deployment/argo-ephemeral-operator-controller-manager

# Check events
kubectl describe ephapp my-first-ephemeral-app
```

### ArgoCD Application Not Syncing

```bash
# Check ArgoCD application
kubectl describe application -n argocd ephemeral-my-first-ephemeral-app

# Check ArgoCD application controller logs
kubectl logs -n argocd deployment/argocd-application-controller
```

## Next Steps

- Read the [full README](README.md) for more details
- Check out the [Architecture documentation](ARCHITECTURE.md)
- See more examples in `config/samples/`
- Integrate with your CI/CD pipeline

## Using with CI/CD

Example GitHub Actions workflow:

```yaml
- name: Create Ephemeral Environment
  run: |
    cat <<EOF | kubectl apply -f -
    apiVersion: ephemeral.argo.io/v1alpha1
    kind: EphemeralApplication
    metadata:
      name: pr-${{ github.event.pull_request.number }}
      namespace: default
    spec:
      repoURL: ${{ github.event.repository.clone_url }}
      path: k8s
      targetRevision: ${{ github.event.pull_request.head.sha }}
      expirationDate: "$(date -u -d '+7 days' +%Y-%m-%dT%H:%M:%SZ)"
      namespacePrefix: pr
    EOF
```

Enjoy your ephemeral environments! 🚀

