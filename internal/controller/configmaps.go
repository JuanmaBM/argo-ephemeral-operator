package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ephemeralv1alpha1 "github.com/jbarea/argo-ephemeral-operator/api/v1alpha1"
)

// copyConfigMaps copies configmaps from source namespaces or creates them inline
func (r *EphemeralApplicationReconciler) copyConfigMaps(
	ctx context.Context,
	ephApp *ephemeralv1alpha1.EphemeralApplication,
	targetNamespace string,
) error {
	logger := log.FromContext(ctx)

	if len(ephApp.Spec.ConfigMaps) == 0 {
		return nil
	}

	logger.Info("copying configmaps to ephemeral namespace", "count", len(ephApp.Spec.ConfigMaps))

	for _, cmRef := range ephApp.Spec.ConfigMaps {
		if err := r.copyConfigMap(ctx, cmRef, targetNamespace, ephApp); err != nil {
			return fmt.Errorf("failed to copy configmap %s: %w", cmRef.Name, err)
		}
	}

	return nil
}

// copyConfigMap copies a single configmap from source or creates from inline data
func (r *EphemeralApplicationReconciler) copyConfigMap(
	ctx context.Context,
	cmRef ephemeralv1alpha1.ConfigMapReference,
	targetNamespace string,
	ephApp *ephemeralv1alpha1.EphemeralApplication,
) error {
	logger := log.FromContext(ctx)

	var cmData map[string]string

	// Check if creating from inline data or copying from source
	if len(cmRef.Data) > 0 {
		// Use inline data
		logger.Info("creating configmap from inline data",
			"name", cmRef.Name,
			"targetNamespace", targetNamespace)
		cmData = cmRef.Data
	} else {
		// Copy from source namespace
		sourceCM := &corev1.ConfigMap{}
		err := r.Get(ctx, client.ObjectKey{
			Namespace: cmRef.SourceNamespace,
			Name:      cmRef.Name,
		}, sourceCM)
		if err != nil {
			return fmt.Errorf("failed to get source configmap: %w", err)
		}

		logger.Info("copying configmap",
			"sourceNamespace", cmRef.SourceNamespace,
			"sourceName", cmRef.Name,
			"targetNamespace", targetNamespace)

		cmData = sourceCM.Data
	}

	// Prepare labels
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "argo-ephemeral-operator",
		"ephemeral.argo.io/owner":      ephApp.Name,
	}

	annotations := map[string]string{}

	// Add different labels for inline vs copied
	if len(cmRef.Data) > 0 {
		labels["ephemeral.argo.io/inline"] = "true"
	} else {
		labels["ephemeral.argo.io/copied-from"] = cmRef.SourceNamespace
		labels["ephemeral.argo.io/source-name"] = cmRef.Name
		annotations["ephemeral.argo.io/source-namespace"] = cmRef.SourceNamespace
		annotations["ephemeral.argo.io/source-configmap"] = cmRef.Name
	}

	// Create the target configmap
	targetCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        cmRef.Name,
			Namespace:   targetNamespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: cmData,
	}

	// Create or update
	err := r.Create(ctx, targetCM)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Update if already exists
			logger.Info("configmap already exists, updating", "name", cmRef.Name)
			existingCM := &corev1.ConfigMap{}
			if err := r.Get(ctx, client.ObjectKey{
				Namespace: targetNamespace,
				Name:      cmRef.Name,
			}, existingCM); err != nil {
				return err
			}

			existingCM.Data = cmData

			if err := r.Update(ctx, existingCM); err != nil {
				return fmt.Errorf("failed to update configmap: %w", err)
			}

			return nil
		}
		return fmt.Errorf("failed to create configmap: %w", err)
	}

	return nil
}

// buildCopiedConfigMapsList creates a human-readable list of copied configmaps
func (r *EphemeralApplicationReconciler) buildCopiedConfigMapsList(configMaps []ephemeralv1alpha1.ConfigMapReference) []string {
	if len(configMaps) == 0 {
		return nil
	}

	copiedList := make([]string, 0, len(configMaps))
	for _, cm := range configMaps {
		if len(cm.Data) > 0 {
			copiedList = append(copiedList, fmt.Sprintf("%s (inline)", cm.Name))
		} else {
			copiedList = append(copiedList, fmt.Sprintf("%s/%s", cm.SourceNamespace, cm.Name))
		}
	}

	return copiedList
}
