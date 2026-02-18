# argo-ephemeral-operator

A Kubernetes/OpenShift operator written in Golang that uses ArgoCD to create and manage ephemeral environments with automatic expiration.

## Features

- **Automated Environment Creation**: Create ephemeral namespaces and deploy applications automatically
- **ArgoCD Integration**: Leverage ArgoCD's GitOps capabilities for application deployment
- **Time-based Expiration**: Automatically remove environments after a specified expiration date
- **Declarative API**: Simple Custom Resource Definition (CRD) for managing ephemeral applications
- **REST API Server**: Go-based API server with Kubernetes ServiceAccount authentication for programmatic access
- **Web UI**: React-based dashboard (PatternFly) for managing ephemeral environments visually
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

The project is composed of three main components:

### Operator (`cmd/main.go`)

The core Kubernetes controller that watches `EphemeralApplication` custom resources and reconciles the desired state:

- **API Layer** (`api/v1alpha1`): Custom Resource Definitions
- **Controller Layer** (`internal/controller`): Reconciliation logic following Kubernetes controller patterns
- **ArgoCD Client** (`internal/argocd`): Interface-based ArgoCD interaction (follows Interface Segregation Principle)
- **Configuration** (`internal/config`): Environment-based configuration management

### API Server (`cmd/api/main.go`)

A REST API server that provides programmatic access to `EphemeralApplication` resources. It authenticates requests using Kubernetes ServiceAccount tokens (`TokenReview` API).

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/ephemeral-apps` | List all ephemeral applications |
| `GET` | `/api/v1/ephemeral-apps/{name}?namespace=` | Get a single application |
| `POST` | `/api/v1/ephemeral-apps/create` | Create a new application |
| `PATCH` | `/api/v1/ephemeral-apps/{name}?namespace=` | Update an application (e.g. extend expiration) |
| `DELETE` | `/api/v1/ephemeral-apps/{name}?namespace=` | Delete an application |
| `GET` | `/api/v1/metrics` | Get environment metrics (totals, by phase, recent) |
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness probe |

All `/api/v1/` endpoints require a valid `Authorization: Bearer <token>` header with a Kubernetes ServiceAccount token.

### Web UI (`web/`)

A React single-page application built with [PatternFly](https://www.patternfly.org/) that provides a visual dashboard for managing ephemeral environments. It communicates with the API server through an Nginx reverse proxy.

**Pages:**

- **Dashboard** (`/environments`): Lists all ephemeral environments with status, expiration, and metrics cards (total, active, creating, failed). Allows creating new environments via a modal form.
- **Environment Detail** (`/environments/:name`): Shows full details of a single environment including status, namespace, repository, path, revision, expiration, and status messages.
- **Settings** (`/settings`): Authentication page where users configure their Kubernetes ServiceAccount token (stored in `localStorage`).

**Tech stack:** React 18, TypeScript, PatternFly 5, React Router, TanStack React Query, Axios, Vite.

## Prerequisites

- Kubernetes cluster (v1.24+)
- ArgoCD installed and running in the cluster
- ArgoCD authentication token
- `kubectl` configured to access your cluster

## Local Setup (Recommended for Development)

The fastest way to get everything running locally is to use the provided `setup-local.sh` script. It provisions a minikube cluster, installs ArgoCD, builds all images, and deploys the operator, API server, and UI in one step.

### Requirements

| Tool | Version | Description |
|------|---------|-------------|
| [minikube](https://minikube.sigs.k8s.io/docs/start/) | Latest | Local Kubernetes cluster |
| [kubectl](https://kubernetes.io/docs/tasks/tools/) | v1.24+ | Kubernetes CLI |
| [Go](https://go.dev/doc/install) | 1.24+ | Build operator and API server |
| [Node.js](https://nodejs.org/) | 20+ | Build the Web UI (includes npm) |
| Docker or Podman | Latest | Container runtime for building images |

On Fedora/RHEL, Podman works out of the box. minikube will auto-detect the best available driver (KVM2 if `libvirt` is installed, Podman otherwise).

### Full Installation

```bash
./setup-local.sh
```

This will:
1. Check all prerequisites
2. Create a minikube cluster (`argo-ephemeral-local`)
3. Install ArgoCD (v2.14.20, matching the operator's `go.mod`)
4. Build all container images (operator, API server, UI)
5. Load images into minikube
6. Install CRDs, RBAC, and deploy all components

### Script Options

```bash
./setup-local.sh                # Full install (default)
./setup-local.sh --skip-build   # Reuse existing images, deploy/redeploy everything
./setup-local.sh --reload-images # Rebuild images, reload, and restart deployments
./setup-local.sh --teardown     # Destroy the minikube cluster
```

### Accessing the Services

After installation, use `kubectl port-forward` to access each service:

```bash
# ArgoCD UI (https://localhost:9090, user: admin)
kubectl port-forward svc/argocd-server -n argocd 9090:443 &

