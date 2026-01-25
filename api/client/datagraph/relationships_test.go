package datagraph_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Entity Relationship Tests

func TestCreateEntityRelationship(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"User Orders","cardinality":"one-to-many","sourceModelId":"em-123","targetModelId":"em-456","sourceJoinKey":"id","targetJoinKey":"user_id"}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs/dg-123/entity-relationships", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "er-789",
			"name": "User Orders",
			"cardinality": "one-to-many",
			"sourceModelId": "em-123",
			"targetModelId": "em-456",
			"sourceJoinKey": "id",
			"targetJoinKey": "user_id",
			"dataGraphId": "dg-123",
			"workspaceId": "ws-456",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.CreateEntityRelationship(context.Background(), "dg-123", &datagraph.CreateRelationshipRequest{
		Name:         "User Orders",
		Cardinality:  "one-to-many",
		SourceModelID:  "em-123",
		TargetModelID:    "em-456",
		SourceJoinKey: "id",
		TargetJoinKey:   "user_id",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Relationship{
		ID:           "er-789",
		Name:         "User Orders",
		Type:         "entity",
		Cardinality:  "one-to-many",
		SourceModelID:  "em-123",
		TargetModelID:    "em-456",
		SourceJoinKey: "id",
		TargetJoinKey:   "user_id",
		DataGraphID:  "dg-123",
		WorkspaceID:  "ws-456",
		CreatedAt:    &testTime1,
		UpdatedAt:    &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestCreateEntityRelationshipWithExternalID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"User Orders","cardinality":"one-to-many","sourceModelId":"em-123","targetModelId":"em-456","sourceJoinKey":"id","targetJoinKey":"user_id","externalId":"user-orders-rel"}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs/dg-123/entity-relationships", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "er-789",
			"name": "User Orders",
			"cardinality": "one-to-many",
			"sourceModelId": "em-123",
			"targetModelId": "em-456",
			"sourceJoinKey": "id",
			"targetJoinKey": "user_id",
			"dataGraphId": "dg-123",
			"workspaceId": "ws-456",
			"externalId": "user-orders-rel",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.CreateEntityRelationship(context.Background(), "dg-123", &datagraph.CreateRelationshipRequest{
		Name:         "User Orders",
		Cardinality:  "one-to-many",
		SourceModelID:  "em-123",
		TargetModelID:    "em-456",
		SourceJoinKey: "id",
		TargetJoinKey:   "user_id",
		ExternalID:   "user-orders-rel",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Relationship{
		ID:           "er-789",
		Name:         "User Orders",
		Type:         "entity",
		Cardinality:  "one-to-many",
		SourceModelID:  "em-123",
		TargetModelID:    "em-456",
		SourceJoinKey: "id",
		TargetJoinKey:   "user_id",
		DataGraphID:  "dg-123",
		WorkspaceID:  "ws-456",
		ExternalID:   "user-orders-rel",
		CreatedAt:    &testTime1,
		UpdatedAt:    &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestGetEntityRelationship(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/entity-relationships/er-789", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "er-789",
			"name": "User Orders",
			"cardinality": "one-to-many",
			"sourceModelId": "em-123",
			"targetModelId": "em-456",
			"sourceJoinKey": "id",
			"targetJoinKey": "user_id",
			"dataGraphId": "dg-123",
			"workspaceId": "ws-456",
			"externalId": "user-orders-rel",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.GetEntityRelationship(context.Background(), "dg-123", "er-789")
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Relationship{
		ID:           "er-789",
		Name:         "User Orders",
		Type:         "entity",
		Cardinality:  "one-to-many",
		SourceModelID:  "em-123",
		TargetModelID:    "em-456",
		SourceJoinKey: "id",
		TargetJoinKey:   "user_id",
		DataGraphID:  "dg-123",
		WorkspaceID:  "ws-456",
		ExternalID:   "user-orders-rel",
		CreatedAt:    &testTime1,
		UpdatedAt:    &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestUpdateEntityRelationship(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"Updated User Orders","cardinality":"one-to-one","sourceModelId":"em-123","targetModelId":"em-789","sourceJoinKey":"order_id","targetJoinKey":"id"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/entity-relationships/er-789", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "er-789",
			"name": "Updated User Orders",
			"cardinality": "one-to-one",
			"sourceModelId": "em-123",
			"targetModelId": "em-789",
			"sourceJoinKey": "order_id",
			"targetJoinKey": "id",
			"dataGraphId": "dg-123",
			"workspaceId": "ws-456",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T13:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.UpdateEntityRelationship(context.Background(), "dg-123", "er-789", &datagraph.UpdateRelationshipRequest{
		Name:         "Updated User Orders",
		Cardinality:  "one-to-one",
		SourceModelID:  "em-123",
		TargetModelID:    "em-789",
		SourceJoinKey: "order_id",
		TargetJoinKey:   "id",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Relationship{
		ID:           "er-789",
		Name:         "Updated User Orders",
		Type:         "entity",
		Cardinality:  "one-to-one",
		SourceModelID:  "em-123",
		TargetModelID:    "em-789",
		SourceJoinKey: "order_id",
		TargetJoinKey:   "id",
		DataGraphID:  "dg-123",
		WorkspaceID:  "ws-456",
		CreatedAt:    &testTime1,
		UpdatedAt:    &testTime2,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestDeleteEntityRelationship(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/v2/data-graphs/dg-123/entity-relationships/er-789", "")
		},
		ResponseStatus: 204,
		ResponseBody:   "",
	})

	store := newTestStore(t, httpClient)

	err := store.DeleteEntityRelationship(context.Background(), "dg-123", "er-789")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestListEntityRelationships(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/entity-relationships", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "er-1",
					"name": "User Orders",
					"cardinality": "one-to-many",
					"sourceModelId": "em-123",
					"targetModelId": "em-456",
					"sourceJoinKey": "id",
					"targetJoinKey": "user_id",
					"dataGraphId": "dg-123",
					"workspaceId": "ws-456",
					"createdAt": "2024-01-15T12:00:00Z",
					"updatedAt": "2024-01-15T12:00:00Z"
				},
				{
					"id": "er-2",
					"name": "Order Product",
					"cardinality": "many-to-one",
					"sourceModelId": "em-456",
					"targetModelId": "em-789",
					"sourceJoinKey": "product_id",
					"targetJoinKey": "id",
					"dataGraphId": "dg-123",
					"workspaceId": "ws-456",
					"createdAt": "2024-01-16T12:00:00Z",
					"updatedAt": "2024-01-16T12:00:00Z"
				}
			],
			"paging": {"total": 2}
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.ListEntityRelationships(context.Background(), "dg-123", 0, 0, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, &datagraph.ListRelationshipsResponse{
		Data: []datagraph.Relationship{
			{
				ID:           "er-1",
				Name:         "User Orders",
				Type:         "entity",
				Cardinality:  "one-to-many",
				SourceModelID:  "em-123",
				TargetModelID:    "em-456",
				SourceJoinKey: "id",
				TargetJoinKey:   "user_id",
				DataGraphID:  "dg-123",
				WorkspaceID:  "ws-456",
				CreatedAt:    &testTime1,
				UpdatedAt:    &testTime1,
			},
			{
				ID:           "er-2",
				Name:         "Order Product",
				Type:         "entity",
				Cardinality:  "many-to-one",
				SourceModelID:  "em-456",
				TargetModelID:    "em-789",
				SourceJoinKey: "product_id",
				TargetJoinKey:   "id",
				DataGraphID:  "dg-123",
				WorkspaceID:  "ws-456",
				CreatedAt:    &testTime3,
				UpdatedAt:    &testTime3,
			},
		},
		Paging: client.Paging{Total: 2},
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestSetEntityRelationshipExternalID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"externalId":"user-orders-rel"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/entity-relationships/er-789/external-id", expected)
		},
		ResponseStatus: 204,
		ResponseBody:   "",
	})

	store := newTestStore(t, httpClient)

	err := store.SetEntityRelationshipExternalID(context.Background(), "dg-123", "er-789", "user-orders-rel")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

// Event Relationship Tests

func TestCreateEventRelationship(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"PageView User","sourceModelId":"em-123","targetModelId":"evm-456","sourceJoinKey":"user_id","targetJoinKey":"id"}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs/dg-123/event-relationships", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "evr-789",
			"name": "PageView User",
			"sourceModelId": "em-123",
			"targetModelId": "evm-456",
			"sourceJoinKey": "user_id",
			"targetJoinKey": "id",
			"dataGraphId": "dg-123",
			"workspaceId": "ws-456",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.CreateEventRelationship(context.Background(), "dg-123", &datagraph.CreateRelationshipRequest{
		Name:         "PageView User",
		SourceModelID:  "em-123",
		TargetModelID:    "evm-456",
		SourceJoinKey: "user_id",
		TargetJoinKey:   "id",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Relationship{
		ID:           "evr-789",
		Name:         "PageView User",
		Type:         "event",
		SourceModelID:  "em-123",
		TargetModelID:    "evm-456",
		SourceJoinKey: "user_id",
		TargetJoinKey:   "id",
		DataGraphID:  "dg-123",
		WorkspaceID:  "ws-456",
		CreatedAt:    &testTime1,
		UpdatedAt:    &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestGetEventRelationship(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/event-relationships/evr-789", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "evr-789",
			"name": "PageView User",
			"sourceModelId": "em-123",
			"targetModelId": "evm-456",
			"sourceJoinKey": "user_id",
			"targetJoinKey": "id",
			"dataGraphId": "dg-123",
			"workspaceId": "ws-456",
			"externalId": "pageview-user-rel",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.GetEventRelationship(context.Background(), "dg-123", "evr-789")
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Relationship{
		ID:           "evr-789",
		Name:         "PageView User",
		Type:         "event",
		SourceModelID:  "em-123",
		TargetModelID:    "evm-456",
		SourceJoinKey: "user_id",
		TargetJoinKey:   "id",
		DataGraphID:  "dg-123",
		WorkspaceID:  "ws-456",
		ExternalID:   "pageview-user-rel",
		CreatedAt:    &testTime1,
		UpdatedAt:    &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestUpdateEventRelationship(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"Updated PageView User","sourceModelId":"em-123","targetModelId":"evm-999","sourceJoinKey":"uid","targetJoinKey":"user_id"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/event-relationships/evr-789", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "evr-789",
			"name": "Updated PageView User",
			"sourceModelId": "em-123",
			"targetModelId": "evm-999",
			"sourceJoinKey": "uid",
			"targetJoinKey": "user_id",
			"dataGraphId": "dg-123",
			"workspaceId": "ws-456",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T13:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.UpdateEventRelationship(context.Background(), "dg-123", "evr-789", &datagraph.UpdateRelationshipRequest{
		Name:         "Updated PageView User",
		SourceModelID:  "em-123",
		TargetModelID:    "evm-999",
		SourceJoinKey: "uid",
		TargetJoinKey:   "user_id",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Relationship{
		ID:           "evr-789",
		Name:         "Updated PageView User",
		Type:         "event",
		SourceModelID:  "em-123",
		TargetModelID:    "evm-999",
		SourceJoinKey: "uid",
		TargetJoinKey:   "user_id",
		DataGraphID:  "dg-123",
		WorkspaceID:  "ws-456",
		CreatedAt:    &testTime1,
		UpdatedAt:    &testTime2,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestDeleteEventRelationship(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/v2/data-graphs/dg-123/event-relationships/evr-789", "")
		},
		ResponseStatus: 204,
		ResponseBody:   "",
	})

	store := newTestStore(t, httpClient)

	err := store.DeleteEventRelationship(context.Background(), "dg-123", "evr-789")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestListEventRelationships(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/event-relationships", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "evr-1",
					"name": "PageView User",
					"sourceModelId": "em-123",
					"targetModelId": "evm-456",
					"sourceJoinKey": "user_id",
					"targetJoinKey": "id",
					"dataGraphId": "dg-123",
					"workspaceId": "ws-456",
					"createdAt": "2024-01-15T12:00:00Z",
					"updatedAt": "2024-01-15T12:00:00Z"
				},
				{
					"id": "evr-2",
					"name": "Purchase User",
					"sourceModelId": "em-123",
					"targetModelId": "evm-789",
					"sourceJoinKey": "user_id",
					"targetJoinKey": "uid",
					"dataGraphId": "dg-123",
					"workspaceId": "ws-456",
					"createdAt": "2024-01-16T12:00:00Z",
					"updatedAt": "2024-01-16T12:00:00Z"
				}
			],
			"paging": {"total": 2}
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.ListEventRelationships(context.Background(), "dg-123", 0, 0, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, &datagraph.ListRelationshipsResponse{
		Data: []datagraph.Relationship{
			{
				ID:           "evr-1",
				Name:         "PageView User",
				Type:         "event",
				SourceModelID:  "em-123",
				TargetModelID:    "evm-456",
				SourceJoinKey: "user_id",
				TargetJoinKey:   "id",
				DataGraphID:  "dg-123",
				WorkspaceID:  "ws-456",
				CreatedAt:    &testTime1,
				UpdatedAt:    &testTime1,
			},
			{
				ID:           "evr-2",
				Name:         "Purchase User",
				Type:         "event",
				SourceModelID:  "em-123",
				TargetModelID:    "evm-789",
				SourceJoinKey: "user_id",
				TargetJoinKey:   "uid",
				DataGraphID:  "dg-123",
				WorkspaceID:  "ws-456",
				CreatedAt:    &testTime3,
				UpdatedAt:    &testTime3,
			},
		},
		Paging: client.Paging{Total: 2},
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestSetEventRelationshipExternalID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"externalId":"pageview-user-rel"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/event-relationships/evr-789/external-id", expected)
		},
		ResponseStatus: 204,
		ResponseBody:   "",
	})

	store := newTestStore(t, httpClient)

	err := store.SetEventRelationshipExternalID(context.Background(), "dg-123", "evr-789", "pageview-user-rel")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

// Error handling tests

func TestRelationshipErrorHandling(t *testing.T) {
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
			name:           "CreateEntityRelationship - API error",
			method:         "POST",
			path:           "/v2/data-graphs/dg-123/entity-relationships",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.CreateEntityRelationship(context.Background(), "dg-123", &datagraph.CreateRelationshipRequest{
					Name:         "Test",
					SourceModelID:  "em-123",
					TargetModelID:    "em-456",
					SourceJoinKey: "id",
					TargetJoinKey:   "uid",
				})
				return err
			},
			expectedError: "creating entity relationship",
		},
		{
			name:           "CreateEventRelationship - API error",
			method:         "POST",
			path:           "/v2/data-graphs/dg-123/event-relationships",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.CreateEventRelationship(context.Background(), "dg-123", &datagraph.CreateRelationshipRequest{
					Name:         "Test",
					SourceModelID:  "em-123",
					TargetModelID:    "evm-456",
					SourceJoinKey: "id",
					TargetJoinKey:   "uid",
				})
				return err
			},
			expectedError: "creating event relationship",
		},
		{
			name:           "GetEntityRelationship - not found",
			method:         "GET",
			path:           "/v2/data-graphs/dg-123/entity-relationships/er-999",
			responseStatus: 404,
			responseBody:   `{"error":"Not Found"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.GetEntityRelationship(context.Background(), "dg-123", "er-999")
				return err
			},
			expectedError: "getting entity relationship",
		},
		{
			name:           "UpdateEntityRelationship - API error",
			method:         "PUT",
			path:           "/v2/data-graphs/dg-123/entity-relationships/er-789",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.UpdateEntityRelationship(context.Background(), "dg-123", "er-789", &datagraph.UpdateRelationshipRequest{
					Name:         "Test",
					SourceModelID:  "em-123",
					TargetModelID:    "em-456",
					SourceJoinKey: "id",
					TargetJoinKey:   "uid",
				})
				return err
			},
			expectedError: "updating entity relationship",
		},
		{
			name:           "DeleteEntityRelationship - API error",
			method:         "DELETE",
			path:           "/v2/data-graphs/dg-123/entity-relationships/er-789",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				return store.DeleteEntityRelationship(context.Background(), "dg-123", "er-789")
			},
			expectedError: "deleting entity relationship",
		},
		{
			name:           "ListEntityRelationships - API error",
			method:         "GET",
			path:           "/v2/data-graphs/dg-123/entity-relationships",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.ListEntityRelationships(context.Background(), "dg-123", 0, 0, nil, nil)
				return err
			},
			expectedError: "listing entity relationships",
		},
		{
			name:           "SetEntityRelationshipExternalID - API error",
			method:         "PUT",
			path:           "/v2/data-graphs/dg-123/entity-relationships/er-789/external-id",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				return store.SetEntityRelationshipExternalID(context.Background(), "dg-123", "er-789", "ext-123")
			},
			expectedError: "setting external ID",
		},
		{
			name:           "GetEventRelationship - not found",
			method:         "GET",
			path:           "/v2/data-graphs/dg-123/event-relationships/evr-999",
			responseStatus: 404,
			responseBody:   `{"error":"Not Found"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.GetEventRelationship(context.Background(), "dg-123", "evr-999")
				return err
			},
			expectedError: "getting event relationship",
		},
		{
			name:           "ListEventRelationships - API error",
			method:         "GET",
			path:           "/v2/data-graphs/dg-123/event-relationships",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.ListEventRelationships(context.Background(), "dg-123", 0, 0, nil, nil)
				return err
			},
			expectedError: "listing event relationships",
		},
		{
			name:           "SetEventRelationshipExternalID - API error",
			method:         "PUT",
			path:           "/v2/data-graphs/dg-123/event-relationships/evr-789/external-id",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				return store.SetEventRelationshipExternalID(context.Background(), "dg-123", "evr-789", "ext-123")
			},
			expectedError: "setting external ID",
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
