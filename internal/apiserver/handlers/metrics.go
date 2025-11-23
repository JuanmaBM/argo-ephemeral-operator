package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ephemeralv1alpha1 "github.com/jbarea/argo-ephemeral-operator/api/v1alpha1"
)

// MetricsHandler handles metrics endpoints
type MetricsHandler struct {
	client client.Client
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(client client.Client) *MetricsHandler {
	return &MetricsHandler{client: client}
}

// MetricsResponse contains aggregated metrics
type MetricsResponse struct {
	TotalEnvironments    int                  `json:"totalEnvironments"`
	ActiveEnvironments   int                  `json:"activeEnvironments"`
	CreatingEnvironments int                  `json:"creatingEnvironments"`
	FailedEnvironments   int                  `json:"failedEnvironments"`
	EnvironmentsByPhase  map[string]int       `json:"environmentsByPhase"`
	RecentEnvironments   []EnvironmentSummary `json:"recentEnvironments"`
}

// EnvironmentSummary is a simplified view of an environment
type EnvironmentSummary struct {
	Name           string      `json:"name"`
	Namespace      string      `json:"namespace"`
	Phase          string      `json:"phase"`
	ExpirationDate metav1.Time `json:"expirationDate"`
	CreatedAt      metav1.Time `json:"createdAt"`
}

// GetMetrics handles GET /api/v1/metrics
func (h *MetricsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	// List all environments
	list := &ephemeralv1alpha1.EphemeralApplicationList{}
	if err := h.client.List(ctx, list); err != nil {
		http.Error(w, `{"error":"Failed to list environments"}`, http.StatusInternalServerError)
		return
	}

	// Calculate metrics
	metrics := MetricsResponse{
		TotalEnvironments:   len(list.Items),
		EnvironmentsByPhase: make(map[string]int),
		RecentEnvironments:  []EnvironmentSummary{},
	}

	for _, env := range list.Items {
		phase := string(env.Status.Phase)
		if phase == "" {
			phase = "Pending"
		}

		metrics.EnvironmentsByPhase[phase]++

		switch env.Status.Phase {
		case ephemeralv1alpha1.PhaseActive:
			metrics.ActiveEnvironments++
		case ephemeralv1alpha1.PhaseCreating:
			metrics.CreatingEnvironments++
		case ephemeralv1alpha1.PhaseFailed:
			metrics.FailedEnvironments++
		}

		// Add to recent list (limit to 10)
		if len(metrics.RecentEnvironments) < 10 {
			metrics.RecentEnvironments = append(metrics.RecentEnvironments, EnvironmentSummary{
				Name:           env.Name,
				Namespace:      env.Status.Namespace,
				Phase:          phase,
				ExpirationDate: env.Spec.ExpirationDate,
				CreatedAt:      env.CreationTimestamp,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}
