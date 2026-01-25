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
	// Entity Relationship operations
	ListEntityRelationships(ctx context.Context, dataGraphID string, page, pageSize int, sourceModelID *string, hasExternalID *bool) (*ListRelationshipsResponse, error)
	GetEntityRelationship(ctx context.Context, dataGraphID, relationshipID string) (*Relationship, error)
	CreateEntityRelationship(ctx context.Context, dataGraphID string, req *CreateRelationshipRequest) (*Relationship, error)
	UpdateEntityRelationship(ctx context.Context, dataGraphID, relationshipID string, req *UpdateRelationshipRequest) (*Relationship, error)
	DeleteEntityRelationship(ctx context.Context, dataGraphID, relationshipID string) error
	SetEntityRelationshipExternalID(ctx context.Context, dataGraphID, relationshipID, externalID string) error

	// Event Relationship operations
	ListEventRelationships(ctx context.Context, dataGraphID string, page, pageSize int, sourceModelID *string, hasExternalID *bool) (*ListRelationshipsResponse, error)
	GetEventRelationship(ctx context.Context, dataGraphID, relationshipID string) (*Relationship, error)
	CreateEventRelationship(ctx context.Context, dataGraphID string, req *CreateRelationshipRequest) (*Relationship, error)
	UpdateEventRelationship(ctx context.Context, dataGraphID, relationshipID string, req *UpdateRelationshipRequest) (*Relationship, error)
	DeleteEventRelationship(ctx context.Context, dataGraphID, relationshipID string) error
	SetEventRelationshipExternalID(ctx context.Context, dataGraphID, relationshipID, externalID string) error
}

// ListEntityRelationships lists entity relationships in a data graph
func (s *rudderDataGraphClient) ListEntityRelationships(ctx context.Context, dataGraphID string, page, pageSize int, sourceModelID *string, hasExternalID *bool) (*ListRelationshipsResponse, error) {
	filters := map[string]interface{}{
		"sourceModelId": sourceModelID,
		"hasExternalId": hasExternalID,
	}
	return s.listRelationships(ctx, dataGraphID, page, pageSize, "entity", filters)
}

// ListEventRelationships lists event relationships in a data graph
func (s *rudderDataGraphClient) ListEventRelationships(ctx context.Context, dataGraphID string, page, pageSize int, sourceModelID *string, hasExternalID *bool) (*ListRelationshipsResponse, error) {
	filters := map[string]interface{}{
		"sourceModelId": sourceModelID,
		"hasExternalId": hasExternalID,
	}
	return s.listRelationships(ctx, dataGraphID, page, pageSize, "event", filters)
}

// GetEntityRelationship retrieves an entity relationship by ID
func (s *rudderDataGraphClient) GetEntityRelationship(ctx context.Context, dataGraphID, relationshipID string) (*Relationship, error) {
	return s.getRelationship(ctx, dataGraphID, relationshipID, "entity")
}

// GetEventRelationship retrieves an event relationship by ID
func (s *rudderDataGraphClient) GetEventRelationship(ctx context.Context, dataGraphID, relationshipID string) (*Relationship, error) {
	return s.getRelationship(ctx, dataGraphID, relationshipID, "event")
}

// CreateEntityRelationship creates a new entity relationship
func (s *rudderDataGraphClient) CreateEntityRelationship(ctx context.Context, dataGraphID string, req *CreateRelationshipRequest) (*Relationship, error) {
	return s.createRelationship(ctx, dataGraphID, req, "entity")
}

// CreateEventRelationship creates a new event relationship
func (s *rudderDataGraphClient) CreateEventRelationship(ctx context.Context, dataGraphID string, req *CreateRelationshipRequest) (*Relationship, error) {
	return s.createRelationship(ctx, dataGraphID, req, "event")
}

// UpdateEntityRelationship updates an existing entity relationship
func (s *rudderDataGraphClient) UpdateEntityRelationship(ctx context.Context, dataGraphID, relationshipID string, req *UpdateRelationshipRequest) (*Relationship, error) {
	return s.updateRelationship(ctx, dataGraphID, relationshipID, req, "entity")
}

// UpdateEventRelationship updates an existing event relationship
func (s *rudderDataGraphClient) UpdateEventRelationship(ctx context.Context, dataGraphID, relationshipID string, req *UpdateRelationshipRequest) (*Relationship, error) {
	return s.updateRelationship(ctx, dataGraphID, relationshipID, req, "event")
}

// DeleteEntityRelationship deletes an entity relationship by ID
func (s *rudderDataGraphClient) DeleteEntityRelationship(ctx context.Context, dataGraphID, relationshipID string) error {
	return s.deleteRelationship(ctx, dataGraphID, relationshipID, "entity")
}

