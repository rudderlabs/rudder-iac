package datagraph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ModelStore is the interface for Model operations
type ModelStore interface {
	// Entity Model operations
	ListEntityModels(ctx context.Context, dataGraphID string, page, pageSize int, isRoot *bool, hasExternalID *bool) (*ListModelsResponse, error)
	GetEntityModel(ctx context.Context, dataGraphID, modelID string) (*Model, error)
	CreateEntityModel(ctx context.Context, dataGraphID string, req *CreateEntityModelRequest) (*Model, error)
	UpdateEntityModel(ctx context.Context, dataGraphID, modelID string, req *UpdateEntityModelRequest) (*Model, error)
	DeleteEntityModel(ctx context.Context, dataGraphID, modelID string) error
	SetEntityModelExternalID(ctx context.Context, dataGraphID, modelID, externalID string) error

	// Event Model operations
	ListEventModels(ctx context.Context, dataGraphID string, page, pageSize int, hasExternalID *bool) (*ListModelsResponse, error)
	GetEventModel(ctx context.Context, dataGraphID, modelID string) (*Model, error)
	CreateEventModel(ctx context.Context, dataGraphID string, req *CreateEventModelRequest) (*Model, error)
	UpdateEventModel(ctx context.Context, dataGraphID, modelID string, req *UpdateEventModelRequest) (*Model, error)
	DeleteEventModel(ctx context.Context, dataGraphID, modelID string) error
	SetEventModelExternalID(ctx context.Context, dataGraphID, modelID, externalID string) error
}

// ListEntityModels lists entity models in a data graph
func (s *rudderDataGraphClient) ListEntityModels(ctx context.Context, dataGraphID string, page, pageSize int, isRoot *bool, hasExternalID *bool) (*ListModelsResponse, error) {
	filters := map[string]*bool{
		"isRoot":        isRoot,
		"hasExternalId": hasExternalID,
	}
	return s.listModels(ctx, dataGraphID, page, pageSize, "entity", filters)
}

// ListEventModels lists event models in a data graph
func (s *rudderDataGraphClient) ListEventModels(ctx context.Context, dataGraphID string, page, pageSize int, hasExternalID *bool) (*ListModelsResponse, error) {
	filters := map[string]*bool{
		"hasExternalId": hasExternalID,
	}
	return s.listModels(ctx, dataGraphID, page, pageSize, "event", filters)
}

// GetEntityModel retrieves an entity model by ID
func (s *rudderDataGraphClient) GetEntityModel(ctx context.Context, dataGraphID, modelID string) (*Model, error) {
	return s.getModel(ctx, dataGraphID, modelID, "entity")
}

// GetEventModel retrieves an event model by ID
func (s *rudderDataGraphClient) GetEventModel(ctx context.Context, dataGraphID, modelID string) (*Model, error) {
	return s.getModel(ctx, dataGraphID, modelID, "event")
}

// CreateEntityModel creates a new entity model
func (s *rudderDataGraphClient) CreateEntityModel(ctx context.Context, dataGraphID string, req *CreateEntityModelRequest) (*Model, error) {
	return s.createModel(ctx, dataGraphID, req, "entity")
}

// CreateEventModel creates a new event model
func (s *rudderDataGraphClient) CreateEventModel(ctx context.Context, dataGraphID string, req *CreateEventModelRequest) (*Model, error) {
	return s.createModel(ctx, dataGraphID, req, "event")
}

// UpdateEntityModel updates an existing entity model
func (s *rudderDataGraphClient) UpdateEntityModel(ctx context.Context, dataGraphID, modelID string, req *UpdateEntityModelRequest) (*Model, error) {
	return s.updateModel(ctx, dataGraphID, modelID, req, "entity")
}

// UpdateEventModel updates an existing event model
func (s *rudderDataGraphClient) UpdateEventModel(ctx context.Context, dataGraphID, modelID string, req *UpdateEventModelRequest) (*Model, error) {
	return s.updateModel(ctx, dataGraphID, modelID, req, "event")
}

// DeleteEntityModel deletes an entity model by ID
func (s *rudderDataGraphClient) DeleteEntityModel(ctx context.Context, dataGraphID, modelID string) error {
	return s.deleteModel(ctx, dataGraphID, modelID, "entity")
}