# Ephemeral Operator API (http://localhost:8080)
kubectl port-forward svc/argo-ephemeral-api-service -n argo-ephemeral-operator-system 8080:8080 &

# Ephemeral Operator UI (http://localhost:8888)
kubectl port-forward svc/argo-ephemeral-ui-service -n argo-ephemeral-operator-system 8888:80 &
```

### Authenticating the Web UI

The Web UI requires a Kubernetes ServiceAccount token. Create one and paste it in the Settings page (`http://localhost:8888/settings`):

```bash
# Create a ServiceAccount with permissions to manage EphemeralApplications
kubectl create serviceaccount ephemeral-user -n default
kubectl create clusterrolebinding ephemeral-user-binding \
  --clusterrole=cluster-admin \
  --serviceaccount=default:ephemeral-user

# Generate a token (valid for 24h)
kubectl create token ephemeral-user -n default --duration=24h
```

Copy the token output and paste it into the Settings page of the UI.

## Manual Installation

If you prefer to install components manually on an existing cluster, follow the steps below.

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
make deploy-operator
```

Or manually:

```bash
kubectl apply -f config/manager/namespace.yaml
kubectl apply -f config/rbac/service_account.yaml
kubectl apply -f config/rbac/role.yaml
kubectl apply -f config/rbac/role_binding.yaml
kubectl apply -f config/manager/deployment.yaml
```

### 4. Deploy the API Server

```bash
make deploy-api
```

Or manually:

```bash
kubectl apply -f config/api/
```

This deploys the API server with its ServiceAccount, RBAC (ClusterRole + ClusterRoleBinding), Deployment, and Service.

### 5. Deploy the Web UI

```bash
make deploy-ui
```

Or manually:

```bash
kubectl apply -f config/ui/
```

This deploys the Nginx-based UI with its Deployment, Service, and Ingress. Edit `config/ui/ingress.yaml` to set your domain before applying.

### 6. Deploy All Components at Once

```bash
make deploy
```

This runs `make install` (CRDs) followed by deploying the namespace, API server, and UI.

### 7. Verify Installation

```bash
kubectl get pods -n argo-ephemeral-operator-system
```

You should see pods for the operator (`controller-manager`), API server (`argo-ephemeral-api`), and UI (`argo-ephemeral-ui`).

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

### Extending Expiration Time

You can extend the expiration time of an ephemeral environment by updating the `expirationDate` field:

```bash
# Option 1: Edit interactively
kubectl edit ephapp my-feature-branch

# Option 2: Patch with new expiration date
kubectl patch ephapp my-feature-branch --type=merge -p '{"spec":{"expirationDate":"2025-11-05T23:59:59Z"}}'

# Verify the change
kubectl get ephapp my-feature-branch -o jsonpath='{.spec.expirationDate}'
```

The controller will detect the change in the next reconciliation cycle and update the expiration accordingly.

### Injecting Secrets

Ephemeral environments often need access to shared resources (databases, APIs, caches). Instead of hardcoding credentials in Git repositories, you can inject secrets into the ephemeral namespace.

#### Deployment Scenarios

**Scenario 1: Same Cluster Deployment**

If you deploy ephemeral environments in the **same cluster** where your production/shared services run, you can directly copy secrets from the namespaces where those services exist:

```yaml
spec:
  secrets:
  # Copy from the namespace where PostgreSQL is running
  - name: postgres-credentials
    sourceNamespace: databases
  
  # Copy from the namespace where Redis is running
  - name: redis-password
    sourceNamespace: cache
  
  # Copy from the namespace where Kafka is running
  - name: kafka-config
    sourceNamespace: messaging