// DeleteEventRelationship deletes an event relationship by ID
func (s *rudderDataGraphClient) DeleteEventRelationship(ctx context.Context, dataGraphID, relationshipID string) error {
	return s.deleteRelationship(ctx, dataGraphID, relationshipID, "event")
}

// SetEntityRelationshipExternalID sets the external ID for an entity relationship
func (s *rudderDataGraphClient) SetEntityRelationshipExternalID(ctx context.Context, dataGraphID, relationshipID, externalID string) error {
	return s.setRelationshipExternalID(ctx, dataGraphID, relationshipID, externalID, "entity")
}

// SetEventRelationshipExternalID sets the external ID for an event relationship
func (s *rudderDataGraphClient) SetEventRelationshipExternalID(ctx context.Context, dataGraphID, relationshipID, externalID string) error {
	return s.setRelationshipExternalID(ctx, dataGraphID, relationshipID, externalID, "event")
}

// Private helper functions

// listRelationships is a common helper for listing entity or event relationships with optional filters
func (s *rudderDataGraphClient) listRelationships(ctx context.Context, dataGraphID string, page, pageSize int, relationshipType string, filters map[string]interface{}) (*ListRelationshipsResponse, error) {
	if dataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s-relationships", dataGraphsBasePath, dataGraphID, relationshipType)

	query := url.Values{}
	if page > 0 {
		query.Add("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		query.Add("pageSize", strconv.Itoa(pageSize))
	}

	// Add filters
	for key, value := range filters {
		switch v := value.(type) {
		case *string:
			if v != nil {
				query.Add(key, *v)
			}
		case *bool:
			if v != nil {
				query.Add(key, strconv.FormatBool(*v))
			}
		}
	}

	if len(query) > 0 {
		path = fmt.Sprintf("%s?%s", path, query.Encode())
	}

	resp, err := s.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing %s relationships: %w", relationshipType, err)
	}

	var result ListRelationshipsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	// Set type on each relationship (not returned by API)
	// DataGraphID and WorkspaceID are populated from API response
	for i := range result.Data {
		result.Data[i].Type = relationshipType
	}

	return &result, nil
}

// getRelationship is a common helper for getting entity or event relationships
func (s *rudderDataGraphClient) getRelationship(ctx context.Context, dataGraphID, relationshipID, relationshipType string) (*Relationship, error) {
	if dataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if relationshipID == "" {
		return nil, fmt.Errorf("relationship ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s-relationships/%s", dataGraphsBasePath, dataGraphID, relationshipType, relationshipID)
	resp, err := s.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting %s relationship: %w", relationshipType, err)
	}

	var result Relationship
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	result.Type = relationshipType
	return &result, nil
}

// createRelationship is a common helper for creating entity or event relationships
func (s *rudderDataGraphClient) createRelationship(ctx context.Context, dataGraphID string, req *CreateRelationshipRequest, relationshipType string) (*Relationship, error) {
	if dataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s-relationships", dataGraphsBasePath, dataGraphID, relationshipType)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	resp, err := s.client.Do(ctx, "POST", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating %s relationship: %w", relationshipType, err)
	}

	var result Relationship
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	result.Type = relationshipType
	return &result, nil
}

// updateRelationship is a common helper for updating entity or event relationships
func (s *rudderDataGraphClient) updateRelationship(ctx context.Context, dataGraphID, relationshipID string, req *UpdateRelationshipRequest, relationshipType string) (*Relationship, error) {
	if dataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if relationshipID == "" {
		return nil, fmt.Errorf("relationship ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s-relationships/%s", dataGraphsBasePath, dataGraphID, relationshipType, relationshipID)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	resp, err := s.client.Do(ctx, "PUT", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("updating %s relationship: %w", relationshipType, err)
	}

	var result Relationship
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	result.Type = relationshipType
	return &result, nil
}

// deleteRelationship is a common helper for deleting entity or event relationships
func (s *rudderDataGraphClient) deleteRelationship(ctx context.Context, dataGraphID, relationshipID, relationshipType string) error {
	if dataGraphID == "" {
		return fmt.Errorf("data graph ID cannot be empty")
	}
	if relationshipID == "" {
		return fmt.Errorf("relationship ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s-relationships/%s", dataGraphsBasePath, dataGraphID, relationshipType, relationshipID)
	_, err := s.client.Do(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("deleting %s relationship: %w", relationshipType, err)
	}

	return nil
}

// setRelationshipExternalID is a common helper for setting external IDs on entity or event relationships
func (s *rudderDataGraphClient) setRelationshipExternalID(ctx context.Context, dataGraphID, relationshipID, externalID, relationshipType string) error {
	if dataGraphID == "" {
		return fmt.Errorf("data graph ID cannot be empty")
	}
	if relationshipID == "" {
		return fmt.Errorf("relationship ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/%s-relationships/%s/external-id", dataGraphsBasePath, dataGraphID, relationshipType, relationshipID)
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
