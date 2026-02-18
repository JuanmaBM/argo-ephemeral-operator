package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ephemeralv1alpha1 "github.com/jbarea/argo-ephemeral-operator/api/v1alpha1"
	"github.com/jbarea/argo-ephemeral-operator/internal/apiserver/auth"
)

// EphemeralAppHandler handles EphemeralApplication CRUD operations
type EphemeralAppHandler struct {
	client client.Client
}

// NewEphemeralAppHandler creates a new handler
func NewEphemeralAppHandler(client client.Client) *EphemeralAppHandler {
	return &EphemeralAppHandler{client: client}
}

// List handles GET /api/v1/ephemeral-apps
func (h *EphemeralAppHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// List all EphemeralApplications
	list := &ephemeralv1alpha1.EphemeralApplicationList{}
	if err := h.client.List(ctx, list); err != nil {
		respondError(w, "Failed to list ephemeral apps", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, list)
}

// HandleSingle routes single resource operations
func (h *EphemeralAppHandler) HandleSingle(w http.ResponseWriter, r *http.Request) {
	// Extract name from path: /api/v1/ephemeral-apps/{name}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/ephemeral-apps/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		respondError(w, "Name required", http.StatusBadRequest)
		return
	}

	name := parts[0]

	switch r.Method {
	case http.MethodGet:
		h.Get(w, r, name)
	case http.MethodPatch:
		h.Update(w, r, name)
	case http.MethodDelete:
		h.Delete(w, r, name)
	default:
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Get retrieves a single EphemeralApplication
func (h *EphemeralAppHandler) Get(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()

	// Parse namespace from query param, default to "default"
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	ephApp := &ephemeralv1alpha1.EphemeralApplication{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := h.client.Get(ctx, key, ephApp); err != nil {
		if client.IgnoreNotFound(err) == nil {
			respondError(w, "Not found", http.StatusNotFound)
			return
		}
		respondError(w, "Failed to get ephemeral app", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, ephApp)
}

// Create handles POST /api/v1/ephemeral-apps/create
func (h *EphemeralAppHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	user, _ := auth.GetUserFromContext(ctx)

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var ephApp ephemeralv1alpha1.EphemeralApplication
	if err := json.Unmarshal(body, &ephApp); err != nil {
		respondError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Set default namespace if not provided
	if ephApp.Namespace == "" {
		ephApp.Namespace = "default"
	}

	// Add annotation with creator info
	if ephApp.Annotations == nil {
		ephApp.Annotations = make(map[string]string)
	}
	if user != nil {
		ephApp.Annotations["ephemeral.argo.io/created-by"] = user.Username
	}

	if err := h.client.Create(ctx, &ephApp); err != nil {
		respondError(w, "Failed to create ephemeral app: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, ephApp)
}

// Update handles PATCH /api/v1/ephemeral-apps/{name}
func (h *EphemeralAppHandler) Update(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()

	// Parse namespace from query param
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Get existing resource
	ephApp := &ephemeralv1alpha1.EphemeralApplication{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := h.client.Get(ctx, key, ephApp); err != nil {
		if client.IgnoreNotFound(err) == nil {
			respondError(w, "Not found", http.StatusNotFound)
			return
		}
		respondError(w, "Failed to get ephemeral app", http.StatusInternalServerError)
		return
	}

	// Read patch data
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse patch
	var patch map[string]interface{}
	if err := json.Unmarshal(body, &patch); err != nil {
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Apply simple patches (only spec.expirationDate for now)
	if spec, ok := patch["spec"].(map[string]interface{}); ok {
		if expDate, ok := spec["expirationDate"].(string); ok {
			var parsedTime metav1.Time
			if err := parsedTime.UnmarshalText([]byte(expDate)); err == nil {
				ephApp.Spec.ExpirationDate = parsedTime
			}
		}
	}

	// Update resource
	if err := h.client.Update(ctx, ephApp); err != nil {
		respondError(w, "Failed to update ephemeral app", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, ephApp)
}

// Delete handles DELETE /api/v1/ephemeral-apps/{name}
func (h *EphemeralAppHandler) Delete(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()

	// Parse namespace from query param
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	ephApp := &ephemeralv1alpha1.EphemeralApplication{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := h.client.Get(ctx, key, ephApp); err != nil {
		if client.IgnoreNotFound(err) == nil {
			respondError(w, "Not found", http.StatusNotFound)
			return
		}
		respondError(w, "Failed to get ephemeral app", http.StatusInternalServerError)
		return
	}

	if err := h.client.Delete(ctx, ephApp); err != nil {
		respondError(w, "Failed to delete ephemeral app", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
