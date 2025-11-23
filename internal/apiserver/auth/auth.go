package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// User represents an authenticated user
type User struct {
	Username string
	UID      string
	Groups   []string
}

// Authenticator handles ServiceAccount token validation
type Authenticator struct {
	clientset *kubernetes.Clientset
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(clientset *kubernetes.Clientset) *Authenticator {
	return &Authenticator{clientset: clientset}
}

// ValidateToken validates a ServiceAccount token against Kubernetes API
func (a *Authenticator) ValidateToken(ctx context.Context, token string) (*User, error) {
	// Create TokenReview to validate the token
	tr := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token: token,
		},
	}

	result, err := a.clientset.AuthenticationV1().TokenReviews().Create(ctx, tr, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	if !result.Status.Authenticated {
		return nil, fmt.Errorf("token not authenticated")
	}

	// Extract user info
	user := &User{
		Username: result.Status.User.Username,
		UID:      result.Status.User.UID,
		Groups:   result.Status.User.Groups,
	}

	return user, nil
}

// Middleware provides authentication middleware for HTTP handlers
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for health checks
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}

		// Only authenticate API routes
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": "Authorization header required"}`, http.StatusUnauthorized)
			return
		}

		// Expected format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error": "Invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]
		user, err := a.ValidateToken(r.Context(), token)
		if err != nil {
			http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type contextKey string

const userContextKey contextKey = "user"

// GetUserFromContext extracts user from request context
func GetUserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userContextKey).(*User)
	return user, ok
}
