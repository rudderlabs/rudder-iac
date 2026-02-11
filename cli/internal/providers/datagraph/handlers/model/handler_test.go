package model

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateModelResource(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	h := &HandlerImpl{client: mockClient}

	graph := resources.NewGraph()
	// Add a data graph node for dependency validation
	graph.AddResource(resources.NewResource(
		"my-dg",
		"data-graph",
		resources.ResourceData{},
		[]string{},
	))

	// Create URNs and PropertyRefs
	dataGraphURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
	dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)

	nonExistentURN := resources.URN("non-existent-dg", datagraph.HandlerMetadata.ResourceType)
	nonExistentRef := datagraph.CreateDataGraphReference(nonExistentURN)

	tests := []struct {
		name     string
		resource *model.ModelResource
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid entity model",
			resource: &model.ModelResource{
				ID:           "user",
				DisplayName:  "User",
				Type:         "entity",
				Table:        "users",
				DataGraphRef: dataGraphRef,
				PrimaryID:    "id",
				Root:         true,
			},
			wantErr: false,
		},
		{
			name: "valid event model",
			resource: &model.ModelResource{
				ID:           "purchase",
				DisplayName:  "Purchase",
				Type:         "event",
				Table:        "purchases",
				DataGraphRef: dataGraphRef,
				Timestamp:    "event_time",
			},
			wantErr: false,
		},
		{
			name: "missing display_name",
			resource: &model.ModelResource{
				ID:           "user",
				Type:         "entity",
				Table:        "users",
				DataGraphRef: dataGraphRef,
				PrimaryID:    "id",
			},
			wantErr: true,
			errMsg:  "display_name is required",
		},
		{
			name: "invalid type",
			resource: &model.ModelResource{
				ID:           "user",
				DisplayName:  "User",
				Type:         "invalid",
				Table:        "users",
				DataGraphRef: dataGraphRef,
			},
			wantErr: true,
			errMsg:  "type must be 'entity' or 'event'",
		},
		{
			name: "missing table",
			resource: &model.ModelResource{
				ID:           "user",
				DisplayName:  "User",
				Type:         "entity",
				DataGraphRef: dataGraphRef,
				PrimaryID:    "id",
			},
			wantErr: true,
			errMsg:  "table is required",
		},
		{
			name: "missing data_graph reference",
			resource: &model.ModelResource{
				ID:          "user",
				DisplayName: "User",
				Type:        "entity",
				Table:       "users",
				PrimaryID:   "id",
			},
			wantErr: true,
			errMsg:  "data_graph reference is required",
		},
		{
			name: "entity model missing primary_id",
			resource: &model.ModelResource{
				ID:           "user",
				DisplayName:  "User",
				Type:         "entity",
				Table:        "users",
				DataGraphRef: dataGraphRef,
			},
			wantErr: true,
			errMsg:  "primary_id is required for entity models",
		},
		{
			name: "event model missing timestamp",
			resource: &model.ModelResource{
				ID:           "purchase",
				DisplayName:  "Purchase",
				Type:         "event",
				Table:        "purchases",
				DataGraphRef: dataGraphRef,
			},
			wantErr: true,
			errMsg:  "timestamp is required for event models",
		},
		{
			name: "referenced data graph does not exist",
			resource: &model.ModelResource{
				ID:           "user",
				DisplayName:  "User",
				Type:         "entity",
				Table:        "users",
				DataGraphRef: nonExistentRef,
				PrimaryID:    "id",
			},
			wantErr: true,
			errMsg:  "referenced data graph data-graph:non-existent-dg does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := h.ValidateResource(tt.resource, graph)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// MapRemoteToState Tests
// ============================================================================

func TestMapRemoteToState(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	h := &HandlerImpl{client: mockClient}

	t.Run("entity model with external ID", func(t *testing.T) {
		// Create mock URN resolver
		urnResolver := testutils.NewMockURNResolver()
		urnResolver.AddMapping("data-graph", "dg-remote-1", "data-graph:my-dg")

		remote := &model.RemoteModel{
			Model: &dgClient.Model{
				ID:          "em-1",
				ExternalID:  "user",
				Name:        "User",
				Type:        "entity",
				TableRef:    "users",
				DataGraphID: "dg-remote-1",
				PrimaryID:   "id",
				Root:        true,
			},
		}

		resource, state, err := h.MapRemoteToState(remote, urnResolver)
		require.NoError(t, err)

		assert.Equal(t, "user", resource.ID)
		assert.Equal(t, "User", resource.DisplayName)
		assert.Equal(t, "entity", resource.Type)
		assert.Equal(t, "users", resource.Table)
		assert.Equal(t, "id", resource.PrimaryID)
		assert.True(t, resource.Root)

		require.NotNil(t, resource.DataGraphRef)
		assert.Equal(t, "data-graph:my-dg", resource.DataGraphRef.URN)

		expectedState := &model.ModelState{
			ID: "em-1",
		}
		assert.Equal(t, expectedState, state)
	})

	t.Run("event model with external ID", func(t *testing.T) {
		urnResolver := testutils.NewMockURNResolver()
		urnResolver.AddMapping("data-graph", "dg-remote-1", "data-graph:my-dg")

		remote := &model.RemoteModel{
			Model: &dgClient.Model{
				ID:          "evm-1",
				ExternalID:  "purchase",
				Name:        "Purchase",
				Type:        "event",
				TableRef:    "purchases",
				DataGraphID: "dg-remote-1",
				Timestamp:   "event_time",
			},
		}

		resource, state, err := h.MapRemoteToState(remote, urnResolver)
		require.NoError(t, err)

		assert.Equal(t, "purchase", resource.ID)
		assert.Equal(t, "Purchase", resource.DisplayName)
		assert.Equal(t, "event", resource.Type)
		assert.Equal(t, "purchases", resource.Table)
		assert.Equal(t, "event_time", resource.Timestamp)

		require.NotNil(t, resource.DataGraphRef)
		assert.Equal(t, "data-graph:my-dg", resource.DataGraphRef.URN)

		expectedState := &model.ModelState{
			ID: "evm-1",
		}
		assert.Equal(t, expectedState, state)
	})

	t.Run("model without external ID", func(t *testing.T) {
		remote := &model.RemoteModel{
			Model: &dgClient.Model{
				ID:          "em-2",
				Name:        "Account",
				Type:        "entity",
				TableRef:    "accounts",
				DataGraphID: "dg-remote-1",
				PrimaryID:   "account_id",
			},
		}

		resource, state, err := h.MapRemoteToState(remote, nil)
		require.NoError(t, err)
		assert.Nil(t, resource)
		assert.Nil(t, state)
	})
}

// ============================================================================
// Create Tests
// ============================================================================

func TestCreate(t *testing.T) {
	t.Run("create entity model", func(t *testing.T) {
		mockClient := &testutils.MockDataGraphClient{
			CreateModelFunc: func(ctx context.Context, req *dgClient.CreateModelRequest) (*dgClient.Model, error) {
				assert.Equal(t, "dg-remote-123", req.DataGraphID)
				assert.Equal(t, "entity", req.Type)
				assert.Equal(t, "User", req.Name)
				assert.Equal(t, "users", req.TableRef)
				assert.Equal(t, "user", req.ExternalID)
				assert.Equal(t, "id", req.PrimaryID)
				assert.True(t, req.Root)

				return &dgClient.Model{
					ID:         "em-456",
					Name:       req.Name,
					Type:       req.Type,
					TableRef:   req.TableRef,
					ExternalID: req.ExternalID,
					PrimaryID:  req.PrimaryID,
					Root:       req.Root,
				}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}

		dataGraphURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
		dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)
		dataGraphRef.IsResolved = true
		dataGraphRef.Value = "dg-remote-123"

		data := &model.ModelResource{
			ID:           "user",
			DisplayName:  "User",
			Type:         "entity",
			Table:        "users",
			DataGraphRef: dataGraphRef,
			PrimaryID:    "id",
			Root:         true,
		}

		state, err := h.Create(context.Background(), data)
		require.NoError(t, err)

		expectedState := &model.ModelState{
			ID: "em-456",
		}
		assert.Equal(t, expectedState, state)
	})

	t.Run("create event model", func(t *testing.T) {
		mockClient := &testutils.MockDataGraphClient{
			CreateModelFunc: func(ctx context.Context, req *dgClient.CreateModelRequest) (*dgClient.Model, error) {
				assert.Equal(t, "dg-remote-123", req.DataGraphID)
				assert.Equal(t, "event", req.Type)
				assert.Equal(t, "Purchase", req.Name)
				assert.Equal(t, "purchases", req.TableRef)
				assert.Equal(t, "purchase", req.ExternalID)
				assert.Equal(t, "event_time", req.Timestamp)

				return &dgClient.Model{
					ID:         "evm-789",
					Name:       req.Name,
					Type:       req.Type,
					TableRef:   req.TableRef,
					ExternalID: req.ExternalID,
					Timestamp:  req.Timestamp,
				}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}

		dataGraphURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
		dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)
		dataGraphRef.IsResolved = true
		dataGraphRef.Value = "dg-remote-123"

		data := &model.ModelResource{
			ID:           "purchase",
			DisplayName:  "Purchase",
			Type:         "event",
			Table:        "purchases",
			DataGraphRef: dataGraphRef,
			Timestamp:    "event_time",
		}

		state, err := h.Create(context.Background(), data)
		require.NoError(t, err)

		expectedState := &model.ModelState{
			ID: "evm-789",
		}
		assert.Equal(t, expectedState, state)
	})
}

// ============================================================================
// Update Tests
// ============================================================================

func TestUpdate(t *testing.T) {
	t.Run("update entity model", func(t *testing.T) {
		mockClient := &testutils.MockDataGraphClient{
			UpdateModelFunc: func(ctx context.Context, req *dgClient.UpdateModelRequest) (*dgClient.Model, error) {
				assert.Equal(t, "dg-remote-123", req.DataGraphID)
				assert.Equal(t, "em-456", req.ModelID)
				assert.Equal(t, "entity", req.Type)
				assert.Equal(t, "Updated User", req.Name)
				assert.Equal(t, "users_v2", req.TableRef)
				assert.Equal(t, "user_id", req.PrimaryID)
				assert.False(t, req.Root)

				return &dgClient.Model{
					ID:        req.ModelID,
					Name:      req.Name,
					Type:      req.Type,
					TableRef:  req.TableRef,
					PrimaryID: req.PrimaryID,
					Root:      req.Root,
				}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}

		dataGraphURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
		dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)
		dataGraphRef.IsResolved = true
		dataGraphRef.Value = "dg-remote-123"

		newData := &model.ModelResource{
			ID:           "user",
			DisplayName:  "Updated User",
			Type:         "entity",
			Table:        "users_v2",
			DataGraphRef: dataGraphRef,
			PrimaryID:    "user_id",
			Root:         false,
		}

		oldData := &model.ModelResource{
			ID:           "user",
			DisplayName:  "User",
			Type:         "entity",
			Table:        "users",
			DataGraphRef: dataGraphRef,
			PrimaryID:    "id",
			Root:         true,
		}

		oldState := &model.ModelState{
			ID: "em-456",
		}

		state, err := h.Update(context.Background(), newData, oldData, oldState)
		require.NoError(t, err)

		expectedState := &model.ModelState{
			ID: "em-456",
		}
		assert.Equal(t, expectedState, state)
	})

	t.Run("update event model", func(t *testing.T) {
		mockClient := &testutils.MockDataGraphClient{
			UpdateModelFunc: func(ctx context.Context, req *dgClient.UpdateModelRequest) (*dgClient.Model, error) {
				assert.Equal(t, "dg-remote-123", req.DataGraphID)
				assert.Equal(t, "evm-789", req.ModelID)
				assert.Equal(t, "event", req.Type)
				assert.Equal(t, "Updated Purchase", req.Name)
				assert.Equal(t, "purchases_v2", req.TableRef)
				assert.Equal(t, "ts", req.Timestamp)

				return &dgClient.Model{
					ID:        req.ModelID,
					Name:      req.Name,
					Type:      req.Type,
					TableRef:  req.TableRef,
					Timestamp: req.Timestamp,
				}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}

		dataGraphURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
		dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)
		dataGraphRef.IsResolved = true
		dataGraphRef.Value = "dg-remote-123"

		newData := &model.ModelResource{
			ID:           "purchase",
			DisplayName:  "Updated Purchase",
			Type:         "event",
			Table:        "purchases_v2",
			DataGraphRef: dataGraphRef,
			Timestamp:    "ts",
		}

		oldData := &model.ModelResource{
			ID:           "purchase",
			DisplayName:  "Purchase",
			Type:         "event",
			Table:        "purchases",
			DataGraphRef: dataGraphRef,
			Timestamp:    "event_time",
		}

		oldState := &model.ModelState{
			ID: "evm-789",
		}

		state, err := h.Update(context.Background(), newData, oldData, oldState)
		require.NoError(t, err)

		expectedState := &model.ModelState{
			ID: "evm-789",
		}
		assert.Equal(t, expectedState, state)
	})
}

// ============================================================================
// Import Tests
// ============================================================================

func TestImport(t *testing.T) {
	t.Run("import entity model", func(t *testing.T) {
		mockClient := &testutils.MockDataGraphClient{
			SetModelExternalIDFunc: func(ctx context.Context, req *dgClient.SetModelExternalIDRequest) (*dgClient.Model, error) {
				assert.Equal(t, "dg-123", req.DataGraphID)
				assert.Equal(t, "em-456", req.ModelID)
				assert.Equal(t, "user", req.ExternalID)
				return &dgClient.Model{
					ID:          req.ModelID,
					ExternalID:  req.ExternalID,
					Type:        "entity",
					DataGraphID: req.DataGraphID,
				}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}

		dataGraphURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
		dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)
		dataGraphRef.IsResolved = true
		dataGraphRef.Value = "dg-123"

		data := &model.ModelResource{
			ID:           "user",
			DisplayName:  "User",
			Type:         "entity",
			Table:        "users",
			DataGraphRef: dataGraphRef,
			PrimaryID:    "id",
			Root:         true,
		}

		state, err := h.Import(context.Background(), data, "em-456")
		require.NoError(t, err)

		expectedState := &model.ModelState{
			ID: "em-456",
		}
		assert.Equal(t, expectedState, state)
	})

	t.Run("import event model", func(t *testing.T) {
		mockClient := &testutils.MockDataGraphClient{
			SetModelExternalIDFunc: func(ctx context.Context, req *dgClient.SetModelExternalIDRequest) (*dgClient.Model, error) {
				assert.Equal(t, "dg-123", req.DataGraphID)
				assert.Equal(t, "evm-789", req.ModelID)
				assert.Equal(t, "purchase", req.ExternalID)
				return &dgClient.Model{
					ID:          req.ModelID,
					ExternalID:  req.ExternalID,
					Type:        "event",
					DataGraphID: req.DataGraphID,
				}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}

		dataGraphURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
		dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)
		dataGraphRef.IsResolved = true
		dataGraphRef.Value = "dg-123"

		data := &model.ModelResource{
			ID:           "purchase",
			DisplayName:  "Purchase",
			Type:         "event",
			Table:        "purchases",
			DataGraphRef: dataGraphRef,
			Timestamp:    "event_time",
		}

		state, err := h.Import(context.Background(), data, "evm-789")
		require.NoError(t, err)

		expectedState := &model.ModelState{
			ID: "evm-789",
		}
		assert.Equal(t, expectedState, state)
	})
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestDelete(t *testing.T) {
	t.Run("delete entity model", func(t *testing.T) {
		mockClient := &testutils.MockDataGraphClient{
			DeleteModelFunc: func(ctx context.Context, req *dgClient.DeleteModelRequest) error {
				assert.Equal(t, "dg-remote-123", req.DataGraphID)
				assert.Equal(t, "em-456", req.ModelID)
				return nil
			},
		}

		h := &HandlerImpl{client: mockClient}

		dataGraphURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
		dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)
		dataGraphRef.IsResolved = true
		dataGraphRef.Value = "dg-remote-123"

		oldData := &model.ModelResource{
			ID:           "user",
			DisplayName:  "User",
			Type:         "entity",
			Table:        "users",
			DataGraphRef: dataGraphRef,
			PrimaryID:    "id",
		}

		oldState := &model.ModelState{
			ID: "em-456",
		}

		err := h.Delete(context.Background(), "user", oldData, oldState)
		require.NoError(t, err)
	})

	t.Run("delete event model", func(t *testing.T) {
		mockClient := &testutils.MockDataGraphClient{
			DeleteModelFunc: func(ctx context.Context, req *dgClient.DeleteModelRequest) error {
				assert.Equal(t, "dg-remote-123", req.DataGraphID)
				assert.Equal(t, "evm-789", req.ModelID)
				return nil
			},
		}

		h := &HandlerImpl{client: mockClient}

		dataGraphURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
		dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)
		dataGraphRef.IsResolved = true
		dataGraphRef.Value = "dg-remote-123"

		oldData := &model.ModelResource{
			ID:           "purchase",
			DisplayName:  "Purchase",
			Type:         "event",
			Table:        "purchases",
			DataGraphRef: dataGraphRef,
			Timestamp:    "event_time",
		}

		oldState := &model.ModelState{
			ID: "evm-789",
		}

		err := h.Delete(context.Background(), "purchase", oldData, oldState)
		require.NoError(t, err)
	})
}

