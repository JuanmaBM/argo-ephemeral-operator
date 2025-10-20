package argocd

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ephemeralv1alpha1 "github.com/jbarea/argo-ephemeral-operator/api/v1alpha1"
)

// ArgoCD Application types (simplified for this implementation)
// In production, you would import these from github.com/argoproj/argo-cd/v2
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ApplicationSpec   `json:"spec"`
	Status            ApplicationStatus `json:"status,omitempty"`
}

type ApplicationSpec struct {
	Project     string                 `json:"project"`
	Source      *ApplicationSource     `json:"source,omitempty"`
	Destination ApplicationDestination `json:"destination"`
	SyncPolicy  *SyncPolicy            `json:"syncPolicy,omitempty"`
}

type ApplicationSource struct {
	RepoURL        string `json:"repoURL"`
	Path           string `json:"path,omitempty"`
	TargetRevision string `json:"targetRevision,omitempty"`
}

type ApplicationDestination struct {
	Server    string `json:"server,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type SyncPolicy struct {
	Automated   *SyncPolicyAutomated `json:"automated,omitempty"`
	SyncOptions SyncOptions          `json:"syncOptions,omitempty"`
}

type SyncPolicyAutomated struct {
	Prune    bool `json:"prune,omitempty"`
	SelfHeal bool `json:"selfHeal,omitempty"`
}

type SyncOptions []string

type ApplicationStatus struct {
	Sync   SyncStatus   `json:"sync,omitempty"`
	Health HealthStatus `json:"health,omitempty"`
}

type SyncStatus struct {
	Status string `json:"status,omitempty"`
}

type HealthStatus struct {
	Status string `json:"status,omitempty"`
}

// Client defines the interface for interacting with ArgoCD
type Client interface {
	// CreateApplication creates an ArgoCD Application
	CreateApplication(ctx context.Context, app *Application) error
	// GetApplication retrieves an ArgoCD Application
	GetApplication(ctx context.Context, namespace, name string) (*Application, error)
	// DeleteApplication deletes an ArgoCD Application
	DeleteApplication(ctx context.Context, namespace, name string) error
	// UpdateApplication updates an ArgoCD Application
	UpdateApplication(ctx context.Context, app *Application) error
}

// clientImpl implements the Client interface
type clientImpl struct {
	k8sClient client.Client
	namespace string
}

// NewClient creates a new ArgoCD client
func NewClient(k8sClient client.Client, namespace string) Client {
	return &clientImpl{
		k8sClient: k8sClient,
		namespace: namespace,
	}
}

// CreateApplication creates an ArgoCD Application
func (c *clientImpl) CreateApplication(ctx context.Context, app *Application) error {
	if err := c.k8sClient.Create(ctx, app); err != nil {
		return fmt.Errorf("failed to create ArgoCD application: %w", err)
	}
	return nil
}

// GetApplication retrieves an ArgoCD Application
func (c *clientImpl) GetApplication(ctx context.Context, namespace, name string) (*Application, error) {
	app := &Application{}
	if err := c.k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, app); err != nil {
		return nil, fmt.Errorf("failed to get ArgoCD application: %w", err)
	}
	return app, nil
}

// DeleteApplication deletes an ArgoCD Application
func (c *clientImpl) DeleteApplication(ctx context.Context, namespace, name string) error {
	app := &Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := c.k8sClient.Delete(ctx, app); err != nil {
		return fmt.Errorf("failed to delete ArgoCD application: %w", err)
	}
	return nil
}

// UpdateApplication updates an ArgoCD Application
func (c *clientImpl) UpdateApplication(ctx context.Context, app *Application) error {
	if err := c.k8sClient.Update(ctx, app); err != nil {
		return fmt.Errorf("failed to update ArgoCD application: %w", err)
	}
	return nil
}

// ApplicationBuilder builds ArgoCD Applications from EphemeralApplications
type ApplicationBuilder struct {
	scheme *runtime.Scheme
}

// NewApplicationBuilder creates a new ApplicationBuilder
func NewApplicationBuilder(scheme *runtime.Scheme) *ApplicationBuilder {
	return &ApplicationBuilder{
		scheme: scheme,
	}
}

// BuildApplication builds an ArgoCD Application from an EphemeralApplication
func (b *ApplicationBuilder) BuildApplication(
	ephApp *ephemeralv1alpha1.EphemeralApplication,
	namespace string,
	argoNamespace string,
) *Application {
	appName := fmt.Sprintf("ephemeral-%s", ephApp.Name)

	app := &Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: argoNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "argo-ephemeral-operator",
				"ephemeral.argo.io/owner":      ephApp.Name,
			},
		},
		Spec: ApplicationSpec{
			Project: "default",
			Source: &ApplicationSource{
				RepoURL:        ephApp.Spec.RepoURL,
				Path:           ephApp.Spec.Path,
				TargetRevision: b.getTargetRevision(ephApp),
			},
			Destination: ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: namespace,
			},
			SyncPolicy: b.buildSyncPolicy(ephApp),
		},
	}

	return app
}

// getTargetRevision returns the target revision or default
func (b *ApplicationBuilder) getTargetRevision(ephApp *ephemeralv1alpha1.EphemeralApplication) string {
	if ephApp.Spec.TargetRevision != "" {
		return ephApp.Spec.TargetRevision
	}
	return "HEAD"
}

// buildSyncPolicy builds the sync policy from the EphemeralApplication
func (b *ApplicationBuilder) buildSyncPolicy(ephApp *ephemeralv1alpha1.EphemeralApplication) *SyncPolicy {
	if ephApp.Spec.SyncPolicy == nil {
		return &SyncPolicy{
			Automated: &SyncPolicyAutomated{
				Prune:    true,
				SelfHeal: true,
			},
			SyncOptions: SyncOptions{
				"CreateNamespace=true",
			},
		}
	}

	policy := &SyncPolicy{
		SyncOptions: SyncOptions{
			"CreateNamespace=true",
		},
	}

	if ephApp.Spec.SyncPolicy.Automated != nil {
		policy.Automated = &SyncPolicyAutomated{
			Prune:    ephApp.Spec.SyncPolicy.Automated.Prune,
			SelfHeal: ephApp.Spec.SyncPolicy.Automated.SelfHeal,
		}
	}

	return policy
}
