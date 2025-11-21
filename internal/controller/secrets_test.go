package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	ephemeralv1alpha1 "github.com/jbarea/argo-ephemeral-operator/api/v1alpha1"
)

func TestCopySecret_FromSource(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = ephemeralv1alpha1.AddToScheme(scheme)

	// Create a source secret
	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "db-credentials",
			Namespace: "shared-secrets",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("secret123"),
			"url":      []byte("postgres://db:5432"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sourceSecret).
		Build()

	reconciler := &EphemeralApplicationReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	ephApp := &ephemeralv1alpha1.EphemeralApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
	}

	secretRef := ephemeralv1alpha1.SecretReference{
		Name:            "db-credentials",
		SourceNamespace: "shared-secrets",
	}

	ctx := context.Background()
	err := reconciler.copySecret(ctx, secretRef, "ephemeral-test", ephApp)
	if err != nil {
		t.Fatalf("copySecret failed: %v", err)
	}

	// Verify secret was created in target namespace
	targetSecret := &corev1.Secret{}
	err = fakeClient.Get(ctx, client.ObjectKey{
		Namespace: "ephemeral-test",
		Name:      "db-credentials",
	}, targetSecret)
	if err != nil {
		t.Fatalf("failed to get copied secret: %v", err)
	}

	// Verify data was copied
	if string(targetSecret.Data["username"]) != "admin" {
		t.Errorf("expected username 'admin', got '%s'", string(targetSecret.Data["username"]))
	}
	if string(targetSecret.Data["password"]) != "secret123" {
		t.Errorf("expected password 'secret123', got '%s'", string(targetSecret.Data["password"]))
	}

	// Verify labels
	if targetSecret.Labels["ephemeral.argo.io/owner"] != "test-app" {
		t.Errorf("expected owner label 'test-app', got '%s'", targetSecret.Labels["ephemeral.argo.io/owner"])
	}
}

func TestCopySecret_InlineValues(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = ephemeralv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &EphemeralApplicationReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	ephApp := &ephemeralv1alpha1.EphemeralApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
	}

	secretRef := ephemeralv1alpha1.SecretReference{
		Name: "inline-secret",
		Values: map[string]string{
			"api-key":    "test-key-123",
			"api-secret": "test-secret-456",
			"endpoint":   "https://api.example.com",
		},
	}

	ctx := context.Background()
	err := reconciler.copySecret(ctx, secretRef, "ephemeral-test", ephApp)
	if err != nil {
		t.Fatalf("copySecret with inline values failed: %v", err)
	}

	// Verify secret was created
	targetSecret := &corev1.Secret{}
	err = fakeClient.Get(ctx, client.ObjectKey{
		Namespace: "ephemeral-test",
		Name:      "inline-secret",
	}, targetSecret)
	if err != nil {
		t.Fatalf("failed to get created secret: %v", err)
	}

	// Verify inline data
	if string(targetSecret.Data["api-key"]) != "test-key-123" {
		t.Errorf("expected api-key 'test-key-123', got '%s'", string(targetSecret.Data["api-key"]))
	}
	if string(targetSecret.Data["endpoint"]) != "https://api.example.com" {
		t.Errorf("expected endpoint 'https://api.example.com', got '%s'", string(targetSecret.Data["endpoint"]))
	}

	// Verify inline label
	if targetSecret.Labels["ephemeral.argo.io/inline"] != "true" {
		t.Error("expected inline label to be 'true'")
	}
}

func TestBuildCopiedSecretsList(t *testing.T) {
	tests := []struct {
		name     string
		secrets  []ephemeralv1alpha1.SecretReference
		expected []string
	}{
		{
			name:     "empty secrets",
			secrets:  []ephemeralv1alpha1.SecretReference{},
			expected: nil,
		},
		{
			name: "copied secret",
			secrets: []ephemeralv1alpha1.SecretReference{
				{
					Name:            "db-creds",
					SourceNamespace: "databases",
				},
			},
			expected: []string{"databases/db-creds -> db-creds"},
		},
		{
			name: "inline secret",
			secrets: []ephemeralv1alpha1.SecretReference{
				{
					Name: "api-keys",
					Values: map[string]string{
						"key": "value",
					},
				},
			},
			expected: []string{"/api-keys -> api-keys"},
		},
		{
			name: "copied with rename",
			secrets: []ephemeralv1alpha1.SecretReference{
				{
					Name:            "postgres",
					SourceNamespace: "databases",
					TargetName:      "pg-creds",
				},
			},
			expected: []string{"databases/postgres -> pg-creds"},
		},
		{
			name: "mixed secrets",
			secrets: []ephemeralv1alpha1.SecretReference{
				{
					Name:            "postgres",
					SourceNamespace: "databases",
				},
				{
					Name: "test-data",
					Values: map[string]string{
						"data": "test",
					},
				},
			},
			expected: []string{"databases/postgres -> postgres", "/test-data -> test-data"},
		},
	}

	reconciler := &EphemeralApplicationReconciler{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reconciler.buildCopiedSecretsList(tt.secrets)

			if len(got) != len(tt.expected) {
				t.Errorf("expected %d items, got %d", len(tt.expected), len(got))
				return
			}

			for i, expected := range tt.expected {
				if got[i] != expected {
					t.Errorf("item %d: expected '%s', got '%s'", i, expected, got[i])
				}
			}
		})
	}
}

func TestCopySecrets_EmptyList(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = ephemeralv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &EphemeralApplicationReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	ephApp := &ephemeralv1alpha1.EphemeralApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
		Spec: ephemeralv1alpha1.EphemeralApplicationSpec{
			Secrets: []ephemeralv1alpha1.SecretReference{},
		},
	}

	ctx := context.Background()
	err := reconciler.copySecrets(ctx, ephApp, "ephemeral-test")
	if err != nil {
		t.Fatalf("copySecrets with empty list should not fail: %v", err)
	}
}
