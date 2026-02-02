package testutils

import (
	"context"
	"fmt"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/stretchr/testify/assert"
)

// MockDataGraphClient implements dgClient.DataGraphClient for testing
type MockDataGraphClient struct {
	// DataGraph methods
	DataGraphs          map[string]*dgClient.DataGraph
	ListDataGraphsFunc  func(ctx context.Context, page, perPage int, hasExternalID *bool) (*dgClient.ListDataGraphsResponse, error)
	GetDataGraphFunc    func(ctx context.Context, id string) (*dgClient.DataGraph, error)
	CreateDataGraphFunc func(ctx context.Context, req *dgClient.CreateDataGraphRequest) (*dgClient.DataGraph, error)
	DeleteDataGraphFunc func(ctx context.Context, id string) error
	SetExternalIDFunc   func(ctx context.Context, id string, externalID string) (*dgClient.DataGraph, error)

	// Model methods
	Models                  map[string]*dgClient.Model
	ListModelsFunc          func(ctx context.Context, req *dgClient.ListModelsRequest) (*dgClient.ListModelsResponse, error)
	GetModelFunc            func(ctx context.Context, req *dgClient.GetModelRequest) (*dgClient.Model, error)
	CreateModelFunc         func(ctx context.Context, req *dgClient.CreateModelRequest) (*dgClient.Model, error)
	UpdateModelFunc         func(ctx context.Context, req *dgClient.UpdateModelRequest) (*dgClient.Model, error)
	DeleteModelFunc         func(ctx context.Context, req *dgClient.DeleteModelRequest) error
	SetModelExternalIDFunc  func(ctx context.Context, req *dgClient.SetModelExternalIDRequest) (*dgClient.Model, error)
}

// DataGraph methods

func (m *MockDataGraphClient) ListDataGraphs(ctx context.Context, page, perPage int, hasExternalID *bool) (*dgClient.ListDataGraphsResponse, error) {
	if m.ListDataGraphsFunc != nil {
		return m.ListDataGraphsFunc(ctx, page, perPage, hasExternalID)
	}
	return &dgClient.ListDataGraphsResponse{}, nil
}

func (m *MockDataGraphClient) GetDataGraph(ctx context.Context, id string) (*dgClient.DataGraph, error) {
	if m.GetDataGraphFunc != nil {
		return m.GetDataGraphFunc(ctx, id)
	}
	if dg, ok := m.DataGraphs[id]; ok {
		return dg, nil
	}
	return nil, assert.AnError
}

func (m *MockDataGraphClient) CreateDataGraph(ctx context.Context, req *dgClient.CreateDataGraphRequest) (*dgClient.DataGraph, error) {
	if m.CreateDataGraphFunc != nil {
		return m.CreateDataGraphFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockDataGraphClient) DeleteDataGraph(ctx context.Context, id string) error {
	if m.DeleteDataGraphFunc != nil {
		return m.DeleteDataGraphFunc(ctx, id)
	}
	return nil
}

func (m *MockDataGraphClient) SetExternalID(ctx context.Context, id string, externalID string) (*dgClient.DataGraph, error) {
	if m.SetExternalIDFunc != nil {
		return m.SetExternalIDFunc(ctx, id, externalID)
	}
	return nil, nil
}

// Model methods

func (m *MockDataGraphClient) ListModels(ctx context.Context, req *dgClient.ListModelsRequest) (*dgClient.ListModelsResponse, error) {
	if m.ListModelsFunc != nil {
		return m.ListModelsFunc(ctx, req)
	}
	return &dgClient.ListModelsResponse{}, nil
}

func (m *MockDataGraphClient) GetModel(ctx context.Context, req *dgClient.GetModelRequest) (*dgClient.Model, error) {
	if m.GetModelFunc != nil {
		return m.GetModelFunc(ctx, req)
	}
	if mdl, ok := m.Models[req.ModelID]; ok {
		return mdl, nil
	}
	return nil, assert.AnError
}

func (m *MockDataGraphClient) CreateModel(ctx context.Context, req *dgClient.CreateModelRequest) (*dgClient.Model, error) {
	if m.CreateModelFunc != nil {
		return m.CreateModelFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockDataGraphClient) UpdateModel(ctx context.Context, req *dgClient.UpdateModelRequest) (*dgClient.Model, error) {
	if m.UpdateModelFunc != nil {
		return m.UpdateModelFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockDataGraphClient) DeleteModel(ctx context.Context, req *dgClient.DeleteModelRequest) error {
	if m.DeleteModelFunc != nil {
		return m.DeleteModelFunc(ctx, req)
	}
	return nil
}

func (m *MockDataGraphClient) SetModelExternalID(ctx context.Context, req *dgClient.SetModelExternalIDRequest) (*dgClient.Model, error) {
	if m.SetModelExternalIDFunc != nil {
		return m.SetModelExternalIDFunc(ctx, req)
	}
	return nil, nil
}

// MockURNResolver implements handler.URNResolver for testing
type MockURNResolver struct {
	urnByID map[string]map[string]string // map[resourceType]map[remoteID]urn
}

func (m *MockURNResolver) GetURNByID(resourceType string, remoteID string) (string, error) {
	if m.urnByID == nil {
		return "", fmt.Errorf("URN not found for resource type %s with ID %s", resourceType, remoteID)
	}
	if typeMap, ok := m.urnByID[resourceType]; ok {
		if urn, ok := typeMap[remoteID]; ok {
			return urn, nil
		}
	}
	return "", fmt.Errorf("URN not found for resource type %s with ID %s", resourceType, remoteID)
}

func NewMockURNResolver() *MockURNResolver {
	return &MockURNResolver{
		urnByID: make(map[string]map[string]string),
	}
}

func (m *MockURNResolver) AddMapping(resourceType, remoteID, urn string) {
	if m.urnByID[resourceType] == nil {
		m.urnByID[resourceType] = make(map[string]string)
	}
	m.urnByID[resourceType][remoteID] = urn
}
