package controller

import (
	"strings"
	"testing"
)

func TestDefaultNameGenerator_GenerateNamespace(t *testing.T) {
	tests := []struct {
		name          string
		namespaceName string
		wantLen       int
		validate      func(t *testing.T, result string)
	}{
		{
			name:          "custom namespace name",
			namespaceName: "my-custom-namespace",
			wantLen:       63,
			validate: func(t *testing.T, result string) {
				if result != "my-custom-namespace" {
					t.Errorf("expected 'my-custom-namespace', got '%s'", result)
				}
			},
		},
		{
			name:          "auto-generated namespace",
			namespaceName: "",
			wantLen:       63,
			validate: func(t *testing.T, result string) {
				if !strings.HasPrefix(result, "ephemeral-") {
					t.Errorf("expected prefix 'ephemeral-', got '%s'", result)
				}
				if len(result) != 17 { // "ephemeral-" (10) + 7 random chars
					t.Errorf("expected length 17, got %d", len(result))
				}
			},
		},
		{
			name:          "long custom name",
			namespaceName: "this-is-a-very-long-namespace-name-that-exceeds-kubernetes-limits",
			wantLen:       63,
			validate: func(t *testing.T, result string) {
				if len(result) > 63 {
					t.Errorf("result length %d exceeds 63 characters", len(result))
				}
			},
		},
		{
			name:          "name with underscores",
			namespaceName: "my_custom_namespace",
			wantLen:       63,
			validate: func(t *testing.T, result string) {
				if result != "my-custom-namespace" {
					t.Errorf("expected 'my-custom-namespace', got '%s'", result)
				}
			},
		},
	}

	gen := NewDefaultNameGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.GenerateNamespace(tt.namespaceName, "")

			if len(got) > tt.wantLen {
				t.Errorf("GenerateNamespace() length = %v, want <= %v", len(got), tt.wantLen)
			}

			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

// Mock implementations for testing would go here
// Example:
// type mockArgoClient struct{}
// func (m *mockArgoClient) CreateApplication(ctx context.Context, app *argocdv1alpha1.Application) error {
//     return nil
// }
