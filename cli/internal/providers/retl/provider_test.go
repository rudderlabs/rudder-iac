package retl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

type mockRETLStore struct {
	retlClient.RETLStore
	readStateFunc   func(ctx context.Context) (*retlClient.State, error)
	putStateFunc    func(ctx context.Context, req retlClient.PutStateRequest) error
	deleteStateFunc func(ctx context.Context, ID string) error
	// Adding mock functions for RETL source operations
	createRetlSourceFunc func(ctx context.Context, source *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error)
	updateRetlSourceFunc func(ctx context.Context, source *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error)
	deleteRetlSourceFunc func(ctx context.Context, id string) error
	getRetlSourceFunc    func(ctx context.Context, id string) (*retlClient.RETLSource, error)
	listRetlSourcesFunc  func(ctx context.Context) (*retlClient.RETLSources, error)
}

func (m *mockRETLStore) ReadState(ctx context.Context) (*retlClient.State, error) {
	if m.readStateFunc != nil {
		return m.readStateFunc(ctx)
	}
	return &retlClient.State{Resources: map[string]retlClient.ResourceState{}}, nil
}

func (m *mockRETLStore) PutResourceState(ctx context.Context, req retlClient.PutStateRequest) error {
	if m.putStateFunc != nil {
		return m.putStateFunc(ctx, req)
	}
	return nil
}

func (m *mockRETLStore) DeleteResourceState(ctx context.Context, ID string) error {
	if m.deleteStateFunc != nil {
		return m.deleteStateFunc(ctx, ID)
	}
	return nil
}

// Add implementations for RETL source operations
func (m *mockRETLStore) CreateRetlSource(ctx context.Context, source *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
	if m.createRetlSourceFunc != nil {
		return m.createRetlSourceFunc(ctx, source)
	}
	return &retlClient.RETLSource{ID: "test-source-id"}, nil
}

func (m *mockRETLStore) UpdateRetlSource(ctx context.Context, source *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error) {
	if m.updateRetlSourceFunc != nil {
		return m.updateRetlSourceFunc(ctx, source)
	}
	return &retlClient.RETLSource{
		ID:                   source.SourceID,
		SourceType:           "model",
		SourceDefinitionName: "postgres",
		Name:                 source.Name,
		Config:               source.Config,
		AccountID:            source.AccountID,
	}, nil
}

func (m *mockRETLStore) DeleteRetlSource(ctx context.Context, id string) error {
	if m.deleteRetlSourceFunc != nil {
		return m.deleteRetlSourceFunc(ctx, id)
	}
	return nil
}

