package datagraph_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListColumnMetadata(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			if req.Header.Get("Authorization") != "Bearer test-token" {
				return false
			}
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/models/em-456/column-metadata", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"columns": [
				{"name": "email", "displayName": "Email", "updatedAt": "2024-01-15T12:00:00Z"},
				{"name": "user_id", "displayName": "User ID", "updatedAt": "2024-01-15T13:00:00Z"}
			]
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.ListColumnMetadata(context.Background(), "dg-123", "em-456")
	require.NoError(t, err)

	assert.Equal(t, &datagraph.ColumnMetadataListResponse{
		Columns: []datagraph.ColumnMetadataRow{
			{Name: "email", DisplayName: "Email", UpdatedAt: "2024-01-15T12:00:00Z"},
			{Name: "user_id", DisplayName: "User ID", UpdatedAt: "2024-01-15T13:00:00Z"},
		},
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestListColumnMetadata_EmptyDataGraphID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t)
	store := newTestStore(t, httpClient)

	_, err := store.ListColumnMetadata(context.Background(), "", "em-456")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data graph ID cannot be empty")
}

func TestListColumnMetadata_EmptyModelID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t)
	store := newTestStore(t, httpClient)

	_, err := store.ListColumnMetadata(context.Background(), "dg-123", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model ID cannot be empty")
}

func TestListColumnMetadata_NotFound(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return req.Method == "GET" && req.URL.Path == "/v2/data-graphs/dg-123/models/em-456/column-metadata"
		},
		ResponseStatus: 404,
		ResponseBody:   `{"error":"Not Found"}`,
	})

	store := newTestStore(t, httpClient)

	_, err := store.ListColumnMetadata(context.Background(), "dg-123", "em-456")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing column metadata")

	httpClient.AssertNumberOfCalls()
}

func TestListColumnMetadata_Unauthorized(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return req.Method == "GET" && req.URL.Path == "/v2/data-graphs/dg-123/models/em-456/column-metadata"
		},
		ResponseStatus: 401,
		ResponseBody:   `{"error":"Unauthorized"}`,
	})

	store := newTestStore(t, httpClient)

	_, err := store.ListColumnMetadata(context.Background(), "dg-123", "em-456")
	require.Error(t, err)

	var apiErr *client.APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, 401, apiErr.HTTPStatusCode)

	httpClient.AssertNumberOfCalls()
}

