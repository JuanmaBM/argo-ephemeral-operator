package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the operator configuration
type Config struct {
	// ArgoCD configuration
	ArgoServer    string
	ArgoToken     string
	ArgoNamespace string
	ArgoInsecure  bool

	// Operator configuration
	MetricsAddr          string
	ProbeAddr            string
	LeaderElectionID     string
	EnableLeaderElection bool
	ReconcileInterval    time.Duration
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		// ArgoCD defaults
		ArgoServer:    getEnvOrDefault("ARGO_SERVER", "argocd-server.argocd.svc.cluster.local"),
		ArgoToken:     os.Getenv("ARGO_TOKEN"),
		ArgoNamespace: getEnvOrDefault("ARGO_NAMESPACE", "argocd"),
		ArgoInsecure:  getEnvBoolOrDefault("ARGO_INSECURE", true),

		// Operator defaults
		MetricsAddr:          getEnvOrDefault("METRICS_ADDR", ":8080"),
		ProbeAddr:            getEnvOrDefault("HEALTH_PROBE_ADDR", ":8081"),
		LeaderElectionID:     getEnvOrDefault("LEADER_ELECTION_ID", "argo-ephemeral-operator-lock"),
		EnableLeaderElection: getEnvBoolOrDefault("ENABLE_LEADER_ELECTION", false),
		ReconcileInterval:    getEnvDurationOrDefault("RECONCILE_INTERVAL", 5*time.Minute),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ArgoServer == "" {
		return fmt.Errorf("ARGO_SERVER is required")
	}
	if c.ArgoToken == "" {
		return fmt.Errorf("ARGO_TOKEN is required")
	}
	if c.ArgoNamespace == "" {
		return fmt.Errorf("ARGO_NAMESPACE is required")
	}
	return nil
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBoolOrDefault returns the boolean value of an environment variable or a default value
func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getEnvDurationOrDefault returns the duration value of an environment variable or a default value
func getEnvDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		parsed, err := time.ParseDuration(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}
