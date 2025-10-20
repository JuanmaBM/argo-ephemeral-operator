package controller

import (
	"testing"
)

func TestDefaultNameGenerator_GenerateNamespace(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		suffix   string
		wantLen  int
		validate func(t *testing.T, result string)
	}{
		{
			name:    "simple case",
			prefix:  "ephemeral",
			suffix:  "test",
			wantLen: 63, // max kubernetes namespace length
			validate: func(t *testing.T, result string) {
				if result != "ephemeral-test" {
					t.Errorf("expected 'ephemeral-test', got '%s'", result)
				}
			},
		},
		{
			name:    "long suffix",
			prefix:  "ephemeral",
			suffix:  "this-is-a-very-long-suffix-that-exceeds-kubernetes-limits-for-namespace-names",
			wantLen: 63,
			validate: func(t *testing.T, result string) {
				if len(result) > 63 {
					t.Errorf("result length %d exceeds 63 characters", len(result))
				}
				if result[:9] != "ephemeral" {
					t.Errorf("expected prefix 'ephemeral', got '%s'", result[:9])
				}
			},
		},
		{
			name:    "suffix with underscores",
			prefix:  "test",
			suffix:  "my_app_name",
			wantLen: 63,
			validate: func(t *testing.T, result string) {
				if result != "test-my-app-name" {
					t.Errorf("expected 'test-my-app-name', got '%s'", result)
				}
			},
		},
		{
			name:    "uppercase suffix",
			prefix:  "dev",
			suffix:  "MyApp",
			wantLen: 63,
			validate: func(t *testing.T, result string) {
				if result != "dev-myapp" {
					t.Errorf("expected 'dev-myapp', got '%s'", result)
				}
			},
		},
	}

	gen := &DefaultNameGenerator{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.GenerateNamespace(tt.prefix, tt.suffix)

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
