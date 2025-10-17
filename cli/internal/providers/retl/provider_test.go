package retl_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
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
	listRetlSourcesFunc  func(ctx context.Context, hasExternalID *bool) (*retlClient.RETLSources, error)
	// Preview functions
	submitPreviewFunc    func(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error)
	getPreviewResultFunc func(ctx context.Context, resultID string) (*retlClient.PreviewResultResponse, error)
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

func (m *mockRETLStore) ListRetlSources(ctx context.Context, hasExternalID *bool) (*retlClient.RETLSources, error) {
	if m.listRetlSourcesFunc != nil {
		return m.listRetlSourcesFunc(ctx, hasExternalID)
	}
	return &retlClient.RETLSources{}, nil
}

// Preview methods
func (m *mockRETLStore) SubmitSourcePreview(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error) {
	if m.submitPreviewFunc != nil {
		return m.submitPreviewFunc(ctx, request)
	}
	return &retlClient.PreviewSubmitResponse{ID: "req-123"}, nil
}

func (m *mockRETLStore) GetSourcePreviewResult(ctx context.Context, resultID string) (*retlClient.PreviewResultResponse, error) {
	if m.getPreviewResultFunc != nil {
		return m.getPreviewResultFunc(ctx, resultID)
	}
	return &retlClient.PreviewResultResponse{Status: retlClient.Completed}, nil
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
		getRetlSourceFunc: func(ctx context.Context, id string) (*retlClient.RETLSource, error) {

			if id == "remote-id-not-found" {
				return nil, fmt.Errorf("retl source not found")
			}

			return &retlClient.RETLSource{
				ID:                   "remote-id",
				Name:                 "Imported Model",
				SourceType:           retlClient.ModelSourceType,
				SourceDefinitionName: "postgres",
				AccountID:            "acc123",
				IsEnabled:            true,
				Config:               retlClient.RETLSQLModelConfig{Description: "desc", PrimaryKey: "id", Sql: "SELECT * FROM t"},
			}, nil
		},
		submitPreviewFunc: func(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error) {
			return &retlClient.PreviewSubmitResponse{
				ID: "req-123",
			}, nil
		},
		getPreviewResultFunc: func(ctx context.Context, resultID string) (*retlClient.PreviewResultResponse, error) {
			return &retlClient.PreviewResultResponse{
				Status: retlClient.Completed,
				Rows:   []map[string]any{{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}},
			}, nil
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

	t.Run("Import", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			provider := retl.New(newDefaultMockClient())

			args := importremote.ImportArgs{RemoteID: "remote-id", LocalID: "local-id"}
			results, err := provider.FetchImportData(context.Background(), sqlmodel.ResourceType, args)
			assert.NoError(t, err)
			assert.Len(t, results, 1)
			imported := results[0]
			assert.Equal(t, "local-id", (*imported.ResourceData)[sqlmodel.IDKey])
			assert.Equal(t, "Imported Model", (*imported.ResourceData)[sqlmodel.DisplayNameKey])
			assert.Equal(t, "desc", (*imported.ResourceData)[sqlmodel.DescriptionKey])
			assert.Equal(t, "id", (*imported.ResourceData)[sqlmodel.PrimaryKeyKey])
			assert.Equal(t, "SELECT * FROM t", (*imported.ResourceData)[sqlmodel.SQLKey])
			assert.Equal(t, "postgres", (*imported.ResourceData)[sqlmodel.SourceDefinitionKey])
			assert.Equal(t, true, (*imported.ResourceData)[sqlmodel.EnabledKey])
			assert.Equal(t, "acc123", (*imported.ResourceData)[sqlmodel.AccountIDKey])
			assert.Equal(t, "local-id", imported.Metadata.Name)
			assert.Equal(t, "remote-id", imported.Metadata.Import.Workspaces[0].Resources[0].RemoteID)
			assert.Equal(t, "local-id", imported.Metadata.Import.Workspaces[0].Resources[0].LocalID)
		})

		t.Run("Unsupported resource type", func(t *testing.T) {
			mockClient := newDefaultMockClient()
			provider := retl.New(mockClient)
			args := importremote.ImportArgs{RemoteID: "remote-id", LocalID: "local-id"}
			results, err := provider.FetchImportData(context.Background(), "unsupported-type", args)
			assert.Error(t, err)
			assert.Nil(t, results)
			assert.Contains(t, err.Error(), "import is only supported for SQL models")
		})

		t.Run("Handler error", func(t *testing.T) {
			mockClient := newDefaultMockClient()
			provider := retl.New(mockClient)
			args := importremote.ImportArgs{RemoteID: "remote-id-not-found", LocalID: "local-id"}
			results, err := provider.FetchImportData(context.Background(), sqlmodel.ResourceType, args)
			assert.Error(t, err)
			assert.Nil(t, results)
			assert.Contains(t, err.Error(), "getting RETL source for import")
		})
	})
}

