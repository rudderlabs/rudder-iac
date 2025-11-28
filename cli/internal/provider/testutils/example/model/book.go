package model

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// BookItem represents a single book in the spec
type BookItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Author string `json:"author"` // URN reference to a writer
}

// BookSpec represents the configuration for books (can contain multiple)
type BookSpec struct {
	Books []BookItem `json:"books"`
}

// BookResource represents the input data for a book
type BookResource struct {
	ID     string                 `json:"id"`
	Name   string                 `json:"name"`
	Author *resources.PropertyRef `json:"author"` // Reference to a writer
}

// BookState represents the output state of a book from the remote system
// Only contains computed fields (remote ID)
type BookState struct {
	ID string // Remote ID
}

// RemoteBook wraps backend.RemoteBook to implement RemoteResource interface
type RemoteBook struct {
	*backend.RemoteBook
}

// Metadata implements the RemoteResource interface
func (r RemoteBook) Metadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:         r.ID,
		ExternalID: r.ExternalID,
		Name:       r.Name,
	}
}
