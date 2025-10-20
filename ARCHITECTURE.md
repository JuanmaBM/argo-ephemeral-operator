# Architecture Documentation

## Overview

The argo-ephemeral-operator is a Kubernetes operator built using the Operator SDK pattern. It follows clean architecture principles and implements SOLID design patterns.

## Component Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes API Server                     │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       │ Watch/Reconcile
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                 Controller Manager                           │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  EphemeralApplication Reconciler                       │ │
│  │  - Watches EphemeralApplication CRs                    │ │
│  │  - Manages application lifecycle                       │ │
│  │  - Handles expiration logic                            │ │
│  └───────┬────────────────────────────────────────────────┘ │
│          │                                                    │
│          │ Uses                                               │
│          │                                                    │
│  ┌───────▼────────────────────────────────────────────────┐ │
│  │  ArgoCD Client (Interface)                             │ │
│  │  - CreateApplication()                                 │ │
│  │  - GetApplication()                                    │ │
│  │  - DeleteApplication()                                 │ │
│  │  - UpdateApplication()                                 │ │
│  └───────┬────────────────────────────────────────────────┘ │
│          │                                                    │
│  ┌───────▼────────────────────────────────────────────────┐ │
│  │  Application Builder                                   │ │
│  │  - Converts EphemeralApp to ArgoCD App                 │ │
│  │  - Builds sync policies                                │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                       │
                       │ Creates/Deletes
                       │
         ┌─────────────┴─────────────┐
         │                           │
┌────────▼──────────┐     ┌─────────▼──────────┐
│   ArgoCD Apps     │     │  Ephemeral         │
│   (in argocd ns)  │     │  Namespaces        │
└───────────────────┘     └────────────────────┘
```

## Directory Structure

```
argo-ephemeral-operator/
├── api/
│   └── v1alpha1/
│       ├── ephemeralapplication_types.go  # CRD definitions
│       └── groupversion_info.go           # API group metadata
├── cmd/
│   └── main.go                            # Entry point
├── config/
│   ├── crd/
│   │   └── bases/                         # CRD manifests
│   ├── manager/                           # Deployment manifests
│   ├── rbac/                              # RBAC manifests
│   ├── samples/                           # Example resources
│   └── kustomization.yaml                 # Kustomize config
├── internal/
│   ├── argocd/
│   │   └── client.go                      # ArgoCD client interface & impl
│   ├── config/
│   │   └── config.go                      # Configuration management
│   └── controller/
│       ├── ephemeralapplication_controller.go  # Main reconciler
│       ├── namegen.go                          # Namespace name generator
│       └── ephemeralapplication_controller_test.go  # Tests
├── .github/
│   └── workflows/
│       └── ci.yaml                        # CI/CD pipeline
├── Dockerfile                             # Container image definition
├── Makefile                               # Build automation
├── go.mod                                 # Go module definition
└── README.md                              # Documentation
```

## Key Design Decisions

### 1. Interface-Based Design (Dependency Inversion)

The controller depends on the `ArgoCD Client` interface, not the concrete implementation. This allows:
- Easy unit testing with mock implementations
- Flexibility to change underlying implementation
- Clear contract between components

```go
type Client interface {
    CreateApplication(ctx context.Context, app *argocdv1alpha1.Application) error
    GetApplication(ctx context.Context, namespace, name string) (*argocdv1alpha1.Application, error)
    DeleteApplication(ctx context.Context, namespace, name string) error
    UpdateApplication(ctx context.Context, app *argocdv1alpha1.Application) error
}
```

### 2. Separation of Concerns (Single Responsibility)

Each package has a single, well-defined responsibility:

- **api/**: Defines the API schema
- **controller/**: Handles reconciliation logic
- **argocd/**: Manages ArgoCD interactions
- **config/**: Manages configuration

### 3. Builder Pattern

The `ApplicationBuilder` separates the construction of ArgoCD Applications from the controller logic:

```go
type ApplicationBuilder struct {
    scheme *runtime.Scheme
}

func (b *ApplicationBuilder) BuildApplication(
    ephApp *ephemeralv1alpha1.EphemeralApplication,
    namespace string,
    argoNamespace string,
) *argocdv1alpha1.Application
```

### 4. Configuration Management

Environment-based configuration with validation:

```go
type Config struct {
    ArgoServer    string
    ArgoToken     string
    ArgoNamespace string
    // ...
}

func (c *Config) Validate() error
```

## Reconciliation Flow

```
1. Watch Event → EphemeralApplication created/updated
                 ↓
2. Check Expiration → If expired, delete resource
                 ↓
3. Check Phase → Determine current state
                 ↓
4. Phase Handling:
   - Pending   → Create namespace & ArgoCD Application
   - Creating  → Monitor ArgoCD sync status
   - Active    → Monitor health & check expiration
   - Failed    → Wait for manual intervention or expiration
                 ↓
5. Update Status → Reflect current state
                 ↓
6. Requeue → Schedule next reconciliation
```

## State Machine

```
┌─────────┐
│ Pending │ ──────┐
└────┬────┘       │
     │            │
     │ Create     │
     │            │
┌────▼─────┐     │ Error
│ Creating │ ────┼──────┐
└────┬─────┘     │      │
     │           │      │
     │ Synced    │      │
     │           │      │
┌────▼────┐      │   ┌──▼────┐
│ Active  │ ─────┘   │Failed │
└────┬────┘          └───────┘
     │
     │ Expired
     │
┌────▼─────┐
│ Expiring │
└──────────┘
```

## Extension Points

The operator is designed to be extensible:

### 1. Custom Name Generators

Implement the `NameGenerator` interface:

```go
type NameGenerator interface {
    GenerateNamespace(prefix, suffix string) string
}
```

### 2. Custom ArgoCD Clients

Implement the `Client` interface to support different ArgoCD backends or versions.

### 3. Additional Reconciliation Logic

The controller's modular design makes it easy to add new phases or behaviors.

## Security Considerations

1. **Least Privilege**: RBAC rules grant only necessary permissions
2. **Secret Management**: ArgoCD credentials stored in Kubernetes Secrets
3. **Non-Root Container**: Runs as user 65532
4. **Read-Only Root Filesystem**: Container filesystem is read-only
5. **Namespace Isolation**: Each ephemeral environment is isolated

## Performance Considerations

1. **Reconciliation Interval**: Configurable (default: 5 minutes)
2. **Leader Election**: Prevents multiple controller instances from conflicting
3. **Finalizers**: Ensures proper cleanup before deletion
4. **Incremental Updates**: Only updates status when changed

## Testing Strategy

1. **Unit Tests**: Test individual components (name generator, builders)
2. **Integration Tests**: Test controller with mock clients
3. **E2E Tests**: Test full workflow in a real cluster (future work)

## Observability

1. **Metrics**: Prometheus metrics at `:8080/metrics`
2. **Health Checks**: 
   - Liveness probe at `:8081/healthz`
   - Readiness probe at `:8081/readyz`
3. **Logging**: Structured logging using controller-runtime logger
4. **Status Conditions**: Rich status information in CRD

## Future Enhancements

1. **Webhook Validation**: Validate resources before creation
2. **Admission Controller**: Enforce policies on ephemeral applications
3. **Metrics Collection**: Expose metrics about ephemeral environments
4. **Notification System**: Alert before expiration
5. **Multi-Cluster**: Support deploying to remote clusters

