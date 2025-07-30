package core

import "github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"

// File represents a generated file
type File struct {
	// Path relative to the output directory
	Path string
	// Content is the file content
	Content string
}

// GeneratorStrategy defines the interface for platform-specific code generators
type GeneratorStrategy interface {
	// Generate produces code files from a tracking plan
	Generate(plan plan.TrackingPlan, options GeneratorOptions) ([]File, error)
}

// GeneratorOptions contains configuration options for generators
type GeneratorOptions struct {
	// OutputDir is the base directory for generated files
	OutputDir string
	// Additional platform-specific options can be added here
	Platform map[string]any
}
