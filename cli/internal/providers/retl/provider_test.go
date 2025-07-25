package retl_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

// mockRETLStore mocks the RETL client for testing
type mockRETLStore struct {
	retlClient.RETLStore
	sourceID        string
	readStateFunc   func(ctx context.Context) (*retlClient.State, error)
	putStateFunc    func(ctx context.Context, id string, req retlClient.PutStateRequest) error
	deleteStateFunc func(ctx context.Context, ID string) error
	// Adding mock functions for RETL source operations
	createRetlSourceFunc func(ctx context.Context, source *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error)
	updateRetlSourceFunc func(ctx context.Context, sourceID string, source *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error)
	deleteRetlSourceFunc func(ctx context.Context, id string) error
	getRetlSourceFunc    func(ctx context.Context, id string) (*retlClient.RETLSource, error)
	listRetlSourcesFunc  func(ctx context.Context) (*retlClient.RETLSources, error)
}

func (m *mockRETLStore) ReadState(ctx context.Context) (*retlClient.State, error) {
	if m.readStateFunc != nil {
		return m.readStateFunc(ctx)
	}
	return nil, nil
}

func (m *mockRETLStore) PutResourceState(ctx context.Context, id string, req retlClient.PutStateRequest) error {
	if m.putStateFunc != nil {
		return m.putStateFunc(ctx, id, req)
	}
	return nil
}

func (m *mockRETLStore) DeleteResourceState(ctx context.Context, ID string) error {
	if m.deleteStateFunc != nil {
		return m.deleteStateFunc(ctx, ID)
	}
	return nil
}

// Mock RETL source operations

func (m *mockRETLStore) CreateRetlSource(ctx context.Context, source *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
	if m.createRetlSourceFunc != nil {
		return m.createRetlSourceFunc(ctx, source)
	}
	return nil, nil
}

