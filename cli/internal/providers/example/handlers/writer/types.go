package writer

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/example/backend"
)

// WriterSpec represents the configuration for a writer resource
type WriterSpec struct {
	ID   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
}

// WriterResource represents the input data for a writer
type WriterResource struct {
	ID   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
}

// WriterState represents the output state of a writer from the remote system
// Only contains computed fields (remote ID)
type WriterState struct {
	ID string // Remote ID
}

// RemoteWriter wraps backend.RemoteWriter to implement RemoteResource interface
type RemoteWriter struct {
	*backend.RemoteWriter
}

// GetResourceMetadata implements the RemoteResource interface
func (r RemoteWriter) GetResourceMetadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:         r.ID,
		ExternalID: r.ExternalID,
		Name:       r.Name,
	}
}
