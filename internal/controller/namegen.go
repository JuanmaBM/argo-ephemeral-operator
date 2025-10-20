package controller

import (
	"fmt"
	"strings"
)

// DefaultNameGenerator is the default implementation of NameGenerator
type DefaultNameGenerator struct{}

// GenerateNamespace generates a namespace name with prefix and suffix
func (g *DefaultNameGenerator) GenerateNamespace(prefix, suffix string) string {
	// Sanitize suffix to be DNS compliant
	sanitized := strings.ToLower(suffix)
	sanitized = strings.ReplaceAll(sanitized, "_", "-")

	// Ensure total length is <= 63 characters (Kubernetes limit)
	maxLength := 63
	combined := fmt.Sprintf("%s-%s", prefix, sanitized)

	if len(combined) > maxLength {
		// Truncate suffix to fit
		allowedSuffixLen := maxLength - len(prefix) - 1 // -1 for the dash
		if allowedSuffixLen > 0 {
			sanitized = sanitized[:allowedSuffixLen]
			combined = fmt.Sprintf("%s-%s", prefix, sanitized)
		} else {
			// If prefix itself is too long, just use it truncated
			combined = prefix[:maxLength]
		}
	}

	return combined
}
