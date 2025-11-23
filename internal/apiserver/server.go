package apiserver

import (
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jbarea/argo-ephemeral-operator/internal/apiserver/auth"
	"github.com/jbarea/argo-ephemeral-operator/internal/apiserver/handlers"
	"github.com/jbarea/argo-ephemeral-operator/internal/apiserver/middleware"
)

// Server represents the API server
type Server struct {
	client        client.Client
	authenticator *auth.Authenticator
}

// NewServer creates a new API server
func NewServer(client client.Client, authenticator *auth.Authenticator) *Server {
	return &Server{
		client:        client,
		authenticator: authenticator,
	}
}

// Routes configures and returns the HTTP handler with all routes
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Health checks (no auth required)
	mux.HandleFunc("/healthz", handlers.HealthCheck)
	mux.HandleFunc("/readyz", handlers.ReadyCheck)

	// Create handlers
	ephemeralHandler := handlers.NewEphemeralAppHandler(s.client)
	metricsHandler := handlers.NewMetricsHandler(s.client)

	// API routes (require authentication)
	mux.HandleFunc("/api/v1/ephemeral-apps", ephemeralHandler.List)
	mux.HandleFunc("/api/v1/ephemeral-apps/", ephemeralHandler.HandleSingle)
	mux.HandleFunc("/api/v1/ephemeral-apps/create", ephemeralHandler.Create)
	mux.HandleFunc("/api/v1/metrics", metricsHandler.GetMetrics)

	// Apply middleware chain (order matters!)
	var handler http.Handler = mux
	handler = s.authenticator.Middleware(handler) // Auth must be before logging for security
	handler = middleware.Logging(handler)
	handler = middleware.CORS(handler)

	return handler
}
