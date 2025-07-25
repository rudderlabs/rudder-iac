package retl

import (
	"context"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// RETLStore is the interface for RETL operations
type RETLStore interface {
	StateStore
	RETLSourceStore
}

// StateStore is the interface for RETL state operations
type StateStore interface {
	// ReadState retrieves the complete RETL state
	ReadState(ctx context.Context) (*State, error)

	// PutResourceState saves a resource state record
	PutResourceState(ctx context.Context, id string, req PutStateRequest) error
}

// RETLSourceStore is the interface for RETL source operations
type RETLSourceStore interface {
	// CreateRetlSource creates a new RETL source
	CreateRetlSource(ctx context.Context, source *RETLSourceCreateRequest) (*RETLSource, error)

	// UpdateRetlSource updates an existing RETL source
	UpdateRetlSource(ctx context.Context, id string, source *RETLSourceUpdateRequest) (*RETLSource, error)

	// DeleteRetlSource deletes a RETL source by ID
	DeleteRetlSource(ctx context.Context, id string) error

	// GetRetlSource retrieves a RETL source by ID
	GetRetlSource(ctx context.Context, id string) (*RETLSource, error)

	// ListRetlSources lists all RETL sources
	ListRetlSources(ctx context.Context) (*RETLSources, error)
}

// RudderRETLStore implements the RETLStore interface
type RudderRETLStore struct {
	client *client.Client
}

// NewRudderRETLStore creates a new RETLStore implementation
func NewRudderRETLStore(client *client.Client) RETLStore {
	store := &RudderRETLStore{
		client: client,
	}
	return store
}
