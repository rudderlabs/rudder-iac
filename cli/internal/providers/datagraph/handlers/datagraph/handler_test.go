package datagraph

import (
	"context"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRemoteResources(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		ListDataGraphsFunc: func(ctx context.Context, req *dgClient.ListDataGraphsRequest) (*dgClient.ListDataGraphsResponse, error) {
			// Verify that hasExternalID is set to true
			require.NotNil(t, req.HasExternalID)
			assert.True(t, *req.HasExternalID)

			if req.Page == 1 {
				return &dgClient.ListDataGraphsResponse{
					Data: []dgClient.DataGraph{
						{
							ID:         "remote-1",
							ExternalID: "dg-1",
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
			AccountID:  "account-1",
		},
	}

	assert.Equal(t, expected, remotes[0])
}

func TestLoadImportableResources(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		ListDataGraphsFunc: func(ctx context.Context, req *dgClient.ListDataGraphsRequest) (*dgClient.ListDataGraphsResponse, error) {
			// Verify that hasExternalID is set to false
			require.NotNil(t, req.HasExternalID)
			assert.False(t, *req.HasExternalID)

			if req.Page == 1 {
				return &dgClient.ListDataGraphsResponse{
					Data: []dgClient.DataGraph{
						{
							ID:         "remote-2",
							ExternalID: "", // No external ID - importable
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

func TestLoadImportableResources_WithAccountNameResolution(t *testing.T) {
	t.Run("resolves account name", func(t *testing.T) {
		mockClient := &testutils.MockDataGraphClient{
			ListDataGraphsFunc: func(ctx context.Context, req *dgClient.ListDataGraphsRequest) (*dgClient.ListDataGraphsResponse, error) {
				if req.Page == 1 {
					return &dgClient.ListDataGraphsResponse{
						Data: []dgClient.DataGraph{
							{ID: "remote-1", AccountID: "account-1"},
							{ID: "remote-2", AccountID: "account-2"},
						},
						Paging: client.Paging{Next: ""},
					}, nil
				}
				return &dgClient.ListDataGraphsResponse{}, nil
			},
		}

		mockResolver := &testutils.MockAccountNameResolver{
			GetAccountNameFunc: func(ctx context.Context, accountID string) (string, error) {
				names := map[string]string{
					"account-1": "My Warehouse",
					"account-2": "Production DB",
				}
				if name, ok := names[accountID]; ok {
					return name, nil
				}
				return "", assert.AnError
			},
		}

		h := &HandlerImpl{client: mockClient, accountResolver: mockResolver}
		remotes, err := h.LoadImportableResources(context.Background())
		require.NoError(t, err)
		require.Len(t, remotes, 2)
		assert.Equal(t, "My Warehouse", remotes[0].AccountName)
		assert.Equal(t, "Production DB", remotes[1].AccountName)

		// Verify Metadata() returns account name as Name
		assert.Equal(t, "My Warehouse", remotes[0].Metadata().Name)
		assert.Equal(t, "Production DB", remotes[1].Metadata().Name)
	})

	t.Run("returns error when account resolution fails", func(t *testing.T) {
		mockClient := &testutils.MockDataGraphClient{
			ListDataGraphsFunc: func(ctx context.Context, req *dgClient.ListDataGraphsRequest) (*dgClient.ListDataGraphsResponse, error) {
				return &dgClient.ListDataGraphsResponse{
					Data: []dgClient.DataGraph{
						{ID: "remote-1", AccountID: "bad-account"},
					},
					Paging: client.Paging{Next: ""},
				}, nil
			},
		}

		mockResolver := &testutils.MockAccountNameResolver{
			GetAccountNameFunc: func(ctx context.Context, accountID string) (string, error) {
				return "", assert.AnError
			},
		}

		h := &HandlerImpl{client: mockClient, accountResolver: mockResolver}
		_, err := h.LoadImportableResources(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resolving account name")
	})
}

func TestMapRemoteToState(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	h := &HandlerImpl{client: mockClient}

	t.Run("with external ID", func(t *testing.T) {
		remote := &model.RemoteDataGraph{
			DataGraph: &dgClient.DataGraph{
				ID:         "remote-1",
				ExternalID: "dg-1",
				AccountID:  "account-1",
			},
		}

		resource, state, err := h.MapRemoteToState(remote, nil)
		require.NoError(t, err)

		expectedResource := &model.DataGraphResource{
			ID:        "dg-1",
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
			assert.Equal(t, "account-123", req.AccountID)
			assert.Equal(t, "test-dg", req.ExternalID)
			return &dgClient.DataGraph{
				ID:         "remote-1",
				AccountID:  req.AccountID,
				ExternalID: req.ExternalID,
			}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}

	data := &model.DataGraphResource{
		ID:        "test-dg",
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
	mockClient := &testutils.MockDataGraphClient{}

	h := &HandlerImpl{client: mockClient}

	newData := &model.DataGraphResource{
		ID:        "test-dg",
		AccountID: "account-123",
	}

	oldData := &model.DataGraphResource{
		ID:        "test-dg",
		AccountID: "account-123",
	}

	oldState := &model.DataGraphState{
		ID: "remote-1",
	}

	_, err := h.Update(context.Background(), newData, oldData, oldState)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "do not support updates")
}

func TestImport(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		SetExternalIDFunc: func(ctx context.Context, req *dgClient.SetExternalIDRequest) (*dgClient.DataGraph, error) {
			assert.Equal(t, "remote-1", req.ID)
			assert.Equal(t, "test-dg", req.ExternalID)
			return &dgClient.DataGraph{
				ID:         req.ID,
				ExternalID: req.ExternalID,
			}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}

	data := &model.DataGraphResource{
		ID:        "test-dg",
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
		AccountID: "account-123",
	}

	oldState := &model.DataGraphState{
		ID: "remote-1",
	}

	err := h.Delete(context.Background(), "test-dg", oldData, oldState)
	require.NoError(t, err)
}

// TestDataGraphResourceMapstructureTags verifies that mapstructure.Decode produces
// snake_case keys from DataGraphResource, matching what the diff engine expects.
func TestDataGraphResourceMapstructureTags(t *testing.T) {
	resource := &model.DataGraphResource{
		ID:        "test-dg",
		AccountID: "account-123",
	}

	var result map[string]interface{}
	err := mapstructure.Decode(resource, &result)
	require.NoError(t, err)

	assert.Equal(t, map[string]interface{}{
		"id":         "test-dg",
		"account_id": "account-123",
	}, result)
}
