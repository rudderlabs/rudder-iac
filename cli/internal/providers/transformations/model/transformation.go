package model

import (
	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
)

// TransformationSpec is an alias to the spec package type
type TransformationSpec = specs.TransformationSpec

// TransformationResource represents the input data for a transformation
type TransformationResource struct {
	ID          string
	Name        string
	Description string
	Language    string
	Code        string // Resolved from inline or file
	Tests       []specs.TransformationTest
}

// TransformationState represents the output state from the remote system
// Contains computed fields (remote ID, version ID)
type TransformationState struct {
	ID        string // Remote ID
	VersionID string // Remote version ID
}

// RemoteTransformation wraps transformations.Transformation to implement RemoteResource interface
type RemoteTransformation struct {
	*transformations.Transformation
}

// Metadata implements the RemoteResource interface
func (r RemoteTransformation) Metadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:         r.ID,
		ExternalID: r.ExternalID,
		Name:       r.Name,
	}
}
