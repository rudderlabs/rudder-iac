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
}

// ModelState represents the output state from the remote system
type ModelState struct {
	ID string // Remote model ID
}

// RemoteModel wraps datagraph.Model to implement RemoteResource interface
type RemoteModel struct {
	*datagraph.Model
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
