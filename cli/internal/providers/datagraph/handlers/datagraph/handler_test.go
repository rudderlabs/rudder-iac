package datagraph

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateResource(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	h := &HandlerImpl{client: mockClient}
	graph := resources.NewGraph()

	tests := []struct {
		name     string
		resource *model.DataGraphResource
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid resource",
			resource: &model.DataGraphResource{
				ID:        "test-dg",
				Name:      "Test Data Graph",
				AccountID: "account-123",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			resource: &model.DataGraphResource{
				ID:        "test-dg",
				AccountID: "account-123",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing account_id",
			resource: &model.DataGraphResource{
				ID:   "test-dg",
				Name: "Test Data Graph",
			},
			wantErr: true,
			errMsg:  "account_id is required",
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

func TestLoadRemoteResources(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		ListDataGraphsFunc: func(ctx context.Context, page, perPage int, hasExternalID *bool) (*dgClient.ListDataGraphsResponse, error) {
			// Verify that hasExternalID is set to true
			require.NotNil(t, hasExternalID)
			assert.True(t, *hasExternalID)

			if page == 1 {
				return &dgClient.ListDataGraphsResponse{
					Data: []dgClient.DataGraph{
						{
							ID:         "remote-1",
							ExternalID: "dg-1",
							Name:       "Data Graph 1",
							AccountID:  "account-1",
						},
					},
					Paging: client.Paging{Next: ""},
				}, nil
			}
			return &dgClient.ListDataGraphsResponse{}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}
	remotes, err := h.LoadRemoteResources(context.Background())
	require.NoError(t, err)
	require.Len(t, remotes, 1)

	expected := &model.RemoteDataGraph{
		DataGraph: &dgClient.DataGraph{
			ID:         "remote-1",
			ExternalID: "dg-1",
			Name:       "Data Graph 1",
			AccountID:  "account-1",
		},
	}

	assert.Equal(t, expected, remotes[0])
}

func TestLoadImportableResources(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		ListDataGraphsFunc: func(ctx context.Context, page, perPage int, hasExternalID *bool) (*dgClient.ListDataGraphsResponse, error) {
			// Verify that hasExternalID is set to false
			require.NotNil(t, hasExternalID)
			assert.False(t, *hasExternalID)

			if page == 1 {
				return &dgClient.ListDataGraphsResponse{
					Data: []dgClient.DataGraph{
						{
							ID:         "remote-2",
							ExternalID: "", // No external ID - importable
							Name:       "Data Graph 2",
							AccountID:  "account-2",
						},
					},
					Paging: client.Paging{Next: ""},
				}, nil
			}
			return &dgClient.ListDataGraphsResponse{}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}
	remotes, err := h.LoadImportableResources(context.Background())
	require.NoError(t, err)
	require.Len(t, remotes, 1)
}

func TestMapRemoteToState(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	h := &HandlerImpl{client: mockClient}

	t.Run("with external ID", func(t *testing.T) {
		remote := &model.RemoteDataGraph{
			DataGraph: &dgClient.DataGraph{
				ID:         "remote-1",
				ExternalID: "dg-1",
				Name:       "Data Graph 1",
				AccountID:  "account-1",
			},
		}

		resource, state, err := h.MapRemoteToState(remote, nil)
		require.NoError(t, err)

		expectedResource := &model.DataGraphResource{
			ID:        "dg-1",
			Name:      "Data Graph 1",
			AccountID: "account-1",
		}
		expectedState := &model.DataGraphState{
			ID: "remote-1",
		}

		assert.Equal(t, expectedResource, resource)
		assert.Equal(t, expectedState, state)
	})

	t.Run("without external ID", func(t *testing.T) {
		remote := &model.RemoteDataGraph{
			DataGraph: &dgClient.DataGraph{
				ID:         "remote-2",
				ExternalID: "",
				Name:       "Data Graph 2",
				AccountID:  "account-2",
			},
		}

		resource, state, err := h.MapRemoteToState(remote, nil)
		require.NoError(t, err)
		assert.Nil(t, resource)
		assert.Nil(t, state)
	})
}

func TestCreate(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		CreateDataGraphFunc: func(ctx context.Context, req *dgClient.CreateDataGraphRequest) (*dgClient.DataGraph, error) {
			assert.Equal(t, "Test Data Graph", req.Name)
			assert.Equal(t, "account-123", req.AccountID)
			assert.Equal(t, "test-dg", req.ExternalID)
			return &dgClient.DataGraph{
				ID:         "remote-1",
				Name:       req.Name,
				AccountID:  req.AccountID,
				ExternalID: req.ExternalID,
			}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}

	data := &model.DataGraphResource{
		ID:        "test-dg",
		Name:      "Test Data Graph",
		AccountID: "account-123",
	}

	state, err := h.Create(context.Background(), data)
	require.NoError(t, err)

	expectedState := &model.DataGraphState{
		ID: "remote-1",
	}
	assert.Equal(t, expectedState, state)
}

func TestUpdate(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		UpdateDataGraphFunc: func(ctx context.Context, id string, req *dgClient.UpdateDataGraphRequest) (*dgClient.DataGraph, error) {
			assert.Equal(t, "remote-1", id)
			assert.Equal(t, "Updated Data Graph", req.Name)
			return &dgClient.DataGraph{
				ID:   id,
				Name: req.Name,
			}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}

	newData := &model.DataGraphResource{
		ID:        "test-dg",
		Name:      "Updated Data Graph",
		AccountID: "account-123",
	}

	oldData := &model.DataGraphResource{
		ID:        "test-dg",
		Name:      "Test Data Graph",
		AccountID: "account-123",
	}

	oldState := &model.DataGraphState{
		ID: "remote-1",
	}

	state, err := h.Update(context.Background(), newData, oldData, oldState)
	require.NoError(t, err)

	expectedState := &model.DataGraphState{
		ID: "remote-1",
	}
	assert.Equal(t, expectedState, state)
}

func TestImport(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		SetExternalIDFunc: func(ctx context.Context, id string, externalID string) error {
			assert.Equal(t, "remote-1", id)
			assert.Equal(t, "test-dg", externalID)
			return nil
		},
		GetDataGraphFunc: func(ctx context.Context, id string) (*dgClient.DataGraph, error) {
			assert.Equal(t, "remote-1", id)
			return &dgClient.DataGraph{
				ID:         id,
				ExternalID: "test-dg",
			}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}

	data := &model.DataGraphResource{
		ID:        "test-dg",
		Name:      "Test Data Graph",
		AccountID: "account-123",
	}

	state, err := h.Import(context.Background(), data, "remote-1")
	require.NoError(t, err)

	expectedState := &model.DataGraphState{
		ID: "remote-1",
	}
	assert.Equal(t, expectedState, state)
}

func TestDelete(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		DeleteDataGraphFunc: func(ctx context.Context, id string) error {
			assert.Equal(t, "remote-1", id)
			return nil
		},
	}

	h := &HandlerImpl{client: mockClient}

	oldData := &model.DataGraphResource{
		ID:        "test-dg",
		Name:      "Test Data Graph",
		AccountID: "account-123",
	}

	oldState := &model.DataGraphState{
		ID: "remote-1",
	}

	err := h.Delete(context.Background(), "test-dg", oldData, oldState)
	require.NoError(t, err)
}