// ============================================================================
// Load Remote Resources Tests
// ============================================================================

func TestLoadRemoteResources(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		ListDataGraphsFunc: func(ctx context.Context, page, perPage int, hasExternalID *bool) (*dgClient.ListDataGraphsResponse, error) {
			require.NotNil(t, hasExternalID)
			assert.True(t, *hasExternalID)

			if page == 1 {
				return &dgClient.ListDataGraphsResponse{
					Data: []dgClient.DataGraph{
						{
							ID:         "dg-1",
							ExternalID: "my-dg",
						},
					},
					Paging: client.Paging{Next: ""},
				}, nil
			}
			return &dgClient.ListDataGraphsResponse{}, nil
		},
		ListModelsFunc: func(ctx context.Context, req *dgClient.ListModelsRequest) (*dgClient.ListModelsResponse, error) {
			assert.Equal(t, "dg-1", req.DataGraphID)
			require.NotNil(t, req.HasExternalID)
			assert.True(t, *req.HasExternalID)

			// Return both entity and event models based on type filter
			if req.ModelType != nil && *req.ModelType == "entity" {
				if req.Page == 1 {
					return &dgClient.ListModelsResponse{
						Data: []dgClient.Model{
							{
								ID:          "em-1",
								ExternalID:  "user",
								Name:        "User",
								Type:        "entity",
								TableRef:    "users",
								DataGraphID: "dg-1",
								PrimaryID:   "id",
								Root:        true,
							},
						},
						Paging: client.Paging{Next: ""},
					}, nil
				}
			} else if req.ModelType != nil && *req.ModelType == "event" {
				if req.Page == 1 {
					return &dgClient.ListModelsResponse{
						Data: []dgClient.Model{
							{
								ID:          "evm-1",
								ExternalID:  "purchase",
								Name:        "Purchase",
								Type:        "event",
								TableRef:    "purchases",
								DataGraphID: "dg-1",
								Timestamp:   "event_time",
							},
						},
						Paging: client.Paging{Next: ""},
					}, nil
				}
			}
			return &dgClient.ListModelsResponse{}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}
	remotes, err := h.LoadRemoteResources(context.Background())
	require.NoError(t, err)
	require.Len(t, remotes, 2)

	// Check entity model
	assert.Equal(t, "em-1", remotes[0].ID)
	assert.Equal(t, "user", remotes[0].ExternalID)
	assert.Equal(t, "entity", remotes[0].Type)
	assert.Equal(t, "dg-1", remotes[0].DataGraphID)

	// Check event model
	assert.Equal(t, "evm-1", remotes[1].ID)
	assert.Equal(t, "purchase", remotes[1].ExternalID)
	assert.Equal(t, "event", remotes[1].Type)
	assert.Equal(t, "dg-1", remotes[1].DataGraphID)
}

