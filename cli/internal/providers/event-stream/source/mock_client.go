package source

import (
	"context"

	"github.com/rudderlabs/rudder-iac/api/client"
	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	trackingplanClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/tracking-plan-connection"
)

type MockSourceClient struct {
	createCalled             bool
	updateCalled             bool
	deleteCalled             bool
	linkTPCalled             bool
	unlinkTPCalled           bool
	updateTPConnectionCalled bool
	getSourcesCalled         bool
	setExternalIDCalled      bool
	getSourcesFunc           func(ctx context.Context) ([]sourceClient.EventStreamSource, error)
}

func (m *MockSourceClient) Create(ctx context.Context, req *sourceClient.CreateSourceRequest) (*sourceClient.CreateUpdateSourceResponse, error) {
	m.createCalled = true
	return &sourceClient.CreateUpdateSourceResponse{
		ExternalID: req.ExternalID,
		Name:       req.Name,
		Type:       req.Type,
		Enabled:    req.Enabled,
	}, nil
}

func (m *MockSourceClient) Update(ctx context.Context, sourceID string, req *sourceClient.UpdateSourceRequest) (*sourceClient.CreateUpdateSourceResponse, error) {
	m.updateCalled = true
	return &sourceClient.CreateUpdateSourceResponse{
		ID:         sourceID,
		ExternalID: "external-123",
		Name:       req.Name,
		Type:       "javascript",
		Enabled:    req.Enabled,
	}, nil
}

func (m *MockSourceClient) Delete(ctx context.Context, sourceID string) error {
	m.deleteCalled = true
	return nil
}

func (m *MockSourceClient) GetSources(ctx context.Context) ([]sourceClient.EventStreamSource, error) {
	m.getSourcesCalled = true
	if m.getSourcesFunc != nil {
		return m.getSourcesFunc(ctx)
	}
	return []sourceClient.EventStreamSource{}, nil
}

func (m *MockSourceClient) LinkTP(ctx context.Context, trackingPlanID string, sourceID string, req *trackingplanClient.ConnectionConfig) error {
	m.linkTPCalled = true
	return nil
}

func (m *MockSourceClient) UnlinkTP(ctx context.Context, trackingPlanID string, sourceID string) error {
	m.unlinkTPCalled = true
	return nil
}

func (m *MockSourceClient) UpdateTPConnection(ctx context.Context, trackingPlanID string, sourceId string, config *trackingplanClient.ConnectionConfig) error {
	m.updateTPConnectionCalled = true
	return nil
}

func (m *MockSourceClient) SetExternalID(ctx context.Context, sourceID string, externalID string) error {
	m.setExternalIDCalled = true
	return nil
}

func NewMockSourceClient() *MockSourceClient {
	return &MockSourceClient{}
}

func (m *MockSourceClient) SetGetSourcesFunc(f func(ctx context.Context) ([]sourceClient.EventStreamSource, error)) {
	m.getSourcesFunc = f
}

func (m *MockSourceClient) CreateCalled() bool {
	return m.createCalled
}

func (m *MockSourceClient) UpdateCalled() bool {
	return m.updateCalled
}

func (m *MockSourceClient) DeleteCalled() bool {
	return m.deleteCalled
}

func (m *MockSourceClient) GetSourcesCalled() bool {
	return m.getSourcesCalled
}

func (m *MockSourceClient) LinkTPCalled() bool {
	return m.linkTPCalled
}

func (m *MockSourceClient) UnlinkTPCalled() bool {
	return m.unlinkTPCalled
}

func (m *MockSourceClient) UpdateTPConnectionCalled() bool {
	return m.updateTPConnectionCalled
}

func (m *MockSourceClient) SetExternalIDCalled() bool {
	return m.setExternalIDCalled
}

// Connection methods keep MockSourceClient satisfying the full
// EventStreamStore. Tests that exercise connection behaviour use the
// richer connection.MockConnectionClient instead.

func (m *MockSourceClient) ListConnections(ctx context.Context, opts ...client.ListConnectionsOption) (*client.ConnectionsPage, error) {
	return &client.ConnectionsPage{}, nil
}

func (m *MockSourceClient) NextConnections(ctx context.Context, paging client.Paging) (*client.ConnectionsPage, error) {
	return nil, nil
}

func (m *MockSourceClient) CreateConnection(ctx context.Context, connection *client.Connection) (*client.Connection, error) {
	created := *connection
	created.ID = "remote-connection-id"
	return &created, nil
}

func (m *MockSourceClient) UpdateConnection(ctx context.Context, connection *client.Connection) (*client.Connection, error) {
	updated := *connection
	return &updated, nil
}

func (m *MockSourceClient) DeleteConnection(ctx context.Context, id string) error {
	return nil
}

func (m *MockSourceClient) GetConnection(ctx context.Context, id string) (*client.Connection, error) {
	return &client.Connection{ID: id}, nil
}

func (m *MockSourceClient) SetConnectionExternalID(ctx context.Context, id string, externalID string) error {
	return nil
}

func (m *MockSourceClient) GetDestinations(ctx context.Context) ([]client.Destination, error) {
	return []client.Destination{}, nil
}
