package kotlin

import (
	"fmt"
	"path/filepath"
	"regexp"
)

// KotlinOptions contains Kotlin-specific generation options
type KotlinOptions struct {
	PackageName    string `mapstructure:"packageName" description:"Package name for generated Kotlin code (e.g., com.example.analytics)"`
	OutputFileName string `mapstructure:"outputFileName" description:"Name of the generated Kotlin file (e.g., MyEvents.kt). Defaults to Main.kt"`
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
	if k.PackageName != "" && !packageNameRegex.MatchString(k.PackageName) {
		return fmt.Errorf(
			"invalid package name %q: must be lowercase with dot-separated segments "+
				"starting with letters (e.g., com.example.analytics)",
			k.PackageName,
		)
	}

	if k.OutputFileName != "" && filepath.Ext(k.OutputFileName) != ".kt" {
		return fmt.Errorf("invalid output file name %q: must have a .kt extension", k.OutputFileName)
	}

	return nil
}
