package connection

import (
	"context"

	apiClient "github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
)

// MockConnectionClient is a recording RETLStore double for the connection
// handler tests. The embedded interface covers the source and preview surfaces
// the handler never touches, so an accidental call panics instead of quietly
// returning a zero value.
//
// Every method refuses a context that is already done, the way a real client
// would: a double that ignores cancellation would hide exactly the bugs the
// import path's detached read-back exists to prevent.
type MockConnectionClient struct {
	retlClient.RETLStore

	CreateCalls        []retlClient.CreateRETLConnectionRequest
	UpdateCalls        []string
	DeleteCalls        []string
	ListCalls          []retlClient.ListRETLConnectionsRequest
	GetCalls           []string
	SetExternalIDCalls []retlClient.SetRETLConnectionExternalIDRequest

	// Sources and Destinations are the two endpoint catalogs eligibility reads;
	// EndpointsErr fails both, and the counters say how often they were read.
	Sources           []retlClient.RETLSource
	Destinations      []apiClient.Destination
	EndpointsErr      error
	SourceListCalls   int
	DestinationsCalls int

	CreateFunc        func(req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error)
	UpdateFunc        func(id string, req *retlClient.UpdateRETLConnectionRequest) (*retlClient.RETLConnection, error)
	DeleteFunc        func(id string) error
	ListFunc          func(req *retlClient.ListRETLConnectionsRequest) (*retlClient.RETLConnectionsPage, error)
	GetFunc           func(id string) (*retlClient.RETLConnection, error)
	SetExternalIDFunc func(req *retlClient.SetRETLConnectionExternalIDRequest) error
}

var _ retlClient.RETLStore = (*MockConnectionClient)(nil)

func (m *MockConnectionClient) CreateConnection(ctx context.Context, req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.CreateCalls = append(m.CreateCalls, *req)
	if m.CreateFunc != nil {
		return m.CreateFunc(req)
	}
	return &retlClient.RETLConnection{
		ID:            "conn-remote-1",
		SourceID:      req.SourceID,
		DestinationID: req.DestinationID,
		ExternalID:    req.ExternalID,
	}, nil
}

func (m *MockConnectionClient) UpdateConnection(ctx context.Context, id string, req *retlClient.UpdateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.UpdateCalls = append(m.UpdateCalls, id)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(id, req)
	}
	return &retlClient.RETLConnection{ID: id}, nil
}

func (m *MockConnectionClient) DeleteConnection(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.DeleteCalls = append(m.DeleteCalls, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *MockConnectionClient) ListConnections(ctx context.Context, req *retlClient.ListRETLConnectionsRequest) (*retlClient.RETLConnectionsPage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.ListCalls = append(m.ListCalls, *req)
	if m.ListFunc != nil {
		return m.ListFunc(req)
	}
	return &retlClient.RETLConnectionsPage{}, nil
}

func (m *MockConnectionClient) GetConnection(ctx context.Context, id string) (*retlClient.RETLConnection, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.GetCalls = append(m.GetCalls, id)
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return &retlClient.RETLConnection{ID: id}, nil
}

func (m *MockConnectionClient) SetConnectionExternalId(ctx context.Context, req *retlClient.SetRETLConnectionExternalIDRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.SetExternalIDCalls = append(m.SetExternalIDCalls, *req)
	if m.SetExternalIDFunc != nil {
		return m.SetExternalIDFunc(req)
	}
	return nil
}

func (m *MockConnectionClient) ListRetlSources(ctx context.Context, _ ...retlClient.ListRetlSourcesOption) (*retlClient.RETLSources, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.SourceListCalls++
	if m.EndpointsErr != nil {
		return nil, m.EndpointsErr
	}
	return &retlClient.RETLSources{Data: m.Sources}, nil
}

func (m *MockConnectionClient) GetDestinations(ctx context.Context) ([]apiClient.Destination, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.DestinationsCalls++
	if m.EndpointsErr != nil {
		return nil, m.EndpointsErr
	}
	return m.Destinations, nil
}
