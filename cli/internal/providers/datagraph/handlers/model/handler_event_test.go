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

func TestValidateEventModelResource(t *testing.T) {
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

	// Create URN and PropertyRef
	dataGraphURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
	dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)

	tests := []struct {
		name     string
		resource *model.ModelResource
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid event model resource",
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

func TestMapRemoteEventModelToState(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	h := &HandlerImpl{client: mockClient}

	t.Run("event model with external ID", func(t *testing.T) {
		// Create mock URN resolver
		urnResolver := testutils.NewMockURNResolver()
		urnResolver.AddMapping("data-graph", "dg-remote-1", "data-graph:my-dg")

		remote := &model.RemoteModel{
			Model: &dgClient.Model{
				ID:          "evm-1",
				ExternalID:  "purchase",
				Name:        "Purchase",
				Type:        "event",
				TableRef:    "purchases",
				DataGraphID: "dg-remote-1", // Remote ID from API
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

		// Verify DataGraphRef is created correctly
		require.NotNil(t, resource.DataGraphRef)
		assert.Equal(t, "data-graph:my-dg", resource.DataGraphRef.URN)

		expectedState := &model.ModelState{
			ID: "evm-1",
		}
		assert.Equal(t, expectedState, state)
	})
}

func TestCreateEventModel(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		CreateEventModelFunc: func(ctx context.Context, dataGraphID string, req *dgClient.CreateEventModelRequest) (*dgClient.Model, error) {
			assert.Equal(t, "dg-remote-123", dataGraphID)
			assert.Equal(t, "Purchase", req.Name)
			assert.Equal(t, "purchases", req.TableRef)
			assert.Equal(t, "purchase", req.ExternalID)
			assert.Equal(t, "event_time", req.Timestamp)

			return &dgClient.Model{
				ID:         "evm-789",
				Name:       req.Name,
				Type:       "event",
				TableRef:   req.TableRef,
				ExternalID: req.ExternalID,
				Timestamp:  req.Timestamp,
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
}

func TestUpdateEventModel(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		UpdateEventModelFunc: func(ctx context.Context, dataGraphID, modelID string, req *dgClient.UpdateEventModelRequest) (*dgClient.Model, error) {
			assert.Equal(t, "dg-remote-123", dataGraphID)
			assert.Equal(t, "evm-789", modelID)
			assert.Equal(t, "Updated Purchase", req.Name)
			assert.Equal(t, "purchases_v2", req.TableRef)
			assert.Equal(t, "ts", req.Timestamp)

			return &dgClient.Model{
				ID:        modelID,
				Name:      req.Name,
				Type:      "event",
				TableRef:  req.TableRef,
				Timestamp: req.Timestamp,
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
}

func TestImportEventModel(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		SetEventModelExternalIDFunc: func(ctx context.Context, dataGraphID, modelID, externalID string) error {
			assert.Equal(t, "dg-123", dataGraphID)
			assert.Equal(t, "evm-789", modelID)
			assert.Equal(t, "purchase", externalID)
			return nil
		},
		GetEventModelFunc: func(ctx context.Context, dataGraphID, modelID string) (*dgClient.Model, error) {
			assert.Equal(t, "dg-123", dataGraphID)
			assert.Equal(t, "evm-789", modelID)
			return &dgClient.Model{
				ID:          modelID,
				ExternalID:  "purchase",
				Type:        "event",
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
}

func TestDeleteEventModel(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		DeleteEventModelFunc: func(ctx context.Context, dataGraphID, modelID string) error {
			assert.Equal(t, "dg-remote-123", dataGraphID)
			assert.Equal(t, "evm-789", modelID)
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
}
