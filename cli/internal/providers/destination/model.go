package destination

import (
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type DestinationSpec struct {
	ID                string         `mapstructure:"id"`
	DisplayName       string         `mapstructure:"display_name"`
	Type              string         `mapstructure:"type"`
	Enabled           bool           `mapstructure:"enabled"`
	DefinitionVersion int64          `mapstructure:"definition_version"`
	Transformation    string         `mapstructure:"transformation"`
	Config            map[string]any `mapstructure:"config"`
}

type DestinationResource struct {
	ID                string
	DisplayName       string
	Type              string
	Enabled           bool
	DefinitionVersion int64
	Transformation    *resources.PropertyRef
	Config            map[string]any
}

type DestinationState struct {
	ID               string
	TransformationID string
}

type RemoteDestination struct {
	*client.Destination
}

func (r RemoteDestination) Metadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:         r.ID,
		ExternalID: r.ExternalID,
		Name:       r.Name,
	}
}
