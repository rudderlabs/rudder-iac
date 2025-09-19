package source

import (
	"context"

	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
)

type MockSourceClient struct {
	createCalled     bool
	updateCalled     bool
	deleteCalled     bool
	getSourcesCalled bool
	getSourcesFunc   func(ctx context.Context) ([]sourceClient.EventStreamSource, error)
}

func (m *MockSourceClient) Create(ctx context.Context, req *sourceClient.CreateSourceRequest) (*sourceClient.EventStreamSource, error) {
	m.createCalled = true
	return &sourceClient.EventStreamSource{
		ExternalID:  req.ExternalID,
		Name:        req.Name,
		Type:        req.Type,
		Enabled:     req.Enabled,
	}, nil
}

func (m *MockSourceClient) Update(ctx context.Context, sourceID string, req *sourceClient.UpdateSourceRequest) (*sourceClient.EventStreamSource, error) {
	m.updateCalled = true
	return &sourceClient.EventStreamSource{
		ID:          sourceID,
		ExternalID:  "external-123",
		Name:        req.Name,
		Type:        "Javascript",
		Enabled:     req.Enabled,
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