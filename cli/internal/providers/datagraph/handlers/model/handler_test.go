package model

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		ListEntityModelsFunc: func(ctx context.Context, dataGraphID string, page, pageSize int, isRoot *bool, hasExternalID *bool) (*dgClient.ListModelsResponse, error) {
			assert.Equal(t, "dg-1", dataGraphID)
			require.NotNil(t, hasExternalID)
			assert.True(t, *hasExternalID)

			if page == 1 {
				return &dgClient.ListModelsResponse{
					Data: []dgClient.Model{
						{
							ID:          "em-1",
							ExternalID:  "user",
							Name:        "User",
							Type:        "entity",
							TableRef:    "users",
							DataGraphID: "dg-1", // From API
							PrimaryID:   "id",
							Root:        true,
						},
					},
					Paging: client.Paging{Next: ""},
				}, nil
			}
			return &dgClient.ListModelsResponse{}, nil
		},
		ListEventModelsFunc: func(ctx context.Context, dataGraphID string, page, pageSize int, hasExternalID *bool) (*dgClient.ListModelsResponse, error) {
			assert.Equal(t, "dg-1", dataGraphID)
			require.NotNil(t, hasExternalID)
			assert.True(t, *hasExternalID)

			if page == 1 {
				return &dgClient.ListModelsResponse{
					Data: []dgClient.Model{
						{
							ID:          "evm-1",
							ExternalID:  "purchase",
							Name:        "Purchase",
							Type:        "event",
							TableRef:    "purchases",
							DataGraphID: "dg-1", // From API
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
		ListDataGraphsFunc: func(ctx context.Context, page, perPage int, hasExternalID *bool) (*dgClient.ListDataGraphsResponse, error) {
			// LoadImportableResources should not filter by external ID
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
		ListEntityModelsFunc: func(ctx context.Context, dataGraphID string, page, pageSize int, isRoot *bool, hasExternalID *bool) (*dgClient.ListModelsResponse, error) {
			assert.Equal(t, "dg-1", dataGraphID)
			require.NotNil(t, hasExternalID)
			assert.False(t, *hasExternalID)

			if page == 1 {
				return &dgClient.ListModelsResponse{
					Data: []dgClient.Model{
						{
							ID:          "em-2",
							Name:        "Account",
							Type:        "entity",
							TableRef:    "accounts",
							DataGraphID: "dg-1", // From API
							PrimaryID:   "account_id",
						},
					},
					Paging: client.Paging{Next: ""},
				}, nil
			}
			return &dgClient.ListModelsResponse{}, nil
		},
		ListEventModelsFunc: func(ctx context.Context, dataGraphID string, page, pageSize int, hasExternalID *bool) (*dgClient.ListModelsResponse, error) {
			assert.Equal(t, "dg-1", dataGraphID)
			require.NotNil(t, hasExternalID)
			assert.False(t, *hasExternalID)

			if page == 1 {
				return &dgClient.ListModelsResponse{
					Data: []dgClient.Model{
						{
							ID:          "evm-2",
							Name:        "PageView",
							Type:        "event",
							TableRef:    "page_views",
							DataGraphID: "dg-1", // From API
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
