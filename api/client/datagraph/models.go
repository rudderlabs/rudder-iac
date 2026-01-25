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
	// ListModels lists models in a data graph with optional type filtering
	ListModels(ctx context.Context, req *ListModelsRequest) (*ListModelsResponse, error)

	// GetModel retrieves a model by ID (works for both entity and event models)
	GetModel(ctx context.Context, req *GetModelRequest) (*Model, error)

	// CreateModel creates a new model (entity or event based on type field in request)
	CreateModel(ctx context.Context, req *CreateModelRequest) (*Model, error)

	// UpdateModel updates a model (entity or event based on type field in request)
	UpdateModel(ctx context.Context, req *UpdateModelRequest) (*Model, error)

	// DeleteModel deletes a model by ID (works for both entity and event models)
	DeleteModel(ctx context.Context, req *DeleteModelRequest) error

	// SetModelExternalID sets the external ID for a model and returns the updated model (works for both entity and event models)
	SetModelExternalID(ctx context.Context, req *SetModelExternalIDRequest) (*Model, error)
}

// ListModels lists models in a data graph with optional type filtering
func (s *rudderDataGraphClient) ListModels(ctx context.Context, req *ListModelsRequest) (*ListModelsResponse, error) {
	if req.DataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/models", dataGraphsBasePath, req.DataGraphID)

	query := url.Values{}
	if req.Page > 0 {
		query.Add("page", strconv.Itoa(req.Page))
	}
	if req.PageSize > 0 {
		query.Add("pageSize", strconv.Itoa(req.PageSize))
	}
	if req.ModelType != nil {
		query.Add("type", *req.ModelType)
	}
	if req.IsRoot != nil {
		query.Add("isRoot", strconv.FormatBool(*req.IsRoot))
	}
	if req.HasExternalID != nil {
		query.Add("hasExternalId", strconv.FormatBool(*req.HasExternalID))
	}

	if len(query) > 0 {
		path = fmt.Sprintf("%s?%s", path, query.Encode())
	}

	resp, err := s.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing models: %w", err)
	}

	var result ListModelsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// GetModel retrieves a model by ID
func (s *rudderDataGraphClient) GetModel(ctx context.Context, req *GetModelRequest) (*Model, error) {
	if req.DataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if req.ModelID == "" {
		return nil, fmt.Errorf("model ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/models/%s", dataGraphsBasePath, req.DataGraphID, req.ModelID)
	resp, err := s.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting model: %w", err)
	}

	var result Model
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// CreateModel creates a new model
func (s *rudderDataGraphClient) CreateModel(ctx context.Context, req *CreateModelRequest) (*Model, error) {
	if req.DataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("model type cannot be empty")
	}
	if req.Type != "entity" && req.Type != "event" {
		return nil, fmt.Errorf("model type must be 'entity' or 'event'")
	}

	// Validate required fields based on type
	if req.Type == "entity" && req.PrimaryID == "" {
		return nil, fmt.Errorf("primaryId is required for entity models")
	}
	if req.Type == "event" && req.Timestamp == "" {
		return nil, fmt.Errorf("timestamp is required for event models")
	}

	path := fmt.Sprintf("%s/%s/models", dataGraphsBasePath, req.DataGraphID)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	resp, err := s.client.Do(ctx, "POST", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating model: %w", err)
	}

	var result Model
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// UpdateModel updates a model
func (s *rudderDataGraphClient) UpdateModel(ctx context.Context, req *UpdateModelRequest) (*Model, error) {
	if req.DataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if req.ModelID == "" {
		return nil, fmt.Errorf("model ID cannot be empty")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("model type cannot be empty")
	}
	if req.Type != "entity" && req.Type != "event" {
		return nil, fmt.Errorf("model type must be 'entity' or 'event'")
	}

	// Validate required fields based on type
	if req.Type == "entity" && req.PrimaryID == "" {
		return nil, fmt.Errorf("primaryId is required for entity models")
	}
	if req.Type == "event" && req.Timestamp == "" {
		return nil, fmt.Errorf("timestamp is required for event models")
	}

	path := fmt.Sprintf("%s/%s/models/%s", dataGraphsBasePath, req.DataGraphID, req.ModelID)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	resp, err := s.client.Do(ctx, "PUT", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("updating model: %w", err)
	}

	var result Model
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// DeleteModel deletes a model by ID
func (s *rudderDataGraphClient) DeleteModel(ctx context.Context, req *DeleteModelRequest) error {
	if req.DataGraphID == "" {
		return fmt.Errorf("data graph ID cannot be empty")
	}
	if req.ModelID == "" {
		return fmt.Errorf("model ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/models/%s", dataGraphsBasePath, req.DataGraphID, req.ModelID)
	_, err := s.client.Do(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("deleting model: %w", err)
	}

	return nil
}

// SetModelExternalID sets the external ID for a model and returns the updated model
func (s *rudderDataGraphClient) SetModelExternalID(ctx context.Context, req *SetModelExternalIDRequest) (*Model, error) {
	if req.DataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if req.ModelID == "" {
		return nil, fmt.Errorf("model ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/models/%s/external-id", dataGraphsBasePath, req.DataGraphID, req.ModelID)
	data, err := json.Marshal(map[string]string{"externalId": req.ExternalID})
	if err != nil {
		return nil, fmt.Errorf("marshalling external ID: %w", err)
	}

	resp, err := s.client.Do(ctx, "PUT", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("setting external ID: %w", err)
	}

	var result Model
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}
