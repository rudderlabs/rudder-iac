package model

import (
	"github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ModelResource represents the resource for a model (entity or event)
type ModelResource struct {
	ID           string
	DisplayName  string
	Type         string // "entity" or "event"
	Table        string
	Description  string
	DataGraphRef *resources.PropertyRef // Reference to parent data graph's remote ID

	// Entity model fields
	PrimaryID string
	Root      bool

	// Event model fields
	Timestamp string
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
	// AIFIX: Include WorkspaceID in the metadata
	return handler.RemoteResourceMetadata{
		ID:         r.ID,
		ExternalID: r.ExternalID,
		Name:       r.Name,
	}
}