```

This allows your ephemeral apps to use the same databases, queues, and services as your other environments.

**Scenario 2: Dedicated Ephemeral Cluster (Recommended)**

If you use a **dedicated cluster for ephemeral environments**, the best practice is to create a centralized namespace (e.g., `shared-secrets`) containing all the secrets your ephemeral applications might need:

```yaml
# One-time setup: Create shared-secrets namespace with all credentials
apiVersion: v1
kind: Namespace
metadata:
  name: shared-secrets
---
# Add all shared secrets here
apiVersion: v1
kind: Secret
metadata:
  name: postgres-dev
  namespace: shared-secrets
data:
  url: <base64-encoded-url>
  username: <base64-encoded-username>
  password: <base64-encoded-password>
---
apiVersion: v1
kind: Secret
metadata:
  name: redis-dev
  namespace: shared-secrets
# ... more secrets
```

Then, all EphemeralApplications copy from this centralized namespace:

```yaml
spec:
  secrets:
  - name: postgres-dev
    sourceNamespace: shared-secrets
  - name: redis-dev
    sourceNamespace: shared-secrets
  - name: api-keys-dev
    sourceNamespace: shared-secrets
```

**Benefits of the centralized approach**:
- ✅ Single place to manage all credentials for ephemeral environments
- ✅ Easy to rotate secrets (update once, affects all future ephemeral envs)
- ✅ Clear separation between production and ephemeral credentials
- ✅ Simplified RBAC (operator only needs access to one namespace)

#### Secret Injection Methods

**1. Copy from existing namespace**:

```yaml
spec:
  secrets:
  - name: postgres-credentials
    sourceNamespace: shared-secrets
  - name: api-keys
    sourceNamespace: shared-secrets
    targetName: external-api-keys  # Optional: rename in target
```

**2. Create inline** (useful for test data, non-sensitive config):

```yaml
spec:
  secrets:
  - name: test-config
    values:
      environment: "ephemeral-test"
      log-level: "debug"
      feature-flags: "new-ui:true,beta:false"
```

**How it works**:
1. Secrets are copied/created **before** ArgoCD deploys your application
2. Your deployments reference these secrets (already defined in your Git repo)
3. Secrets are automatically cleaned up when the namespace is deleted

**Example deployment in your Git repository**:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-server
spec:
  template:
    spec:
      containers:
      - name: api
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: postgres-credentials  # Injected by operator
              key: url
        - name: REDIS_HOST
          valueFrom:
            secretKeyRef:
              name: redis-creds
              key: host
```

See `examples/with-secrets.yaml` for a complete example.

### Injecting ConfigMaps

Similar to secrets, you can inject ConfigMaps with environment-specific configuration (service endpoints, feature flags, etc.):

**Copy from existing namespace**:

```yaml
spec:
  configMaps:
  - name: shared-config
    sourceNamespace: shared-configs
```

**Create inline** (recommended for env-specific values):

```yaml
spec:
  configMaps:
  - name: app-config
    data:
      DATABASE_HOST: "postgres.databases.svc.cluster.local"
      DATABASE_PORT: "5432"
      REDIS_HOST: "redis.cache.svc"
      API_ENDPOINT: "https://api-dev.example.com"
      LOG_LEVEL: "debug"
```

**Your deployments use these ConfigMaps**:

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: api
        envFrom:
        - configMapRef:
            name: app-config  # Injected by operator
```

This approach works with **any source type** (Helm, Kustomize, plain YAML) and keeps your Git repository clean from environment-specific values.

**⚠️ Important Limitations**

The argo-ephemeral-operator is designed for **cloud-native applications** following **GitOps principles** and deployed with **ArgoCD**. 

**This operator CANNOT override configuration if**:
- Environment variables are **hardcoded** in your Deployment YAML files (e.g., `env: - name: DB_HOST value: "prod-db"`)
- Configuration is **embedded in application code** instead of externalized
- Applications don't use ConfigMaps or Secrets for configuration

**For the operator to work correctly**, your applications must:
- ✅ Use `envFrom` with ConfigMaps/Secrets references
- ✅ Externalize all environment-specific configuration
- ✅ Be designed for cloud-native deployment patterns

**If your application has hardcoded configuration**, you must refactor it to use ConfigMaps/Secrets before using ephemeral environments. This is a best practice for cloud-native applications regardless of ephemeral environments.

See `examples/with-configmaps.yaml` for a complete example.

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

- Go 1.24+
- Node.js 20+ and npm (for the Web UI)
- Docker or Podman (for building images)
- Access to a Kubernetes cluster

### Building

```bash
# Build operator and API server binaries
make build

