package datagraph_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListColumnMetadata(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(
				t,
				req,
				"GET",
				"https://api.rudderstack.com/v2/data-graphs/dg-123/models/m-456/column-metadata",
				"",
			)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"columnMetadata": [
				{
					"columnName": "ltv",
					"displayName": "Lifetime Value",
					"updatedAt": "2024-01-15T12:00:00Z"
				}
			]
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.ListColumnMetadata(context.Background(), "dg-123", "m-456")
	require.NoError(t, err)
	require.Len(t, result.ColumnMetadata, 1)
	assert.Equal(t, "ltv", result.ColumnMetadata[0].ColumnName)
	assert.Equal(t, "Lifetime Value", result.ColumnMetadata[0].DisplayName)

	httpClient.AssertNumberOfCalls()
}

func TestUpsertColumnMetadata(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"displayName":"Lifetime Value"}`
			return testutils.ValidateRequest(
				t,
				req,
				"PATCH",
				"https://api.rudderstack.com/v2/data-graphs/dg-123/models/m-456/column-metadata/ltv",
				expected,
			)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"columnName": "ltv",
			"displayName": "Lifetime Value",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.UpsertColumnMetadata(
		context.Background(),
		"dg-123",
		"m-456",
		"ltv",
		"Lifetime Value",
	)
	require.NoError(t, err)
	assert.Equal(t, "ltv", result.ColumnName)
	assert.Equal(t, "Lifetime Value", result.DisplayName)

	httpClient.AssertNumberOfCalls()
}

func TestUpsertColumnMetadata_PathEscape(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(
				t,
				req,
				"PATCH",
				"https://api.rudderstack.com/v2/data-graphs/dg-123/models/m-456/column-metadata/col%2Fname",
				`{"displayName":"Alias"}`,
			)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"columnName": "col/name",
			"displayName": "Alias",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	_, err := store.UpsertColumnMetadata(context.Background(), "dg-123", "m-456", "col/name", "Alias")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestDeleteColumnMetadata(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(
				t,
				req,
				"DELETE",
				"https://api.rudderstack.com/v2/data-graphs/dg-123/models/m-456/column-metadata/ltv",
				"",
			)
		},
		ResponseStatus: 204,
	})

	store := newTestStore(t, httpClient)

	err := store.DeleteColumnMetadata(context.Background(), "dg-123", "m-456", "ltv")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestListColumnMetadata_Validation(t *testing.T) {
	store := newTestStore(t, testutils.NewMockHTTPClient(t))

	_, err := store.ListColumnMetadata(context.Background(), "", "m-456")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data graph ID")

	_, err = store.ListColumnMetadata(context.Background(), "dg-123", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model ID")
}
