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

// copySecrets copies secrets from source namespaces to the target ephemeral namespace
func (r *EphemeralApplicationReconciler) copySecrets(
	ctx context.Context,
	ephApp *ephemeralv1alpha1.EphemeralApplication,
	targetNamespace string,
) error {
	logger := log.FromContext(ctx)

	if len(ephApp.Spec.Secrets) == 0 {
		return nil
	}

	logger.Info("copying secrets to ephemeral namespace", "count", len(ephApp.Spec.Secrets))

	for _, secretRef := range ephApp.Spec.Secrets {
		if err := r.copySecret(ctx, secretRef, targetNamespace, ephApp); err != nil {
			return fmt.Errorf("failed to copy secret %s from %s: %w",
				secretRef.Name, secretRef.SourceNamespace, err)
		}
	}

	return nil
}

// copySecret copies a single secret from source to target namespace
func (r *EphemeralApplicationReconciler) copySecret(
	ctx context.Context,
	secretRef ephemeralv1alpha1.SecretReference,
	targetNamespace string,
	ephApp *ephemeralv1alpha1.EphemeralApplication,
) error {
	logger := log.FromContext(ctx)

	// Get the source secret or use the values if provided
	sourceSecret := &corev1.Secret{}
	if len(secretRef.Values) == 0 {
		err := r.Get(ctx, client.ObjectKey{
			Namespace: secretRef.SourceNamespace,
			Name:      secretRef.Name,
		}, sourceSecret)
		if err != nil {
			return fmt.Errorf("failed to get source secret: %w", err)
		}
	} else {
		sourceSecret.Data = make(map[string][]byte)
		for key, value := range secretRef.Values {
			sourceSecret.Data[key] = []byte(value)
		}
		sourceSecret.Type = corev1.SecretTypeOpaque
	}

	// Determine target secret name
	targetName := secretRef.Name
	if secretRef.TargetName != "" {
		targetName = secretRef.TargetName
	}

	logger.Info("copying secret",
		"sourceNamespace", secretRef.SourceNamespace,
		"sourceName", secretRef.Name,
		"targetNamespace", targetNamespace,
		"targetName", targetName)

	// Create the target secret
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "argo-ephemeral-operator",
		"ephemeral.argo.io/owner":      ephApp.Name,
	}

	annotations := map[string]string{}

	// Add different labels for inline vs copied secrets
	if len(secretRef.Values) > 0 {
		labels["ephemeral.argo.io/inline"] = "true"
	} else {
		labels["ephemeral.argo.io/copied-from"] = secretRef.SourceNamespace
		labels["ephemeral.argo.io/source-name"] = secretRef.Name
		annotations["ephemeral.argo.io/source-namespace"] = secretRef.SourceNamespace
		annotations["ephemeral.argo.io/source-secret"] = secretRef.Name
	}

	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        targetName,
			Namespace:   targetNamespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Type: sourceSecret.Type,
		Data: sourceSecret.Data,
	}

	// Create or update the secret
	err := r.Create(ctx, targetSecret)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Update if already exists
			logger.Info("secret already exists, updating", "name", targetName)
			existingSecret := &corev1.Secret{}
			if err := r.Get(ctx, client.ObjectKey{
				Namespace: targetNamespace,
				Name:      targetName,
			}, existingSecret); err != nil {
				return err
			}

			existingSecret.Data = sourceSecret.Data
			existingSecret.Type = sourceSecret.Type

			if err := r.Update(ctx, existingSecret); err != nil {
				return fmt.Errorf("failed to update secret: %w", err)
			}

			return nil
		}
		return fmt.Errorf("failed to create secret: %w", err)
	}

	return nil
}

// buildCopiedSecretsList creates a human-readable list of copied secrets
func (r *EphemeralApplicationReconciler) buildCopiedSecretsList(secrets []ephemeralv1alpha1.SecretReference) []string {
	if len(secrets) == 0 {
		return nil
	}

	copiedList := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		targetName := secret.Name
		if secret.TargetName != "" {
			targetName = secret.TargetName
		}
		copiedList = append(copiedList, fmt.Sprintf("%s/%s -> %s", secret.SourceNamespace, secret.Name, targetName))
	}

	return copiedList
}
