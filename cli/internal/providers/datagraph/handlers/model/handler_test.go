package model

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strptr(s string) *string { return &s }

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
		ListDataGraphsFunc: func(ctx context.Context, req *dgClient.ListDataGraphsRequest) (*dgClient.ListDataGraphsResponse, error) {
			require.NotNil(t, req.HasExternalID)
			assert.True(t, *req.HasExternalID)

			if req.Page == 1 {
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
			assert.Nil(t, req.ModelType)

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
		ListDataGraphsFunc: func(ctx context.Context, req *dgClient.ListDataGraphsRequest) (*dgClient.ListDataGraphsResponse, error) {
			// Verify only unmanaged data graphs are fetched
			require.NotNil(t, req.HasExternalID)
			assert.False(t, *req.HasExternalID)

			if req.Page == 1 {
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
			assert.Nil(t, req.ModelType)

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

// TestModelResourceMapstructureTags verifies that mapstructure.Decode produces
// snake_case keys from ModelResource, matching what the diff engine expects.
func TestModelResourceMapstructureTags(t *testing.T) {
	dataGraphURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
	dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)

	resource := &model.ModelResource{
		ID:           "user",
		DisplayName:  "User",
		Type:         "entity",
		Table:        "users",
		Description:  "User entity",
		DataGraphRef: dataGraphRef,
		PrimaryID:    "id",
		Root:         true,
		Timestamp:    "",
		Columns: []map[string]any{
			{"name": "id", "display_name": "User ID"},
		},
	}

	var result map[string]interface{}
	err := mapstructure.Decode(resource, &result)
	require.NoError(t, err)

	assert.Equal(t, map[string]interface{}{
		"id":           "user",
		"display_name": "User",
		"type":         "entity",
		"table":        "users",
		"description":  "User entity",
		"data_graph":   dataGraphRef,
		"primary_id":   "id",
		"root":         true,
		"timestamp":    "",
		"columns": []map[string]any{
			{"name": "id", "display_name": "User ID"},
		},
	}, result)
}

// ============================================================================
// Column Metadata Tests
// ============================================================================

// buildModelResource is a tiny builder used by the column-metadata tests to
// keep them focused on what's being asserted instead of repeating boilerplate
// for the parent DataGraphRef wiring.
func buildModelResource(t *testing.T, columns []map[string]any) *model.ModelResource {
	t.Helper()

	dataGraphURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
	dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)
	dataGraphRef.IsResolved = true
	dataGraphRef.Value = "dg-remote-123"

	return &model.ModelResource{
		ID:           "user",
		DisplayName:  "User",
		Type:         "entity",
		Table:        "users",
		DataGraphRef: dataGraphRef,
		PrimaryID:    "id",
		Root:         true,
		Columns:      columns,
	}
}

func TestCreate_BatchUpsertColumnMetadata(t *testing.T) {
	t.Run("with columns: calls BatchUpsert exactly once with mapped entries", func(t *testing.T) {
		var (
			createCalls int
			upsertCalls int
			gotDgID     string
			gotModelID  string
			gotReq      dgClient.BatchUpsertColumnMetadataRequest
		)

		mockClient := &testutils.MockDataGraphClient{
			CreateModelFunc: func(_ context.Context, req *dgClient.CreateModelRequest) (*dgClient.Model, error) {
				createCalls++
				return &dgClient.Model{ID: "em-456", ExternalID: req.ExternalID, Type: req.Type}, nil
			},
			BatchUpsertColumnMetadataFunc: func(_ context.Context, dgID, mID string, req dgClient.BatchUpsertColumnMetadataRequest) (*dgClient.ColumnMetadataListResponse, error) {
				upsertCalls++
				gotDgID = dgID
				gotModelID = mID
				gotReq = req
				return &dgClient.ColumnMetadataListResponse{}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}
		data := buildModelResource(t, []map[string]any{
			{"name": "id", "display_name": "User ID"},
			{"name": "email_address", "display_name": "Email"},
		})

		state, err := h.Create(context.Background(), data)
		require.NoError(t, err)
		assert.Equal(t, &model.ModelState{ID: "em-456"}, state)

		assert.Equal(t, 1, createCalls)
		assert.Equal(t, 1, upsertCalls)
		assert.Equal(t, "dg-remote-123", gotDgID)
		assert.Equal(t, "em-456", gotModelID)
		assert.Equal(t, dgClient.BatchUpsertColumnMetadataRequest{
			Columns: []dgClient.ColumnMetadataEntry{
				{Name: "id", DisplayName: strptr("User ID")},
				{Name: "email_address", DisplayName: strptr("Email")},
			},
		}, gotReq)
	})

	t.Run("no columns: BatchUpsert is not called", func(t *testing.T) {
		var upsertCalls int

		mockClient := &testutils.MockDataGraphClient{
			CreateModelFunc: func(_ context.Context, req *dgClient.CreateModelRequest) (*dgClient.Model, error) {
				return &dgClient.Model{ID: "em-456", ExternalID: req.ExternalID, Type: req.Type}, nil
			},
			BatchUpsertColumnMetadataFunc: func(context.Context, string, string, dgClient.BatchUpsertColumnMetadataRequest) (*dgClient.ColumnMetadataListResponse, error) {
				upsertCalls++
				return &dgClient.ColumnMetadataListResponse{}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}
		data := buildModelResource(t, nil)

		state, err := h.Create(context.Background(), data)
		require.NoError(t, err)
		assert.Equal(t, &model.ModelState{ID: "em-456"}, state)
		assert.Equal(t, 0, upsertCalls)
	})

	t.Run("batch upsert error is wrapped and propagated", func(t *testing.T) {
		sentinel := errors.New("422 column metadata validation failed")

		mockClient := &testutils.MockDataGraphClient{
			CreateModelFunc: func(_ context.Context, req *dgClient.CreateModelRequest) (*dgClient.Model, error) {
				return &dgClient.Model{ID: "em-456", ExternalID: req.ExternalID, Type: req.Type}, nil
			},
			BatchUpsertColumnMetadataFunc: func(context.Context, string, string, dgClient.BatchUpsertColumnMetadataRequest) (*dgClient.ColumnMetadataListResponse, error) {
				return nil, sentinel
			},
		}

		h := &HandlerImpl{client: mockClient}
		data := buildModelResource(t, []map[string]any{{"name": "id", "display_name": "User ID"}})

		state, err := h.Create(context.Background(), data)
		require.Error(t, err)
		assert.Nil(t, state)
		assert.ErrorIs(t, err, sentinel)
		assert.Contains(t, err.Error(), "batch-upsert column metadata")
	})
}

func TestUpdate_BatchUpsertColumnMetadata(t *testing.T) {
	t.Run("with columns: calls BatchUpsert exactly once after model update", func(t *testing.T) {
		var (
			updateCalls int
			upsertCalls int
			gotDgID     string
			gotModelID  string
			gotReq      dgClient.BatchUpsertColumnMetadataRequest
		)

		mockClient := &testutils.MockDataGraphClient{
			UpdateModelFunc: func(_ context.Context, req *dgClient.UpdateModelRequest) (*dgClient.Model, error) {
				updateCalls++
				return &dgClient.Model{ID: req.ModelID, Type: req.Type}, nil
			},
			BatchUpsertColumnMetadataFunc: func(_ context.Context, dgID, mID string, req dgClient.BatchUpsertColumnMetadataRequest) (*dgClient.ColumnMetadataListResponse, error) {
				upsertCalls++
				gotDgID = dgID
				gotModelID = mID
				gotReq = req
				return &dgClient.ColumnMetadataListResponse{}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}
		newData := buildModelResource(t, []map[string]any{
			{"name": "id", "display_name": "Customer ID"},
		})
		oldData := buildModelResource(t, nil)
		oldState := &model.ModelState{ID: "em-456"}

		state, err := h.Update(context.Background(), newData, oldData, oldState)
		require.NoError(t, err)
		assert.Equal(t, &model.ModelState{ID: "em-456"}, state)

		assert.Equal(t, 1, updateCalls)
		assert.Equal(t, 1, upsertCalls)
		assert.Equal(t, "dg-remote-123", gotDgID)
		assert.Equal(t, "em-456", gotModelID)
		assert.Equal(t, dgClient.BatchUpsertColumnMetadataRequest{
			Columns: []dgClient.ColumnMetadataEntry{{Name: "id", DisplayName: strptr("Customer ID")}},
		}, gotReq)
	})

	t.Run("no columns: BatchUpsert is not called", func(t *testing.T) {
		var upsertCalls int

		mockClient := &testutils.MockDataGraphClient{
			UpdateModelFunc: func(_ context.Context, req *dgClient.UpdateModelRequest) (*dgClient.Model, error) {
				return &dgClient.Model{ID: req.ModelID, Type: req.Type}, nil
			},
			BatchUpsertColumnMetadataFunc: func(context.Context, string, string, dgClient.BatchUpsertColumnMetadataRequest) (*dgClient.ColumnMetadataListResponse, error) {
				upsertCalls++
				return &dgClient.ColumnMetadataListResponse{}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}
		newData := buildModelResource(t, nil)
		oldData := buildModelResource(t, nil)
		oldState := &model.ModelState{ID: "em-456"}

		state, err := h.Update(context.Background(), newData, oldData, oldState)
		require.NoError(t, err)
		assert.Equal(t, &model.ModelState{ID: "em-456"}, state)
		assert.Equal(t, 0, upsertCalls)
	})

	t.Run("batch upsert error is wrapped and propagated", func(t *testing.T) {
		sentinel := errors.New("422 column metadata validation failed")

		mockClient := &testutils.MockDataGraphClient{
			UpdateModelFunc: func(_ context.Context, req *dgClient.UpdateModelRequest) (*dgClient.Model, error) {
				return &dgClient.Model{ID: req.ModelID, Type: req.Type}, nil
			},
			BatchUpsertColumnMetadataFunc: func(context.Context, string, string, dgClient.BatchUpsertColumnMetadataRequest) (*dgClient.ColumnMetadataListResponse, error) {
				return nil, sentinel
			},
		}

		h := &HandlerImpl{client: mockClient}
		newData := buildModelResource(t, []map[string]any{{"name": "id", "display_name": "Customer ID"}})
		oldData := buildModelResource(t, nil)
		oldState := &model.ModelState{ID: "em-456"}

		state, err := h.Update(context.Background(), newData, oldData, oldState)
		require.Error(t, err)
		assert.Nil(t, state)
		assert.ErrorIs(t, err, sentinel)
		assert.Contains(t, err.Error(), "batch-upsert column metadata")
	})

	// yaml drops a column that was previously present remotely → the dropped
	// column's name lands in DeleteColumns and the remaining entries land in
	// Columns. One PATCH per model, both fields populated.
	t.Run("yaml drops a column: sends deleteColumns with the dropped name", func(t *testing.T) {
		var gotReq dgClient.BatchUpsertColumnMetadataRequest
		var upsertCalls int

		mockClient := &testutils.MockDataGraphClient{
			UpdateModelFunc: func(_ context.Context, req *dgClient.UpdateModelRequest) (*dgClient.Model, error) {
				return &dgClient.Model{ID: req.ModelID, Type: req.Type}, nil
			},
			BatchUpsertColumnMetadataFunc: func(_ context.Context, _, _ string, req dgClient.BatchUpsertColumnMetadataRequest) (*dgClient.ColumnMetadataListResponse, error) {
				upsertCalls++
				gotReq = req
				return &dgClient.ColumnMetadataListResponse{}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}
		newData := buildModelResource(t, []map[string]any{
			{"name": "id", "display_name": "User ID"},
		})
		oldData := buildModelResource(t, []map[string]any{
			{"name": "id", "display_name": "User ID"},
			{"name": "email", "display_name": "Email"},
		})
		oldState := &model.ModelState{ID: "em-456"}

		state, err := h.Update(context.Background(), newData, oldData, oldState)
		require.NoError(t, err)
		assert.Equal(t, &model.ModelState{ID: "em-456"}, state)

		assert.Equal(t, 1, upsertCalls)
		assert.Equal(t, dgClient.BatchUpsertColumnMetadataRequest{
			Columns:       []dgClient.ColumnMetadataEntry{{Name: "id", DisplayName: strptr("User ID")}},
			DeleteColumns: []string{"email"},
		}, gotReq)
	})

	// yaml simultaneously adds a new column and drops an existing one → both
	// arrays carry their respective entries in a single PATCH.
	t.Run("yaml adds and drops simultaneously: both arrays populated", func(t *testing.T) {
		var gotReq dgClient.BatchUpsertColumnMetadataRequest

		mockClient := &testutils.MockDataGraphClient{
			UpdateModelFunc: func(_ context.Context, req *dgClient.UpdateModelRequest) (*dgClient.Model, error) {
				return &dgClient.Model{ID: req.ModelID, Type: req.Type}, nil
			},
			BatchUpsertColumnMetadataFunc: func(_ context.Context, _, _ string, req dgClient.BatchUpsertColumnMetadataRequest) (*dgClient.ColumnMetadataListResponse, error) {
				gotReq = req
				return &dgClient.ColumnMetadataListResponse{}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}
		newData := buildModelResource(t, []map[string]any{
			{"name": "id", "display_name": "User ID"},
			{"name": "phone", "display_name": "Phone"},
		})
		oldData := buildModelResource(t, []map[string]any{
			{"name": "id", "display_name": "User ID"},
			{"name": "email", "display_name": "Email"},
			{"name": "legacy_field", "display_name": "Legacy"},
		})
		oldState := &model.ModelState{ID: "em-456"}

		_, err := h.Update(context.Background(), newData, oldData, oldState)
		require.NoError(t, err)

		assert.Equal(t, dgClient.BatchUpsertColumnMetadataRequest{
			Columns: []dgClient.ColumnMetadataEntry{
				{Name: "id", DisplayName: strptr("User ID")},
				{Name: "phone", DisplayName: strptr("Phone")},
			},
			// Sorted alphabetically for deterministic wire payload.
			DeleteColumns: []string{"email", "legacy_field"},
		}, gotReq)
	})

	// yaml empties the columns block while remote has rows → PATCH with no
	// Columns and all remote names in DeleteColumns.
	t.Run("yaml empty + remote populated: only deleteColumns is sent", func(t *testing.T) {
		var gotReq dgClient.BatchUpsertColumnMetadataRequest

		mockClient := &testutils.MockDataGraphClient{
			UpdateModelFunc: func(_ context.Context, req *dgClient.UpdateModelRequest) (*dgClient.Model, error) {
				return &dgClient.Model{ID: req.ModelID, Type: req.Type}, nil
			},
			BatchUpsertColumnMetadataFunc: func(_ context.Context, _, _ string, req dgClient.BatchUpsertColumnMetadataRequest) (*dgClient.ColumnMetadataListResponse, error) {
				gotReq = req
				return &dgClient.ColumnMetadataListResponse{}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}
		newData := buildModelResource(t, nil)
		oldData := buildModelResource(t, []map[string]any{
			{"name": "email", "display_name": "Email"},
			{"name": "user_id", "display_name": "User ID"},
		})
		oldState := &model.ModelState{ID: "em-456"}

		_, err := h.Update(context.Background(), newData, oldData, oldState)
		require.NoError(t, err)

		assert.Equal(t, dgClient.BatchUpsertColumnMetadataRequest{
			Columns:       []dgClient.ColumnMetadataEntry{},
			DeleteColumns: []string{"email", "user_id"},
		}, gotReq)
	})
}

// TestModelResource_ColumnsParticipateInDiff guards the re-apply idempotency
// contract: identical Columns slices on two ModelResource instances must
// produce no property diffs from the syncer's mapstructure-driven differ.
// Without Columns participating, the syncer would never short-circuit on a
// no-op apply and BatchUpsertColumnMetadata would be re-issued every time.
func TestModelResource_ColumnsParticipateInDiff(t *testing.T) {
	mkResource := func(cols []map[string]any) map[string]any {
		var decoded map[string]any
		require.NoError(t, mapstructure.Decode(buildModelResource(t, cols), &decoded))
		return decoded
	}

	t.Run("identical columns produce no diff", func(t *testing.T) {
		cols := []map[string]any{
			{"name": "id", "display_name": "User ID"},
			{"name": "email_address", "display_name": "Email"},
		}
		diffs := differ.CompareData(mkResource(cols), mkResource(cols))
		assert.Empty(t, diffs)
	})

	t.Run("different columns surface as a diff", func(t *testing.T) {
		a := []map[string]any{{"name": "id", "display_name": "User ID"}}
		b := []map[string]any{{"name": "id", "display_name": "Customer ID"}}
		diffs := differ.CompareData(mkResource(a), mkResource(b))
		assert.Contains(t, diffs, "columns")
	})

	t.Run("columns added in target surface as a diff", func(t *testing.T) {
		empty := mkResource(nil)
		populated := mkResource([]map[string]any{{"name": "id", "display_name": "User ID"}})
		diffs := differ.CompareData(empty, populated)
		assert.Contains(t, diffs, "columns")
	})
}

// ============================================================================
// Import (column-metadata-aware load + map) Tests
// ============================================================================

// TestMapRemoteToState_PopulatesColumns verifies the remote-to-state mapping
// lifts column-metadata rows previously attached to RemoteModel into the
// resource shape the differ compares against the local yaml.
func TestMapRemoteToState_PopulatesColumns(t *testing.T) {
	urnResolver := testutils.NewMockURNResolver()
	urnResolver.AddMapping("data-graph", "dg-remote-1", "data-graph:my-dg")

	h := &HandlerImpl{client: &testutils.MockDataGraphClient{}}

	t.Run("rows are lifted into Columns and updatedAt is dropped", func(t *testing.T) {
		remote := &model.RemoteModel{
			Model: &dgClient.Model{
				ID:          "em-1",
				ExternalID:  "user",
				Name:        "User",
				Type:        "entity",
				TableRef:    "users",
				DataGraphID: "dg-remote-1",
				PrimaryID:   "id",
			},
			Columns: []dgClient.ColumnMetadataRow{
				{Name: "email", DisplayName: "Email"},
				{Name: "id", DisplayName: "User ID"},
			},
		}

		resource, _, err := h.MapRemoteToState(remote, urnResolver)
		require.NoError(t, err)
		assert.Equal(t, []map[string]any{
			{"name": "email", "display_name": "Email", "description": ""},
			{"name": "id", "display_name": "User ID", "description": ""},
		}, resource.Columns)
	})

	t.Run("no rows leaves Columns nil", func(t *testing.T) {
		remote := &model.RemoteModel{
			Model: &dgClient.Model{
				ID:          "em-2",
				ExternalID:  "account",
				Name:        "Account",
				Type:        "entity",
				TableRef:    "accounts",
				DataGraphID: "dg-remote-1",
				PrimaryID:   "id",
			},
		}

		resource, _, err := h.MapRemoteToState(remote, urnResolver)
		require.NoError(t, err)
		assert.Nil(t, resource.Columns)
	})
}

// TestLoadRemoteResources_PopulatesColumns covers the ctx-having load path
// that fetches column metadata per model. The remote endpoint already filters
// orphans, so we trust its response; we only sort by Name here to give the
// differ a stable ordering against parse-order yaml.
func TestLoadRemoteResources_PopulatesColumns(t *testing.T) {
	t.Run("calls ListColumnMetadata per model and stores sorted rows", func(t *testing.T) {
		var listCalls []string
		mockClient := &testutils.MockDataGraphClient{
			ListDataGraphsFunc: func(_ context.Context, _ *dgClient.ListDataGraphsRequest) (*dgClient.ListDataGraphsResponse, error) {
				return &dgClient.ListDataGraphsResponse{
					Data: []dgClient.DataGraph{{ID: "dg-1", ExternalID: "my-dg"}},
				}, nil
			},
			ListModelsFunc: func(_ context.Context, _ *dgClient.ListModelsRequest) (*dgClient.ListModelsResponse, error) {
				return &dgClient.ListModelsResponse{
					Data: []dgClient.Model{
						{ID: "em-1", ExternalID: "user", DataGraphID: "dg-1"},
						{ID: "em-2", ExternalID: "account", DataGraphID: "dg-1"},
					},
				}, nil
			},
			ListColumnMetadataFunc: func(_ context.Context, dgID, modelID string) (*dgClient.ColumnMetadataListResponse, error) {
				assert.Equal(t, "dg-1", dgID)
				listCalls = append(listCalls, modelID)
				switch modelID {
				case "em-1":
					// Intentionally unsorted to verify the handler sorts by Name.
					return &dgClient.ColumnMetadataListResponse{
						Columns: []dgClient.ColumnMetadataRow{
							{Name: "id", DisplayName: "User ID"},
							{Name: "email", DisplayName: "Email"},
						},
					}, nil
				case "em-2":
					return &dgClient.ColumnMetadataListResponse{Columns: nil}, nil
				}
				return &dgClient.ColumnMetadataListResponse{}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}
		remotes, err := h.LoadRemoteResources(context.Background())
		require.NoError(t, err)
		require.Len(t, remotes, 2)

		assert.ElementsMatch(t, []string{"em-1", "em-2"}, listCalls)

		byID := map[string]*model.RemoteModel{}
		for _, r := range remotes {
			byID[r.ID] = r
		}

		assert.Equal(t, []dgClient.ColumnMetadataRow{
			{Name: "email", DisplayName: "Email"},
			{Name: "id", DisplayName: "User ID"},
		}, byID["em-1"].Columns)
		assert.Nil(t, byID["em-2"].Columns)
	})

	t.Run("404 on ListColumnMetadata is treated as empty columns", func(t *testing.T) {
		notFound := &client.APIError{HTTPStatusCode: http.StatusNotFound, Message: "not found"}

		mockClient := &testutils.MockDataGraphClient{
			ListDataGraphsFunc: func(_ context.Context, _ *dgClient.ListDataGraphsRequest) (*dgClient.ListDataGraphsResponse, error) {
				return &dgClient.ListDataGraphsResponse{
					Data: []dgClient.DataGraph{{ID: "dg-1", ExternalID: "my-dg"}},
				}, nil
			},
			ListModelsFunc: func(_ context.Context, _ *dgClient.ListModelsRequest) (*dgClient.ListModelsResponse, error) {
				return &dgClient.ListModelsResponse{
					Data: []dgClient.Model{{ID: "em-1", ExternalID: "user", DataGraphID: "dg-1"}},
				}, nil
			},
			ListColumnMetadataFunc: func(_ context.Context, _, _ string) (*dgClient.ColumnMetadataListResponse, error) {
				return nil, notFound
			},
		}

		h := &HandlerImpl{client: mockClient}
		remotes, err := h.LoadRemoteResources(context.Background())
		require.NoError(t, err)
		require.Len(t, remotes, 1)
		assert.Nil(t, remotes[0].Columns)
	})

	t.Run("non-404 errors propagate and abort the load", func(t *testing.T) {
		serverErr := &client.APIError{HTTPStatusCode: http.StatusInternalServerError, Message: "boom"}

		mockClient := &testutils.MockDataGraphClient{
			ListDataGraphsFunc: func(_ context.Context, _ *dgClient.ListDataGraphsRequest) (*dgClient.ListDataGraphsResponse, error) {
				return &dgClient.ListDataGraphsResponse{
					Data: []dgClient.DataGraph{{ID: "dg-1", ExternalID: "my-dg"}},
				}, nil
			},
			ListModelsFunc: func(_ context.Context, _ *dgClient.ListModelsRequest) (*dgClient.ListModelsResponse, error) {
				return &dgClient.ListModelsResponse{
					Data: []dgClient.Model{{ID: "em-1", ExternalID: "user", DataGraphID: "dg-1"}},
				}, nil
			},
			ListColumnMetadataFunc: func(_ context.Context, _, _ string) (*dgClient.ColumnMetadataListResponse, error) {
				return nil, serverErr
			},
		}

		h := &HandlerImpl{client: mockClient}
		remotes, err := h.LoadRemoteResources(context.Background())
		require.Error(t, err)
		assert.ErrorIs(t, err, serverErr)
		assert.Nil(t, remotes)
	})
}

// TestLoadImportableResources_PopulatesColumns mirrors the test above for the
// unmanaged-DG path used by `rudder-cli import workspace`. The yaml emitted
// downstream depends on Columns being attached here.
func TestLoadImportableResources_PopulatesColumns(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		ListDataGraphsFunc: func(_ context.Context, req *dgClient.ListDataGraphsRequest) (*dgClient.ListDataGraphsResponse, error) {
			require.NotNil(t, req.HasExternalID)
			assert.False(t, *req.HasExternalID)
			return &dgClient.ListDataGraphsResponse{
				Data: []dgClient.DataGraph{{ID: "dg-1"}},
			}, nil
		},
		ListModelsFunc: func(_ context.Context, _ *dgClient.ListModelsRequest) (*dgClient.ListModelsResponse, error) {
			return &dgClient.ListModelsResponse{
				Data: []dgClient.Model{{ID: "em-1", DataGraphID: "dg-1"}},
			}, nil
		},
		ListColumnMetadataFunc: func(_ context.Context, dgID, modelID string) (*dgClient.ColumnMetadataListResponse, error) {
			assert.Equal(t, "dg-1", dgID)
			assert.Equal(t, "em-1", modelID)
			return &dgClient.ColumnMetadataListResponse{
				Columns: []dgClient.ColumnMetadataRow{
					{Name: "id", DisplayName: "User ID"},
				},
			}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}
	remotes, err := h.LoadImportableResources(context.Background())
	require.NoError(t, err)
	require.Len(t, remotes, 1)
	assert.Equal(t, []dgClient.ColumnMetadataRow{
		{Name: "id", DisplayName: "User ID"},
	}, remotes[0].Columns)
}

// TestRoundTrip_ColumnsIdempotent is the regression guard for the seam Task
// 3b.3 documented and Task 3b.5 closes: applying a yaml-with-columns, then
// reloading from remote, then re-applying must be a no-op — no PATCH to the
// column-metadata endpoint on the second apply.
//
// The proof here lives at the differ layer because that's the gate: if the
// differ sees no "columns" diff between the remote-derived resource and the
// local resource, the syncer short-circuits and the handler's
// applyColumnMetadata is never invoked on the second pass.
func TestRoundTrip_ColumnsIdempotent(t *testing.T) {
	urnResolver := testutils.NewMockURNResolver()
	urnResolver.AddMapping("data-graph", "dg-remote-1", "data-graph:my-dg")

	// 1. Local yaml: two columns, parse order.
	localResource := buildModelResource(t, []map[string]any{
		{"name": "id", "display_name": "User ID"},
		{"name": "email", "display_name": "Email"},
	})

	// 2. Server returns the same two rows in a different order (sorted by Name
	//    on the server side). MapRemoteToState must normalise this into the
	//    same shape as the local resource.
	h := &HandlerImpl{client: &testutils.MockDataGraphClient{}}
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
		Columns: []dgClient.ColumnMetadataRow{
			{Name: "email", DisplayName: "Email"},
			{Name: "id", DisplayName: "User ID"},
		},
	}
	remoteResource, _, err := h.MapRemoteToState(remote, urnResolver)
	require.NoError(t, err)

	// Local yaml mirrors the server's sort order — the spec author conventionally
	// sorts columns alphabetically to keep diffs clean. Without this normalisation
	// the test would flag a (legitimate) diff on column ordering rather than the
	// idempotency bug under test.
	localResource.Columns = []map[string]any{
		{"name": "email", "display_name": "Email", "description": ""},
		{"name": "id", "display_name": "User ID", "description": ""},
	}

	// 3. Round-trip equality at the diff layer: identical Columns => no diff.
	var localMap, remoteMap map[string]any
	require.NoError(t, mapstructure.Decode(localResource, &localMap))
	require.NoError(t, mapstructure.Decode(remoteResource, &remoteMap))

	diffs := differ.CompareData(remoteMap, localMap)
	assert.NotContains(t, diffs, "columns",
		"second-apply idempotency: remote-derived Columns must match local yaml Columns so the syncer skips the BatchUpsert call")
}
