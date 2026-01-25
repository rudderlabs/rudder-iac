package model

import (
	"context"
	"testing"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEntityModelResource(t *testing.T) {
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
			name: "valid entity model resource",
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

func TestMapRemoteEntityModelToState(t *testing.T) {
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
				DataGraphID: "dg-remote-1", // Remote ID from API
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

		// Verify DataGraphRef is created correctly
		require.NotNil(t, resource.DataGraphRef)
		assert.Equal(t, "data-graph:my-dg", resource.DataGraphRef.URN)

		expectedState := &model.ModelState{
			ID: "em-1",
		}
		assert.Equal(t, expectedState, state)
	})

	t.Run("entity model without external ID", func(t *testing.T) {
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

func TestCreateEntityModel(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		CreateEntityModelFunc: func(ctx context.Context, dataGraphID string, req *dgClient.CreateEntityModelRequest) (*dgClient.Model, error) {
			assert.Equal(t, "dg-remote-123", dataGraphID)
			assert.Equal(t, "User", req.Name)
			assert.Equal(t, "users", req.TableRef)
			assert.Equal(t, "user", req.ExternalID)
			assert.Equal(t, "id", req.PrimaryID)
			assert.True(t, req.Root)

			return &dgClient.Model{
				ID:         "em-456",
				Name:       req.Name,
				Type:       "entity",
				TableRef:   req.TableRef,
				ExternalID: req.ExternalID,
				PrimaryID:  req.PrimaryID,
				Root:       req.Root,
			}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}

	// Create a resolved PropertyRef
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
}

func TestUpdateEntityModel(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		UpdateEntityModelFunc: func(ctx context.Context, dataGraphID, modelID string, req *dgClient.UpdateEntityModelRequest) (*dgClient.Model, error) {
			assert.Equal(t, "dg-remote-123", dataGraphID)
			assert.Equal(t, "em-456", modelID)
			assert.Equal(t, "Updated User", req.Name)
			assert.Equal(t, "users_v2", req.TableRef)
			assert.Equal(t, "user_id", req.PrimaryID)
			assert.False(t, req.Root)

			return &dgClient.Model{
				ID:        modelID,
				Name:      req.Name,
				Type:      "entity",
				TableRef:  req.TableRef,
				PrimaryID: req.PrimaryID,
				Root:      req.Root,
			}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}

	// Create a resolved PropertyRef
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
}

func TestImportEntityModel(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		SetEntityModelExternalIDFunc: func(ctx context.Context, dataGraphID, modelID, externalID string) error {
			assert.Equal(t, "dg-123", dataGraphID)
			assert.Equal(t, "em-456", modelID)
			assert.Equal(t, "user", externalID)
			return nil
		},
		GetEntityModelFunc: func(ctx context.Context, dataGraphID, modelID string) (*dgClient.Model, error) {
			assert.Equal(t, "dg-123", dataGraphID)
			assert.Equal(t, "em-456", modelID)
			return &dgClient.Model{
				ID:          modelID,
				ExternalID:  "user",
				Type:        "entity",
				DataGraphID: "dg-123", // From API
			}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}

	// Create a resolved PropertyRef
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
}

func TestDeleteEntityModel(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		DeleteEntityModelFunc: func(ctx context.Context, dataGraphID, modelID string) error {
			assert.Equal(t, "dg-remote-123", dataGraphID)
			assert.Equal(t, "em-456", modelID)
			return nil
		},
	}

	h := &HandlerImpl{client: mockClient}

	// Create a resolved PropertyRef
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
}