func TestProviderList(t *testing.T) {
	t.Run("DelegatesToHandler", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			t.Parallel()
			mockClient := newDefaultMockClient()
			provider := retl.New(mockClient)

			// Mock successful listing in the client that the handler will use
			mockClient.listRetlSourcesFunc = func(ctx context.Context, hasExternalID *bool) (*retlClient.RETLSources, error) {
				return &retlClient.RETLSources{
					Data: []retlClient.RETLSource{
						{
							ID:                   "source-1",
							Name:                 "Test Source 1",
							IsEnabled:            true,
							SourceType:           retlClient.ModelSourceType,
							SourceDefinitionName: "postgres",
							AccountID:            "account-1",
							Config: retlClient.RETLSQLModelConfig{
								Description: "Test description 1",
								PrimaryKey:  "id",
								Sql:         "SELECT * FROM table1",
							},
						},
					},
				}, nil
			}

			ctx := context.Background()
			results, err := provider.List(ctx, "retl-source-sql-model", map[string]string{})

			require.NoError(t, err)
			assert.Len(t, results, 1)

			// Verify the handler correctly converted the data
			assert.Equal(t, "source-1", results[0]["id"])
			assert.Equal(t, "Test Source 1", results[0]["name"]) // Handler uses "name" not "display_name"
			// Note: EnabledKey and SourceTypeKey are not set by the current List implementation
			assert.Equal(t, "postgres", results[0]["source_definition"])
			assert.Equal(t, "account-1", results[0]["account_id"])
		})

		t.Run("HandlerError", func(t *testing.T) {
			t.Parallel()
			mockClient := newDefaultMockClient()
			provider := retl.New(mockClient)

			// Mock error from client that the handler will encounter
			mockClient.listRetlSourcesFunc = func(ctx context.Context, hasExternalID *bool) (*retlClient.RETLSources, error) {
				return nil, fmt.Errorf("API error")
			}

			ctx := context.Background()
			results, err := provider.List(ctx, "retl-source-sql-model", map[string]string{})

			assert.Error(t, err)
			assert.Nil(t, results)
			assert.Contains(t, err.Error(), "listing RETL sources")
		})

		t.Run("UnsupportedResourceType", func(t *testing.T) {
			t.Parallel()
			mockClient := newDefaultMockClient()
			provider := retl.New(mockClient)

			ctx := context.Background()
			results, err := provider.List(ctx, "unsupported-type", map[string]string{})

			assert.Error(t, err)
			assert.Nil(t, results)
			assert.Contains(t, err.Error(), "no handler for resource type")
		})
	})
}

func TestProviderPreview(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)

		ctx := context.Background()
		data := resources.ResourceData{
			sqlmodel.SQLKey:              "SELECT 1 as id, 'Alice' as name UNION ALL SELECT 2, 'Bob'",
			sqlmodel.AccountIDKey:        "acc123",
			sqlmodel.SourceDefinitionKey: string(sqlmodel.SourceDefinitionPostgres),
		}

		rows, err := provider.Preview(ctx, "test", sqlmodel.ResourceType, data, 10)
		require.NoError(t, err)
		require.Len(t, rows, 2)
		assert.Equal(t, any(1), rows[0]["id"])
		assert.Equal(t, any("Alice"), rows[0]["name"])
	})

	t.Run("UnsupportedType", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)
		ctx := context.Background()
		_, err := provider.Preview(ctx, "test", "unknown-type", resources.ResourceData{}, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no handler for resource type")
	})

	t.Run("MissingSQL", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)
		ctx := context.Background()
		data := resources.ResourceData{
			sqlmodel.AccountIDKey:        "acc123",
			sqlmodel.SourceDefinitionKey: string(sqlmodel.SourceDefinitionPostgres),
		}
		_, err := provider.Preview(ctx, "test", sqlmodel.ResourceType, data, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SQL not found")
	})

	t.Run("MissingAccountID", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)
		ctx := context.Background()
		data := resources.ResourceData{
			sqlmodel.SQLKey:              "SELECT 1",
			sqlmodel.SourceDefinitionKey: string(sqlmodel.SourceDefinitionPostgres),
		}
		_, err := provider.Preview(ctx, "test", sqlmodel.ResourceType, data, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account ID not found")
	})

	t.Run("SubmitError", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		mockClient.submitPreviewFunc = func(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error) {
			return nil, fmt.Errorf("network down")
		}
		provider := retl.New(mockClient)
		ctx := context.Background()
		data := resources.ResourceData{
			sqlmodel.SQLKey:              "SELECT 1",
			sqlmodel.AccountIDKey:        "acc123",
			sqlmodel.SourceDefinitionKey: string(sqlmodel.SourceDefinitionPostgres),
		}
		_, err := provider.Preview(ctx, "test", sqlmodel.ResourceType, data, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "submitting preview request")
	})

	t.Run("SubmitFailureResponse", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		mockClient.submitPreviewFunc = func(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error) {
			return nil, fmt.Errorf("bad sql")
		}
		provider := retl.New(mockClient)
		ctx := context.Background()
		data := resources.ResourceData{
			sqlmodel.SQLKey:              "SELECT BROKEN",
			sqlmodel.AccountIDKey:        "acc123",
			sqlmodel.SourceDefinitionKey: string(sqlmodel.SourceDefinitionPostgres),
		}
		_, err := provider.Preview(ctx, "test", sqlmodel.ResourceType, data, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "submitting preview request")
	})

	t.Run("ResultFailure", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		mockClient.getPreviewResultFunc = func(ctx context.Context, resultID string) (*retlClient.PreviewResultResponse, error) {
			return &retlClient.PreviewResultResponse{
				Status: retlClient.Failed,
				Error:  "execution error",
			}, nil
		}
		provider := retl.New(mockClient)
		ctx := context.Background()
		data := resources.ResourceData{
			sqlmodel.SQLKey:              "SELECT 1",
			sqlmodel.AccountIDKey:        "acc123",
			sqlmodel.SourceDefinitionKey: string(sqlmodel.SourceDefinitionPostgres),
		}
		_, err := provider.Preview(ctx, "test", sqlmodel.ResourceType, data, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "preview request failed")
	})
}
