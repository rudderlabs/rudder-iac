package retl

import (
	"context"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// RETLStore is the interface for RETL operations
type RETLStore interface {
	RETLSourceStore
	RETLConnectionStore
	PreviewStore
}

// RETLConnectionStore is the interface for RETL connection operations.
type RETLConnectionStore interface {
	// CreateConnection creates a new RETL connection.
	CreateConnection(ctx context.Context, req *CreateRETLConnectionRequest) (*RETLConnection, error)

	// UpdateConnection updates mutable fields of a RETL connection.
	UpdateConnection(ctx context.Context, id string, req *UpdateRETLConnectionRequest) (*RETLConnection, error)

	// DeleteConnection soft-deletes a RETL connection by ID.
	DeleteConnection(ctx context.Context, id string) error

	// GetConnection retrieves a RETL connection by ID.
	GetConnection(ctx context.Context, id string) (*RETLConnection, error)

	// ListConnections returns a paginated list of RETL connections matching the provided filters.
	ListConnections(ctx context.Context, req *ListRETLConnectionsRequest) (*RETLConnectionsPage, error)

	// SetConnectionExternalID sets the external ID for a RETL connection.
	SetConnectionExternalID(ctx context.Context, req *SetRETLConnectionExternalIDRequest) error
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