func (m *mockRETLStore) GetRetlSource(ctx context.Context, id string) (*retlClient.RETLSource, error) {
	if m.getRetlSourceFunc != nil {
		return m.getRetlSourceFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockRETLStore) ListRetlSources(ctx context.Context) (*retlClient.RETLSources, error) {
	if m.listRetlSourcesFunc != nil {
		return m.listRetlSourcesFunc(ctx)
	}
	return &retlClient.RETLSources{}, nil
}

func TestProvider(t *testing.T) {
	t.Parallel()

	mockClient := &mockRETLStore{}

	// Set up mock client functions
	mockClient.createRetlSourceFunc = func(ctx context.Context, source *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
		return &retlClient.RETLSource{
			ID:                   "test-source-id",
			SourceType:           source.SourceType,
			SourceDefinitionName: source.SourceDefinitionName,
			Name:                 source.Name,
			Config:               source.Config,
			AccountID:            source.AccountID,
		}, nil
	}

	mockClient.updateRetlSourceFunc = func(ctx context.Context, source *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error) {
		return &retlClient.RETLSource{
			ID:                   source.SourceID,
			SourceType:           "model",
			SourceDefinitionName: "postgres",
			Name:                 source.Name,
			Config:               source.Config,
			AccountID:            source.AccountID,
		}, nil
	}

	mockClient.deleteRetlSourceFunc = func(ctx context.Context, id string) error {
		return nil
	}

	provider := New(mockClient)

	t.Run("GetSupportedKinds", func(t *testing.T) {
		t.Parallel()

		kinds := provider.GetSupportedKinds()
		assert.Contains(t, kinds, "retl-source-sql-model")
	})

	t.Run("GetSupportedTypes", func(t *testing.T) {
		t.Parallel()

		types := provider.GetSupportedTypes()
		assert.Contains(t, types, "sql-model")
	})

	t.Run("LoadSpec", func(t *testing.T) {
		t.Parallel()

		t.Run("UnsupportedKind", func(t *testing.T) {
			t.Parallel()

			err := provider.LoadSpec("test.yaml", &specs.Spec{Kind: "unsupported"})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported kind")
		})

		t.Run("ValidKind", func(t *testing.T) {
			t.Parallel()

			err := provider.LoadSpec("test.yaml", &specs.Spec{
				Kind: "retl-source-sql-model",
				Spec: map[string]interface{}{
					"id":           "test-model",
					"display_name": "Test Model",
					"description":  "Test Description",
					"account_id":   "test-account",
					"primary_key":  "id",
					"sql":          "SELECT * FROM users",
				},
			})
			assert.NoError(t, err)
		})
	})

	t.Run("GetResourceGraph", func(t *testing.T) {
		t.Parallel()

		graph, err := provider.GetResourceGraph()
		require.NoError(t, err)
		assert.NotNil(t, graph)
	})

	t.Run("LoadState", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockClient.readStateFunc = func(ctx context.Context) (*retlClient.State, error) {
			return &retlClient.State{
				Resources: map[string]retlClient.ResourceState{
					"sql-model:test": {
						ID:   "test",
						Type: "sql-model",
					},
				},
			}, nil
		}

		s, err := provider.LoadState(ctx)
		require.NoError(t, err)
		assert.NotNil(t, s)

		rs := s.GetResource("sql-model:test")
		require.NotNil(t, rs)
		assert.Equal(t, "test", rs.ID)
		assert.Equal(t, "sql-model", rs.Type)
	})

	t.Run("PutResourceState", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		called := false
		mockClient.putStateFunc = func(ctx context.Context, req retlClient.PutStateRequest) error {
			called = true
			assert.Equal(t, "test", req.ID)
			assert.Equal(t, "test:resource", req.URN)
			return nil
		}

		err := provider.PutResourceState(ctx, "test:resource", &state.ResourceState{
			ID:   "test",
			Type: "sql-model",
		})
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("DeleteResourceState", func(t *testing.T) {
		t.Parallel()

		// DeleteResourceState is now a no-op, so just check that it returns nil
		ctx := context.Background()
		err := provider.DeleteResourceState(ctx, &state.ResourceState{ID: "test"})
		require.NoError(t, err)
	})

	t.Run("CRUD Operations", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		// Create complete test data for SQL model
		createData := resources.ResourceData{
			"id":           "test-model",
			"display_name": "Test Model",
			"description":  "Test Description",
			"account_id":   "test-account",
			"primary_key":  "id",
			"sql":          "SELECT * FROM users",
		}

		t.Run("Create", func(t *testing.T) {
			t.Parallel()

			result, err := provider.Create(ctx, "test", "sql-model", createData)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, "test-source-id", (*result)["source_id"])

			_, err = provider.Create(ctx, "test", "unknown", createData)
			assert.Error(t, err)
		})

		t.Run("Update", func(t *testing.T) {
			t.Parallel()

			// For update, we need state data with a source_id
			stateData := resources.ResourceData{
				"id":           "test-model",
				"display_name": "Test Model",
				"description":  "Test Description",
				"account_id":   "test-account",
				"primary_key":  "id",
				"sql":          "SELECT * FROM users",
				"source_id":    "test-source-id",
			}

			result, err := provider.Update(ctx, "test", "sql-model", createData, stateData)
			require.NoError(t, err)
			require.NotNil(t, result)

			_, err = provider.Update(ctx, "test", "unknown", createData, stateData)
			assert.Error(t, err)
		})

		t.Run("Delete", func(t *testing.T) {
			t.Parallel()

			// For delete, we need state data with a source_id
			stateData := resources.ResourceData{
				"source_id": "test-source-id",
			}

			err := provider.Delete(ctx, "test", "sql-model", stateData)
			require.NoError(t, err)

			err = provider.Delete(ctx, "test", "unknown", stateData)
			assert.Error(t, err)
		})
	})
}
