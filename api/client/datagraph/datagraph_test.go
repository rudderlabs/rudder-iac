package datagraph_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers

func newTestStore(t *testing.T, httpClient client.HTTPClient) datagraph.DataGraphClient {
	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	return datagraph.NewRudderDataGraphClient(c)
}

// Common test timestamps
var (
	testTime1 = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	testTime2 = time.Date(2024, 1, 15, 13, 0, 0, 0, time.UTC)
	testTime3 = time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC)
)

// Helper function to create bool pointers
func boolPtr(b bool) *bool {
	return &b
}

// Happy path tests

func TestCreateDataGraph(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"accountId":"wh-123"}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "dg-123",
			"workspaceId": "ws-456",
			"accountId": "wh-123",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.CreateDataGraph(context.Background(), &datagraph.CreateDataGraphRequest{
		AccountID: "wh-123",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.DataGraph{
		ID:          "dg-123",
		WorkspaceID: "ws-456",
		AccountID:   "wh-123",
		CreatedAt:   &testTime1,
		UpdatedAt:   &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestCreateDataGraphWithExternalID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"accountId":"wh-123","externalId":"ext-123"}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "dg-123",
			"workspaceId": "ws-456",
			"accountId": "wh-123",
			"externalId": "ext-123",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.CreateDataGraph(context.Background(), &datagraph.CreateDataGraphRequest{
		AccountID:  "wh-123",
		ExternalID: "ext-123",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.DataGraph{
		ID:          "dg-123",
		WorkspaceID: "ws-456",
		AccountID:   "wh-123",
		ExternalID:  "ext-123",
		CreatedAt:   &testTime1,
		UpdatedAt:   &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestCreateDataGraphWithExternalID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"Test Graph","accountId":"wh-123","externalId":"ext-123"}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "dg-123",
			"name": "Test Graph",
			"workspaceId": "ws-456",
			"accountId": "wh-123",
			"externalId": "ext-123",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.CreateDataGraph(context.Background(), &datagraph.CreateDataGraphRequest{
		Name:       "Test Graph",
		AccountID:  "wh-123",
		ExternalID: "ext-123",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.DataGraph{
		ID:          "dg-123",
		Name:        "Test Graph",
		WorkspaceID: "ws-456",
		AccountID:   "wh-123",
		ExternalID:  "ext-123",
		CreatedAt:   &testTime1,
		UpdatedAt:   &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestGetDataGraph(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "dg-123",
			"workspaceId": "ws-456",
			"accountId": "wh-123",
			"externalId": "ext-123",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.GetDataGraph(context.Background(), "dg-123")
	require.NoError(t, err)

	assert.Equal(t, &datagraph.DataGraph{
		ID:          "dg-123",
		WorkspaceID: "ws-456",
		AccountID:   "wh-123",
		ExternalID:  "ext-123",
		CreatedAt:   &testTime1,
		UpdatedAt:   &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestDeleteDataGraph(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/v2/data-graphs/dg-123", "")
		},
		ResponseStatus: 204,
		ResponseBody:   "",
	})

	store := newTestStore(t, httpClient)

	err := store.DeleteDataGraph(context.Background(), "dg-123")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestListDataGraphs(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "dg-1",
					"workspaceId": "ws-456",
					"accountId": "wh-123",
					"createdAt": "2024-01-15T12:00:00Z",
					"updatedAt": "2024-01-15T12:00:00Z"
				},
				{
					"id": "dg-2",
					"workspaceId": "ws-456",
					"accountId": "wh-456",
					"createdAt": "2024-01-16T12:00:00Z",
					"updatedAt": "2024-01-16T12:00:00Z"
				}
			],
			"paging": {"total": 2}
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.ListDataGraphs(context.Background(), 0, 0, nil)
	require.NoError(t, err)

	assert.Equal(t, &datagraph.ListDataGraphsResponse{
		Data: []datagraph.DataGraph{
			{
				ID:          "dg-1",
				WorkspaceID: "ws-456",
				AccountID:   "wh-123",
				CreatedAt:   &testTime1,
				UpdatedAt:   &testTime1,
			},
			{
				ID:          "dg-2",
				WorkspaceID: "ws-456",
				AccountID:   "wh-456",
				CreatedAt:   &testTime3,
				UpdatedAt:   &testTime3,
			},
		},
		Paging: client.Paging{Total: 2},
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestListDataGraphsWithPagination(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			query := req.URL.Query()
			return req.Method == "GET" &&
				req.URL.Path == "/v2/data-graphs" &&
				query.Get("page") == "2" &&
				query.Get("pageSize") == "10"
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "dg-11",
					"workspaceId": "ws-456",
					"accountId": "wh-123",
					"createdAt": "2024-01-15T12:00:00Z",
					"updatedAt": "2024-01-15T12:00:00Z"
				},
				{
					"id": "dg-12",
					"workspaceId": "ws-456",
					"accountId": "wh-456",
					"createdAt": "2024-01-16T12:00:00Z",
					"updatedAt": "2024-01-16T12:00:00Z"
				}
			],
			"paging": {"total": 42, "next": "/v2/data-graphs?page=3&pageSize=10"}
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.ListDataGraphs(context.Background(), 2, 10, nil)
	require.NoError(t, err)

	assert.Equal(t, &datagraph.ListDataGraphsResponse{
		Data: []datagraph.DataGraph{
			{
				ID:          "dg-11",
				WorkspaceID: "ws-456",
				AccountID:   "wh-123",
				CreatedAt:   &testTime1,
				UpdatedAt:   &testTime1,
			},
			{
				ID:          "dg-12",
				WorkspaceID: "ws-456",
				AccountID:   "wh-456",
				CreatedAt:   &testTime3,
				UpdatedAt:   &testTime3,
			},
		},
		Paging: client.Paging{Total: 42, Next: "/v2/data-graphs?page=3&pageSize=10"},
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestListDataGraphsWithExternalIDFilter(t *testing.T) {
	tests := []struct {
		name              string
		hasExternalID     *bool
		expectedQueryKey  string
		expectedQueryVal  string
		responseWithExtID bool
	}{
		{
			name:              "filter for graphs with external ID",
			hasExternalID:     boolPtr(true),
			expectedQueryKey:  "hasExternalId",
			expectedQueryVal:  "true",
			responseWithExtID: true,
		},
		{
			name:              "filter for graphs without external ID",
			hasExternalID:     boolPtr(false),
			expectedQueryKey:  "hasExternalId",
			expectedQueryVal:  "false",
			responseWithExtID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
				Validate: func(req *http.Request) bool {
					query := req.URL.Query()
					return req.Method == "GET" &&
						req.URL.Path == "/v2/data-graphs" &&
						query.Get(tt.expectedQueryKey) == tt.expectedQueryVal
				},
				ResponseStatus: 200,
				ResponseBody: func() string {
					if tt.responseWithExtID {
						return `{
							"data": [
								{
									"id": "dg-1",
									"workspaceId": "ws-456",
									"accountId": "wh-123",
									"externalId": "ext-1",
									"createdAt": "2024-01-15T12:00:00Z",
									"updatedAt": "2024-01-15T12:00:00Z"
								}
							],
							"paging": {"total": 1}
						}`
					}
					return `{
						"data": [
							{
								"id": "dg-2",
								"workspaceId": "ws-456",
								"accountId": "wh-456",
								"createdAt": "2024-01-16T12:00:00Z",
								"updatedAt": "2024-01-16T12:00:00Z"
							}
						],
						"paging": {"total": 1}
					}`
				}(),
			})

			store := newTestStore(t, httpClient)

			result, err := store.ListDataGraphs(context.Background(), 0, 0, tt.hasExternalID)
			require.NoError(t, err)

			require.Len(t, result.Data, 1)
			if tt.responseWithExtID {
				assert.Equal(t, "ext-1", result.Data[0].ExternalID)
			} else {
				assert.Empty(t, result.Data[0].ExternalID)
			}

			httpClient.AssertNumberOfCalls()
		})
	}
}

func TestSetExternalID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"externalId":"ext-123"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/external-id", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "dg-123",
			"workspaceId": "ws-456",
			"accountId": "wh-123",
			"externalId": "ext-123",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T13:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.SetExternalID(context.Background(), "dg-123", "ext-123")
	require.NoError(t, err)

	assert.Equal(t, &datagraph.DataGraph{
		ID:          "dg-123",
		WorkspaceID: "ws-456",
		AccountID:   "wh-123",
		ExternalID:  "ext-123",
		CreatedAt:   &testTime1,
		UpdatedAt:   &testTime2,
	}, result)

	httpClient.AssertNumberOfCalls()
}

// Empty ID validation tests (table-driven)

func TestEmptyIDValidation(t *testing.T) {
	// Mock HTTP client is not called since validation happens before HTTP request
	httpClient := testutils.NewMockHTTPClient(t)
	store := newTestStore(t, httpClient)

	tests := []struct {
		name      string
		operation func() error
	}{
		{
			name: "GetDataGraph",
			operation: func() error {
				_, err := store.GetDataGraph(context.Background(), "")
				return err
			},
		},
		{
			name: "DeleteDataGraph",
			operation: func() error {
				return store.DeleteDataGraph(context.Background(), "")
			},
		},
		{
			name: "SetExternalID",
			operation: func() error {
				_, err := store.SetExternalID(context.Background(), "", "ext-123")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "data graph ID cannot be empty")
		})
	}
}

// API error tests (table-driven)

func TestAPIErrors(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		responseStatus int
		responseBody   string
		operation      func(store datagraph.DataGraphClient) error
		expectedError  string
	}{
		{
			name:           "CreateDataGraph",
			method:         "POST",
			path:           "/v2/data-graphs",
			responseStatus: 400,
			responseBody:   `{"error":"Bad Request"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.CreateDataGraph(context.Background(), &datagraph.CreateDataGraphRequest{
					AccountID: "wh-123",
				})
				return err
			},
			expectedError: "creating data graph",
		},
		{
			name:           "GetDataGraph",
			method:         "GET",
			path:           "/v2/data-graphs/dg-123",
			responseStatus: 404,
			responseBody:   `{"error":"Not Found"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.GetDataGraph(context.Background(), "dg-123")
				return err
			},
			expectedError: "getting data graph",
		},
		{
			name:           "DeleteDataGraph",
			method:         "DELETE",
			path:           "/v2/data-graphs/dg-123",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				return store.DeleteDataGraph(context.Background(), "dg-123")
			},
			expectedError: "deleting data graph",
		},
		{
			name:           "ListDataGraphs",
			method:         "GET",
			path:           "/v2/data-graphs",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.ListDataGraphs(context.Background(), 0, 0, nil)
				return err
			},
			expectedError: "listing data graphs",
		},
		{
			name:           "SetExternalID",
			method:         "PUT",
			path:           "/v2/data-graphs/dg-123/external-id",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.SetExternalID(context.Background(), "dg-123", "ext-123")
				return err
			},
			expectedError: "setting external ID",
		},
		{
			name:           "CreateDataGraphConflict",
			method:         "POST",
			path:           "/v2/data-graphs",
			responseStatus: 409,
			responseBody:   `{"error":"Data graph with the same warehouse id already exists"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.CreateDataGraph(context.Background(), &datagraph.CreateDataGraphRequest{
					AccountID: "wh-123",
				})
				return err
			},
			expectedError: "creating data graph",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
				Validate: func(req *http.Request) bool {
					return req.Method == tt.method && req.URL.Path == tt.path
				},
				ResponseStatus: tt.responseStatus,
				ResponseBody:   tt.responseBody,
			})

			store := newTestStore(t, httpClient)
			err := tt.operation(store)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
			httpClient.AssertNumberOfCalls()
		})
	}
}

// Malformed response tests (table-driven)

func TestMalformedResponses(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		path          string
		operation     func(store datagraph.DataGraphClient) error
		expectedError string
	}{
		{
			name:   "CreateDataGraph",
			method: "POST",
			path:   "/v2/data-graphs",
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.CreateDataGraph(context.Background(), &datagraph.CreateDataGraphRequest{
					AccountID: "wh-123",
				})
				return err
			},
			expectedError: "unmarshalling response",
		},
		{
			name:   "GetDataGraph",
			method: "GET",
			path:   "/v2/data-graphs/dg-123",
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.GetDataGraph(context.Background(), "dg-123")
				return err
			},
			expectedError: "unmarshalling response",
		},
		{
			name:   "ListDataGraphs",
			method: "GET",
			path:   "/v2/data-graphs",
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.ListDataGraphs(context.Background(), 0, 0, nil)
				return err
			},
			expectedError: "unmarshalling response",
		},
		{
			name:   "SetExternalID",
			method: "PUT",
			path:   "/v2/data-graphs/dg-123/external-id",
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.SetExternalID(context.Background(), "dg-123", "ext-123")
				return err
			},
			expectedError: "unmarshalling response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
				Validate: func(req *http.Request) bool {
					return req.Method == tt.method && req.URL.Path == tt.path
				},
				ResponseStatus: 200,
				ResponseBody:   `{malformed_json`,
			})

			store := newTestStore(t, httpClient)
			err := tt.operation(store)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
			httpClient.AssertNumberOfCalls()
		})
	}
}