# Build only the operator
make build-operator

# Build only the API server
make build-api

# Build the Web UI (npm install + vite build)
make build-ui

# Run tests
make test

# Format code
make fmt

# Run linter
make vet
```

### Building Docker Images

```bash
# Build all images (operator, API, UI)
make docker-build

# Or individually
make docker-build-operator
make docker-build-api
make docker-build-ui
```

### Running Components Locally (Development Mode)

Each component can be run individually from your host during development:

```bash
# Set required environment variables for the operator
export ARGO_SERVER="argocd-server.argocd.svc.cluster.local"
export ARGO_PORT="443"
export ARGO_USERNAME="admin"
export ARGO_PASSWORD="your-argocd-password"
export ARGO_NAMESPACE="argocd"

# Run the operator (requires kubeconfig)
make run-operator

# Run the API server on port 8080 (requires kubeconfig)
make run-api

# Run the UI in Vite dev mode on port 3000 (proxies /api to localhost:8080)
make run-ui
```

When running the UI in development mode (`make run-ui`), Vite proxies all `/api` requests to `http://localhost:8080`, so make sure the API server is also running.

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

### Example 4: With Secret Injection

```yaml
apiVersion: ephemeral.argo.io/v1alpha1
kind: EphemeralApplication
metadata:
  name: app-with-secrets
spec:
  repoURL: https://github.com/company/app.git
  path: k8s
  targetRevision: main
  expirationDate: "2025-11-05T18:00:00Z"
  
  # Copy shared secrets
  secrets:
  - name: postgres-credentials
    sourceNamespace: shared-databases
  - name: redis-password
    sourceNamespace: shared-cache
  
  # Create test secrets inline
  - name: test-api-keys
    values:
      api-key: "test-key-123"
      api-secret: "test-secret-456"
  
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
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

## Project Structure

```
argo-ephemeral-operator/
├── api/v1alpha1/              # CRD types and deepcopy
├── cmd/
│   ├── main.go               # Operator entry point
│   └── api/main.go           # API server entry point
├── internal/
│   ├── apiserver/            # REST API server
│   │   ├── auth/             # Kubernetes TokenReview authenticator
│   │   ├── handlers/         # HTTP handlers (CRUD, metrics, health)
│   │   └── middleware/       # CORS, logging middleware
│   ├── argocd/               # ArgoCD gRPC client implementation
│   ├── config/               # Configuration management
│   └── controller/           # Reconciliation logic and state machine
├── web/                       # React Web UI (PatternFly)
│   ├── src/
│   │   ├── api/              # API client (Axios) and TypeScript types
│   │   ├── components/       # Reusable UI components
│   │   ├── hooks/            # React Query hooks
│   │   └── pages/            # Dashboard, EnvironmentDetail, Settings
│   ├── package.json
│   └── vite.config.ts        # Vite config (dev proxy to API on :8080)
├── config/                    # Kubernetes manifests
│   ├── crd/                  # Custom Resource Definition
│   ├── rbac/                 # Operator RBAC
│   ├── manager/              # Operator deployment + namespace
│   ├── api/                  # API server deployment, RBAC, service
│   ├── ui/                   # UI deployment, service, ingress
│   └── samples/              # Example resources
├── Dockerfile                 # Operator image (multi-stage)
├── Dockerfile.api             # API server image (multi-stage)
├── Dockerfile.ui              # UI image (Node build + Nginx)
├── nginx.conf                 # Nginx config (serves UI + proxies /api)
├── Makefile                   # Build and deployment automation
└── setup-local.sh             # One-command local setup with minikube
```

## Contributing

Contributions are welcome! Please ensure your code follows the established patterns and principles.

## License

See [LICENSE](LICENSE) file for details.
