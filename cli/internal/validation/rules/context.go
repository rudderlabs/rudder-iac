package rules

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ValidationContext provides all the information a rule needs to perform validation.
// Different rules may use different parts of this context depending on their validation needs.
// Syntactic rules typically use FilePath, FileName, Spec, Kind, and Metadata fields,
// while semantic rules additionally use the Graph field for cross-resource validation.
type ValidationContext struct {
	// FilePath is the absolute path to the spec file being validated
	FilePath string

	// FileName is just the name of the file (without directory path)
	FileName string

	// Spec is the raw spec data (typically map[string]any) from the YAML.
	// Rules can inspect this for syntactic validation before resource graph construction.
	Spec any

	// Kind is the spec kind (e.g., "properties", "events", "tp", "custom-types")
	Kind string

	// Version is the spec version (e.g., "rudder/v1")
	Version string

	// Metadata contains parsed common metadata from the spec.
	// This includes name, import metadata, etc.
	Metadata map[string]any

	// Graph is the complete resource graph built from all loaded specs.
	// This is nil for syntactic validation (pre-graph construction)
	// and populated for semantic validation (post-graph construction).
	Graph *resources.Graph
}

// HasGraph returns true if the context includes a resource graph for semantic validation.
// Rules can use this to determine if they have access to cross-resource validation data.
func (vc *ValidationContext) HasGraph() bool {
	return vc.Graph != nil
}
