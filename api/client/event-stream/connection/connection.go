package connection

import (
	"context"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// ConnectionStore is the event stream slice of the generic /v2/connections
// API, plus the narrow destinations read connection import needs. Method
// names carry the Connection token so the interface can be embedded in
// EventStreamStore alongside SourceStore without clashing.
type ConnectionStore interface {
	ListConnections(ctx context.Context, opts ...client.ListConnectionsOption) (*client.ConnectionsPage, error)
	NextConnections(ctx context.Context, paging client.Paging) (*client.ConnectionsPage, error)
	GetConnection(ctx context.Context, id string) (*client.Connection, error)
	CreateConnection(ctx context.Context, connection *client.Connection) (*client.Connection, error)
	UpdateConnection(ctx context.Context, connection *client.Connection) (*client.Connection, error)
	DeleteConnection(ctx context.Context, id string) error
	SetConnectionExternalID(ctx context.Context, id string, externalID string) error
	// GetDestinations lists the workspace's destinations — connection import
	// needs destination names to mint readable connection ids.
	GetDestinations(ctx context.Context) ([]client.Destination, error)
}

type rudderConnectionStore struct {
	client *client.Client
}

func NewRudderConnectionStore(client *client.Client) ConnectionStore {
	return &rudderConnectionStore{
		client: client,
	}
}

func (r *rudderConnectionStore) ListConnections(ctx context.Context, opts ...client.ListConnectionsOption) (*client.ConnectionsPage, error) {
	return r.client.Connections.List(ctx, opts...)
}

func (r *rudderConnectionStore) NextConnections(ctx context.Context, paging client.Paging) (*client.ConnectionsPage, error) {
	return r.client.Connections.Next(ctx, paging)
}

func (r *rudderConnectionStore) GetConnection(ctx context.Context, id string) (*client.Connection, error) {
	return r.client.Connections.Get(ctx, id)
}

func (r *rudderConnectionStore) CreateConnection(ctx context.Context, connection *client.Connection) (*client.Connection, error) {
	return r.client.Connections.Create(ctx, connection)
}

func (r *rudderConnectionStore) UpdateConnection(ctx context.Context, connection *client.Connection) (*client.Connection, error) {
	return r.client.Connections.Update(ctx, connection)
}

func (r *rudderConnectionStore) DeleteConnection(ctx context.Context, id string) error {
	return r.client.Connections.Delete(ctx, id)
}

func (r *rudderConnectionStore) SetConnectionExternalID(ctx context.Context, id string, externalID string) error {
	return r.client.Connections.SetExternalID(ctx, id, externalID)
}

func (r *rudderConnectionStore) GetDestinations(ctx context.Context) ([]client.Destination, error) {
	return r.client.Destinations.GetAll(ctx)
}
