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

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Entity Model Tests

func TestCreateModel_Entity(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"type":"entity","name":"User","tableRef":"users","primaryId":"id","root":true}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs/dg-123/models", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "em-456",
			"type": "entity",
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

	result, err := store.CreateModel(context.Background(), &datagraph.CreateModelRequest{DataGraphID: "dg-123", Type: "entity",
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

func TestCreateModel_EntityWithExternalID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"type":"entity","name":"User","tableRef":"users","externalId":"user-model","primaryId":"id","root":true}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs/dg-123/models", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "em-456",
			"type": "entity",
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

	result, err := store.CreateModel(context.Background(), &datagraph.CreateModelRequest{DataGraphID: "dg-123", Type: "entity",
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

func TestGetModel_Entity(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/models/em-456", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "em-456",
			"type": "entity",
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

	result, err := store.GetModel(context.Background(), &datagraph.GetModelRequest{DataGraphID: "dg-123", ModelID: "em-456"})
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

func TestUpdateModel_Entity(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"type":"entity","name":"Updated User","tableRef":"users_v2","primaryId":"user_id"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/models/em-456", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "em-456",
			"type": "entity",
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

	result, err := store.UpdateModel(context.Background(), &datagraph.UpdateModelRequest{DataGraphID: "dg-123", ModelID: "em-456", Type: "entity",
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

func TestDeleteModel_Entity(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/v2/data-graphs/dg-123/models/em-456", "")
		},
		ResponseStatus: 204,
		ResponseBody:   "",
	})

	store := newTestStore(t, httpClient)

	err := store.DeleteModel(context.Background(), &datagraph.DeleteModelRequest{DataGraphID: "dg-123", ModelID: "em-456"})
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestListModels_Entity(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/models", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "em-1",
			"type": "entity",
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
			"type": "entity",
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

	result, err := store.ListModels(context.Background(), &datagraph.ListModelsRequest{DataGraphID: "dg-123", Page: 0, PageSize: 0, ModelType: stringPtr("entity"), IsRoot: nil, HasExternalID: nil})
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

func TestListModels_EntityWithFilters(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			query := req.URL.Query()
			return req.Method == "GET" &&
				req.URL.Path == "/v2/data-graphs/dg-123/models" &&
				query.Get("type") == "entity" &&
				query.Get("isRoot") == "true" &&
				query.Get("hasExternalId") == "true"
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "em-1",
			"type": "entity",
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

	result, err := store.ListModels(context.Background(), &datagraph.ListModelsRequest{DataGraphID: "dg-123", Page: 0, PageSize: 0, ModelType: stringPtr("entity"), IsRoot: boolPtr(true), HasExternalID: boolPtr(true)})
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

func TestSetModelExternalID_Entity(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"externalId":"user-model"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/models/em-456/external-id", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "em-456",
			"type": "entity",
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

	result, err := store.SetModelExternalID(context.Background(), &datagraph.SetModelExternalIDRequest{DataGraphID: "dg-123", ModelID: "em-456", ExternalID: "user-model"})
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
		UpdatedAt:   &testTime2,
	}, result)

	httpClient.AssertNumberOfCalls()
}

// Event Model Tests

func TestCreateModel_Event(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"type":"event","name":"Purchase","tableRef":"purchases","timestamp":"event_time"}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs/dg-123/models", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "evm-789",
			"type": "event",
			"name": "Purchase",
			"tableRef": "purchases",
			"dataGraphId": "dg-123",
			"timestamp": "event_time",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.CreateModel(context.Background(), &datagraph.CreateModelRequest{DataGraphID: "dg-123", Type: "event",
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

func TestCreateModel_EventWithDescription(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"type":"event","name":"Purchase","description":"Purchase events","tableRef":"purchases","timestamp":"event_time"}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs/dg-123/models", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "evm-789",
			"type": "event",
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

	result, err := store.CreateModel(context.Background(), &datagraph.CreateModelRequest{DataGraphID: "dg-123", Type: "event",
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

func TestGetModel_Event(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/models/evm-789", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "evm-789",
			"type": "event",
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

	result, err := store.GetModel(context.Background(), &datagraph.GetModelRequest{DataGraphID: "dg-123", ModelID: "evm-789"})
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

func TestUpdateModel_Event(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"type":"event","name":"Updated Purchase","tableRef":"purchases_v2","timestamp":"ts"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/models/evm-789", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "evm-789",
			"type": "event",
			"name": "Updated Purchase",
			"tableRef": "purchases_v2",
			"dataGraphId": "dg-123",
			"timestamp": "ts",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T13:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.UpdateModel(context.Background(), &datagraph.UpdateModelRequest{DataGraphID: "dg-123", ModelID: "evm-789", Type: "event",
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

func TestDeleteModel_Event(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/v2/data-graphs/dg-123/models/evm-789", "")
		},
		ResponseStatus: 204,
		ResponseBody:   "",
	})

	store := newTestStore(t, httpClient)

	err := store.DeleteModel(context.Background(), &datagraph.DeleteModelRequest{DataGraphID: "dg-123", ModelID: "evm-789"})
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestListModels_Event(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/models", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "evm-1",
			"type": "event",
					"name": "Purchase",
					"tableRef": "purchases",
					"dataGraphId": "dg-123",
					"timestamp": "event_time",
					"createdAt": "2024-01-15T12:00:00Z",
					"updatedAt": "2024-01-15T12:00:00Z"
				},
				{
					"id": "evm-2",
			"type": "event",
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

	result, err := store.ListModels(context.Background(), &datagraph.ListModelsRequest{DataGraphID: "dg-123", Page: 0, PageSize: 0, ModelType: stringPtr("event"), HasExternalID: nil})
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

func TestListModels_EventWithPagination(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			query := req.URL.Query()
			return req.Method == "GET" &&
				req.URL.Path == "/v2/data-graphs/dg-123/models" &&
				query.Get("type") == "event" &&
				query.Get("page") == "2" &&
				query.Get("pageSize") == "10"
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "evm-11",
			"type": "event",
					"name": "Event 11",
					"tableRef": "events",
					"dataGraphId": "dg-123",
					"timestamp": "ts",
					"createdAt": "2024-01-15T12:00:00Z",
					"updatedAt": "2024-01-15T12:00:00Z"
				}
			],
			"paging": {"total": 42, "next": "/v2/data-graphs/dg-123/models?page=3&pageSize=10"}
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.ListModels(context.Background(), &datagraph.ListModelsRequest{DataGraphID: "dg-123", Page: 2, PageSize: 10, ModelType: stringPtr("event"), HasExternalID: nil})
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
		Paging: client.Paging{Total: 42, Next: "/v2/data-graphs/dg-123/models?page=3&pageSize=10"},
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestSetModelExternalID_Event(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"externalId":"purchase-event"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/models/evm-789/external-id", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "evm-789",
			"type": "event",
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

	result, err := store.SetModelExternalID(context.Background(), &datagraph.SetModelExternalIDRequest{DataGraphID: "dg-123", ModelID: "evm-789", ExternalID: "purchase-event"})
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
		UpdatedAt:   &testTime2,
	}, result)

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
				_, err := store.CreateModel(context.Background(), &datagraph.CreateModelRequest{DataGraphID: "", Type: "entity",
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
				_, err := store.CreateModel(context.Background(), &datagraph.CreateModelRequest{DataGraphID: "", Type: "event",
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
				_, err := store.GetModel(context.Background(), &datagraph.GetModelRequest{DataGraphID: "", ModelID: "em-456"})
				return err
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "GetEntityModel - empty model ID",
			operation: func() error {
				_, err := store.GetModel(context.Background(), &datagraph.GetModelRequest{DataGraphID: "dg-123", ModelID: ""})
				return err
			},
			expectedError: "model ID cannot be empty",
		},
		{
			name: "GetEventModel - empty data graph ID",
			operation: func() error {
				_, err := store.GetModel(context.Background(), &datagraph.GetModelRequest{DataGraphID: "", ModelID: "evm-789"})
				return err
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "GetEventModel - empty model ID",
			operation: func() error {
				_, err := store.GetModel(context.Background(), &datagraph.GetModelRequest{DataGraphID: "dg-123", ModelID: ""})
				return err
			},
			expectedError: "model ID cannot be empty",
		},
		{
			name: "UpdateEntityModel - empty data graph ID",
			operation: func() error {
				_, err := store.UpdateModel(context.Background(), &datagraph.UpdateModelRequest{DataGraphID: "", ModelID: "em-456", Type: "entity",
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
				_, err := store.UpdateModel(context.Background(), &datagraph.UpdateModelRequest{DataGraphID: "dg-123", ModelID: "", Type: "entity",
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
				_, err := store.UpdateModel(context.Background(), &datagraph.UpdateModelRequest{DataGraphID: "", ModelID: "evm-789", Type: "event",
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
				_, err := store.UpdateModel(context.Background(), &datagraph.UpdateModelRequest{DataGraphID: "dg-123", ModelID: "", Type: "event",
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
				return store.DeleteModel(context.Background(), &datagraph.DeleteModelRequest{DataGraphID: "", ModelID: "em-456"})
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "DeleteEntityModel - empty model ID",
			operation: func() error {
				return store.DeleteModel(context.Background(), &datagraph.DeleteModelRequest{DataGraphID: "dg-123", ModelID: ""})
			},
			expectedError: "model ID cannot be empty",
		},
		{
			name: "DeleteEventModel - empty data graph ID",
			operation: func() error {
				return store.DeleteModel(context.Background(), &datagraph.DeleteModelRequest{DataGraphID: "", ModelID: "evm-789"})
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "DeleteEventModel - empty model ID",
			operation: func() error {
				return store.DeleteModel(context.Background(), &datagraph.DeleteModelRequest{DataGraphID: "dg-123", ModelID: ""})
			},
			expectedError: "model ID cannot be empty",
		},
		{
			name: "SetEntityModelExternalID - empty data graph ID",
			operation: func() error {
				_, err := store.SetModelExternalID(context.Background(), &datagraph.SetModelExternalIDRequest{DataGraphID: "", ModelID: "em-456", ExternalID: "ext-123"})
				return err
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "SetEntityModelExternalID - empty model ID",
			operation: func() error {
				_, err := store.SetModelExternalID(context.Background(), &datagraph.SetModelExternalIDRequest{DataGraphID: "dg-123", ModelID: "", ExternalID: "ext-123"})
				return err
			},
			expectedError: "model ID cannot be empty",
		},
		{
			name: "SetEventModelExternalID - empty data graph ID",
			operation: func() error {
				_, err := store.SetModelExternalID(context.Background(), &datagraph.SetModelExternalIDRequest{DataGraphID: "", ModelID: "evm-789", ExternalID: "ext-123"})
				return err
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "SetEventModelExternalID - empty model ID",
			operation: func() error {
				_, err := store.SetModelExternalID(context.Background(), &datagraph.SetModelExternalIDRequest{DataGraphID: "dg-123", ModelID: "", ExternalID: "ext-123"})
				return err
			},
			expectedError: "model ID cannot be empty",
		},
		{
			name: "ListEntityModels - empty data graph ID",
			operation: func() error {
				_, err := store.ListModels(context.Background(), &datagraph.ListModelsRequest{DataGraphID: "", ModelType: stringPtr("entity")})
				return err
			},
			expectedError: "data graph ID cannot be empty",
		},
		{
			name: "ListEventModels - empty data graph ID",
			operation: func() error {
				_, err := store.ListModels(context.Background(), &datagraph.ListModelsRequest{DataGraphID: "", ModelType: stringPtr("event")})
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
			path:           "/v2/data-graphs/dg-123/models",
			responseStatus: 400,
			responseBody:   `{"error":"Bad Request"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.CreateModel(context.Background(), &datagraph.CreateModelRequest{DataGraphID: "dg-123", Type: "entity",
					Name:      "User",
					TableRef:  "users",
					PrimaryID: "id",
				})
				return err
			},
			expectedError: "creating model",
		},
		{
			name:           "CreateEventModel - API error",
			method:         "POST",
			path:           "/v2/data-graphs/dg-123/models",
			responseStatus: 400,
			responseBody:   `{"error":"Bad Request"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.CreateModel(context.Background(), &datagraph.CreateModelRequest{DataGraphID: "dg-123", Type: "event",
					Name:      "Purchase",
					TableRef:  "purchases",
					Timestamp: "ts",
				})
				return err
			},
			expectedError: "creating model",
		},
		{
			name:           "GetEntityModel - not found",
			method:         "GET",
			path:           "/v2/data-graphs/dg-123/models/em-456",
			responseStatus: 404,
			responseBody:   `{"error":"Not Found"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.GetModel(context.Background(), &datagraph.GetModelRequest{DataGraphID: "dg-123", ModelID: "em-456"})
				return err
			},
			expectedError: "getting model",
		},
		{
			name:           "GetEventModel - not found",
			method:         "GET",
			path:           "/v2/data-graphs/dg-123/models/evm-789",
			responseStatus: 404,
			responseBody:   `{"error":"Not Found"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.GetModel(context.Background(), &datagraph.GetModelRequest{DataGraphID: "dg-123", ModelID: "evm-789"})
				return err
			},
			expectedError: "getting model",
		},
		{
			name:           "UpdateEntityModel - API error",
			method:         "PUT",
			path:           "/v2/data-graphs/dg-123/models/em-456",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.UpdateModel(context.Background(), &datagraph.UpdateModelRequest{DataGraphID: "dg-123", ModelID: "em-456", Type: "entity",
					Name:      "User",
					TableRef:  "users",
					PrimaryID: "id",
				})
				return err
			},
			expectedError: "updating model",
		},
		{
			name:           "UpdateEventModel - API error",
			method:         "PUT",
			path:           "/v2/data-graphs/dg-123/models/evm-789",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.UpdateModel(context.Background(), &datagraph.UpdateModelRequest{DataGraphID: "dg-123", ModelID: "evm-789", Type: "event",
					Name:      "Purchase",
					TableRef:  "purchases",
					Timestamp: "ts",
				})
				return err
			},
			expectedError: "updating model",
		},
		{
			name:           "DeleteEntityModel - API error",
			method:         "DELETE",
			path:           "/v2/data-graphs/dg-123/models/em-456",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				return store.DeleteModel(context.Background(), &datagraph.DeleteModelRequest{DataGraphID: "dg-123", ModelID: "em-456"})
			},
			expectedError: "deleting model",
		},
		{
			name:           "DeleteEventModel - API error",
			method:         "DELETE",
			path:           "/v2/data-graphs/dg-123/models/evm-789",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				return store.DeleteModel(context.Background(), &datagraph.DeleteModelRequest{DataGraphID: "dg-123", ModelID: "evm-789"})
			},
			expectedError: "deleting model",
		},
		{
			name:           "ListEntityModels - API error",
			method:         "GET",
			path:           "/v2/data-graphs/dg-123/models",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.ListModels(context.Background(), &datagraph.ListModelsRequest{DataGraphID: "dg-123", Page: 0, PageSize: 0, ModelType: stringPtr("entity"), IsRoot: nil, HasExternalID: nil})
				return err
			},
			expectedError: "listing models",
		},
		{
			name:           "ListEventModels - API error",
			method:         "GET",
			path:           "/v2/data-graphs/dg-123/models",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.ListModels(context.Background(), &datagraph.ListModelsRequest{DataGraphID: "dg-123", Page: 0, PageSize: 0, ModelType: stringPtr("event"), HasExternalID: nil})
				return err
			},
			expectedError: "listing models",
		},
		{
			name:           "SetEntityModelExternalID - API error",
			method:         "PUT",
			path:           "/v2/data-graphs/dg-123/models/em-456/external-id",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.SetModelExternalID(context.Background(), &datagraph.SetModelExternalIDRequest{DataGraphID: "dg-123", ModelID: "em-456", ExternalID: "ext-123"})
				return err
			},
			expectedError: "setting external ID",
		},
		{
			name:           "SetEventModelExternalID - API error",
			method:         "PUT",
			path:           "/v2/data-graphs/dg-123/models/evm-789/external-id",
			responseStatus: 500,
			responseBody:   `{"error":"Internal Server Error"}`,
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.SetModelExternalID(context.Background(), &datagraph.SetModelExternalIDRequest{DataGraphID: "dg-123", ModelID: "evm-789", ExternalID: "ext-123"})
				return err
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
