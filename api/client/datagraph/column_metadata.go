package datagraph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// ColumnMetadata is display name metadata for a warehouse column on a model.
type ColumnMetadata struct {
	ColumnName  string `json:"columnName"`
	DisplayName string `json:"displayName"`
	UpdatedAt   string `json:"updatedAt"`
}

// ColumnMetadataListResponse is the response for listing column metadata.
type ColumnMetadataListResponse struct {
	ColumnMetadata []ColumnMetadata `json:"columnMetadata"`
}

// ColumnMetadataStore is the interface for column metadata operations.
type ColumnMetadataStore interface {
	ListColumnMetadata(ctx context.Context, dataGraphID, modelID string) (*ColumnMetadataListResponse, error)
	UpsertColumnMetadata(ctx context.Context, dataGraphID, modelID, columnName, displayName string) (*ColumnMetadata, error)
	DeleteColumnMetadata(ctx context.Context, dataGraphID, modelID, columnName string) error
}

// ListColumnMetadata lists display name aliases for columns on a model.
func (s *rudderDataGraphClient) ListColumnMetadata(
	ctx context.Context,
	dataGraphID, modelID string,
) (*ColumnMetadataListResponse, error) {
	if dataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if modelID == "" {
		return nil, fmt.Errorf("model ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/models/%s/column-metadata", dataGraphsBasePath, dataGraphID, modelID)
	resp, err := s.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing column metadata: %w", err)
	}

	var result ColumnMetadataListResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// UpsertColumnMetadata sets or updates the display name alias for a column.
func (s *rudderDataGraphClient) UpsertColumnMetadata(
	ctx context.Context,
	dataGraphID, modelID, columnName, displayName string,
) (*ColumnMetadata, error) {
	if dataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if modelID == "" {
		return nil, fmt.Errorf("model ID cannot be empty")
	}
	if columnName == "" {
		return nil, fmt.Errorf("column name cannot be empty")
	}

	path := fmt.Sprintf(
		"%s/%s/models/%s/column-metadata/%s",
		dataGraphsBasePath,
		dataGraphID,
		modelID,
		url.PathEscape(columnName),
	)

	body, err := json.Marshal(map[string]string{"displayName": displayName})
	if err != nil {
		return nil, fmt.Errorf("marshalling request body: %w", err)
	}

	resp, err := s.client.Do(ctx, "PATCH", path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("upserting column metadata: %w", err)
	}

	var result ColumnMetadata
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// DeleteColumnMetadata removes the display name alias for a column.
func (s *rudderDataGraphClient) DeleteColumnMetadata(
	ctx context.Context,
	dataGraphID, modelID, columnName string,
) error {
	if dataGraphID == "" {
		return fmt.Errorf("data graph ID cannot be empty")
	}
	if modelID == "" {
		return fmt.Errorf("model ID cannot be empty")
	}
	if columnName == "" {
		return fmt.Errorf("column name cannot be empty")
	}

	path := fmt.Sprintf(
		"%s/%s/models/%s/column-metadata/%s",
		dataGraphsBasePath,
		dataGraphID,
		modelID,
		url.PathEscape(columnName),
	)

	_, err := s.client.Do(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("deleting column metadata: %w", err)
	}

	return nil
}
