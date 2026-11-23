package kotlin

import (
	"fmt"
	"regexp"
)

const (
	// EmitAll writes the models and the client built on the RudderStack Kotlin SDK.
	EmitAll = "all"
	// EmitCommon writes the models and a client that has no SDK dependency.
	EmitCommon = "common"
)

// KotlinOptions contains Kotlin-specific generation options
type KotlinOptions struct {
	PackageName      string `mapstructure:"packageName" description:"Package name for generated Kotlin code (e.g., com.example.analytics)"`
	OutputFileName   string `mapstructure:"outputFileName" description:"Name of the generated Kotlin file (e.g., MyEvents.kt). Defaults to Main.kt"`
	Emit             string `mapstructure:"emit" description:"What to generate: 'all' (default) writes the models and a RudderAnalytics client that needs the RudderStack Kotlin SDK, 'common' writes the models and an SDK-free RudderTyper client that sends events through a RudderTyperTransport you implement once per platform, so it compiles in Kotlin Multiplatform common code and pure JVM modules"`
	ComposeImmutable bool   `mapstructure:"composeImmutable" description:"Add @androidx.compose.runtime.Immutable to generated data classes (and concrete sealed subclasses) so Jetpack Compose treats them as stable. Requires androidx.compose.runtime on the consuming module's classpath."`
}

// GetAvailableOptions returns metadata about all supported Kotlin options
func (k *Generator) DefaultOptions() any {
	return KotlinOptions{
		PackageName:    "com.rudderstack.ruddertyper",
		OutputFileName: "Main.kt",
		Emit:           EmitAll,
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

	switch k.Emit {
	case "", EmitAll, EmitCommon:
	default:
		return fmt.Errorf("invalid emit value %q: must be %q or %q", k.Emit, EmitAll, EmitCommon)
	}

	return nil
}
