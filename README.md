# argo-ephemeral-operator

A Kubernetes/OpenShift operator written in Golang that uses ArgoCD to create and manage ephemeral environments with automatic expiration.

## Features

- **Automated Environment Creation**: Create ephemeral namespaces and deploy applications automatically
- **ArgoCD Integration**: Leverage ArgoCD's GitOps capabilities for application deployment
- **Time-based Expiration**: Automatically remove environments after a specified expiration date
- **Declarative API**: Simple Custom Resource Definition (CRD) for managing ephemeral applications
- **Production Ready**: Built following SOLID, DRY, and YAGNI principles with comprehensive error handling

## How It Works

1. **Create**: User creates an `EphemeralApplication` custom resource
2. **Deploy**: Operator detects the resource and:
   - Creates a new ephemeral namespace
   - Creates an ArgoCD Application pointing to the specified Git repository
   - ArgoCD deploys all resources into the ephemeral namespace
3. **Monitor**: Operator continuously monitors the application health and sync status
4. **Cleanup**: When the expiration date is reached:
   - Operator deletes the ArgoCD Application
   - Kubernetes removes the namespace and all resources within it

## Architecture

The operator follows clean architecture principles:

- **API Layer** (`api/v1alpha1`): Custom Resource Definitions
- **Controller Layer** (`internal/controller`): Reconciliation logic following Kubernetes controller patterns
- **ArgoCD Client** (`internal/argocd`): Interface-based ArgoCD interaction (follows Interface Segregation Principle)
- **Configuration** (`internal/config`): Environment-based configuration management

## Prerequisites

- Kubernetes cluster (v1.24+)
- ArgoCD installed and running in the cluster
- ArgoCD authentication token
- `kubectl` configured to access your cluster

## Installation

### 1. Install the CRD

```bash
make install
```

Or manually:

```bash
kubectl apply -f config/crd/bases/ephemeral.argo.io_ephemeralapplications.yaml
```

### 2. Create ArgoCD Access Secret

First, get your ArgoCD admin password:

```bash
kubectl get secret argocd-initial-admin-secret -n argocd -o jsonpath='{.data.password}' | base64 -d
```

Create the secret with your ArgoCD credentials:

```bash
kubectl create secret generic argo-ephemeral-operator-config \
  --from-literal=argo-server="argocd-server.argocd.svc.cluster.local" \
  --from-literal=argo-port="443" \
  --from-literal=argo-username="admin" \
  --from-literal=argo-password="YOUR_ARGOCD_PASSWORD" \
  --from-literal=argo-namespace="argocd" \
  --from-literal=argo-insecure="true" \
  -n argo-ephemeral-operator-system
```

Or use the example template:

```bash
# Edit the secret with your values
cp config/samples/secret-example.yaml my-secret.yaml
# Edit my-secret.yaml with your actual values
kubectl apply -f my-secret.yaml
```

### 3. Deploy the Operator

```bash
make deploy
```

Or manually:

```bash
kubectl apply -f config/manager/namespace.yaml
kubectl apply -f config/rbac/service_account.yaml
kubectl apply -f config/rbac/role.yaml
kubectl apply -f config/rbac/role_binding.yaml
kubectl apply -f config/manager/deployment.yaml
```

### 4. Verify Installation

```bash
kubectl get pods -n argo-ephemeral-operator-system
```

## Usage

### Creating an Ephemeral Application

Create a YAML file with your ephemeral application definition:

```yaml
apiVersion: ephemeral.argo.io/v1alpha1
kind: EphemeralApplication
metadata:
  name: my-feature-branch
  namespace: default
spec:
  # Git repository containing your application manifests
  repoURL: https://github.com/your-org/your-app.git
  
  # Path within the repository
  path: kubernetes/manifests
  
  # Git revision (branch, tag, or commit SHA)
  targetRevision: feature/new-feature
  
  # Expiration date (RFC3339 format)
  expirationDate: "2025-10-27T23:59:59Z"
  
  # Optional: Namespace name (if not specified, auto-generates as ephemeral-{random})
  namespaceName: feature-new-feature
  
  # Optional: Sync policy
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

Apply the resource:

```bash
kubectl apply -f my-ephemeral-app.yaml
```

### Checking Status

```bash
# List all ephemeral applications
kubectl get ephapp

# Get detailed status
kubectl describe ephapp my-feature-branch

