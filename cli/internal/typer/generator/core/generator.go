package core

import "github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"

// File represents a generated file
type File struct {
	// Path relative to the output directory
	Path string
	// Content is the file content
	Content string
}

// Generator defines the interface for platform-specific code generators
type Generator interface {
	// Generate produces code files from a tracking plan
	Generate(plan *plan.TrackingPlan, options GenerateOptions, platformOptions any) ([]*File, error)

	// DefaultOptions returns the default platform-specific options, in a platform-specific struct
	// By convetion, fields in the returned struct should have `mapstructure` tags for proper mapping
	// from string key-value pairs to struct fields, as well as `description` tags for documentation purposes
	DefaultOptions() any
}

// GenerateOptions contains configuration for code generation
type GenerateOptions struct {
	RudderCLIVersion string
	Platform         string
	OutputPath       string
	PlatformOptions  map[string]string // Platform-specific options map (e.g., KotlinOptions)
}
