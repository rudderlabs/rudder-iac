package model

import (
	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
)

// LibrarySpec is an alias to the spec package type
type LibrarySpec = specs.TransformationLibrarySpec

// LibraryResource represents the input data for a transformation library
type LibraryResource struct {
	ID          string
	Name        string
	Description string
	Language    string
	Code        string // Resolved from inline or file
	ImportName  string
}

// LibraryState represents the output state from the remote system
// Contains computed fields (remote ID, version ID)
type LibraryState struct {
	ID        string // Remote ID
	VersionID string // Remote version ID
}

// RemoteLibrary wraps transformations.TransformationLibrary to implement RemoteResource interface
type RemoteLibrary struct {
	*transformations.TransformationLibrary
}

// Metadata implements the RemoteResource interface
func (r RemoteLibrary) Metadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:         r.ID,
		ExternalID: r.ExternalID,
		Name:       r.Name,
	}
}