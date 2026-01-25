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

// Entity Model Tests

func TestCreateEntityModel(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"User","tableRef":"users","primaryId":"id","root":true}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs/dg-123/entity-models", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "em-456",
			"name": "User",
			"tableRef": "users",
			"dataGraphId": "dg-123",
			"primaryId": "id",
			"root": true,
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.CreateEntityModel(context.Background(), "dg-123", &datagraph.CreateEntityModelRequest{
		Name:      "User",
		TableRef:  "users",
		PrimaryID: "id",
		Root:      true,
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Model{
		ID:          "em-456",
		Name:        "User",
		Type:        "entity",
		TableRef:    "users",
		DataGraphID: "dg-123",
		PrimaryID:   "id",
		Root:        true,
		CreatedAt:   &testTime1,
		UpdatedAt:   &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestCreateEntityModelWithExternalID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"User","tableRef":"users","externalId":"user-model","primaryId":"id","root":true}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs/dg-123/entity-models", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "em-456",
			"name": "User",
			"tableRef": "users",
			"dataGraphId": "dg-123",
			"externalId": "user-model",
			"primaryId": "id",
			"root": true,
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.CreateEntityModel(context.Background(), "dg-123", &datagraph.CreateEntityModelRequest{
		Name:       "User",
		TableRef:   "users",
		ExternalID: "user-model",
		PrimaryID:  "id",
		Root:       true,
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Model{
		ID:          "em-456",
		Name:        "User",
		Type:        "entity",
		TableRef:    "users",
		DataGraphID: "dg-123",
		ExternalID:  "user-model",
		PrimaryID:   "id",
		Root:        true,
		CreatedAt:   &testTime1,
		UpdatedAt:   &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestGetEntityModel(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/entity-models/em-456", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "em-456",
			"name": "User",
			"tableRef": "users",
			"dataGraphId": "dg-123",
			"externalId": "user-model",
			"primaryId": "id",
			"root": true,
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.GetEntityModel(context.Background(), "dg-123", "em-456")
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Model{
		ID:          "em-456",
		Name:        "User",
		Type:        "entity",
		TableRef:    "users",
		DataGraphID: "dg-123",
		ExternalID:  "user-model",
		PrimaryID:   "id",
		Root:        true,
		CreatedAt:   &testTime1,
		UpdatedAt:   &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestUpdateEntityModel(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"Updated User","tableRef":"users_v2","primaryId":"user_id","root":false}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/entity-models/em-456", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "em-456",
			"name": "Updated User",
			"tableRef": "users_v2",
			"dataGraphId": "dg-123",
			"primaryId": "user_id",
			"root": false,
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T13:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.UpdateEntityModel(context.Background(), "dg-123", "em-456", &datagraph.UpdateEntityModelRequest{
		Name:      "Updated User",
		TableRef:  "users_v2",
		PrimaryID: "user_id",
		Root:      false,
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Model{
		ID:          "em-456",
		Name:        "Updated User",
		Type:        "entity",
		TableRef:    "users_v2",
		DataGraphID: "dg-123",
		PrimaryID:   "user_id",
		Root:        false,
		CreatedAt:   &testTime1,
		UpdatedAt:   &testTime2,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestDeleteEntityModel(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/v2/data-graphs/dg-123/entity-models/em-456", "")
		},
		ResponseStatus: 204,
		ResponseBody:   "",
	})

	store := newTestStore(t, httpClient)

	err := store.DeleteEntityModel(context.Background(), "dg-123", "em-456")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestListEntityModels(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/entity-models", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "em-1",
					"name": "User",
					"tableRef": "users",
					"dataGraphId": "dg-123",
					"primaryId": "id",
					"root": true,
					"createdAt": "2024-01-15T12:00:00Z",
					"updatedAt": "2024-01-15T12:00:00Z"
				},
				{
					"id": "em-2",
					"name": "Account",
					"tableRef": "accounts",
					"dataGraphId": "dg-123",
					"primaryId": "account_id",
					"root": false,
					"createdAt": "2024-01-16T12:00:00Z",
					"updatedAt": "2024-01-16T12:00:00Z"
				}
			],
			"paging": {"total": 2}
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.ListEntityModels(context.Background(), "dg-123", 0, 0, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, &datagraph.ListModelsResponse{
		Data: []datagraph.Model{
			{
				ID:          "em-1",
				Name:        "User",
				Type:        "entity",
				TableRef:    "users",
				DataGraphID: "dg-123",
				PrimaryID:   "id",
				Root:        true,
				CreatedAt:   &testTime1,
				UpdatedAt:   &testTime1,
			},
			{
				ID:          "em-2",
				Name:        "Account",
				Type:        "entity",
				TableRef:    "accounts",
				DataGraphID: "dg-123",
				PrimaryID:   "account_id",
				Root:        false,
				CreatedAt:   &testTime3,
				UpdatedAt:   &testTime3,
			},
		},
		Paging: client.Paging{Total: 2},
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestListEntityModelsWithFilters(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			query := req.URL.Query()
			return req.Method == "GET" &&
				req.URL.Path == "/v2/data-graphs/dg-123/entity-models" &&
				query.Get("isRoot") == "true" &&
				query.Get("hasExternalId") == "true"
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "em-1",
					"name": "User",
					"tableRef": "users",
					"dataGraphId": "dg-123",
					"externalId": "user-model",
					"primaryId": "id",
					"root": true,
					"createdAt": "2024-01-15T12:00:00Z",
					"updatedAt": "2024-01-15T12:00:00Z"
				}
			],
			"paging": {"total": 1}
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.ListEntityModels(context.Background(), "dg-123", 0, 0, boolPtr(true), boolPtr(true))
	require.NoError(t, err)

	assert.Equal(t, &datagraph.ListModelsResponse{
		Data: []datagraph.Model{
			{
				ID:          "em-1",
				Name:        "User",
				Type:        "entity",
				TableRef:    "users",
				DataGraphID: "dg-123",
				ExternalID:  "user-model",
				PrimaryID:   "id",
				Root:        true,
				CreatedAt:   &testTime1,
				UpdatedAt:   &testTime1,
			},
		},
		Paging: client.Paging{Total: 1},
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestSetEntityModelExternalID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"externalId":"user-model"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/entity-models/em-456/external-id", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "em-456",
			"name": "User",
			"tableRef": "users",
			"dataGraphId": "dg-123",
			"externalId": "user-model",
			"primaryId": "id",
			"root": true,
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T13:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	err := store.SetEntityModelExternalID(context.Background(), "dg-123", "em-456", "user-model")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

// Event Model Tests

func TestCreateEventModel(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"Purchase","tableRef":"purchases","timestamp":"event_time"}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs/dg-123/event-models", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "evm-789",
			"name": "Purchase",
			"tableRef": "purchases",
			"dataGraphId": "dg-123",
			"timestamp": "event_time",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.CreateEventModel(context.Background(), "dg-123", &datagraph.CreateEventModelRequest{
		Name:      "Purchase",
		TableRef:  "purchases",
		Timestamp: "event_time",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Model{
		ID:          "evm-789",
		Name:        "Purchase",
		Type:        "event",
		TableRef:    "purchases",
		DataGraphID: "dg-123",
		Timestamp:   "event_time",
		CreatedAt:   &testTime1,
		UpdatedAt:   &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestCreateEventModelWithDescription(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"Purchase","description":"Purchase events","tableRef":"purchases","timestamp":"event_time"}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs/dg-123/event-models", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "evm-789",
			"name": "Purchase",
			"description": "Purchase events",
			"tableRef": "purchases",
			"dataGraphId": "dg-123",
			"timestamp": "event_time",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.CreateEventModel(context.Background(), "dg-123", &datagraph.CreateEventModelRequest{
		Name:        "Purchase",
		Description: "Purchase events",
		TableRef:    "purchases",
		Timestamp:   "event_time",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Model{
		ID:          "evm-789",
		Name:        "Purchase",
		Type:        "event",
		Description: "Purchase events",
		TableRef:    "purchases",
		DataGraphID: "dg-123",
		Timestamp:   "event_time",
		CreatedAt:   &testTime1,
		UpdatedAt:   &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestGetEventModel(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/event-models/evm-789", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "evm-789",
			"name": "Purchase",
			"tableRef": "purchases",
			"dataGraphId": "dg-123",
			"externalId": "purchase-event",
			"timestamp": "event_time",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.GetEventModel(context.Background(), "dg-123", "evm-789")
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Model{
		ID:          "evm-789",
		Name:        "Purchase",
		Type:        "event",
		TableRef:    "purchases",
		DataGraphID: "dg-123",
		ExternalID:  "purchase-event",
		Timestamp:   "event_time",
		CreatedAt:   &testTime1,
		UpdatedAt:   &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestUpdateEventModel(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"Updated Purchase","tableRef":"purchases_v2","timestamp":"ts"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/event-models/evm-789", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "evm-789",
			"name": "Updated Purchase",
			"tableRef": "purchases_v2",
			"dataGraphId": "dg-123",
			"timestamp": "ts",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T13:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.UpdateEventModel(context.Background(), "dg-123", "evm-789", &datagraph.UpdateEventModelRequest{
		Name:      "Updated Purchase",
		TableRef:  "purchases_v2",
		Timestamp: "ts",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Model{
		ID:          "evm-789",
		Name:        "Updated Purchase",
		Type:        "event",
		TableRef:    "purchases_v2",
		DataGraphID: "dg-123",
		Timestamp:   "ts",
		CreatedAt:   &testTime1,
		UpdatedAt:   &testTime2,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestDeleteEventModel(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/v2/data-graphs/dg-123/event-models/evm-789", "")
		},
		ResponseStatus: 204,
		ResponseBody:   "",
	})

	store := newTestStore(t, httpClient)

	err := store.DeleteEventModel(context.Background(), "dg-123", "evm-789")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestListEventModels(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/event-models", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "evm-1",
					"name": "Purchase",
					"tableRef": "purchases",
					"dataGraphId": "dg-123",
					"timestamp": "event_time",
					"createdAt": "2024-01-15T12:00:00Z",
					"updatedAt": "2024-01-15T12:00:00Z"
				},
				{
					"id": "evm-2",
					"name": "PageView",
					"tableRef": "page_views",
					"dataGraphId": "dg-123",
					"timestamp": "ts",
					"createdAt": "2024-01-16T12:00:00Z",
					"updatedAt": "2024-01-16T12:00:00Z"
				}
			],
			"paging": {"total": 2}
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.ListEventModels(context.Background(), "dg-123", 0, 0, nil)
	require.NoError(t, err)

	assert.Equal(t, &datagraph.ListModelsResponse{
		Data: []datagraph.Model{
			{
				ID:          "evm-1",
				Name:        "Purchase",
				Type:        "event",
				TableRef:    "purchases",
				DataGraphID: "dg-123",
				Timestamp:   "event_time",
				CreatedAt:   &testTime1,
				UpdatedAt:   &testTime1,
			},
			{
				ID:          "evm-2",
				Name:        "PageView",
				Type:        "event",
				TableRef:    "page_views",
				DataGraphID: "dg-123",
				Timestamp:   "ts",
				CreatedAt:   &testTime3,
				UpdatedAt:   &testTime3,
			},
		},
		Paging: client.Paging{Total: 2},
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestListEventModelsWithPagination(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			query := req.URL.Query()
			return req.Method == "GET" &&
				req.URL.Path == "/v2/data-graphs/dg-123/event-models" &&
				query.Get("page") == "2" &&
				query.Get("pageSize") == "10"
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "evm-11",
					"name": "Event 11",
					"tableRef": "events",
					"dataGraphId": "dg-123",
					"timestamp": "ts",
					"createdAt": "2024-01-15T12:00:00Z",
					"updatedAt": "2024-01-15T12:00:00Z"
				}
			],
			"paging": {"total": 42, "next": "/v2/data-graphs/dg-123/event-models?page=3&pageSize=10"}
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.ListEventModels(context.Background(), "dg-123", 2, 10, nil)
	require.NoError(t, err)

	assert.Equal(t, &datagraph.ListModelsResponse{
		Data: []datagraph.Model{
			{
				ID:          "evm-11",
				Name:        "Event 11",
				Type:        "event",
				TableRef:    "events",
				DataGraphID: "dg-123",
				Timestamp:   "ts",
				CreatedAt:   &testTime1,
				UpdatedAt:   &testTime1,
			},
		},
		Paging: client.Paging{Total: 42, Next: "/v2/data-graphs/dg-123/event-models?page=3&pageSize=10"},
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestSetEventModelExternalID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"externalId":"purchase-event"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/event-models/evm-789/external-id", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "evm-789",
			"name": "Purchase",
			"tableRef": "purchases",
			"dataGraphId": "dg-123",
			"externalId": "purchase-event",
			"timestamp": "event_time",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T13:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	err := store.SetEventModelExternalID(context.Background(), "dg-123", "evm-789", "purchase-event")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

// Validation Tests

func TestModelValidation(t *testing.T) {
	// Mock HTTP client is not called since validation happens before HTTP request
	httpClient := testutils.NewMockHTTPClient(t)
	store := newTestStore(t, httpClient)

	tests := []struct {
		name          string
		operation     func() error
		expectedError string
	}{
		{
			name: "CreateEntityModel - empty data graph ID",
			operation: func() error {
				_, err := store.CreateEntityModel(context.Background(), "", &datagraph.CreateEntityModelRequest{
					Name:      "User",
					TableRef:  "users",
					PrimaryID: "id",
				})
				return err
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "CreateEventModel - empty data graph ID",
			operation: func() error {
				_, err := store.CreateEventModel(context.Background(), "", &datagraph.CreateEventModelRequest{
					Name:      "Purchase",
					TableRef:  "purchases",
					Timestamp: "ts",
				})
				return err
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "GetEntityModel - empty data graph ID",
			operation: func() error {
				_, err := store.GetEntityModel(context.Background(), "", "em-456")
				return err
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "GetEntityModel - empty model ID",
			operation: func() error {
				_, err := store.GetEntityModel(context.Background(), "dg-123", "")
				return err
			},
			expectedError: "model ID cannot be empty",
		},
		{
			name: "GetEventModel - empty data graph ID",
			operation: func() error {
				_, err := store.GetEventModel(context.Background(), "", "evm-789")
				return err
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "GetEventModel - empty model ID",
			operation: func() error {
				_, err := store.GetEventModel(context.Background(), "dg-123", "")
				return err
			},
			expectedError: "model ID cannot be empty",
		},
		{
			name: "UpdateEntityModel - empty data graph ID",
			operation: func() error {
				_, err := store.UpdateEntityModel(context.Background(), "", "em-456", &datagraph.UpdateEntityModelRequest{
					Name:      "User",
					TableRef:  "users",
					PrimaryID: "id",
				})
				return err
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "UpdateEntityModel - empty model ID",
			operation: func() error {
				_, err := store.UpdateEntityModel(context.Background(), "dg-123", "", &datagraph.UpdateEntityModelRequest{
					Name:      "User",
					TableRef:  "users",
					PrimaryID: "id",
				})
				return err
			},
			expectedError: "model ID cannot be empty",
		},
		{
			name: "UpdateEventModel - empty data graph ID",
			operation: func() error {
				_, err := store.UpdateEventModel(context.Background(), "", "evm-789", &datagraph.UpdateEventModelRequest{
					Name:      "Purchase",
					TableRef:  "purchases",
					Timestamp: "ts",
				})
				return err
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "UpdateEventModel - empty model ID",
			operation: func() error {
				_, err := store.UpdateEventModel(context.Background(), "dg-123", "", &datagraph.UpdateEventModelRequest{
					Name:      "Purchase",
					TableRef:  "purchases",
					Timestamp: "ts",
				})
				return err
			},
			expectedError: "model ID cannot be empty",
		},
		{
			name: "DeleteEntityModel - empty data graph ID",
			operation: func() error {
				return store.DeleteEntityModel(context.Background(), "", "em-456")
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "DeleteEntityModel - empty model ID",
			operation: func() error {
				return store.DeleteEntityModel(context.Background(), "dg-123", "")
			},
			expectedError: "model ID cannot be empty",
		},
		{
			name: "DeleteEventModel - empty data graph ID",
			operation: func() error {
				return store.DeleteEventModel(context.Background(), "", "evm-789")
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "DeleteEventModel - empty model ID",
			operation: func() error {
				return store.DeleteEventModel(context.Background(), "dg-123", "")
			},
			expectedError: "model ID cannot be empty",
		},
		{
			name: "SetEntityModelExternalID - empty data graph ID",
			operation: func() error {
				return store.SetEntityModelExternalID(context.Background(), "", "em-456", "ext-123")
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "SetEntityModelExternalID - empty model ID",
			operation: func() error {
				return store.SetEntityModelExternalID(context.Background(), "dg-123", "", "ext-123")
			},
			expectedError: "model ID cannot be empty",
		},
		{
			name: "SetEventModelExternalID - empty data graph ID",
			operation: func() error {
				return store.SetEventModelExternalID(context.Background(), "", "evm-789", "ext-123")
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "SetEventModelExternalID - empty model ID",
			operation: func() error {
				return store.SetEventModelExternalID(context.Background(), "dg-123", "", "ext-123")
			},
			expectedError: "model ID cannot be empty",
		},
		{
			name: "ListEntityModels - empty data graph ID",
			operation: func() error {
				_, err := store.ListEntityModels(context.Background(), "", 0, 0, nil, nil)
				return err
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "ListEventModels - empty data graph ID",
			operation: func() error {
				_, err := store.ListEventModels(context.Background(), "", 0, 0, nil)
				return err
			},
			expectedError: "data graph ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// API Error Tests

func TestModelAPIErrors(t *testing.T) {
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
			name:           "CreateEntityModel - API error",
			method:         "POST",
			path:           "/v2/data-graphs/dg-123/entity-models",
			responseStatus: 400,
			responseBody:   `{"error":"Bad Request"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.CreateEntityModel(context.Background(), "dg-123", &datagraph.CreateEntityModelRequest{
					Name:      "User",
					TableRef:  "users",
					PrimaryID: "id",
				})
				return err
			},
			expectedError: "creating entity model",
		},
		{
			name:           "CreateEventModel - API error",
			method:         "POST",
			path:           "/v2/data-graphs/dg-123/event-models",
			responseStatus: 400,
			responseBody:   `{"error":"Bad Request"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.CreateEventModel(context.Background(), "dg-123", &datagraph.CreateEventModelRequest{
					Name:      "Purchase",
					TableRef:  "purchases",
					Timestamp: "ts",
				})
				return err
			},
			expectedError: "creating event model",
		},
		{
			name:           "GetEntityModel - not found",
			method:         "GET",
			path:           "/v2/data-graphs/dg-123/entity-models/em-456",
			responseStatus: 404,
			responseBody:   `{"error":"Not Found"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.GetEntityModel(context.Background(), "dg-123", "em-456")
				return err
			},
			expectedError: "getting entity model",
		},
		{
			name:           "GetEventModel - not found",
			method:         "GET",
			path:           "/v2/data-graphs/dg-123/event-models/evm-789",
			responseStatus: 404,
			responseBody:   `{"error":"Not Found"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.GetEventModel(context.Background(), "dg-123", "evm-789")
				return err
			},
			expectedError: "getting event model",
		},
		{
			name:           "UpdateEntityModel - API error",
			method:         "PUT",
			path:           "/v2/data-graphs/dg-123/entity-models/em-456",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.UpdateEntityModel(context.Background(), "dg-123", "em-456", &datagraph.UpdateEntityModelRequest{
					Name:      "User",
					TableRef:  "users",
					PrimaryID: "id",
				})
				return err
			},
			expectedError: "updating entity model",
		},
		{
			name:           "UpdateEventModel - API error",
			method:         "PUT",
			path:           "/v2/data-graphs/dg-123/event-models/evm-789",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.UpdateEventModel(context.Background(), "dg-123", "evm-789", &datagraph.UpdateEventModelRequest{
					Name:      "Purchase",
					TableRef:  "purchases",
					Timestamp: "ts",
				})
				return err
			},
			expectedError: "updating event model",
		},
		{
			name:           "DeleteEntityModel - API error",
			method:         "DELETE",
			path:           "/v2/data-graphs/dg-123/entity-models/em-456",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				return store.DeleteEntityModel(context.Background(), "dg-123", "em-456")
			},
			expectedError: "deleting entity model",
		},
		{
			name:           "DeleteEventModel - API error",
			method:         "DELETE",
			path:           "/v2/data-graphs/dg-123/event-models/evm-789",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				return store.DeleteEventModel(context.Background(), "dg-123", "evm-789")
			},
			expectedError: "deleting event model",
		},
		{
			name:           "ListEntityModels - API error",
			method:         "GET",
			path:           "/v2/data-graphs/dg-123/entity-models",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.ListEntityModels(context.Background(), "dg-123", 0, 0, nil, nil)
				return err
			},
			expectedError: "listing entity models",
		},
		{
			name:           "ListEventModels - API error",
			method:         "GET",
			path:           "/v2/data-graphs/dg-123/event-models",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.ListEventModels(context.Background(), "dg-123", 0, 0, nil)
				return err
			},
			expectedError: "listing event models",
		},
		{
			name:           "SetEntityModelExternalID - API error",
			method:         "PUT",
			path:           "/v2/data-graphs/dg-123/entity-models/em-456/external-id",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				return store.SetEntityModelExternalID(context.Background(), "dg-123", "em-456", "ext-123")
			},
			expectedError: "setting external ID",
		},
		{
			name:           "SetEventModelExternalID - API error",
			method:         "PUT",
			path:           "/v2/data-graphs/dg-123/event-models/evm-789/external-id",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				return store.SetEventModelExternalID(context.Background(), "dg-123", "evm-789", "ext-123")
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
