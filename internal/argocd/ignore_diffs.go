package argocd

import (
	v1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	ephemeralv1alpha1 "github.com/jbarea/argo-ephemeral-operator/api/v1alpha1"
)

// BuildIgnoreDifferences constructs the ignore differences list for injected resources
func BuildIgnoreDifferences(ephApp *ephemeralv1alpha1.EphemeralApplication) []v1alpha1.ResourceIgnoreDifferences {
	ignoreDiffs := []v1alpha1.ResourceIgnoreDifferences{}

	// Ignore injected secrets
	for _, secret := range ephApp.Spec.Secrets {
		name := secret.Name
		if secret.TargetName != "" {
			name = secret.TargetName
		}
		ignoreDiffs = append(ignoreDiffs, v1alpha1.ResourceIgnoreDifferences{
			Group: "",
			Kind:  "Secret",
			Name:  name,
			// Ignore everything in the secret to prevent ArgoCD from reverting it
			JSONPointers: []string{"/data", "/stringData"},
		})
	}

	// Ignore injected configmaps
	for _, cm := range ephApp.Spec.ConfigMaps {
		ignoreDiffs = append(ignoreDiffs, v1alpha1.ResourceIgnoreDifferences{
			Group: "",
			Kind:  "ConfigMap",
			Name:  cm.Name,
			// Ignore data to prevent ArgoCD from reverting it
			JSONPointers: []string{"/data", "/binaryData"},
		})
	}

	return ignoreDiffs
}