func TestLoadImportableResources(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		ListDataGraphsFunc: func(ctx context.Context, page, perPage int, hasExternalID *bool) (*dgClient.ListDataGraphsResponse, error) {
			if page == 1 {
				return &dgClient.ListDataGraphsResponse{
					Data: []dgClient.DataGraph{
						{
							ID:         "dg-1",
							ExternalID: "",
						},
					},
					Paging: client.Paging{Next: ""},
				}, nil
			}
			return &dgClient.ListDataGraphsResponse{}, nil
		},
		ListModelsFunc: func(ctx context.Context, req *dgClient.ListModelsRequest) (*dgClient.ListModelsResponse, error) {
			assert.Equal(t, "dg-1", req.DataGraphID)
			require.NotNil(t, req.HasExternalID)
			assert.False(t, *req.HasExternalID)

			// Return both entity and event models based on type filter
			if req.ModelType != nil && *req.ModelType == "entity" {
				if req.Page == 1 {
					return &dgClient.ListModelsResponse{
						Data: []dgClient.Model{
							{
								ID:          "em-2",
								Name:        "Account",
								Type:        "entity",
								TableRef:    "accounts",
								DataGraphID: "dg-1",
								PrimaryID:   "account_id",
							},
						},
						Paging: client.Paging{Next: ""},
					}, nil
				}
			} else if req.ModelType != nil && *req.ModelType == "event" {
				if req.Page == 1 {
					return &dgClient.ListModelsResponse{
						Data: []dgClient.Model{
							{
								ID:          "evm-2",
								Name:        "PageView",
								Type:        "event",
								TableRef:    "page_views",
								DataGraphID: "dg-1",
								Timestamp:   "ts",
							},
						},
						Paging: client.Paging{Next: ""},
					}, nil
				}
			}
			return &dgClient.ListModelsResponse{}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}
	remotes, err := h.LoadImportableResources(context.Background())
	require.NoError(t, err)
	require.Len(t, remotes, 2)

	// Check entity model
	assert.Equal(t, "em-2", remotes[0].ID)
	assert.Equal(t, "", remotes[0].ExternalID)
	assert.Equal(t, "entity", remotes[0].Type)
	assert.Equal(t, "dg-1", remotes[0].DataGraphID)

	// Check event model
	assert.Equal(t, "evm-2", remotes[1].ID)
	assert.Equal(t, "", remotes[1].ExternalID)
	assert.Equal(t, "event", remotes[1].Type)
	assert.Equal(t, "dg-1", remotes[1].DataGraphID)
}
