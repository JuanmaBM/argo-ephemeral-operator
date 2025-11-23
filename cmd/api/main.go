package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ephemeralv1alpha1 "github.com/jbarea/argo-ephemeral-operator/api/v1alpha1"
	"github.com/jbarea/argo-ephemeral-operator/internal/apiserver"
	"github.com/jbarea/argo-ephemeral-operator/internal/apiserver/auth"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = ephemeralv1alpha1.AddToScheme(scheme)
}

func main() {
	var port int
	flag.IntVar(&port, "port", 8080, "API server port")
	flag.Parse()

	log.Println("Starting Argo Ephemeral Operator API Server...")

	// Setup Kubernetes clients
	cfg := ctrl.GetConfigOrDie()

	// Client for authentication (TokenReview)
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	// Controller-runtime client for CRD access
	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("Failed to create controller-runtime client: %v", err)
	}

	// Create authenticator
	authenticator := auth.NewAuthenticator(clientset)

	// Create API server
	srv := apiserver.NewServer(k8sClient, authenticator)

	// HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      srv.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("API server listening on port %d", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received signal %v, shutting down server...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}
