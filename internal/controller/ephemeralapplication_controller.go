package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	ephemeralv1alpha1 "github.com/jbarea/argo-ephemeral-operator/api/v1alpha1"
	"github.com/jbarea/argo-ephemeral-operator/internal/argocd"
	"github.com/jbarea/argo-ephemeral-operator/internal/config"
)

const (
	finalizerName = "ephemeral.argo.io/finalizer"
)

// EphemeralApplicationReconciler reconciles a EphemeralApplication object
type EphemeralApplicationReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	ArgoClient    argocd.Client
	Config        *config.Config
	NameGenerator NameGenerator
}

// NameGenerator generates unique namespace names
type NameGenerator interface {
	GenerateNamespace(prefix, suffix string) string
}

// +kubebuilder:rbac:groups=ephemeral.argo.io,resources=ephemeralapplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ephemeral.argo.io,resources=ephemeralapplications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ephemeral.argo.io,resources=ephemeralapplications/finalizers,verbs=update
// +kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;delete

// Reconcile is the main reconciliation loop
func (r *EphemeralApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the EphemeralApplication
	ephApp := &ephemeralv1alpha1.EphemeralApplication{}
	if err := r.Get(ctx, req.NamespacedName, ephApp); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch EphemeralApplication")
		return ctrl.Result{}, err
	}

	// Check if the resource is being deleted
	if !ephApp.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, ephApp)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(ephApp, finalizerName) {
		controllerutil.AddFinalizer(ephApp, finalizerName)
		if err := r.Update(ctx, ephApp); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check if expired
	if r.isExpired(ephApp) {
		return r.handleExpiration(ctx, ephApp)
	}

	// Handle based on current phase
	switch ephApp.Status.Phase {
	case "", ephemeralv1alpha1.PhasePending:
		return r.handlePendingPhase(ctx, ephApp)
	case ephemeralv1alpha1.PhaseCreating:
		return r.handleCreatingPhase(ctx, ephApp)
	case ephemeralv1alpha1.PhaseActive:
		return r.handleActivePhase(ctx, ephApp)
	case ephemeralv1alpha1.PhaseFailed:
		return r.handleFailedPhase(ctx, ephApp)
	default:
		return ctrl.Result{}, nil
	}
}

// handlePendingPhase handles the pending phase
func (r *EphemeralApplicationReconciler) handlePendingPhase(ctx context.Context, ephApp *ephemeralv1alpha1.EphemeralApplication) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("handling pending phase")

	// Generate namespace name
	namespace := r.NameGenerator.GenerateNamespace(r.getNamespacePrefix(ephApp), ephApp.Name)

	// Create namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "argo-ephemeral-operator",
				"ephemeral.argo.io/owner":      ephApp.Name,
			},
		},
	}

	if err := r.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "failed to create namespace")
		return r.updateStatusWithError(ctx, ephApp, ephemeralv1alpha1.PhaseFailed, "Failed to create namespace", err)
	}

	// Build and create ArgoCD Application
	argoApp, err := r.ArgoClient.CreateApplication(ctx, &application.ApplicationCreateRequest{
		Application: &v1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name: ephApp.Name,
			},
			Spec: v1alpha1.ApplicationSpec{
				Project: "default",
				Source: &v1alpha1.ApplicationSource{
					RepoURL:        ephApp.Spec.RepoURL,
					Path:           ephApp.Spec.Path,
					TargetRevision: ephApp.Spec.TargetRevision,
				},
				Destination: v1alpha1.ApplicationDestination{
					Namespace: namespace,
					Server:    "https://kubernetes.default.svc",
				},
				SyncPolicy: &v1alpha1.SyncPolicy{
					Automated: &v1alpha1.SyncPolicyAutomated{
						Prune:    true,
						SelfHeal: true,
					},
				},
			},
		},
	})

	if err != nil {
		logger.Error(err, "failed to create ArgoCD application")
		return r.updateStatusWithError(ctx, ephApp, ephemeralv1alpha1.PhaseFailed, "Failed to create ArgoCD application", err)
	}

	ephApp.Status.Phase = ephemeralv1alpha1.PhaseCreating
	ephApp.Status.Namespace = namespace
	ephApp.Status.ArgoApplicationName = argoApp.Name
	ephApp.Status.Message = "ArgoCD application created successfully"
	r.setCondition(ephApp, "Ready", metav1.ConditionFalse, "Creating", "Creating ephemeral environment")

	if err := r.Status().Update(ctx, ephApp); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// handleCreatingPhase handles the creating phase
func (r *EphemeralApplicationReconciler) handleCreatingPhase(ctx context.Context, ephApp *ephemeralv1alpha1.EphemeralApplication) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("handling creating phase")

	// Check if ArgoCD Application exists and is synced
	appQuery := application.ApplicationQuery{
		Name:         &ephApp.Status.ArgoApplicationName,
		AppNamespace: &ephApp.Status.Namespace,
	}
	argoApp, err := r.ArgoClient.GetApplication(ctx, appQuery)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.updateStatusWithError(ctx, ephApp, ephemeralv1alpha1.PhaseFailed, "ArgoCD application not found", err)
		}
		return ctrl.Result{}, err
	}

	// Check sync status
	if argoApp.Status.Sync.Status == "Synced" && argoApp.Status.Health.Status == "Healthy" {
		ephApp.Status.Phase = ephemeralv1alpha1.PhaseActive
		ephApp.Status.Message = "Ephemeral environment is active"
		now := metav1.Now()
		ephApp.Status.LastSyncTime = &now
		r.setCondition(ephApp, "Ready", metav1.ConditionTrue, "Active", "Ephemeral environment is active and healthy")

		if err := r.Status().Update(ctx, ephApp); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: r.Config.ReconcileInterval}, nil
	}

	// Still creating, requeue
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// handleActivePhase handles the active phase
func (r *EphemeralApplicationReconciler) handleActivePhase(ctx context.Context, ephApp *ephemeralv1alpha1.EphemeralApplication) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("handling active phase")

	// Verify ArgoCD Application still exists and is healthy
	appQuery := application.ApplicationQuery{
		Name:         &ephApp.Status.ArgoApplicationName,
		AppNamespace: &ephApp.Status.Namespace,
	}
	argoApp, err := r.ArgoClient.GetApplication(ctx, appQuery)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.updateStatusWithError(ctx, ephApp, ephemeralv1alpha1.PhaseFailed, "ArgoCD application disappeared", err)
		}
		return ctrl.Result{}, err
	}

	// Update sync time if synced
	if argoApp.Status.Sync.Status == "Synced" {
		now := metav1.Now()
		ephApp.Status.LastSyncTime = &now
		if err := r.Status().Update(ctx, ephApp); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Requeue for next check
	return ctrl.Result{RequeueAfter: r.Config.ReconcileInterval}, nil
}

