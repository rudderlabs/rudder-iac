package kotlin

import (
	"fmt"
	"regexp"
	"strings"
)

// KotlinOptions contains Kotlin-specific generation options
type KotlinOptions struct {
	PackageName    string `mapstructure:"packageName" description:"Package name for generated Kotlin code (e.g., com.example.analytics)"`
	OutputFileName string `mapstructure:"outputFileName" description:"Name of the generated Kotlin file (e.g., MyEvents.kt). Defaults to Main.kt"`
	Annotations    string `mapstructure:"annotations" description:"Comma-separated fully qualified annotations for generated data classes (e.g., androidx.compose.runtime.Stable)"`
}

// GetAvailableOptions returns metadata about all supported Kotlin options
func (k *Generator) DefaultOptions() any {
	return KotlinOptions{
		PackageName:    "com.rudderstack.ruddertyper",
		OutputFileName: "Main.kt",
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

	for _, fqn := range k.ParsedAnnotations() {
		if !annotationFQNRegex.MatchString(fqn) {
			return fmt.Errorf(
				"invalid annotation %q: must be a fully qualified class name (e.g., androidx.compose.runtime.Stable)",
				fqn,
			)
		}
	}

	return nil
}

// annotationFQNRegex validates fully qualified Kotlin class names (e.g., androidx.compose.runtime.Stable)
var annotationFQNRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*\.[A-Z][a-zA-Z0-9]*$`)

// ParsedAnnotations splits the comma-separated Annotations string into individual entries
func (k *KotlinOptions) ParsedAnnotations() []string {
	if k.Annotations == "" {
		return nil
	}

	var result []string
	for _, a := range strings.Split(k.Annotations, ",") {
		if trimmed := strings.TrimSpace(a); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
