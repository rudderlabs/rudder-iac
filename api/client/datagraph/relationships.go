package datagraph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// RelationshipStore is the interface for Relationship operations
type RelationshipStore interface {
	// ListRelationships lists relationships in a data graph
	ListRelationships(ctx context.Context, req *ListRelationshipsRequest) (*ListRelationshipsResponse, error)

	// GetRelationship retrieves a relationship by ID
	GetRelationship(ctx context.Context, req *GetRelationshipRequest) (*Relationship, error)

	// CreateRelationship creates a new relationship
	CreateRelationship(ctx context.Context, req *CreateRelationshipRequest) (*Relationship, error)

	// UpdateRelationship updates an existing relationship
	UpdateRelationship(ctx context.Context, req *UpdateRelationshipRequest) (*Relationship, error)

	// DeleteRelationship deletes a relationship by ID
	DeleteRelationship(ctx context.Context, req *DeleteRelationshipRequest) error

	// SetRelationshipExternalID sets the external ID for a relationship and returns the updated relationship
	SetRelationshipExternalID(ctx context.Context, req *SetRelationshipExternalIDRequest) (*Relationship, error)
}

// ListRelationships lists relationships in a data graph
func (s *rudderDataGraphClient) ListRelationships(ctx context.Context, req *ListRelationshipsRequest) (*ListRelationshipsResponse, error) {
	if req.DataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/relationships", dataGraphsBasePath, req.DataGraphID)

	query := url.Values{}
	if req.Page > 0 {
		query.Add("page", strconv.Itoa(req.Page))
	}
	if req.PageSize > 0 {
		query.Add("pageSize", strconv.Itoa(req.PageSize))
	}
	if req.SourceModelID != nil {
		query.Add("sourceModelId", *req.SourceModelID)
	}
	if req.HasExternalID != nil {
		query.Add("hasExternalId", strconv.FormatBool(*req.HasExternalID))
	}

	if len(query) > 0 {
		path = fmt.Sprintf("%s?%s", path, query.Encode())
	}

	resp, err := s.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing relationships: %w", err)
	}

	var result ListRelationshipsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// GetRelationship retrieves a relationship by ID
func (s *rudderDataGraphClient) GetRelationship(ctx context.Context, req *GetRelationshipRequest) (*Relationship, error) {
	if req.DataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if req.RelationshipID == "" {
		return nil, fmt.Errorf("relationship ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/relationships/%s", dataGraphsBasePath, req.DataGraphID, req.RelationshipID)
	resp, err := s.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting relationship: %w", err)
	}

	var result Relationship
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// CreateRelationship creates a new relationship
func (s *rudderDataGraphClient) CreateRelationship(ctx context.Context, req *CreateRelationshipRequest) (*Relationship, error) {
	if req.DataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if req.Cardinality == "" {
		return nil, fmt.Errorf("cardinality is required")
	}

	path := fmt.Sprintf("%s/%s/relationships", dataGraphsBasePath, req.DataGraphID)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	resp, err := s.client.Do(ctx, "POST", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating relationship: %w", err)
	}

	var result Relationship
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// UpdateRelationship updates a relationship
func (s *rudderDataGraphClient) UpdateRelationship(ctx context.Context, req *UpdateRelationshipRequest) (*Relationship, error) {
	if req.DataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if req.RelationshipID == "" {
		return nil, fmt.Errorf("relationship ID cannot be empty")
	}
	if req.Cardinality == "" {
		return nil, fmt.Errorf("cardinality is required")
	}

	path := fmt.Sprintf("%s/%s/relationships/%s", dataGraphsBasePath, req.DataGraphID, req.RelationshipID)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	resp, err := s.client.Do(ctx, "PUT", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("updating relationship: %w", err)
	}

	var result Relationship
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// DeleteRelationship deletes a relationship by ID
func (s *rudderDataGraphClient) DeleteRelationship(ctx context.Context, req *DeleteRelationshipRequest) error {
	if req.DataGraphID == "" {
		return fmt.Errorf("data graph ID cannot be empty")
	}
	if req.RelationshipID == "" {
		return fmt.Errorf("relationship ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/relationships/%s", dataGraphsBasePath, req.DataGraphID, req.RelationshipID)
	_, err := s.client.Do(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("deleting relationship: %w", err)
	}

	return nil
}

// SetRelationshipExternalID sets the external ID for a relationship and returns the updated relationship
func (s *rudderDataGraphClient) SetRelationshipExternalID(ctx context.Context, req *SetRelationshipExternalIDRequest) (*Relationship, error) {
	if req.DataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if req.RelationshipID == "" {
		return nil, fmt.Errorf("relationship ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/relationships/%s/external-id", dataGraphsBasePath, req.DataGraphID, req.RelationshipID)
	data, err := json.Marshal(map[string]string{"externalId": req.ExternalID})
	if err != nil {
		return nil, fmt.Errorf("marshalling external ID: %w", err)
	}

	resp, err := s.client.Do(ctx, "PUT", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("setting external ID: %w", err)
	}

	var result Relationship
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}