func TestBatchUpsertColumnMetadata(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			if req.Header.Get("Authorization") != "Bearer test-token" {
				return false
			}
			expected := `{"columns":[{"name":"email","displayName":"Email"},{"name":"user_id","displayName":"User ID"}]}`
			return testutils.ValidateRequest(t, req, "PATCH", "https://api.rudderstack.com/v2/data-graphs/dg-123/models/em-456/column-metadata", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"columns": [
				{"name": "email", "displayName": "Email", "updatedAt": "2024-01-15T12:00:00Z"},
				{"name": "user_id", "displayName": "User ID", "updatedAt": "2024-01-15T12:00:00Z"}
			]
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.BatchUpsertColumnMetadata(
		context.Background(),
		"dg-123",
		"em-456",
		datagraph.BatchUpsertColumnMetadataRequest{
			Columns: []datagraph.ColumnMetadataEntry{
				{Name: "email", DisplayName: "Email"},
				{Name: "user_id", DisplayName: "User ID"},
			},
		},
	)
	require.NoError(t, err)

	assert.Equal(t, &datagraph.ColumnMetadataListResponse{
		Columns: []datagraph.ColumnMetadataRow{
			{Name: "email", DisplayName: "Email", UpdatedAt: "2024-01-15T12:00:00Z"},
			{Name: "user_id", DisplayName: "User ID", UpdatedAt: "2024-01-15T12:00:00Z"},
		},
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestBatchUpsertColumnMetadata_BothColumnsAndDeleteColumns(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			if req.Header.Get("Authorization") != "Bearer test-token" {
				return false
			}
			expected := `{
				"columns": [{"name":"user_id","displayName":"User ID"}],
				"deleteColumns": ["email","legacy_field"]
			}`
			return testutils.ValidateRequest(t, req, "PATCH", "https://api.rudderstack.com/v2/data-graphs/dg-123/models/em-456/column-metadata", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"columns": [
				{"name": "user_id", "displayName": "User ID", "updatedAt": "2024-01-15T12:00:00Z"}
			]
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.BatchUpsertColumnMetadata(
		context.Background(),
		"dg-123",
		"em-456",
		datagraph.BatchUpsertColumnMetadataRequest{
			Columns:       []datagraph.ColumnMetadataEntry{{Name: "user_id", DisplayName: "User ID"}},
			DeleteColumns: []string{"email", "legacy_field"},
		},
	)
	require.NoError(t, err)

	assert.Equal(t, &datagraph.ColumnMetadataListResponse{
		Columns: []datagraph.ColumnMetadataRow{
			{Name: "user_id", DisplayName: "User ID", UpdatedAt: "2024-01-15T12:00:00Z"},
		},
	}, result)

	httpClient.AssertNumberOfCalls()
}

// TestBatchUpsertColumnMetadata_OnlyDeleteColumns covers the all-removals
// shape: yaml drops every previously-managed column, so the call carries only
// deleteColumns. Verifies omitempty on Columns drops the key from the wire
// payload entirely, matching the server's "at least one non-empty" contract.
func TestBatchUpsertColumnMetadata_OnlyDeleteColumns(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			if req.Header.Get("Authorization") != "Bearer test-token" {
				return false
			}
			expected := `{"deleteColumns": ["email", "user_id"]}`
			return testutils.ValidateRequest(t, req, "PATCH", "https://api.rudderstack.com/v2/data-graphs/dg-123/models/em-456/column-metadata", expected)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"columns": []}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.BatchUpsertColumnMetadata(
		context.Background(),
		"dg-123",
		"em-456",
		datagraph.BatchUpsertColumnMetadataRequest{
			DeleteColumns: []string{"email", "user_id"},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, &datagraph.ColumnMetadataListResponse{Columns: []datagraph.ColumnMetadataRow{}}, result)

	httpClient.AssertNumberOfCalls()
}

func TestBatchUpsertColumnMetadata_EmptyDataGraphID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t)
	store := newTestStore(t, httpClient)

	_, err := store.BatchUpsertColumnMetadata(context.Background(), "", "em-456", datagraph.BatchUpsertColumnMetadataRequest{
		Columns: []datagraph.ColumnMetadataEntry{{Name: "email", DisplayName: "Email"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data graph ID cannot be empty")
}

func TestBatchUpsertColumnMetadata_EmptyModelID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t)
	store := newTestStore(t, httpClient)

	_, err := store.BatchUpsertColumnMetadata(context.Background(), "dg-123", "", datagraph.BatchUpsertColumnMetadataRequest{
		Columns: []datagraph.ColumnMetadataEntry{{Name: "email", DisplayName: "Email"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model ID cannot be empty")
}

func TestBatchUpsertColumnMetadata_BadRequest(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return req.Method == "PATCH" && req.URL.Path == "/v2/data-graphs/dg-123/models/em-456/column-metadata"
		},
		ResponseStatus: 400,
		ResponseBody:   `{"message":"request validation failed","error":"request validation failed","details":{"columns.0.displayName":"Too small: expected string to have >=1 characters"}}`,
	})

	store := newTestStore(t, httpClient)

	_, err := store.BatchUpsertColumnMetadata(context.Background(), "dg-123", "em-456", datagraph.BatchUpsertColumnMetadataRequest{
		Columns: []datagraph.ColumnMetadataEntry{{Name: "email", DisplayName: ""}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upserting column metadata")

	httpClient.AssertNumberOfCalls()
}

func TestBatchUpsertColumnMetadata_NotFound(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return req.Method == "PATCH" && req.URL.Path == "/v2/data-graphs/dg-123/models/em-456/column-metadata"
		},
		ResponseStatus: 404,
		ResponseBody:   `{"error":"Not Found"}`,
	})

	store := newTestStore(t, httpClient)

	_, err := store.BatchUpsertColumnMetadata(context.Background(), "dg-123", "em-456", datagraph.BatchUpsertColumnMetadataRequest{
		Columns: []datagraph.ColumnMetadataEntry{{Name: "email", DisplayName: "Email"}},
	})
	require.Error(t, err)

	var apiErr *client.APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, 404, apiErr.HTTPStatusCode)

	httpClient.AssertNumberOfCalls()
}

func TestBatchUpsertColumnMetadata_ServerError(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return req.Method == "PATCH" && req.URL.Path == "/v2/data-graphs/dg-123/models/em-456/column-metadata"
		},
		ResponseStatus: 500,
		ResponseBody:   `{"error":"Internal Server Error"}`,
	})

	store := newTestStore(t, httpClient)

	_, err := store.BatchUpsertColumnMetadata(context.Background(), "dg-123", "em-456", datagraph.BatchUpsertColumnMetadataRequest{
		Columns: []datagraph.ColumnMetadataEntry{{Name: "email", DisplayName: "Email"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upserting column metadata")

	httpClient.AssertNumberOfCalls()
}

func TestBatchUpsertColumnMetadata_ValidationError422(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return req.Method == "PATCH" && req.URL.Path == "/v2/data-graphs/dg-123/models/em-456/column-metadata"
		},
		ResponseStatus: 422,
		ResponseBody: `{
			"message": "Failed to upsert column metadata",
			"error": "Failed to upsert column metadata",
			"code": "column-metadata-validation-failed",
			"details": {
				"code": "column-metadata-validation-failed",
				"errors": [
					{"name": "email", "reason": "duplicate-display-name", "conflictWith": "user_id"},
					{"name": "phone", "reason": "column-not-in-schema"}
				]
			}
		}`,
	})

	store := newTestStore(t, httpClient)

	_, err := store.BatchUpsertColumnMetadata(context.Background(), "dg-123", "em-456", datagraph.BatchUpsertColumnMetadataRequest{
		Columns: []datagraph.ColumnMetadataEntry{
			{Name: "email", DisplayName: "User ID"},
			{Name: "phone", DisplayName: "Phone"},
		},
	})
	require.Error(t, err)

	validationErr, ok := datagraph.AsColumnMetadataValidationError(err)
	require.True(t, ok, "expected error to decode as ColumnMetadataValidationError, got %T: %v", err, err)
	assert.Equal(t, "column-metadata-validation-failed", validationErr.Code)
	assert.Equal(t, []datagraph.ColumnMetadataValidationErrorEntry{
		{Name: "email", Reason: "duplicate-display-name", ConflictWith: "user_id"},
		{Name: "phone", Reason: "column-not-in-schema"},
	}, validationErr.Errors)

	// The underlying APIError should remain accessible.
	var apiErr *client.APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, 422, apiErr.HTTPStatusCode)

	httpClient.AssertNumberOfCalls()
}

func TestAsColumnMetadataValidationError_OtherErrors(t *testing.T) {
	// Plain error: not a column metadata validation error.
	_, ok := datagraph.AsColumnMetadataValidationError(errors.New("boom"))
	assert.False(t, ok)

	// APIError without the expected code: not a column metadata validation error.
	apiErr := &client.APIError{HTTPStatusCode: 422, ErrorCode: "something-else"}
	_, ok = datagraph.AsColumnMetadataValidationError(apiErr)
	assert.False(t, ok)
}
