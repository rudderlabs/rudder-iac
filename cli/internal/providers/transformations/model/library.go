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
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Code        string `json:"code"` // Resolved from inline or file
	ImportName  string `json:"import_name"`
}

// LibraryState represents the output state from the remote system
// Contains computed fields (remote ID, version ID)
type LibraryState struct {
	ID        string `json:"id"`        // Remote ID
	VersionID string `json:"versionId"` // Remote version ID
	Modified  bool   `json:"-" mapstructure:"-"` // Used to determine if it is modified during plan operations
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
