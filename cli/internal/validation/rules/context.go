package rules

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
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
	Spec map[string]any

	// Source is the spec file as written, before variable substitution. Only
	// rules implementing SourceRule receive it, from NewSourceContext.
	Source []byte

	// References is Source with each variable reference swapped for a
	// placeholder. A SourceRule's Spec and Metadata are parsed from this copy,
	// since an unquoted reference may not parse until substitution runs, so a
	// value holding a reference reads as its placeholder, and References maps
	// it back. Only rules implementing SourceRule receive it.
	References varsubst.Replaced

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

	// WorkspaceID is the workspace this run targets. Workspace-scoped semantic
	// rules (e.g. orphaned-URN) read it to validate only the active workspace's
	// entries. Empty when no specific workspace is targeted (e.g. validate with
	// no target) — such rules then apply to all workspaces.
	WorkspaceID string
}

// HasGraph returns true if the context includes a resource graph for semantic validation.
// Rules can use this to determine if they have access to cross-resource validation data.
func (vc *ValidationContext) HasGraph() bool {
	return vc.Graph != nil
}
