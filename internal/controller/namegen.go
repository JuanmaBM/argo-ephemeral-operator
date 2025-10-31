package controller

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// DefaultNameGenerator is the default implementation of NameGenerator
type DefaultNameGenerator struct {
	rnd *rand.Rand
}

// NewDefaultNameGenerator creates a new DefaultNameGenerator with random seed
func NewDefaultNameGenerator() *DefaultNameGenerator {
	return &DefaultNameGenerator{
		rnd: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateNamespace generates a namespace name
// If namespaceName is provided, uses it directly
// Otherwise generates "ephemeral-{random}"
func (g *DefaultNameGenerator) GenerateNamespace(namespaceName, _ string) string {
	if namespaceName != "" {
		// Use provided name directly, sanitize it
		sanitized := strings.ToLower(namespaceName)
		sanitized = strings.ReplaceAll(sanitized, "_", "-")

		// Ensure it's <= 63 characters
		if len(sanitized) > 63 {
			sanitized = sanitized[:63]
		}

		return sanitized
	}

	// Generate random suffix (7 characters)
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	suffix := make([]byte, 7)
	for i := range suffix {
		suffix[i] = charset[g.rnd.Intn(len(charset))]
	}

	return fmt.Sprintf("ephemeral-%s", string(suffix))
}