func (m *mockRETLStore) UpdateRetlSource(ctx context.Context, sourceID string, source *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error) {
	if m.updateRetlSourceFunc != nil {
		return m.updateRetlSourceFunc(ctx, sourceID, source)
	}
	return nil, nil
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

// newDefaultMockClient creates a new mock client with default behavior
func newDefaultMockClient() *mockRETLStore {
	return &mockRETLStore{
		sourceID: "test-source-id",
		createRetlSourceFunc: func(ctx context.Context, source *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
			return &retlClient.RETLSource{
				ID:                   "test-source-id",
				SourceType:           source.SourceType,
				SourceDefinitionName: source.SourceDefinitionName,
				Name:                 source.Name,
				Config:               source.Config,
				AccountID:            source.AccountID,
			}, nil
		},
		updateRetlSourceFunc: func(ctx context.Context, sourceID string, source *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error) {
			return &retlClient.RETLSource{
				SourceType:           "model",
				SourceDefinitionName: "postgres",
				Name:                 source.Name,
				Config:               source.Config,
				AccountID:            source.AccountID,
			}, nil
		},
		deleteRetlSourceFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}
}

func TestProvider(t *testing.T) {
	t.Run("GetSupportedKinds", func(t *testing.T) {
		t.Parallel()
		provider := retl.New(newDefaultMockClient())
		kinds := provider.GetSupportedKinds()
		assert.Contains(t, kinds, "retl-source-sql-model")
	})

	t.Run("GetSupportedTypes", func(t *testing.T) {
		t.Parallel()
		provider := retl.New(newDefaultMockClient())
		types := provider.GetSupportedTypes()
		assert.Contains(t, types, sqlmodel.ResourceType)
	})

	t.Run("LoadSpec", func(t *testing.T) {
		t.Run("UnsupportedKind", func(t *testing.T) {
			t.Parallel()
			provider := retl.New(newDefaultMockClient())
			err := provider.LoadSpec("test.yaml", &specs.Spec{Kind: "unsupported"})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported kind")
		})

		t.Run("ValidKind", func(t *testing.T) {
			t.Parallel()
			provider := retl.New(newDefaultMockClient())
			err := provider.LoadSpec("test.yaml", &specs.Spec{
				Kind: "retl-source-sql-model",
				Spec: map[string]interface{}{
					"id":                     "test-model",
					"display_name":           "Test Model",
					"description":            "Test Description",
					"account_id":             "test-account",
					"primary_key":            "id",
					"sql":                    "SELECT * FROM users",
					"source_definition_name": "postgres",
				},
			})
			assert.NoError(t, err)
		})
	})

	t.Run("LoadState", func(t *testing.T) {
		t.Run("Success with multiple resources", func(t *testing.T) {
			t.Parallel()
			mockClient := newDefaultMockClient()
			provider := retl.New(mockClient)

			ctx := context.Background()
			mockClient.readStateFunc = func(ctx context.Context) (*retlClient.State, error) {
				return &retlClient.State{
					Resources: map[string]retlClient.ResourceState{
						"retl-source-sql-model:test1": {
							ID:   "test1",
							Type: "retl-source-sql-model",
							Input: map[string]interface{}{
								"name": "test1",
							},
							Output: map[string]interface{}{
								"source_id": "src1",
							},
							Dependencies: []string{"dep1"},
						},
						"retl-source-sql-model:test2": {
							ID:   "test2",
							Type: "retl-source-sql-model",
							Input: map[string]interface{}{
								"name": "test2",
							},
							Output: map[string]interface{}{
								"source_id": "src2",
							},
							Dependencies: []string{"dep2"},
						},
					},
				}, nil
			}

			s, err := provider.LoadState(ctx)
			require.NoError(t, err)
			assert.NotNil(t, s)

			// Check first resource
			rs1 := s.GetResource("retl-source-sql-model:test1")
			require.NotNil(t, rs1)
			assert.Equal(t, "test1", rs1.ID)
			assert.Equal(t, "retl-source-sql-model", rs1.Type)
			assert.Equal(t, "test1", rs1.Input["name"])
			assert.Equal(t, "src1", rs1.Output["source_id"])
			assert.Equal(t, []string{"dep1"}, rs1.Dependencies)

			// Check second resource
			rs2 := s.GetResource("retl-source-sql-model:test2")
			require.NotNil(t, rs2)
			assert.Equal(t, "test2", rs2.ID)
			assert.Equal(t, "retl-source-sql-model", rs2.Type)
			assert.Equal(t, "test2", rs2.Input["name"])
			assert.Equal(t, "src2", rs2.Output["source_id"])
			assert.Equal(t, []string{"dep2"}, rs2.Dependencies)
		})

		t.Run("Error reading state", func(t *testing.T) {
			t.Parallel()
			mockClient := newDefaultMockClient()
			provider := retl.New(mockClient)

			ctx := context.Background()
			mockClient.readStateFunc = func(ctx context.Context) (*retlClient.State, error) {
				return nil, fmt.Errorf("failed to read state")
			}

			s, err := provider.LoadState(ctx)
			assert.Error(t, err)
			assert.Nil(t, s)
			assert.Contains(t, err.Error(), "reading remote state")
		})
	})

	t.Run("GetResourceGraph", func(t *testing.T) {
		t.Run("Multiple resources", func(t *testing.T) {
			t.Parallel()
			provider := retl.New(newDefaultMockClient())

			// Load multiple specs to test graph with multiple resources
			err := provider.LoadSpec("test1.yaml", &specs.Spec{
				Kind: "retl-source-sql-model",
				Spec: map[string]interface{}{
					"id":                     "test-model-1",
					"display_name":           "Test Model 1",
					"description":            "Test Description 1",
					"account_id":             "test-account",
					"primary_key":            "id",
					"sql":                    "SELECT * FROM users",
					"source_definition_name": "postgres",
				},
			})
			require.NoError(t, err)

			err = provider.LoadSpec("test2.yaml", &specs.Spec{
				Kind: "retl-source-sql-model",
				Spec: map[string]interface{}{
					"id":                     "test-model-2",
					"display_name":           "Test Model 2",
					"description":            "Test Description 2",
					"account_id":             "test-account",
					"primary_key":            "id",
					"sql":                    "SELECT * FROM orders",
					"source_definition_name": "postgres",
				},
			})
			require.NoError(t, err)

			graph, err := provider.GetResourceGraph()
			require.NoError(t, err)
			assert.NotNil(t, graph)

			// Verify both resources are in the graph
			resources := graph.Resources()
			assert.Len(t, resources, 2)

			// Verify resource IDs
			resourceIDs := make([]string, 0, len(resources))
			for _, r := range resources {
				resourceIDs = append(resourceIDs, r.ID())
			}
			assert.Contains(t, resourceIDs, "test-model-1")
			assert.Contains(t, resourceIDs, "test-model-2")
		})
	})

	t.Run("PutResourceState", func(t *testing.T) {
		t.Run("Success case", func(t *testing.T) {
			t.Parallel()
			mockClient := newDefaultMockClient()
			provider := retl.New(mockClient)

			ctx := context.Background()
			called := false
			mockClient.putStateFunc = func(ctx context.Context, id string, req retlClient.PutStateRequest) error {
				called = true
				assert.Equal(t, "test", id)
				assert.Equal(t, "test:resource", req.URN)
				return nil
			}

			rs := &state.ResourceState{
				ID:   "test",
				Type: sqlmodel.ResourceType,
				Output: map[string]interface{}{
					sqlmodel.IDKey: "test",
				},
			}

			err := provider.PutResourceState(ctx, "test:resource", rs)
			require.NoError(t, err)
			assert.True(t, called)
		})

		t.Run("Error case - client error", func(t *testing.T) {
			t.Parallel()
			mockClient := newDefaultMockClient()
			provider := retl.New(mockClient)

			ctx := context.Background()
			mockClient.putStateFunc = func(ctx context.Context, id string, req retlClient.PutStateRequest) error {
				return fmt.Errorf("failed to put state")
			}

			rs := &state.ResourceState{
				ID:   "test",
				Type: sqlmodel.ResourceType,
				Output: map[string]interface{}{
					sqlmodel.IDKey: "test",
				},
			}

			err := provider.PutResourceState(ctx, "test:resource", rs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to put state")
		})
	})

	t.Run("DeleteResourceState", func(t *testing.T) {
		t.Parallel()
		provider := retl.New(newDefaultMockClient())

		// DeleteResourceState is now a no-op, so just check that it returns nil
		ctx := context.Background()
		err := provider.DeleteResourceState(ctx, &state.ResourceState{ID: "test"})
		require.NoError(t, err)
	})

	t.Run("CRUD Operations", func(t *testing.T) {
		t.Run("Create", func(t *testing.T) {
			t.Parallel()
			provider := retl.New(newDefaultMockClient())
			ctx := context.Background()

			// Create complete test data for SQL model
			createData := resources.ResourceData{
				"id":                "test-model",
				"display_name":      "Test Model",
				"description":       "Test Description",
				"account_id":        "test-account",
				"primary_key":       "id",
				"sql":               "SELECT * FROM users",
				"source_definition": "postgres",
				"enabled":           true,
			}

			result, err := provider.Create(ctx, "test", sqlmodel.ResourceType, createData)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, "test-source-id", (*result)[sqlmodel.IDKey])

			_, err = provider.Create(ctx, "test", "unknown", createData)
			assert.Error(t, err)
		})

		t.Run("Update", func(t *testing.T) {
			t.Parallel()
			provider := retl.New(newDefaultMockClient())
			ctx := context.Background()

			// Create complete test data for SQL model
			createData := resources.ResourceData{
				"id":                     "test-model",
				"display_name":           "Test Model",
				"description":            "Test Description",
				"account_id":             "test-account",
				"primary_key":            "id",
				"sql":                    "SELECT * FROM users",
				"source_definition_name": "postgres",
				"enabled":                true,
			}

			// For update, we need state data with a source_id
			stateData := resources.ResourceData{
				"display_name":           "Test Model",
				"description":            "Test Description",
				"account_id":             "test-account",
				"primary_key":            "id",
				"sql":                    "SELECT * FROM users",
				"id":                     "test-source-id",
				"source_definition_name": "postgres",
				"enabled":                true,
			}

			result, err := provider.Update(ctx, "test", sqlmodel.ResourceType, createData, stateData)
			require.NoError(t, err)
			require.NotNil(t, result)

			_, err = provider.Update(ctx, "test", "unknown", createData, stateData)
			assert.Error(t, err)
		})

		t.Run("Delete", func(t *testing.T) {
			t.Parallel()
			provider := retl.New(newDefaultMockClient())
			ctx := context.Background()

			// For delete, we need state data with a source_id
			stateData := resources.ResourceData{
				sqlmodel.IDKey: "test-source-id",
			}

			err := provider.Delete(ctx, "test", sqlmodel.ResourceType, stateData)
			require.NoError(t, err)

			err = provider.Delete(ctx, "test", "unknown", stateData)
			assert.Error(t, err)
		})
	})

	t.Run("Validate", func(t *testing.T) {
		testCases := []struct {
			name          string
			specs         []*specs.Spec
			expectedError bool
			errorMessage  string
			loadError     bool
		}{
			{
				name: "Valid resources",
				specs: []*specs.Spec{
					{
						Version: "rudder/v0.1",
						Kind:    "retl-source-sql-model",
						Spec: map[string]interface{}{
							"id":                "test-model",
							"display_name":      "Test Model",
							"description":       "Test description",
							"sql":               "SELECT * FROM users",
							"account_id":        "acc123",
							"primary_key":       "id",
							"source_definition": "postgres",
							"enabled":           true,
						},
					},
				},
				expectedError: false,
				loadError:     false,
			},
			{
				name: "Invalid resource - missing required fields",
				specs: []*specs.Spec{
					{
						Version: "rudder/v0.1",
						Kind:    "retl-source-sql-model",
						Spec: map[string]interface{}{
							"id":                "test-model",
							"display_name":      "Test Model",
							"description":       "Test description",
							"account_id":        "acc123",
							"primary_key":       "id",
							"source_definition": "postgres",
							"enabled":           true,
						},
					},
				},
				expectedError: true,
				errorMessage:  "sql or file must be specified",
				loadError:     true,
			},
		}

		for _, tc := range testCases {
			tc := tc // capture range variable
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				provider := retl.New(newDefaultMockClient())

				// Load all specs
				for _, spec := range tc.specs {
					err := provider.LoadSpec("test.yaml", spec)
					if tc.loadError {
						assert.Error(t, err)
						if tc.errorMessage != "" {
							assert.Contains(t, err.Error(), tc.errorMessage)
						}
						return
					}
					require.NoError(t, err, "LoadSpec should not fail")
				}

				// Validate all specs
				err := provider.Validate()
				if tc.expectedError {
					assert.Error(t, err)
					if tc.errorMessage != "" {
						assert.Contains(t, err.Error(), tc.errorMessage)
					}
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}
