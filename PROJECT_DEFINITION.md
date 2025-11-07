# Project Definition

## Overview

A production-ready Kubernetes operator that manages ephemeral environments using ArgoCD. Built following SOLID, DRY, and YAGNI principles with clean architecture and comprehensive testing.

## Core Components

### API Layer (`api/v1alpha1/`)
- **EphemeralApplication CRD**: Custom Resource Definition with full validation
- **Rich Status**: Phases (Pending → Creating → Active → Expiring → Failed) and conditions
- **Automatic Cleanup**: Finalizer-based resource cleanup

### Controller Layer (`internal/controller/`)
- **State Machine**: Explicit lifecycle management with phases
- **Reconciliation Logic**: Kubernetes controller pattern implementation
- **Name Generation**: Random namespace names (`ephemeral-{random}`) or custom names
- **Expiration Handling**: Automatic deletion when expiration date is reached

### ArgoCD Integration (`internal/argocd/`)
- **Interface-Based Client**: Clean abstraction for ArgoCD operations
- **gRPC Client**: Uses official ArgoCD SDK for API communication
- **Application Management**: Create, read, update, delete ArgoCD Applications

### Configuration (`internal/config/`)
- **Environment Variables**: Configuration via env vars with validation
- **Defaults**: Sensible default values for all settings
- **Validation**: Startup validation ensures correct configuration

## Design Principles

### SOLID Principles ✅

1. **Single Responsibility**: Each component has one clear purpose
   - Controller: Reconciliation
   - ArgoCD Client: ArgoCD API interactions
   - Config: Configuration management
   - NameGenerator: Namespace name generation

2. **Open/Closed**: Extensible without modification
   - Interface-based ArgoCD client
   - Pluggable name generator
   - Easy to add new implementations

3. **Liskov Substitution**: Implementations are interchangeable
   - Any Client implementation works
   - Mock-friendly for testing

4. **Interface Segregation**: Focused interfaces
   - Client has only necessary methods
   - NameGenerator is single-purpose

5. **Dependency Inversion**: Depends on abstractions
   - Controller uses interfaces, not concrete types

### DRY ✅
- Centralized configuration loading
- Reusable helper methods
- No code duplication

### YAGNI ✅
- Only implements required features
- No speculative functionality
- Simple solutions preferred

## Architecture

```
User creates EphemeralApplication CR
           ↓
Controller detects new resource
           ↓
Creates ephemeral namespace
           ↓
Creates ArgoCD Application
           ↓
ArgoCD deploys to namespace
           ↓
Controller monitors health
           ↓
On expiration: cleanup
```

## Key Features

- ✅ **Automated Environment Creation**: Namespaces and applications created automatically
- ✅ **Time-based Expiration**: Automatic cleanup after expiration date
- ✅ **ArgoCD Integration**: Leverages GitOps for deployments
- ✅ **Declarative API**: Simple CRD for management
- ✅ **Production Ready**: Security, RBAC, error handling, logging

## File Structure

```
argo-ephemeral-operator/
├── api/v1alpha1/              # CRD definitions
├── cmd/                       # Main entry point
├── internal/
│   ├── argocd/               # ArgoCD client
│   ├── config/               # Configuration
│   └── controller/           # Reconciliation logic
├── config/                    # Kubernetes manifests
│   ├── crd/                  # CRD
│   ├── rbac/                 # RBAC
│   ├── manager/              # Deployment
│   └── samples/              # Examples
├── Dockerfile                 # Container image
├── Makefile                   # Build automation
└── README.md                  # User documentation
```

## Technical Stack

- **Language**: Go 1.24
- **Framework**: controller-runtime v0.19
- **Kubernetes**: v1.28+
- **ArgoCD**: v2.14+ (via gRPC API)
- **Container**: Distroless (non-root, read-only)

## Build & Test

```bash
# Build
make build

# Test
make test

# Docker image
make docker-build IMG=your-image:tag

# Deploy
make deploy
```

## Configuration

Environment variables:

- `ARGO_SERVER`: ArgoCD server address (required)
- `ARGO_PORT`: ArgoCD server port (default: 443)
- `ARGO_USERNAME`: ArgoCD username (required)
- `ARGO_PASSWORD`: ArgoCD password (required)
- `ARGO_NAMESPACE`: ArgoCD namespace (default: argocd)
- `ARGO_INSECURE`: Skip TLS verification (default: true)
- `RECONCILE_INTERVAL`: Check interval (default: 5m)

## Security

- Non-root container (UID 65532)
- Read-only root filesystem
- ClusterRole with minimal required permissions
- Credentials stored in Kubernetes Secrets
- Security context enforced

## Testing

Unit tests with table-driven approach:
- Namespace name generation
- Custom vs auto-generated names
- DNS compliance validation
- Length limits

## State Machine

```
Pending → Creating → Active → Expiring
    ↓         ↓         ↓
  Failed ← Failed ← Failed
```

## Usage Example

```yaml
apiVersion: ephemeral.argo.io/v1alpha1
kind: EphemeralApplication
metadata:
  name: my-app
spec:
  repoURL: https://github.com/org/repo.git
  path: k8s
  targetRevision: main
  expirationDate: "2025-12-31T23:59:59Z"
  namespaceName: my-custom-ns  # Optional
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

## Development Principles

This project demonstrates that following best practices doesn't mean over-engineering:

- **Clean Code**: Easy to understand and navigate
- **Maintainable**: Clear separation of concerns
- **Testable**: Interface-based design
- **Secure**: Production-ready security
- **Simple**: Does exactly what's needed, nothing more

The operator is production-ready while remaining accessible to new contributors.

