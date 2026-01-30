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
	Models                       map[string]*dgClient.Model
	ListEntityModelsFunc         func(ctx context.Context, dataGraphID string, page, pageSize int, isRoot *bool, hasExternalID *bool) (*dgClient.ListModelsResponse, error)
	ListEventModelsFunc          func(ctx context.Context, dataGraphID string, page, pageSize int, hasExternalID *bool) (*dgClient.ListModelsResponse, error)
	GetEntityModelFunc           func(ctx context.Context, dataGraphID, modelID string) (*dgClient.Model, error)
	GetEventModelFunc            func(ctx context.Context, dataGraphID, modelID string) (*dgClient.Model, error)
	CreateEntityModelFunc        func(ctx context.Context, dataGraphID string, req *dgClient.CreateEntityModelRequest) (*dgClient.Model, error)
	CreateEventModelFunc         func(ctx context.Context, dataGraphID string, req *dgClient.CreateEventModelRequest) (*dgClient.Model, error)
	UpdateEntityModelFunc        func(ctx context.Context, dataGraphID, modelID string, req *dgClient.UpdateEntityModelRequest) (*dgClient.Model, error)
	UpdateEventModelFunc         func(ctx context.Context, dataGraphID, modelID string, req *dgClient.UpdateEventModelRequest) (*dgClient.Model, error)
	DeleteEntityModelFunc        func(ctx context.Context, dataGraphID, modelID string) error
	DeleteEventModelFunc         func(ctx context.Context, dataGraphID, modelID string) error
	SetEntityModelExternalIDFunc func(ctx context.Context, dataGraphID, modelID, externalID string) error
	SetEventModelExternalIDFunc  func(ctx context.Context, dataGraphID, modelID, externalID string) error
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

func (m *MockDataGraphClient) ListEntityModels(ctx context.Context, dataGraphID string, page, pageSize int, isRoot *bool, hasExternalID *bool) (*dgClient.ListModelsResponse, error) {
	if m.ListEntityModelsFunc != nil {
		return m.ListEntityModelsFunc(ctx, dataGraphID, page, pageSize, isRoot, hasExternalID)
	}
	return &dgClient.ListModelsResponse{}, nil
}

func (m *MockDataGraphClient) ListEventModels(ctx context.Context, dataGraphID string, page, pageSize int, hasExternalID *bool) (*dgClient.ListModelsResponse, error) {
	if m.ListEventModelsFunc != nil {
		return m.ListEventModelsFunc(ctx, dataGraphID, page, pageSize, hasExternalID)
	}
	return &dgClient.ListModelsResponse{}, nil
}

func (m *MockDataGraphClient) GetEntityModel(ctx context.Context, dataGraphID, modelID string) (*dgClient.Model, error) {
	if m.GetEntityModelFunc != nil {
		return m.GetEntityModelFunc(ctx, dataGraphID, modelID)
	}
	if mdl, ok := m.Models[modelID]; ok && mdl.Type == "entity" {
		return mdl, nil
	}
	return nil, assert.AnError
}

func (m *MockDataGraphClient) GetEventModel(ctx context.Context, dataGraphID, modelID string) (*dgClient.Model, error) {
	if m.GetEventModelFunc != nil {
		return m.GetEventModelFunc(ctx, dataGraphID, modelID)
	}
	if mdl, ok := m.Models[modelID]; ok && mdl.Type == "event" {
		return mdl, nil
	}
	return nil, assert.AnError
}

func (m *MockDataGraphClient) CreateEntityModel(ctx context.Context, dataGraphID string, req *dgClient.CreateEntityModelRequest) (*dgClient.Model, error) {
	if m.CreateEntityModelFunc != nil {
		return m.CreateEntityModelFunc(ctx, dataGraphID, req)
	}
	return nil, nil
}

func (m *MockDataGraphClient) CreateEventModel(ctx context.Context, dataGraphID string, req *dgClient.CreateEventModelRequest) (*dgClient.Model, error) {
	if m.CreateEventModelFunc != nil {
		return m.CreateEventModelFunc(ctx, dataGraphID, req)
	}
	return nil, nil
}

func (m *MockDataGraphClient) UpdateEntityModel(ctx context.Context, dataGraphID, modelID string, req *dgClient.UpdateEntityModelRequest) (*dgClient.Model, error) {
	if m.UpdateEntityModelFunc != nil {
		return m.UpdateEntityModelFunc(ctx, dataGraphID, modelID, req)
	}
	return nil, nil
}

func (m *MockDataGraphClient) UpdateEventModel(ctx context.Context, dataGraphID, modelID string, req *dgClient.UpdateEventModelRequest) (*dgClient.Model, error) {
	if m.UpdateEventModelFunc != nil {
		return m.UpdateEventModelFunc(ctx, dataGraphID, modelID, req)
	}
	return nil, nil
}

func (m *MockDataGraphClient) DeleteEntityModel(ctx context.Context, dataGraphID, modelID string) error {
	if m.DeleteEntityModelFunc != nil {
		return m.DeleteEntityModelFunc(ctx, dataGraphID, modelID)
	}
	return nil
}

func (m *MockDataGraphClient) DeleteEventModel(ctx context.Context, dataGraphID, modelID string) error {
	if m.DeleteEventModelFunc != nil {
		return m.DeleteEventModelFunc(ctx, dataGraphID, modelID)
	}
	return nil
}

func (m *MockDataGraphClient) SetEntityModelExternalID(ctx context.Context, dataGraphID, modelID, externalID string) error {
	if m.SetEntityModelExternalIDFunc != nil {
		return m.SetEntityModelExternalIDFunc(ctx, dataGraphID, modelID, externalID)
	}
	return nil
}

func (m *MockDataGraphClient) SetEventModelExternalID(ctx context.Context, dataGraphID, modelID, externalID string) error {
	if m.SetEventModelExternalIDFunc != nil {
		return m.SetEventModelExternalIDFunc(ctx, dataGraphID, modelID, externalID)
	}
	return nil
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
