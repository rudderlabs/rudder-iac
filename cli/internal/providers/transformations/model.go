package transformations

import (
	transformationsClient "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
)

// TransformationSpec embeds the specs package spec
type TransformationSpec = specs.TransformationSpec

// TransformationResource represents the parsed resource configuration
type TransformationResource struct {
	ID          string
	Name        string
	Description string
	Language    string
	Code        string // Resolved from inline or file
	Tests       []specs.TransformationTest
}

// TransformationState represents the remote state returned from API
type TransformationState struct {
	ID        string
	VersionID string
}

// RemoteTransformation wraps the API client transformation to implement RemoteResource interface
type RemoteTransformation struct {
	transformationsClient.Transformation
}

func (r RemoteTransformation) Metadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:          r.ID,
		ExternalID:  r.ExternalID,
		WorkspaceID: r.WorkspaceID,
		Name:        r.Name,
	}
}
