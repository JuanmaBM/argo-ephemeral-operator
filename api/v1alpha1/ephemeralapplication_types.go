package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EphemeralApplicationSpec defines the desired state of EphemeralApplication
type EphemeralApplicationSpec struct {
	// RepoURL is the Git repository URL containing the application manifests
	// +kubebuilder:validation:Required
	RepoURL string `json:"repoURL"`

	// Path is the path within the Git repository
	// +kubebuilder:validation:Required
	Path string `json:"path"`

	// TargetRevision is the Git revision to deploy (branch, tag, or commit)
	// +kubebuilder:default:="HEAD"
	TargetRevision string `json:"targetRevision,omitempty"`

	// ExpirationDate is the date when this ephemeral environment should be deleted
	// Format: RFC3339 (e.g., "2024-12-31T23:59:59Z")
	// +kubebuilder:validation:Required
	ExpirationDate metav1.Time `json:"expirationDate"`

	// NamespaceName is the name for the ephemeral namespace
	// If not provided, a random name will be generated: ephemeral-{random}
	// +optional
	NamespaceName string `json:"namespaceName,omitempty"`

	// Secrets to copy from other namespaces into the ephemeral namespace
	// Allows applications to access shared credentials (databases, APIs, etc.)
	// +optional
	Secrets []SecretReference `json:"secrets,omitempty"`

	// ConfigMaps to copy from other namespaces or create inline
	// Useful for injecting environment-specific configuration
	// +optional
	ConfigMaps []ConfigMapReference `json:"configMaps,omitempty"`

	// SyncPolicy defines how the application should be synced
	// +optional
	SyncPolicy *SyncPolicy `json:"syncPolicy,omitempty"`
}

// ConfigMapReference defines a configmap to copy or create
type ConfigMapReference struct {
	// Name of the configmap
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// SourceNamespace where the configmap exists (for copying)
	// Mutually exclusive with Data
	// +optional
	SourceNamespace string `json:"sourceNamespace,omitempty"`

	// Data to create a new configmap inline
	// Mutually exclusive with SourceNamespace
	// +optional
	Data map[string]string `json:"data,omitempty"`
}

// SecretReference defines a secret to copy from another namespace
type SecretReference struct {
	// Name of the secret in the source namespace
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// SourceNamespace where the secret exists
	// +kubebuilder:validation:Required
	SourceNamespace string `json:"sourceNamespace"`

	// TargetName is the optional name for the secret in the target namespace
	// If not specified, uses the same name as the source
	// +optional
	TargetName string `json:"targetName,omitempty"`

	// Value is the value of the secret to be used in the ephemeral namespace
	// If not specified, the secret will be copied as is using the secret from the SourceNamespace
	// +optional
	Values map[string]string `json:"values,omitempty"`
}

// SyncPolicy defines the sync behavior
type SyncPolicy struct {
	// Automated defines if the application should auto-sync
	// +optional
	Automated *AutomatedSyncPolicy `json:"automated,omitempty"`

	// Prune specifies whether to delete resources that are no longer defined
	// +optional
	Prune bool `json:"prune,omitempty"`

	// SelfHeal specifies whether to revert resources back to their desired state
	// +optional
	SelfHeal bool `json:"selfHeal,omitempty"`
}

// AutomatedSyncPolicy defines automated sync options
type AutomatedSyncPolicy struct {
	// Prune specifies whether to delete resources during auto-sync
	// +optional
	Prune bool `json:"prune,omitempty"`

	// SelfHeal specifies whether to revert resources during auto-sync
	// +optional
	SelfHeal bool `json:"selfHeal,omitempty"`
}

// EphemeralApplicationStatus defines the observed state of EphemeralApplication
type EphemeralApplicationStatus struct {
	// Phase represents the current phase of the ephemeral application
	// +optional
	Phase EphemeralApplicationPhase `json:"phase,omitempty"`

	// Namespace is the actual namespace created for this ephemeral application
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// ArgoApplicationName is the name of the ArgoCD Application created
	// +optional
	ArgoApplicationName string `json:"argoApplicationName,omitempty"`

	// Conditions represent the latest available observations of the application's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Message provides additional information about the current state
	// +optional
	Message string `json:"message,omitempty"`

	// LastSyncTime is the last time the application was synced
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// CopiedSecrets contains the list of secrets that were copied
	// Format: "source-ns/source-name -> target-name"
	// +optional
	CopiedSecrets []string `json:"copiedSecrets,omitempty"`

	// CopiedConfigMaps contains the list of configmaps that were copied
	// +optional
	CopiedConfigMaps []string `json:"copiedConfigMaps,omitempty"`
}

// EphemeralApplicationPhase represents the phase of an ephemeral application
// +kubebuilder:validation:Enum=Pending;Creating;Active;Expiring;Failed
type EphemeralApplicationPhase string

const (
	// PhasePending indicates the application is waiting to be processed
	PhasePending EphemeralApplicationPhase = "Pending"
	// PhaseCreating indicates the application is being created
	PhaseCreating EphemeralApplicationPhase = "Creating"
	// PhaseActive indicates the application is active and running
	PhaseActive EphemeralApplicationPhase = "Active"
	// PhaseExpiring indicates the application is being deleted due to expiration
	PhaseExpiring EphemeralApplicationPhase = "Expiring"
	// PhaseFailed indicates the application has failed
	PhaseFailed EphemeralApplicationPhase = "Failed"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ephapp
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.status.namespace`
// +kubebuilder:printcolumn:name="Expiration",type=date,JSONPath=`.spec.expirationDate`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// EphemeralApplication is the Schema for the ephemeralapplications API
type EphemeralApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EphemeralApplicationSpec   `json:"spec,omitempty"`
	Status EphemeralApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EphemeralApplicationList contains a list of EphemeralApplication
type EphemeralApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EphemeralApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EphemeralApplication{}, &EphemeralApplicationList{})
}
