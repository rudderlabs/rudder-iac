package retl_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// mockRETLStore mocks the RETL client for testing
type mockRETLStore struct {
	retlClient.RETLStore
	sourceID string
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
				WorkspaceID:          "test-workspace-id",
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
	t.Run("SupportedKinds", func(t *testing.T) {
		t.Parallel()
		provider := retl.New(newDefaultMockClient())
		kinds := provider.SupportedKinds()
		assert.Contains(t, kinds, "retl-source-sql-model")
	})

	t.Run("SupportedTypes", func(t *testing.T) {
		t.Parallel()
		provider := retl.New(newDefaultMockClient())
		types := provider.SupportedTypes()
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
					"id":                "test-model",
					"display_name":      "Test Model",
					"description":       "Test Description",
					"account_id":        "test-account",
					"primary_key":       "id",
					"sql":               "SELECT * FROM users",
					"source_definition": "postgres",
				},
			})
			assert.NoError(t, err)
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
					"id":                "test-model-1",
					"display_name":      "Test Model 1",
					"description":       "Test Description 1",
					"account_id":        "test-account",
					"primary_key":       "id",
					"sql":               "SELECT * FROM users",
					"source_definition": "postgres",
				},
			})
			require.NoError(t, err)

			err = provider.LoadSpec("test2.yaml", &specs.Spec{
				Kind: "retl-source-sql-model",
				Spec: map[string]interface{}{
					"id":                "test-model-2",
					"display_name":      "Test Model 2",
					"description":       "Test Description 2",
					"account_id":        "test-account",
					"primary_key":       "id",
					"sql":               "SELECT * FROM orders",
					"source_definition": "postgres",
				},
			})
			require.NoError(t, err)

			graph, err := provider.ResourceGraph()
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
				"id":                "test-model",
				"display_name":      "Test Model",
				"description":       "Test Description",
				"account_id":        "test-account",
				"primary_key":       "id",
				"sql":               "SELECT * FROM users",
				"source_definition": "postgres",
				"enabled":           true,
			}

			// For update, we need state data with a source_id
			stateData := resources.ResourceData{
				"display_name":      "Test Model",
				"description":       "Test Description",
				"account_id":        "test-account",
				"primary_key":       "id",
				"sql":               "SELECT * FROM users",
				"id":                "test-source-id",
				"source_definition": "postgres",
				"enabled":           true,
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
				err := provider.Validate(nil)
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

			args := specs.ImportIds{RemoteID: "remote-id", LocalID: "local-id"}
			results, err := provider.FetchImportData(context.Background(), sqlmodel.ResourceType, args)
			assert.NoError(t, err)
			assert.Equal(t, writer.FormattableEntity{
				RelativePath: "local-id.yaml",
				Content: &specs.Spec{
					Version: "rudder/v0.1",
					Kind:    "retl-source-sql-model",
					Metadata: map[string]any{
						"name": "local-id",
						"import": map[string]any{
							"workspaces": []any{
								map[string]any{
									"workspace_id": "test-workspace-id",
									"resources": []any{
										map[string]any{
											"local_id":  "local-id",
											"remote_id": "remote-id",
										},
									},
								},
							},
						},
					},
					Spec: map[string]any{
						"id":                "local-id",
						"display_name":      "Imported Model",
						"description":       "desc",
						"primary_key":       "id",
						"sql":               "SELECT * FROM t",
						"source_definition": "postgres",
						"enabled":           true,
						"account_id":        "acc123",
					},
				},
			}, results)
		})

		t.Run("Unsupported resource type", func(t *testing.T) {
			mockClient := newDefaultMockClient()
			provider := retl.New(mockClient)
			args := specs.ImportIds{RemoteID: "remote-id", LocalID: "local-id"}
			results, err := provider.FetchImportData(context.Background(), "unsupported-type", args)
			assert.Error(t, err)
			assert.Equal(t, writer.FormattableEntity{}, results)
			assert.Contains(t, err.Error(), "import is only supported for SQL models")
		})

		t.Run("Handler error", func(t *testing.T) {
			mockClient := newDefaultMockClient()
			provider := retl.New(mockClient)
			args := specs.ImportIds{RemoteID: "remote-id-not-found", LocalID: "local-id"}
			results, err := provider.FetchImportData(context.Background(), sqlmodel.ResourceType, args)
			assert.Error(t, err)
			assert.Equal(t, writer.FormattableEntity{}, results)
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

		t.Run("Success hasExternaId true", func(t *testing.T) {
			t.Parallel()
			mockClient := newDefaultMockClient()
			provider := retl.New(mockClient)

			// Mock successful listing in the client that the handler will use
			mockClient.listRetlSourcesFunc = func(ctx context.Context, hasExternalID *bool) (*retlClient.RETLSources, error) {
				if !assert.True(t, *hasExternalID) {
					return nil, fmt.Errorf("hasExternalID is not true")
				}
				externalId := "ext-123"
				return &retlClient.RETLSources{
					Data: []retlClient.RETLSource{
						{
							ID:                   "source-1",
							Name:                 "Test Source 1",
							IsEnabled:            true,
							SourceType:           retlClient.ModelSourceType,
							SourceDefinitionName: "postgres",
							AccountID:            "account-1",
							ExternalID:           externalId,
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
			results, err := provider.List(ctx, "retl-source-sql-model", map[string]string{"hasExternalId": "true"})

			require.NoError(t, err)
			assert.Len(t, results, 1)
			assert.Equal(t, "source-1", results[0]["id"])
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

func TestProviderParseSpec(t *testing.T) {
	t.Run("UnsupportedKind", func(t *testing.T) {
		t.Parallel()
		provider := retl.New(newDefaultMockClient())
		_, err := provider.ParseSpec("test.yaml", &specs.Spec{
			Kind: "unsupported-kind",
			Spec: map[string]any{"id": "abc"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported kind")
	})

	t.Run("DelegatesToHandlerAndReturnsURNs", func(t *testing.T) {
		t.Parallel()
		provider := retl.New(newDefaultMockClient())
		parsed, err := provider.ParseSpec("test.yaml", &specs.Spec{
			Kind: sqlmodel.ResourceKind,
			Spec: map[string]any{
				"id":                "orders-model",
				"display_name":      "Orders",
				"description":       "desc",
				"sql":               "SELECT 1",
				"account_id":        "acc-1",
				"primary_key":       "id",
				"source_definition": "postgres",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, parsed)
		expectedURN := resources.URN("orders-model", sqlmodel.ResourceType)
		assert.ElementsMatch(t, []string{expectedURN}, parsed.URNs)
		assert.Equal(t, sqlmodel.ResourceType, parsed.LegacyResourceType)
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

func TestProviderLoadResourcesFromRemote(t *testing.T) {
	t.Run("Success collects only resources with ExternalID", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)

		// Prepare two sources: one with ExternalID (should be included), one without (should be skipped)
		externalID := "ext-123"
		mockClient.listRetlSourcesFunc = func(ctx context.Context, hasExternalID *bool) (*retlClient.RETLSources, error) {
			if hasExternalID == nil || !*hasExternalID {
				return nil, fmt.Errorf("expected hasExternalID=true filter")
			}
			return &retlClient.RETLSources{
				Data: []retlClient.RETLSource{
					{
						ID:                   "source-with-ext",
						Name:                 "With External",
						SourceType:           retlClient.ModelSourceType,
						SourceDefinitionName: "postgres",
						AccountID:            "acc-1",
						ExternalID:           externalID,
						Config: retlClient.RETLSQLModelConfig{
							Description: "d1",
							PrimaryKey:  "id",
							Sql:         "SELECT 1",
						},
					},
				},
			}, nil
		}

		ctx := context.Background()
		collection, err := provider.LoadResourcesFromRemote(ctx)
		require.NoError(t, err)
		require.NotNil(t, collection)

		// Only one resource should be present (the one with ExternalID)
		assert.Equal(t, 1, collection.Len())

		// Validate the stored resource
		resourcesMap := collection.GetAll(sqlmodel.ResourceType)
		require.NotNil(t, resourcesMap)
		remoteRes, exists := resourcesMap["source-with-ext"]
		require.True(t, exists, "expected resource with id 'source-with-ext'")
		assert.Equal(t, "ext-123", remoteRes.ExternalID)
		// Ensure Data is the full RETLSource
		_, ok := remoteRes.Data.(retlClient.RETLSource)
		require.True(t, ok, "expected Data to be retlClient.RETLSource")
	})

	t.Run("Handler error surfaces with context", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)
		mockClient.listRetlSourcesFunc = func(ctx context.Context, hasExternalID *bool) (*retlClient.RETLSources, error) {
			return nil, fmt.Errorf("API error")
		}

		ctx := context.Background()
		collection, err := provider.LoadResourcesFromRemote(ctx)
		require.Error(t, err)
		assert.Nil(t, collection)
		assert.Contains(t, err.Error(), "loading retl-source-sql-model")
		assert.Contains(t, err.Error(), "listing RETL sources")
	})
}

func TestProviderMapRemoteToState(t *testing.T) {
	t.Run("Success reconstructs state from collection", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)

		// Build a resource collection with one valid RETLSource carrying ExternalID
		extID := "local-1"
		source := retlClient.RETLSource{
			ID:                   "remote-1",
			Name:                 "Model One",
			SourceType:           retlClient.ModelSourceType,
			SourceDefinitionName: "postgres",
			AccountID:            "acc-1",
			IsEnabled:            true,
			ExternalID:           extID,
			Config: retlClient.RETLSQLModelConfig{
				Description: "desc-1",
				PrimaryKey:  "id",
				Sql:         "SELECT 1",
			},
		}
		collection := resources.NewRemoteResources()
		collection.Set(sqlmodel.ResourceType, map[string]*resources.RemoteResource{
			source.ID: {
				ID:         source.ID,
				ExternalID: source.ExternalID,
				Data:       source,
			},
		})

		st, err := provider.MapRemoteToState(collection)
		require.NoError(t, err)
		require.NotNil(t, st)

		// Validate state contains resource keyed by local (external) id
		rs := st.GetResource("retl-source-sql-model:" + extID)
		require.NotNil(t, rs)
		assert.Equal(t, sqlmodel.ResourceType, rs.Type)
		assert.Equal(t, extID, rs.ID)
		// Inputs reconstructed from source
		assert.Equal(t, any("Model One"), rs.Input[sqlmodel.DisplayNameKey])
		assert.Equal(t, any("desc-1"), rs.Input[sqlmodel.DescriptionKey])
		assert.Equal(t, any("acc-1"), rs.Input[sqlmodel.AccountIDKey])
		assert.Equal(t, any("id"), rs.Input[sqlmodel.PrimaryKeyKey])
		assert.Equal(t, any("SELECT 1"), rs.Input[sqlmodel.SQLKey])
		assert.Equal(t, any(true), rs.Input[sqlmodel.EnabledKey])
		assert.Equal(t, any("postgres"), rs.Input[sqlmodel.SourceDefinitionKey])
		assert.Equal(t, any(extID), rs.Input[sqlmodel.LocalIDKey])
		// Outputs come from toResourceData
		assert.Equal(t, any("remote-1"), rs.Output[sqlmodel.IDKey])
		assert.Equal(t, any(retlClient.ModelSourceType), rs.Output[sqlmodel.SourceTypeKey])
	})

	t.Run("Error when handler fails to cast resource data", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)

		// Insert a malformed resource (Data is wrong type)
		collection := resources.NewRemoteResources()
		collection.Set(sqlmodel.ResourceType, map[string]*resources.RemoteResource{
			"bad-1": {
				ID:         "bad-1",
				ExternalID: "local-bad",
				Data:       "not a retlClient.RETLSource",
			},
		})

		st, err := provider.MapRemoteToState(collection)
		require.Error(t, err)
		assert.Nil(t, st)
		// Wrapped error should mention provider handler and inner cast error
		assert.Contains(t, err.Error(), "loading state from provider handler retl-source-sql-model")
		assert.Contains(t, err.Error(), "unable to cast resource to retl source")
	})
}

func TestProviderLoadImportable(t *testing.T) {
	t.Run("Success generates external IDs and references", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)

		// Mock list to return sources that do NOT have ExternalID yet
		mockClient.listRetlSourcesFunc = func(ctx context.Context, hasExternalID *bool) (*retlClient.RETLSources, error) {
			// Expect false for hasExternalID
			require.NotNil(t, hasExternalID)
			require.False(t, *hasExternalID)
			return &retlClient.RETLSources{
				Data: []retlClient.RETLSource{
					{
						ID:                   "source-1",
						Name:                 "Orders Model",
						SourceType:           retlClient.ModelSourceType,
						SourceDefinitionName: "postgres",
						AccountID:            "acc-1",
						WorkspaceID:          "ws-1",
						Config: retlClient.RETLSQLModelConfig{
							Description: "desc-1",
							PrimaryKey:  "id",
							Sql:         "SELECT 1",
						},
					},
					{
						ID:                   "source-2",
						Name:                 "Users Model",
						SourceType:           retlClient.ModelSourceType,
						SourceDefinitionName: "postgres",
						AccountID:            "acc-1",
						WorkspaceID:          "ws-1",
						Config: retlClient.RETLSQLModelConfig{
							Description: "desc-2",
							PrimaryKey:  "user_id",
							Sql:         "SELECT 2",
						},
					},
				},
			}, nil
		}

		ctx := context.Background()
		idNamer := namer.NewExternalIdNamer(namer.StrategyKebabCase)
		collection, err := provider.LoadImportable(ctx, idNamer)
		require.NoError(t, err)
		require.NotNil(t, collection)

		// Two items imported
		m := collection.GetAll(sqlmodel.ResourceType)
		require.Len(t, m, 2)

		// External IDs should be kebab-cased names
		s1, ok := collection.GetByID(sqlmodel.ResourceType, "source-1")
		require.True(t, ok)
		assert.Equal(t, "orders-model", s1.ExternalID)
		assert.Equal(t, "#/retl-source-sql-model/retl-source-sql-model/orders-model", s1.Reference)

		s2, ok := collection.GetByID(sqlmodel.ResourceType, "source-2")
		require.True(t, ok)
		assert.Equal(t, "users-model", s2.ExternalID)
		assert.Equal(t, "#/retl-source-sql-model/retl-source-sql-model/users-model", s2.Reference)
	})

	t.Run("Handler error surfaces with context", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)
		mockClient.listRetlSourcesFunc = func(ctx context.Context, hasExternalID *bool) (*retlClient.RETLSources, error) {
			return nil, fmt.Errorf("API error")
		}

		ctx := context.Background()
		idNamer := namer.NewExternalIdNamer(namer.StrategyKebabCase)
		collection, err := provider.LoadImportable(ctx, idNamer)
		require.Error(t, err)
		assert.Nil(t, collection)
		assert.Contains(t, err.Error(), "loading importable resources from handler")
	})
}

// minimal resolver implementation; not used by current FormatForExport flow
type noopResolver struct{}

func (n noopResolver) ResolveToReference(entityType string, remoteID string) (string, error) {
	return "", nil
}

func TestProviderFormatForExport(t *testing.T) {
	t.Run("Formats entities with correct paths and metadata", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)

		// Prepare a collection similar to LoadImportable output
		src := retlClient.RETLSource{
			ID:                   "remote-1",
			Name:                 "Orders Model",
			SourceType:           retlClient.ModelSourceType,
			SourceDefinitionName: "postgres",
			AccountID:            "acc-1",
			WorkspaceID:          "ws-1",
			IsEnabled:            true,
			Config: retlClient.RETLSQLModelConfig{
				Description: "desc-1",
				PrimaryKey:  "id",
				Sql:         "SELECT 1",
			},
		}
		collection := resources.NewRemoteResources()
		collection.Set(sqlmodel.ResourceType, map[string]*resources.RemoteResource{
			src.ID: {
				ID:         src.ID,
				ExternalID: "orders-model",
				Data:       &src,
			},
		})

		idNamer := namer.NewExternalIdNamer(namer.StrategyKebabCase)
		entities, err := provider.FormatForExport(collection, idNamer, noopResolver{})
		require.NoError(t, err)
		require.Len(t, entities, 1)

		// Validate relative path and core spec content
		assert.Equal(t, filepath.Join("retl", sqlmodel.ImportPath, "orders-model.yaml"), entities[0].RelativePath)

		spec, ok := entities[0].Content.(*specs.Spec)
		require.True(t, ok)
		assert.Equal(t, sqlmodel.ResourceKind, spec.Kind)
		assert.Equal(t, any("Orders Model"), spec.Spec[sqlmodel.DisplayNameKey])
		assert.Equal(t, any("desc-1"), spec.Spec[sqlmodel.DescriptionKey])
		assert.Equal(t, any("acc-1"), spec.Spec[sqlmodel.AccountIDKey])
		assert.Equal(t, any("id"), spec.Spec[sqlmodel.PrimaryKeyKey])
		assert.Equal(t, any("SELECT 1"), spec.Spec[sqlmodel.SQLKey])
		assert.Equal(t, any("postgres"), spec.Spec[sqlmodel.SourceDefinitionKey])
		assert.Equal(t, any(true), spec.Spec[sqlmodel.EnabledKey])
		assert.Equal(t, any("orders-model"), spec.Spec[sqlmodel.IDKey])
	})

	t.Run("Handler cast error is wrapped by provider", func(t *testing.T) {
		t.Parallel()
		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)

		// Malformed data type inside collection
		collection := resources.NewRemoteResources()
		collection.Set(sqlmodel.ResourceType, map[string]*resources.RemoteResource{
			"bad": {
				ID:         "bad",
				ExternalID: "bad-external",
				Data:       "not a pointer to retlClient.RETLSource",
			},
		})

		idNamer := namer.NewExternalIdNamer(namer.StrategyKebabCase)
		entities, err := provider.FormatForExport(collection, idNamer, noopResolver{})
		require.Error(t, err)
		assert.Nil(t, entities)
		assert.Contains(t, err.Error(), "formatting for export for handler")
	})
}

func TestProviderMigrateSpec(t *testing.T) {
	t.Run("Migrates import metadata from LocalID to URN format", func(t *testing.T) {
		t.Parallel()

		spec := &specs.Spec{
			Kind: "retl-source-sql-model",
			Metadata: map[string]any{
				"name": "test-model",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "ws-123",
							"resources": []any{
								map[string]any{
									"local_id":  "model1",
									"remote_id": "remote-model-1",
								},
								map[string]any{
									"local_id":  "model2",
									"remote_id": "remote-model-2",
								},
							},
						},
					},
				},
			},
			Spec: map[string]any{
				"id":                "test-model",
				"display_name":      "Test Model",
				"description":       "Test description",
				"sql":               "SELECT * FROM users",
				"account_id":        "acc123",
				"primary_key":       "id",
				"source_definition": "postgres",
			},
		}

		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)

		migratedSpec, err := provider.MigrateSpec(spec)
		require.NoError(t, err)
		require.NotNil(t, migratedSpec)

		metadata, err := migratedSpec.CommonMetadata()
		require.NoError(t, err)
		require.NotNil(t, metadata.Import)
		require.Len(t, metadata.Import.Workspaces, 1)
		require.Len(t, metadata.Import.Workspaces[0].Resources, 2)

		// Verify URN field is set and LocalID is cleared
		assert.Equal(t, "retl-source-sql-model:model1", metadata.Import.Workspaces[0].Resources[0].URN)
		assert.Equal(t, "", metadata.Import.Workspaces[0].Resources[0].LocalID)
		assert.Equal(t, "remote-model-1", metadata.Import.Workspaces[0].Resources[0].RemoteID)

		assert.Equal(t, "retl-source-sql-model:model2", metadata.Import.Workspaces[0].Resources[1].URN)
		assert.Equal(t, "", metadata.Import.Workspaces[0].Resources[1].LocalID)
		assert.Equal(t, "remote-model-2", metadata.Import.Workspaces[0].Resources[1].RemoteID)
	})

	t.Run("Skips migration when URN is already set", func(t *testing.T) {
		t.Parallel()

		spec := &specs.Spec{
			Kind: "retl-source-sql-model",
			Metadata: map[string]any{
				"name": "test-model",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "ws-123",
							"resources": []any{
								map[string]any{
									"urn":       "retl-source-sql-model:model1",
									"remote_id": "remote-model-1",
								},
							},
						},
					},
				},
			},
			Spec: map[string]any{
				"id":                "test-model",
				"display_name":      "Test Model",
				"sql":               "SELECT * FROM users",
				"account_id":        "acc123",
				"primary_key":       "id",
				"source_definition": "postgres",
			},
		}

		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)

		migratedSpec, err := provider.MigrateSpec(spec)
		require.NoError(t, err)
		require.NotNil(t, migratedSpec)

		metadata, err := migratedSpec.CommonMetadata()
		require.NoError(t, err)
		require.NotNil(t, metadata.Import)

		// URN should remain unchanged, LocalID should remain empty
		assert.Equal(t, "retl-source-sql-model:model1", metadata.Import.Workspaces[0].Resources[0].URN)
		assert.Equal(t, "", metadata.Import.Workspaces[0].Resources[0].LocalID)
		assert.Equal(t, "remote-model-1", metadata.Import.Workspaces[0].Resources[0].RemoteID)
	})

	t.Run("Handles spec with no import metadata", func(t *testing.T) {
		t.Parallel()

		spec := &specs.Spec{
			Kind: "retl-source-sql-model",
			Metadata: map[string]any{
				"name": "test-model",
			},
			Spec: map[string]any{
				"id":                "test-model",
				"display_name":      "Test Model",
				"sql":               "SELECT * FROM users",
				"account_id":        "acc123",
				"primary_key":       "id",
				"source_definition": "postgres",
			},
		}

		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)

		migratedSpec, err := provider.MigrateSpec(spec)
		require.NoError(t, err)
		require.NotNil(t, migratedSpec)

		metadata, err := migratedSpec.CommonMetadata()
		require.NoError(t, err)
		assert.Nil(t, metadata.Import)
	})

	t.Run("Returns spec unchanged for unsupported kind", func(t *testing.T) {
		t.Parallel()

		spec := &specs.Spec{
			Kind: "unsupported-kind",
			Metadata: map[string]any{
				"name": "test",
			},
			Spec: map[string]any{
				"id": "test",
			},
		}

		mockClient := newDefaultMockClient()
		provider := retl.New(mockClient)

		migratedSpec, err := provider.MigrateSpec(spec)
		require.NoError(t, err)
		require.NotNil(t, migratedSpec)
		assert.Equal(t, spec, migratedSpec)
	})
}
