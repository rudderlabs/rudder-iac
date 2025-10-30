package kotlin

import (
	"fmt"
	"regexp"
)

// KotlinOptions contains Kotlin-specific generation options
type KotlinOptions struct {
	PackageName string `mapstructure:"packageName" description:"Package name for generated Kotlin code (e.g., com.example.analytics)"`
}

// GetAvailableOptions returns metadata about all supported Kotlin options
func (k *Generator) DefaultOptions() any {
	return KotlinOptions{
		PackageName: "com.rudderstack.ruddertyper",
	}
}

// packageNameRegex validates Kotlin package names:
// - Must start with lowercase letter
// - Can contain lowercase letters, digits, underscores
// - Segments separated by dots
// - Each segment must start with a letter
var packageNameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`)

// Validate validates Kotlin-specific options
// Returns error if unknown options are provided or if validation fails
func (k *KotlinOptions) Validate() error {
	// Validate packageName if it was set
	if k.PackageName != "" && !packageNameRegex.MatchString(k.PackageName) {
		return fmt.Errorf(
			"invalid package name %q: must be lowercase with dot-separated segments "+
				"starting with letters (e.g., com.example.analytics)",
			k.PackageName,
		)
	}

	return nil
}
