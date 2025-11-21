package retl

import (
	"context"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// RETLStore is the interface for RETL operations
type RETLStore interface {
	RETLSourceStore
	PreviewStore
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
	ListRetlSources(ctx context.Context, hasExternalId *bool) (*RETLSources, error)

	// SetExternalId sets the external ID for a RETL source
	SetExternalId(ctx context.Context, id string, externalId string) error
}

// PreviewStore is the interface for RETL source preview operations
type PreviewStore interface {
	// SubmitSourcePreview submits a request to preview a RETL source
	SubmitSourcePreview(ctx context.Context, request *PreviewSubmitRequest) (*PreviewSubmitResponse, error)

	// GetSourcePreviewResult retrieves the results of a RETL source preview
	GetSourcePreviewResult(ctx context.Context, resultID string) (*PreviewResultResponse, error)
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
