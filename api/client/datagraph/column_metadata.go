package datagraph

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// columnMetadataValidationCode is the API error code returned for 422 responses
// from the column-metadata endpoints. It is the discriminator used to decide
// whether the structured details payload is safe to decode into a
// ColumnMetadataValidationError.
const columnMetadataValidationCode = "column-metadata-validation-failed"

// ColumnMetadataEntry is a single (name, displayName) pair sent in a batch upsert.
type ColumnMetadataEntry struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// ColumnMetadataRow is a single row returned by the API: the persisted
// (name, displayName) plus the server-assigned updatedAt timestamp.
type ColumnMetadataRow struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	UpdatedAt   string `json:"updatedAt"`
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

// ColumnMetadataValidationErrorEntry describes a single per-column validation
// failure returned in the 422 details payload.
type ColumnMetadataValidationErrorEntry struct {
	Name         string `json:"name"`
	Reason       string `json:"reason"`
	ConflictWith string `json:"conflictWith,omitempty"`
}

// ColumnMetadataValidationError is the typed view of a 422 response from the
// column-metadata endpoints. It wraps the underlying *client.APIError so
// callers can still inspect HTTP status / message via errors.As.
type ColumnMetadataValidationError struct {
	Code   string                               `json:"code"`
	Errors []ColumnMetadataValidationErrorEntry `json:"errors"`

	apiErr *client.APIError
}

func (e *ColumnMetadataValidationError) Error() string {
	return e.apiErr.Error()
}

func (e *ColumnMetadataValidationError) Unwrap() error {
	return e.apiErr
}

// AsColumnMetadataValidationError reports whether err carries a structured
// column-metadata validation failure (HTTP 422 with the documented details
// shape). When true, the returned value gives typed access to the per-column
// reasons; the underlying *client.APIError remains reachable via errors.As.
func AsColumnMetadataValidationError(err error) (*ColumnMetadataValidationError, bool) {
	var validationErr *ColumnMetadataValidationError
	if errors.As(err, &validationErr) {
		return validationErr, true
	}

	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		return nil, false
	}
	if apiErr.ErrorCode != columnMetadataValidationCode {
		return nil, false
	}

	parsed := &ColumnMetadataValidationError{apiErr: apiErr}
	if len(apiErr.Details) > 0 {
		// Best-effort decode: even if details is missing fields the error
		// is still a validation error — callers get the code and an empty
		// Errors slice rather than losing the typing.
		_ = json.Unmarshal(apiErr.Details, parsed)
	}
	if parsed.Code == "" {
		parsed.Code = apiErr.ErrorCode
	}
	return parsed, true
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
		return nil, fmt.Errorf("upserting column metadata: %w", wrapColumnMetadataError(err))
	}

	var result ColumnMetadataListResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}

	return &result, nil
}

// wrapColumnMetadataError promotes a generic *client.APIError carrying the
// column-metadata-validation-failed code into a typed
// ColumnMetadataValidationError so callers can render per-column reasons
// without re-parsing details. Non-matching errors pass through unchanged.
func wrapColumnMetadataError(err error) error {
	if validationErr, ok := AsColumnMetadataValidationError(err); ok {
		return validationErr
	}
	return err
}
