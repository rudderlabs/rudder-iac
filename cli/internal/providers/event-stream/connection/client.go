package connection

import (
	"context"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// ConnectionStore is the slice of the generic connections API client the
// event stream connection handler uses. The client.Client Connections service
// satisfies it.
type ConnectionStore interface {
	List(ctx context.Context, opts ...client.ListConnectionsOption) (*client.ConnectionsPage, error)
	Next(ctx context.Context, paging client.Paging) (*client.ConnectionsPage, error)
	Create(ctx context.Context, connection *client.Connection) (*client.Connection, error)
	Update(ctx context.Context, connection *client.Connection) (*client.Connection, error)
	Delete(ctx context.Context, id string) error
}
