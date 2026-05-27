package model

import (
	"github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ModelResource represents the resource for a model (entity or event)
type ModelResource struct {
	ID           string                 `mapstructure:"id"`
	DisplayName  string                 `mapstructure:"display_name"`
	Type         string                 `mapstructure:"type"` // "entity" or "event"
	Table        string                 `mapstructure:"table"`
	Description  string                 `mapstructure:"description"`
	DataGraphRef *resources.PropertyRef `mapstructure:"data_graph"` // Reference to parent data graph's remote ID

	// Entity model fields
	PrimaryID string `mapstructure:"primary_id"`
	Root      bool   `mapstructure:"root"`

	// Event model fields
	Timestamp string `mapstructure:"timestamp"`

	// Sparse per-column metadata overrides extracted from the yaml model spec.
	// Stored as []map[string]any (not []ColumnMetadataYAML) so the syncer's
	// mapstructure-based diff handles slice equality via reflect.DeepEqual
	// instead of the unsupported `!=` on typed slices. Each entry has shape
	// {"name": <columnName>, "display_name": <displayName>}; the handler maps
	// it back into datagraph.ColumnMetadataEntry before calling the API.
	Columns []map[string]any `mapstructure:"columns,omitempty"`
}

// ModelState represents the output state from the remote system
type ModelState struct {
	ID string // Remote model ID
}

// RemoteModel wraps datagraph.Model to implement RemoteResource interface.
//
// Columns carries the per-model column metadata rows fetched from the
// `/column-metadata` endpoint at remote-load time. It is populated in
// LoadRemoteResources / LoadImportableResources (where context is available)
// so that both MapRemoteToState (apply diff) and FormatForExport (yaml export)
// can read it without re-issuing HTTP calls. The slice is always sorted by
// Name so that comparisons against the local yaml's column list — which the
// handler also normalises by parse order — produce stable diffs across runs.
type RemoteModel struct {
	*datagraph.Model
	Columns []datagraph.ColumnMetadataRow
}

// Metadata implements the RemoteResource interface
func (r RemoteModel) Metadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:          r.ID,
		ExternalID:  r.ExternalID,
		WorkspaceID: r.WorkspaceID,
		Name:        r.Name,
	}
}