// handleFailedPhase handles the failed phase
func (r *EphemeralApplicationReconciler) handleFailedPhase(ctx context.Context, ephApp *ephemeralv1alpha1.EphemeralApplication) (ctrl.Result, error) {
	// In failed state, just requeue to check expiration
	return ctrl.Result{RequeueAfter: r.Config.ReconcileInterval}, nil
}

// handleExpiration handles expired applications
func (r *EphemeralApplicationReconciler) handleExpiration(ctx context.Context, ephApp *ephemeralv1alpha1.EphemeralApplication) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("handling expiration", "expirationDate", ephApp.Spec.ExpirationDate)

	ephApp.Status.Phase = ephemeralv1alpha1.PhaseExpiring
	ephApp.Status.Message = "Ephemeral environment has expired and is being deleted"
	r.setCondition(ephApp, "Ready", metav1.ConditionFalse, "Expiring", "Environment has expired")

	if err := r.Status().Update(ctx, ephApp); err != nil {
		return ctrl.Result{}, err
	}

	// Delete the EphemeralApplication (finalizer will clean up)
	if err := r.Delete(ctx, ephApp); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// handleDeletion handles cleanup when the resource is being deleted
func (r *EphemeralApplicationReconciler) handleDeletion(ctx context.Context, ephApp *ephemeralv1alpha1.EphemeralApplication) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(ephApp, finalizerName) {
		// Delete ArgoCD Application
		if ephApp.Status.ArgoApplicationName != "" {
			logger.Info("deleting ArgoCD application", "name", ephApp.Status.ArgoApplicationName)
			if err := r.ArgoClient.DeleteApplication(ctx, ephApp.Status.ArgoApplicationName, ephApp.Status.Namespace); err != nil {
				if !errors.IsNotFound(err) {
					logger.Error(err, "failed to delete ArgoCD application")
					return ctrl.Result{}, err
				}
			}
		}

		// Delete namespace
		if ephApp.Status.Namespace != "" {
			logger.Info("deleting namespace", "namespace", ephApp.Status.Namespace)
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ephApp.Status.Namespace,
				},
			}
			if err := r.Delete(ctx, ns); err != nil {
				if !errors.IsNotFound(err) {
					logger.Error(err, "failed to delete namespace")
					return ctrl.Result{}, err
				}
			}
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(ephApp, finalizerName)
		if err := r.Update(ctx, ephApp); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// isExpired checks if the application has expired
func (r *EphemeralApplicationReconciler) isExpired(ephApp *ephemeralv1alpha1.EphemeralApplication) bool {
	return time.Now().After(ephApp.Spec.ExpirationDate.Time)
}

// getNamespacePrefix returns the namespace prefix
func (r *EphemeralApplicationReconciler) getNamespacePrefix(ephApp *ephemeralv1alpha1.EphemeralApplication) string {
	if ephApp.Spec.NamespacePrefix != "" {
		return ephApp.Spec.NamespacePrefix
	}
	return "ephemeral"
}

// updateStatusWithError updates the status with an error
func (r *EphemeralApplicationReconciler) updateStatusWithError(
	ctx context.Context,
	ephApp *ephemeralv1alpha1.EphemeralApplication,
	phase ephemeralv1alpha1.EphemeralApplicationPhase,
	message string,
	err error,
) (ctrl.Result, error) {
	ephApp.Status.Phase = phase
	ephApp.Status.Message = fmt.Sprintf("%s: %v", message, err)
	r.setCondition(ephApp, "Ready", metav1.ConditionFalse, "Error", message)

	if updateErr := r.Status().Update(ctx, ephApp); updateErr != nil {
		return ctrl.Result{}, updateErr
	}

	return ctrl.Result{RequeueAfter: r.Config.ReconcileInterval}, nil
}

// setCondition sets a condition on the EphemeralApplication
func (r *EphemeralApplicationReconciler) setCondition(
	ephApp *ephemeralv1alpha1.EphemeralApplication,
	conditionType string,
	status metav1.ConditionStatus,
	reason, message string,
) {
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: ephApp.Generation,
	}

	// Find and update existing condition or append new one
	found := false
	for i, c := range ephApp.Status.Conditions {
		if c.Type == conditionType {
			ephApp.Status.Conditions[i] = condition
			found = true
			break
		}
	}
	if !found {
		ephApp.Status.Conditions = append(ephApp.Status.Conditions, condition)
	}
}

// SetupWithManager sets up the controller with the Manager
func (r *EphemeralApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ephemeralv1alpha1.EphemeralApplication{}).
		Complete(r)
}