// DeleteEventModel deletes an event model by ID
func (s *rudderDataGraphClient) DeleteEventModel(ctx context.Context, dataGraphID, modelID string) error {
	return s.deleteModel(ctx, dataGraphID, modelID, "event")
}

// SetEntityModelExternalID sets the external ID for an entity model
func (s *rudderDataGraphClient) SetEntityModelExternalID(ctx context.Context, dataGraphID, modelID, externalID string) error {
	return s.setModelExternalID(ctx, dataGraphID, modelID, externalID, "entity")
}

// SetEventModelExternalID sets the external ID for an event model
func (s *rudderDataGraphClient) SetEventModelExternalID(ctx context.Context, dataGraphID, modelID, externalID string) error {
	return s.setModelExternalID(ctx, dataGraphID, modelID, externalID, "event")
}

// Private helper functions

// listModels is a common helper for listing entity or event models with optional filters
func (s *rudderDataGraphClient) listModels(ctx context.Context, dataGraphID string, page, pageSize int, modelType string, filters map[string]*bool) (*ListModelsResponse, error) {
	if dataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s-models", dataGraphsBasePath, dataGraphID, modelType)

	query := url.Values{}
	if page > 0 {
		query.Add("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		query.Add("pageSize", strconv.Itoa(pageSize))
	}

	// Add filters
	for key, value := range filters {
		if value != nil {
			query.Add(key, strconv.FormatBool(*value))
		}
	}

	if len(query) > 0 {
		path = fmt.Sprintf("%s?%s", path, query.Encode())
	}

	resp, err := s.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing %s models: %w", modelType, err)
	}

	var result ListModelsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	// Set type on each model
	for i := range result.Data {
		result.Data[i].Type = modelType
	}

	return &result, nil
}

// getModel is a common helper for getting entity or event models
func (s *rudderDataGraphClient) getModel(ctx context.Context, dataGraphID, modelID, modelType string) (*Model, error) {
	if dataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if modelID == "" {
		return nil, fmt.Errorf("model ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s-models/%s", dataGraphsBasePath, dataGraphID, modelType, modelID)
	resp, err := s.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting %s model: %w", modelType, err)
	}

	var result Model
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	result.Type = modelType
	return &result, nil
}

// createModel is a common helper for creating entity or event models
func (s *rudderDataGraphClient) createModel(ctx context.Context, dataGraphID string, req any, modelType string) (*Model, error) {
	if dataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s-models", dataGraphsBasePath, dataGraphID, modelType)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	resp, err := s.client.Do(ctx, "POST", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating %s model: %w", modelType, err)
	}

	var result Model
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	result.Type = modelType
	return &result, nil
}

// updateModel is a common helper for updating entity or event models
func (s *rudderDataGraphClient) updateModel(ctx context.Context, dataGraphID, modelID string, req any, modelType string) (*Model, error) {
	if dataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if modelID == "" {
		return nil, fmt.Errorf("model ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s-models/%s", dataGraphsBasePath, dataGraphID, modelType, modelID)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	resp, err := s.client.Do(ctx, "PUT", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("updating %s model: %w", modelType, err)
	}

	var result Model
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	result.Type = modelType
	return &result, nil
}

// deleteModel is a common helper for deleting entity or event models
func (s *rudderDataGraphClient) deleteModel(ctx context.Context, dataGraphID, modelID, modelType string) error {
	if dataGraphID == "" {
		return fmt.Errorf("data graph ID cannot be empty")
	}
	if modelID == "" {
		return fmt.Errorf("model ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s-models/%s", dataGraphsBasePath, dataGraphID, modelType, modelID)
	_, err := s.client.Do(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("deleting %s model: %w", modelType, err)
	}

	return nil
}

// setModelExternalID is a common helper for setting external IDs on entity or event models
func (s *rudderDataGraphClient) setModelExternalID(ctx context.Context, dataGraphID, modelID, externalID, modelType string) error {
	if dataGraphID == "" {
		return fmt.Errorf("data graph ID cannot be empty")
	}
	if modelID == "" {
		return fmt.Errorf("model ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s-models/%s/external-id", dataGraphsBasePath, dataGraphID, modelType, modelID)
	data, err := json.Marshal(map[string]string{"externalId": externalID})
	if err != nil {
		return fmt.Errorf("marshalling external ID: %w", err)
	}

	_, err = s.client.Do(ctx, "PUT", path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("setting external ID: %w", err)
	}

	return nil
}
