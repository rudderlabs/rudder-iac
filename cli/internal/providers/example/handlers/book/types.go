package book

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// BookItem represents a single book in the spec
type BookItem struct {
	ID     string `mapstructure:"id"`
	Name   string `mapstructure:"name"`
	Author string `mapstructure:"author"` // URN reference to a writer
}

// BookSpec represents the configuration for books (can contain multiple)
type BookSpec struct {
	Books []BookItem `mapstructure:"books"`
}

// BookResource represents the input data for a book
type BookResource struct {
	ID     string                 `mapstructure:"id"`
	Name   string                 `mapstructure:"name"`
	Author *resources.PropertyRef `mapstructure:"author"` // Reference to a writer
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

// GetResourceMetadata implements the RemoteResource interface
func (r RemoteBook) GetResourceMetadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:         r.ID,
		ExternalID: r.ExternalID,
		Name:       r.Name,
	}
}
