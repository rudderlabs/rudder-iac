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

func TestCreateRelationship(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"User Orders","cardinality":"one-to-many","sourceModelId":"m-123","targetModelId":"m-456","sourceJoinKey":"id","targetJoinKey":"user_id"}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs/dg-123/relationships", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "rel-789",
			"name": "User Orders",
			"cardinality": "one-to-many",
			"sourceModelId": "m-123",
			"targetModelId": "m-456",
			"sourceJoinKey": "id",
			"targetJoinKey": "user_id",
			"dataGraphId": "dg-123",
			"workspaceId": "ws-456",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.CreateRelationship(context.Background(), &datagraph.CreateRelationshipRequest{
		DataGraphID:   "dg-123",
		Name:          "User Orders",
		Cardinality:   "one-to-many",
		SourceModelID: "m-123",
		TargetModelID: "m-456",
		SourceJoinKey: "id",
		TargetJoinKey: "user_id",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Relationship{
		ID:            "rel-789",
		Name:          "User Orders",
		Cardinality:   "one-to-many",
		SourceModelID: "m-123",
		TargetModelID: "m-456",
		SourceJoinKey: "id",
		TargetJoinKey: "user_id",
		DataGraphID:   "dg-123",
		WorkspaceID:   "ws-456",
		CreatedAt:     &testTime1,
		UpdatedAt:     &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestCreateRelationshipWithExternalID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"User Orders","cardinality":"one-to-many","sourceModelId":"m-123","targetModelId":"m-456","sourceJoinKey":"id","targetJoinKey":"user_id","externalId":"user-orders-rel"}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/data-graphs/dg-123/relationships", expected)
		},
		ResponseStatus: 201,
		ResponseBody: `{
			"id": "rel-789",
			"name": "User Orders",
			"cardinality": "one-to-many",
			"sourceModelId": "m-123",
			"targetModelId": "m-456",
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

	result, err := store.CreateRelationship(context.Background(), &datagraph.CreateRelationshipRequest{
		DataGraphID:   "dg-123",
		Name:          "User Orders",
		Cardinality:   "one-to-many",
		SourceModelID: "m-123",
		TargetModelID: "m-456",
		SourceJoinKey: "id",
		TargetJoinKey: "user_id",
		ExternalID:    "user-orders-rel",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Relationship{
		ID:            "rel-789",
		Name:          "User Orders",
		Cardinality:   "one-to-many",
		SourceModelID: "m-123",
		TargetModelID: "m-456",
		SourceJoinKey: "id",
		TargetJoinKey: "user_id",
		DataGraphID:   "dg-123",
		WorkspaceID:   "ws-456",
		ExternalID:    "user-orders-rel",
		CreatedAt:     &testTime1,
		UpdatedAt:     &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestGetRelationship(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/relationships/rel-789", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "rel-789",
			"name": "User Orders",
			"cardinality": "one-to-many",
			"sourceModelId": "m-123",
			"targetModelId": "m-456",
			"sourceJoinKey": "id",
			"targetJoinKey": "user_id",
			"dataGraphId": "dg-123",
			"workspaceId": "ws-456",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T12:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.GetRelationship(context.Background(), &datagraph.GetRelationshipRequest{
		DataGraphID:    "dg-123",
		RelationshipID: "rel-789",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Relationship{
		ID:            "rel-789",
		Name:          "User Orders",
		Cardinality:   "one-to-many",
		SourceModelID: "m-123",
		TargetModelID: "m-456",
		SourceJoinKey: "id",
		TargetJoinKey: "user_id",
		DataGraphID:   "dg-123",
		WorkspaceID:   "ws-456",
		CreatedAt:     &testTime1,
		UpdatedAt:     &testTime1,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestUpdateRelationship(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"Updated Orders","cardinality":"one-to-many","sourceModelId":"m-123","targetModelId":"m-456","sourceJoinKey":"user_id","targetJoinKey":"order_user_id"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/relationships/rel-789", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "rel-789",
			"name": "Updated Orders",
			"cardinality": "one-to-many",
			"sourceModelId": "m-123",
			"targetModelId": "m-456",
			"sourceJoinKey": "user_id",
			"targetJoinKey": "order_user_id",
			"dataGraphId": "dg-123",
			"workspaceId": "ws-456",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T13:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.UpdateRelationship(context.Background(), &datagraph.UpdateRelationshipRequest{
		DataGraphID:    "dg-123",
		RelationshipID: "rel-789",
		Name:           "Updated Orders",
		Cardinality:    "one-to-many",
		SourceModelID:  "m-123",
		TargetModelID:  "m-456",
		SourceJoinKey:  "user_id",
		TargetJoinKey:  "order_user_id",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Relationship{
		ID:            "rel-789",
		Name:          "Updated Orders",
		Cardinality:   "one-to-many",
		SourceModelID: "m-123",
		TargetModelID: "m-456",
		SourceJoinKey: "user_id",
		TargetJoinKey: "order_user_id",
		DataGraphID:   "dg-123",
		WorkspaceID:   "ws-456",
		CreatedAt:     &testTime1,
		UpdatedAt:     &testTime2,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestDeleteRelationship(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/v2/data-graphs/dg-123/relationships/rel-789", "")
		},
		ResponseStatus: 204,
	})

	store := newTestStore(t, httpClient)

	err := store.DeleteRelationship(context.Background(), &datagraph.DeleteRelationshipRequest{
		DataGraphID:    "dg-123",
		RelationshipID: "rel-789",
	})
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestListRelationships(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/relationships?page=1&pageSize=20", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "rel-789",
					"name": "User Orders",
					"cardinality": "one-to-many",
					"sourceModelId": "m-123",
					"targetModelId": "m-456",
					"sourceJoinKey": "id",
					"targetJoinKey": "user_id",
					"dataGraphId": "dg-123",
					"workspaceId": "ws-456",
					"createdAt": "2024-01-15T12:00:00Z",
					"updatedAt": "2024-01-15T12:00:00Z"
				}
			],
			"paging": {
				"total": 1
			}
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.ListRelationships(context.Background(), &datagraph.ListRelationshipsRequest{
		DataGraphID: "dg-123",
		Page:        1,
		PageSize:    20,
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.ListRelationshipsResponse{
		Data: []datagraph.Relationship{
			{
				ID:            "rel-789",
				Name:          "User Orders",
				Cardinality:   "one-to-many",
				SourceModelID: "m-123",
				TargetModelID: "m-456",
				SourceJoinKey: "id",
				TargetJoinKey: "user_id",
				DataGraphID:   "dg-123",
				WorkspaceID:   "ws-456",
				CreatedAt:     &testTime1,
				UpdatedAt:     &testTime1,
			},
		},
		Paging: client.Paging{
			Total: 1,
		},
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestListRelationshipsWithFilters(t *testing.T) {
	sourceModelID := "m-123"
	hasExternalID := true

	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/data-graphs/dg-123/relationships?hasExternalId=true&page=1&pageSize=20&sourceModelId=m-123", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "rel-789",
					"name": "User Orders",
					"cardinality": "one-to-many",
					"sourceModelId": "m-123",
					"targetModelId": "m-456",
					"sourceJoinKey": "id",
					"targetJoinKey": "user_id",
					"dataGraphId": "dg-123",
					"workspaceId": "ws-456",
					"externalId": "user-orders-rel",
					"createdAt": "2024-01-15T12:00:00Z",
					"updatedAt": "2024-01-15T12:00:00Z"
				}
			],
			"paging": {
				"total": 1
			}
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.ListRelationships(context.Background(), &datagraph.ListRelationshipsRequest{
		DataGraphID:   "dg-123",
		Page:          1,
		PageSize:      20,
		SourceModelID: &sourceModelID,
		HasExternalID: &hasExternalID,
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.ListRelationshipsResponse{
		Data: []datagraph.Relationship{
			{
				ID:            "rel-789",
				Name:          "User Orders",
				Cardinality:   "one-to-many",
				SourceModelID: "m-123",
				TargetModelID: "m-456",
				SourceJoinKey: "id",
				TargetJoinKey: "user_id",
				DataGraphID:   "dg-123",
				WorkspaceID:   "ws-456",
				ExternalID:    "user-orders-rel",
				CreatedAt:     &testTime1,
				UpdatedAt:     &testTime1,
			},
		},
		Paging: client.Paging{
			Total: 1,
		},
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestSetRelationshipExternalID(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"externalId":"user-orders-rel-v2"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/data-graphs/dg-123/relationships/rel-789/external-id", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "rel-789",
			"name": "User Orders",
			"cardinality": "one-to-many",
			"sourceModelId": "m-123",
			"targetModelId": "m-456",
			"sourceJoinKey": "id",
			"targetJoinKey": "user_id",
			"dataGraphId": "dg-123",
			"workspaceId": "ws-456",
			"externalId": "user-orders-rel-v2",
			"createdAt": "2024-01-15T12:00:00Z",
			"updatedAt": "2024-01-15T13:00:00Z"
		}`,
	})

	store := newTestStore(t, httpClient)

	result, err := store.SetRelationshipExternalID(context.Background(), &datagraph.SetRelationshipExternalIDRequest{
		DataGraphID:    "dg-123",
		RelationshipID: "rel-789",
		ExternalID:     "user-orders-rel-v2",
	})
	require.NoError(t, err)

	assert.Equal(t, &datagraph.Relationship{
		ID:            "rel-789",
		Name:          "User Orders",
		Cardinality:   "one-to-many",
		SourceModelID: "m-123",
		TargetModelID: "m-456",
		SourceJoinKey: "id",
		TargetJoinKey: "user_id",
		DataGraphID:   "dg-123",
		WorkspaceID:   "ws-456",
		ExternalID:    "user-orders-rel-v2",
		CreatedAt:     &testTime1,
		UpdatedAt:     &testTime2,
	}, result)

	httpClient.AssertNumberOfCalls()
}

func TestRelationshipErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		operation func(datagraph.DataGraphClient) error
		wantErr   string
	}{
		{
			name: "ListRelationships - empty data graph ID",
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.ListRelationships(context.Background(), &datagraph.ListRelationshipsRequest{
					DataGraphID: "",
				})
				return err
			},
			wantErr: "data graph ID cannot be empty",
		},
		{
			name: "GetRelationship - empty data graph ID",
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.GetRelationship(context.Background(), &datagraph.GetRelationshipRequest{
					DataGraphID:    "",
					RelationshipID: "rel-123",
				})
				return err
			},
			wantErr: "data graph ID cannot be empty",
		},
		{
			name: "GetRelationship - empty relationship ID",
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.GetRelationship(context.Background(), &datagraph.GetRelationshipRequest{
					DataGraphID:    "dg-123",
					RelationshipID: "",
				})
				return err
			},
			wantErr: "relationship ID cannot be empty",
		},
		{
			name: "CreateRelationship - empty data graph ID",
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.CreateRelationship(context.Background(), &datagraph.CreateRelationshipRequest{
					DataGraphID: "",
					Name:        "Test",
					Cardinality: "one-to-many",
				})
				return err
			},
			wantErr: "data graph ID cannot be empty",
		},
		{
			name: "CreateRelationship - empty cardinality",
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.CreateRelationship(context.Background(), &datagraph.CreateRelationshipRequest{
					DataGraphID: "dg-123",
					Name:        "Test",
					Cardinality: "",
				})
				return err
			},
			wantErr: "cardinality is required",
		},
		{
			name: "UpdateRelationship - empty data graph ID",
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.UpdateRelationship(context.Background(), &datagraph.UpdateRelationshipRequest{
					DataGraphID:    "",
					RelationshipID: "rel-123",
					Name:           "Test",
					Cardinality:    "one-to-many",
				})
				return err
			},
			wantErr: "data graph ID cannot be empty",
		},
		{
			name: "UpdateRelationship - empty relationship ID",
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.UpdateRelationship(context.Background(), &datagraph.UpdateRelationshipRequest{
					DataGraphID:    "dg-123",
					RelationshipID: "",
					Name:           "Test",
					Cardinality:    "one-to-many",
				})
				return err
			},
			wantErr: "relationship ID cannot be empty",
		},
		{
			name: "UpdateRelationship - empty cardinality",
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.UpdateRelationship(context.Background(), &datagraph.UpdateRelationshipRequest{
					DataGraphID:    "dg-123",
					RelationshipID: "rel-123",
					Name:           "Test",
					Cardinality:    "",
				})
				return err
			},
			wantErr: "cardinality is required",
		},
		{
			name: "DeleteRelationship - empty data graph ID",
			operation: func(store datagraph.DataGraphClient) error {
				return store.DeleteRelationship(context.Background(), &datagraph.DeleteRelationshipRequest{
					DataGraphID:    "",
					RelationshipID: "rel-123",
				})
			},
			wantErr: "data graph ID cannot be empty",
		},
		{
			name: "DeleteRelationship - empty relationship ID",
			operation: func(store datagraph.DataGraphClient) error {
				return store.DeleteRelationship(context.Background(), &datagraph.DeleteRelationshipRequest{
					DataGraphID:    "dg-123",
					RelationshipID: "",
				})
			},
			wantErr: "relationship ID cannot be empty",
		},
		{
			name: "SetRelationshipExternalID - empty data graph ID",
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.SetRelationshipExternalID(context.Background(), &datagraph.SetRelationshipExternalIDRequest{
					DataGraphID:    "",
					RelationshipID: "rel-123",
					ExternalID:     "ext-123",
				})
				return err
			},
			wantErr: "data graph ID cannot be empty",
		},
		{
			name: "SetRelationshipExternalID - empty relationship ID",
			operation: func(store datagraph.DataGraphClient) error {
				_, err := store.SetRelationshipExternalID(context.Background(), &datagraph.SetRelationshipExternalIDRequest{
					DataGraphID:    "dg-123",
					RelationshipID: "",
					ExternalID:     "ext-123",
				})
				return err
			},
			wantErr: "relationship ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := testutils.NewMockHTTPClient(t)
			store := newTestStore(t, httpClient)

			err := tt.operation(store)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
