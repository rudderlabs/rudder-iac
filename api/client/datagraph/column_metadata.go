package datagraph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// ColumnMetadataEntry is a single column-metadata write sent in a batch upsert.
// DisplayName / Description are pointers so a nil value serializes as JSON `null`
// (an explicit clear, per the declarative apply contract) rather than being
// omitted — hence no `omitempty`. A non-nil value sets the field. At least one
// of the two is non-nil; an empty string is invalid (use null to clear).
type ColumnMetadataEntry struct {
	Name        string  `json:"name"`
	DisplayName *string `json:"displayName"`
	Description *string `json:"description"`
}

// ColumnMetadataRow is a single column-metadata row returned by the API.
// displayName / description are omitted by the server when unset, so they
// unmarshal to the empty string.
type ColumnMetadataRow struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
}

// ColumnMetadataListResponse is the response from both List and BatchUpsert.
type ColumnMetadataListResponse struct {
	Columns []ColumnMetadataRow `json:"columns"`
}

// BatchUpsertColumnMetadataRequest is the request body for a combined batch
// upsert + remove. Columns carries (name, displayName) pairs to set or update,
// DeleteColumns carries names whose rows should be removed in the same call.
// Both fields use omitempty so empty slices are not serialized; the server
// requires at least one of them to be non-empty. The caller is responsible
// for ensuring no name appears in both arrays.
type BatchUpsertColumnMetadataRequest struct {
	Columns       []ColumnMetadataEntry `json:"columns,omitempty"`
	DeleteColumns []string              `json:"deleteColumns,omitempty"`
}

// ColumnMetadataStore is the interface for per-model column metadata operations.
type ColumnMetadataStore interface {
	// ListColumnMetadata returns the persisted column metadata rows for a model.
	ListColumnMetadata(ctx context.Context, dataGraphID, modelID string) (*ColumnMetadataListResponse, error)

	// BatchUpsertColumnMetadata upserts the supplied column metadata entries
	// for a model in a single call and returns the resulting persisted rows.
	BatchUpsertColumnMetadata(ctx context.Context, dataGraphID, modelID string, req BatchUpsertColumnMetadataRequest) (*ColumnMetadataListResponse, error)
}

// ListColumnMetadata lists column metadata for a model.
func (s *rudderDataGraphClient) ListColumnMetadata(ctx context.Context, dataGraphID, modelID string) (*ColumnMetadataListResponse, error) {
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

// BatchUpsertColumnMetadata upserts column metadata for a model in one batch.
func (s *rudderDataGraphClient) BatchUpsertColumnMetadata(ctx context.Context, dataGraphID, modelID string, req BatchUpsertColumnMetadataRequest) (*ColumnMetadataListResponse, error) {
	if dataGraphID == "" {
		return nil, fmt.Errorf("data graph ID cannot be empty")
	}
	if modelID == "" {
		return nil, fmt.Errorf("model ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/models/%s/column-metadata", dataGraphsBasePath, dataGraphID, modelID)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	resp, err := s.client.Do(ctx, "PATCH", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("upserting column metadata: %w", err)
	}

	var result ColumnMetadataListResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}
