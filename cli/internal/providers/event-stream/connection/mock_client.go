package connection

import (
	"context"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
)

// MockConnectionClient is a hand-rolled EventStreamStore double for connection
// tests. The embedded source mock supplies the source and tracking-plan
// surfaces; the connection methods here record each call — Calls keeps the
// cross-method order so tests can assert sequences like delete-then-create —
// and delegate to an optional per-test func for canned responses.
type MockConnectionClient struct {
	source.MockSourceClient

	Calls []string

	ListCalls   []client.ListConnectionsOptions
	NextCalls   []client.Paging
	CreateCalls []client.Connection
	UpdateCalls []client.Connection
	DeleteCalls []string

	ListFunc   func(options client.ListConnectionsOptions) (*client.ConnectionsPage, error)
	NextFunc   func(paging client.Paging) (*client.ConnectionsPage, error)
	CreateFunc func(connection *client.Connection) (*client.Connection, error)
	UpdateFunc func(connection *client.Connection) (*client.Connection, error)
	DeleteFunc func(id string) error
}

func (m *MockConnectionClient) ListConnections(_ context.Context, opts ...client.ListConnectionsOption) (*client.ConnectionsPage, error) {
	options := client.ListConnectionsOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	m.Calls = append(m.Calls, "ListConnections")
	m.ListCalls = append(m.ListCalls, options)
	if m.ListFunc != nil {
		return m.ListFunc(options)
	}
	return &client.ConnectionsPage{}, nil
}

func (m *MockConnectionClient) NextConnections(_ context.Context, paging client.Paging) (*client.ConnectionsPage, error) {
	m.Calls = append(m.Calls, "NextConnections")
	m.NextCalls = append(m.NextCalls, paging)
	if m.NextFunc != nil {
		return m.NextFunc(paging)
	}
	return nil, nil
}

func (m *MockConnectionClient) CreateConnection(_ context.Context, connection *client.Connection) (*client.Connection, error) {
	m.Calls = append(m.Calls, "CreateConnection")
	m.CreateCalls = append(m.CreateCalls, *connection)
	if m.CreateFunc != nil {
		return m.CreateFunc(connection)
	}
	created := *connection
	created.ID = "remote-connection-id"
	return &created, nil
}

func (m *MockConnectionClient) UpdateConnection(_ context.Context, connection *client.Connection) (*client.Connection, error) {
	m.Calls = append(m.Calls, "UpdateConnection")
	m.UpdateCalls = append(m.UpdateCalls, *connection)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(connection)
	}
	updated := *connection
	return &updated, nil
}

func (m *MockConnectionClient) DeleteConnection(_ context.Context, id string) error {
	m.Calls = append(m.Calls, "DeleteConnection")
	m.DeleteCalls = append(m.DeleteCalls, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}
