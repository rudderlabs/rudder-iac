package destination

import (
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// DestinationSpec is the user-authored YAML representation of a destination.
type DestinationSpec struct {
	ID                string         `mapstructure:"id" validate:"required"`
	DisplayName       string         `mapstructure:"display_name" validate:"required,pattern=destination_display_name"`
	Type              string         `mapstructure:"type" validate:"required"`
	Enabled           bool           `mapstructure:"enabled"`
	DefinitionVersion int64          `mapstructure:"definition_version" validate:"required"`
	Transformation    string         `mapstructure:"transformation"` // scalar "#transformation:<id>" — object form deferred
	Config            map[string]any `mapstructure:"config"`
}

// DestinationResource is the resolved in-memory representation compared by the
// differ. Config is kept snake_case on both sides; conversion to the API's
// camelCase happens at the API boundary in the handler.
type DestinationResource struct {
	ID                string
	DisplayName       string
	Type              string
	Enabled           bool
	DefinitionVersion int64
	Transformation    *resources.PropertyRef
	Config            map[string]any
}

// DestinationState is the persisted apply-cycle state. ID is the remote
// destination API ID; TransformationID is the linked transformation's remote
// ID (empty when no link exists).
type DestinationState struct {
	ID               string
	TransformationID string
}

// RemoteDestination wraps client.Destination to satisfy handler.RemoteResource
// without pulling handler types into api/client.
type RemoteDestination struct {
	*client.Destination
}

// Metadata exposes the identifying fields BaseHandler uses to key the remote
// collection and to name importable resources.
func (r RemoteDestination) Metadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:          r.ID,
		ExternalID:  r.ExternalID,
		WorkspaceID: r.WorkspaceID,
		Name:        r.Name,
	}
}