# Check the created namespace
kubectl get namespaces | grep ephemeral
```

### Deleting an Ephemeral Application

Ephemeral applications are automatically deleted when they expire, but you can manually delete them:

```bash
kubectl delete ephapp my-feature-branch
```

This will trigger cleanup of the ArgoCD Application and the ephemeral namespace.

## Configuration

The operator is configured through environment variables:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ARGO_SERVER` | ArgoCD server address | `argocd-server.argocd.svc.cluster.local` | Yes |
| `ARGO_PORT` | ArgoCD server port | `443` | No |
| `ARGO_USERNAME` | ArgoCD username | `admin` | Yes |
| `ARGO_PASSWORD` | ArgoCD password | - | Yes |
| `ARGO_NAMESPACE` | Namespace where ArgoCD is installed | `argocd` | No |
| `ARGO_INSECURE` | Skip TLS verification | `true` | No |
| `RECONCILE_INTERVAL` | How often to check application status | `5m` | No |
| `METRICS_ADDR` | Metrics endpoint address | `:8080` | No |
| `HEALTH_PROBE_ADDR` | Health probe endpoint address | `:8081` | No |
| `ENABLE_LEADER_ELECTION` | Enable leader election | `false` | No |

## Development

### Prerequisites

- Go 1.21+
- Docker (for building images)
- Access to a Kubernetes cluster

### Building

```bash
# Build the binary
make build

# Run locally (requires kubeconfig)
make run

# Run tests
make test

# Format code
make fmt

# Run linter
make vet
```

### Building Docker Image

```bash
make docker-build IMG=your-registry/argo-ephemeral-operator:tag
make docker-push IMG=your-registry/argo-ephemeral-operator:tag
```

### Running Locally

```bash
# Set required environment variables
export ARGO_SERVER="argocd-server.argocd.svc.cluster.local"
export ARGO_PORT="443"
export ARGO_USERNAME="admin"
export ARGO_PASSWORD="your-argocd-password"
export ARGO_NAMESPACE="argocd"

# Run the operator
make run
```

## Examples

### Example 1: Short-lived Test Environment

```yaml
apiVersion: ephemeral.argo.io/v1alpha1
kind: EphemeralApplication
metadata:
  name: pr-123-test
spec:
  repoURL: https://github.com/company/app.git
  path: deploy/kubernetes
  targetRevision: pull/123/head
  expirationDate: "2025-10-21T18:00:00Z"  # Expires in 6 hours
  namespaceName: pr-123-test
```

### Example 2: Demo Environment

```yaml
apiVersion: ephemeral.argo.io/v1alpha1
kind: EphemeralApplication
metadata:
  name: customer-demo
spec:
  repoURL: https://github.com/company/demo-app.git
  path: k8s
  targetRevision: v2.0.0
  expirationDate: "2025-10-25T23:59:59Z"  # Expires in 5 days
  namespaceName: customer-demo
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### Example 3: Development Environment

```yaml
apiVersion: ephemeral.argo.io/v1alpha1
kind: EphemeralApplication
metadata:
  name: dev-john-doe
spec:
  repoURL: https://github.com/company/microservices.git
  path: services/api
  targetRevision: dev/john-doe
  expirationDate: "2025-11-01T09:00:00Z"  # Expires in 2 weeks
  namespaceName: dev-john-doe
```

## Troubleshooting

### Operator not starting

Check the operator logs:

```bash
kubectl logs -n argo-ephemeral-operator-system deployment/argo-ephemeral-operator-controller-manager
```

### EphemeralApplication stuck in "Creating" phase

1. Check the ArgoCD Application status:
```bash
kubectl get applications -n argocd
kubectl describe application -n argocd ephemeral-<name>
```

2. Verify ArgoCD can access the Git repository
3. Check ArgoCD Application Controller logs

### Namespace not being deleted

Check if there are resources preventing deletion:

```bash
kubectl describe namespace <ephemeral-namespace>
kubectl api-resources --verbs=list --namespaced -o name | xargs -n 1 kubectl get --show-kind --ignore-not-found -n <namespace>
```

### Permission Errors

Ensure the operator has the correct RBAC permissions:

```bash
kubectl auth can-i create namespaces --as=system:serviceaccount:argo-ephemeral-operator-system:argo-ephemeral-operator-controller-manager
kubectl auth can-i create applications.argoproj.io --as=system:serviceaccount:argo-ephemeral-operator-system:argo-ephemeral-operator-controller-manager -n argocd
```

## Contributing

Contributions are welcome! This project follows:

- **SOLID Principles**: Single Responsibility, Open/Closed, Liskov Substitution, Interface Segregation, Dependency Inversion
- **DRY (Don't Repeat Yourself)**: Code reuse and abstraction where appropriate
- **YAGNI (You Aren't Gonna Need It)**: Only implement what's necessary

## License

See [LICENSE](LICENSE) file for details.
