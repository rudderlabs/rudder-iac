package connection

import (
	"context"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
)

// MockConnectionClient is a recording RETLStore double for the connection
// handler tests. The embedded interface covers the source and preview surfaces
// the handler never touches, so an accidental call panics instead of quietly
// returning a zero value.
type MockConnectionClient struct {
	retlClient.RETLStore

	CreateCalls []retlClient.CreateRETLConnectionRequest
	UpdateCalls []string
	DeleteCalls []string

	CreateFunc func(req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error)
	UpdateFunc func(id string, req *retlClient.UpdateRETLConnectionRequest) (*retlClient.RETLConnection, error)
	DeleteFunc func(id string) error
}

var _ retlClient.RETLStore = (*MockConnectionClient)(nil)

func (m *MockConnectionClient) CreateConnection(_ context.Context, req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
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

func (m *MockConnectionClient) UpdateConnection(_ context.Context, id string, req *retlClient.UpdateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
	m.UpdateCalls = append(m.UpdateCalls, id)
	if m.UpdateFunc != nil {
		return m.UpdateFunc(id, req)
	}
	return &retlClient.RETLConnection{ID: id}, nil
}

func (m *MockConnectionClient) DeleteConnection(_ context.Context, id string) error {
	m.DeleteCalls = append(m.DeleteCalls, id)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}
