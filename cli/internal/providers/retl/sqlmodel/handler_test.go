package sqlmodel_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

// createTestRETLSourceWithConfig creates a test RETL source with custom config
func createTestRETLSourceWithConfig(id, name, sourceDefn, accountID string, enabled bool, config retlClient.RETLSQLModelConfig) retlClient.RETLSource {
	return retlClient.RETLSource{
		ID:                   id,
		Name:                 name,
		IsEnabled:            enabled,
		SourceType:           retlClient.ModelSourceType,
		SourceDefinitionName: sourceDefn,
		AccountID:            accountID,
		Config:               config,
	}
}

// createTestResourceData creates test resource data with common defaults
func createTestResourceData(id, displayName, description, sql string) resources.ResourceData {
	return resources.ResourceData{
		"id":                id,
		"display_name":      displayName,
		"description":       description,
		"sql":               sql,
		"account_id":        "acc123",
		"primary_key":       "id",
		"source_definition": "postgres",
		"enabled":           true,
	}
}

// createTestSpec creates a test spec with common defaults
func createTestSpec(id, displayName, description, sql string) *specs.Spec {
	return &specs.Spec{
		Version: "rudder/v0.1",
		Kind:    "retl-source-sql-model",
		Spec: map[string]interface{}{
			"id":                id,
			"display_name":      displayName,
			"description":       description,
			"sql":               sql,
			"account_id":        "acc123",
			"primary_key":       "id",
			"source_definition": "postgres",
			"enabled":           true,
		},
	}
}

// createTestSpecMap creates a test spec map with custom fields
func createTestSpecMap(fields map[string]interface{}) *specs.Spec {
	return &specs.Spec{
		Version: "rudder/v0.1",
		Kind:    "retl-source-sql-model",
		Spec:    fields,
	}
}

// mockListRetlSources creates a mock list function that returns the given sources
func mockListRetlSources(sources ...retlClient.RETLSource) func(ctx context.Context) (*retlClient.RETLSources, error) {
	return func(ctx context.Context) (*retlClient.RETLSources, error) {
		return &retlClient.RETLSources{Data: sources}, nil
	}
}

// mockRETLClient is a mock implementation of the RETL client
type mockRETLClient struct {
	createCalled               bool
	updateCalled               bool
	deleteCalled               bool
	sourceID                   string
	deleteError                bool
	updateError                bool
	createRetlSourceFunc       func(ctx context.Context, req *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error)
	updateRetlSourceFunc       func(ctx context.Context, sourceID string, req *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error)
	listRetlSourcesFunc        func(ctx context.Context) (*retlClient.RETLSources, error)
	getRetlSourceFunc          func(ctx context.Context, sourceID string) (*retlClient.RETLSource, error)
	readStateFunc              func(ctx context.Context) (*retlClient.State, error)
	submitSourcePreviewFunc    func(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error)
	getSourcePreviewResultFunc func(ctx context.Context, resultID string) (*retlClient.PreviewResultResponse, error)
	setExternalIdFunc          func(ctx context.Context, sourceID string, externalId string) error
}

func (m *mockRETLClient) CreateRetlSource(ctx context.Context, req *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
	m.createCalled = true

	if m.createRetlSourceFunc != nil {
		return m.createRetlSourceFunc(ctx, req)
	}

	return &retlClient.RETLSource{
		ID:                   m.sourceID,
		Name:                 req.Name,
		Config:               req.Config,
		SourceType:           req.SourceType,
		SourceDefinitionName: req.SourceDefinitionName,
		AccountID:            req.AccountID,
		IsEnabled:            true,
	}, nil
}

func (m *mockRETLClient) GetRetlSource(ctx context.Context, sourceID string) (*retlClient.RETLSource, error) {
	if m.getRetlSourceFunc != nil {
		return m.getRetlSourceFunc(ctx, sourceID)
	}
	return &retlClient.RETLSource{
		ID:                   sourceID,
		Name:                 "Test Model",
		Config:               retlClient.RETLSQLModelConfig{},
		SourceType:           "model",
		SourceDefinitionName: "postgres",
		AccountID:            "acc123",
		IsEnabled:            true,
	}, nil
}

func (m *mockRETLClient) UpdateRetlSource(ctx context.Context, sourceID string, req *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error) {
	m.updateCalled = true

	if m.updateRetlSourceFunc != nil {
		return m.updateRetlSourceFunc(ctx, sourceID, req)
	}

	if m.updateError {
		return nil, fmt.Errorf("updating RETL source")
	}
	return &retlClient.RETLSource{
		Name:                 req.Name,
		Config:               req.Config,
		SourceType:           "model",
		SourceDefinitionName: "postgres",
		AccountID:            req.AccountID,
		IsEnabled:            req.IsEnabled,
	}, nil
}

func (m *mockRETLClient) DeleteRetlSource(ctx context.Context, sourceID string) error {
	m.deleteCalled = true
	if m.deleteError {
		return errors.New("deleting RETL source")
	}
	return nil
}

func (m *mockRETLClient) ListRetlSources(ctx context.Context, hasExternalID *bool) (*retlClient.RETLSources, error) {
	if m.listRetlSourcesFunc != nil {
		return m.listRetlSourcesFunc(ctx)
	}
	return &retlClient.RETLSources{
		Data: []retlClient.RETLSource{
			{
				ID:                   m.sourceID,
				Name:                 "Test Model",
				Config:               retlClient.RETLSQLModelConfig{},
				SourceType:           "model",
				SourceDefinitionName: "postgres",
				AccountID:            "acc123",
				IsEnabled:            true,
			},
		},
	}, nil
}

func (m *mockRETLClient) ReadState(ctx context.Context) (*retlClient.State, error) {
	if m.readStateFunc != nil {
		return m.readStateFunc(ctx)
	}
	return &retlClient.State{
		Resources: map[string]retlClient.ResourceState{},
	}, nil
}

func (m *mockRETLClient) PutResourceState(ctx context.Context, id string, req retlClient.PutStateRequest) error {
	return nil
}

func (m *mockRETLClient) SubmitSourcePreview(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error) {
	if m.submitSourcePreviewFunc != nil {
		return m.submitSourcePreviewFunc(ctx, request)
	}
	return &retlClient.PreviewSubmitResponse{
		ID: "req-123",
	}, nil
}

func (m *mockRETLClient) GetSourcePreviewResult(ctx context.Context, resultID string) (*retlClient.PreviewResultResponse, error) {
	if m.getSourcePreviewResultFunc != nil {
		return m.getSourcePreviewResultFunc(ctx, resultID)
	}
	return &retlClient.PreviewResultResponse{
		Status: retlClient.Completed,
		Rows:   []map[string]any{},
	}, nil
}

func (m *mockRETLClient) SetExternalId(ctx context.Context, sourceID string, externalId string) error {
	if m.setExternalIdFunc != nil {
		return m.setExternalIdFunc(ctx, sourceID, externalId)
	}
	return nil
}

func TestSQLModelHandler(t *testing.T) {
	t.Parallel()

	t.Run("ParseSpec", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name          string
			spec          *specs.Spec
			expectedIDs   []string
			expectedError bool
			errorContains string
		}{
			{
				name: "success - parse spec with id",
				spec: &specs.Spec{
					Kind: "retl-source-sql-model",
					Spec: map[string]any{
						"id":                "test-model",
						"display_name":      "Test Model",
						"source_definition": "postgres",
					},
				},
				expectedIDs:   []string{"test-model"},
				expectedError: false,
			},
			{
				name: "error - id not found in spec",
				spec: &specs.Spec{
					Kind: "retl-source-sql-model",
					Spec: map[string]any{
						"display_name":      "Test Model",
						"source_definition": "postgres",
					},
				},
				expectedIDs:   nil,
				expectedError: true,
				errorContains: "id not found in sql model spec",
			},
			{
				name: "error - id is not a string",
				spec: &specs.Spec{
					Kind: "retl-source-sql-model",
					Spec: map[string]any{
						"id":                123,
						"display_name":      "Test Model",
						"source_definition": "postgres",
					},
				},
				expectedIDs:   nil,
				expectedError: true,
				errorContains: "id not found in sql model spec",
			},
			{
				name: "error - empty spec",
				spec: &specs.Spec{
					Kind: "retl-source-sql-model",
					Spec: map[string]any{},
				},
				expectedIDs:   nil,
				expectedError: true,
				errorContains: "id not found in sql model spec",
			},
		}

		for _, tc := range cases {
			tc := tc // capture range variable
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				mockClient := &mockRETLClient{}
				handler := sqlmodel.NewHandler(mockClient, "retl")

				parsedSpec, err := handler.ParseSpec("test/path.yaml", tc.spec)

				if tc.expectedError {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tc.errorContains)
					assert.Nil(t, parsedSpec)
				} else {
					require.NoError(t, err)
					require.NotNil(t, parsedSpec)
					assert.Equal(t, tc.expectedIDs, parsedSpec.ExternalIDs)
				}
			})
		}
	})

	t.Run("LoadSpec", func(t *testing.T) {
		t.Parallel()

		// Create a temporary SQL file for testing
		tmpDir := t.TempDir()
		sqlFile := filepath.Join(tmpDir, "test.sql")
		err := os.WriteFile(sqlFile, []byte("SELECT * FROM test_table"), 0644)
		require.NoError(t, err)

		// Test cases
		testCases := []struct {
			name          string
			spec          *specs.Spec
			expectedError bool
			errorMessage  string
		}{
			{
				name: "Valid spec with SQL",
				spec: &specs.Spec{
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
				expectedError: false,
			},
			{
				name: "Valid spec with file path",
				spec: &specs.Spec{
					Version: "rudder/v0.1",
					Kind:    "retl-source-sql-model",
					Spec: map[string]interface{}{
						"id":                "test-model",
						"display_name":      "Test Model",
						"description":       "Test description",
						"file":              sqlFile,
						"account_id":        "acc123",
						"primary_key":       "id",
						"source_definition": "postgres",
						"enabled":           true,
					},
				},
				expectedError: false,
			},
			{
				name: "Missing SQL and File",
				spec: &specs.Spec{
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
				expectedError: true,
				errorMessage:  "sql or file must be specified",
			},
			{
				name: "Both SQL and File specified",
				spec: &specs.Spec{
					Version: "rudder/v0.1",
					Kind:    "retl-source-sql-model",
					Spec: map[string]interface{}{
						"id":                "test-model",
						"display_name":      "Test Model",
						"description":       "Test description",
						"sql":               "SELECT * FROM users",
						"file":              sqlFile,
						"account_id":        "acc123",
						"primary_key":       "id",
						"source_definition": "postgres",
						"enabled":           true,
					},
				},
				expectedError: true,
				errorMessage:  "sql and file cannot be specified together",
			},
			{
				name: "Invalid file path",
				spec: &specs.Spec{
					Version: "rudder/v0.1",
					Kind:    "retl-source-sql-model",
					Spec: map[string]interface{}{
						"id":                "test-model",
						"display_name":      "Test Model",
						"description":       "Test description",
						"file":              "nonexistent.sql",
						"account_id":        "acc123",
						"primary_key":       "id",
						"source_definition": "postgres",
						"enabled":           true,
					},
				},
				expectedError: true,
				errorMessage:  "reading SQL file",
			},
			{
				name: "Enabled is false",
				spec: &specs.Spec{
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
						"enabled":           false,
					},
				},
				expectedError: false,
			},
			{
				name: "Enabled is missing",
				spec: &specs.Spec{
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
					},
				},
				expectedError: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Setup
				mockClient := &mockRETLClient{sourceID: "src123"}
				handler := sqlmodel.NewHandler(mockClient, "retl")

				// Convert spec to JSON and back to simulate loading from file
				specJSON, err := json.Marshal(tc.spec.Spec)
				require.NoError(t, err)

				var specMap map[string]interface{}
				err = json.Unmarshal(specJSON, &specMap)
				require.NoError(t, err)

				tc.spec.Spec = specMap

				// Execute
				err = handler.LoadSpec("test.yaml", tc.spec)

				// Verify
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

	// Preview workflow tests
	t.Run("Preview", func(t *testing.T) {
		// common valid data
		validData := resources.ResourceData{
			sqlmodel.SQLKey:              "SELECT id, name FROM users",
			sqlmodel.AccountIDKey:        "acc123",
			sqlmodel.SourceDefinitionKey: "postgres",
		}

		t.Run("Success with polling", func(t *testing.T) {
			t.Parallel()

			mockClient := &mockRETLClient{}
			// Submit returns success with request id
			mockClient.submitSourcePreviewFunc = func(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error) {
				resp := &retlClient.PreviewSubmitResponse{}
				resp.ID = "req-123"
				return resp, nil
			}

			// First call RUNNING, second call returns result
			call := 0
			mockClient.getSourcePreviewResultFunc = func(ctx context.Context, resultID string) (*retlClient.PreviewResultResponse, error) {
				call++
				if call == 1 {
					resp := &retlClient.PreviewResultResponse{}
					resp.Status = retlClient.Pending
					return resp, nil
				}
				resp := &retlClient.PreviewResultResponse{}
				resp.Status = retlClient.Completed
				resp.Rows = []map[string]any{
					{"id": 1, "name": "alice"},
					{"id": 2, "name": "bob"},
				}
				return resp, nil
			}

			h := sqlmodel.NewHandler(mockClient, "retl")
			rows, err := h.Preview(context.Background(), "id", validData, 10)

			require.NoError(t, err)
			require.Len(t, rows, 2)
			assert.Equal(t, any(1), rows[0]["id"])
			assert.Equal(t, any("alice"), rows[0]["name"])
		})

		t.Run("Submit API error", func(t *testing.T) {
			t.Parallel()
			mockClient := &mockRETLClient{}
			mockClient.submitSourcePreviewFunc = func(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error) {
				return nil, fmt.Errorf("boom")
			}
			h := sqlmodel.NewHandler(mockClient, "retl")
			rows, err := h.Preview(context.Background(), "id", validData, 5)
			assert.Error(t, err)
			assert.Nil(t, rows)
			assert.Contains(t, err.Error(), "submitting preview request")
		})

		t.Run("Submit success=false with message", func(t *testing.T) {
			t.Parallel()
			mockClient := &mockRETLClient{}
			mockClient.submitSourcePreviewFunc = func(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error) {
				return nil, fmt.Errorf("bad query")
			}
			h := sqlmodel.NewHandler(mockClient, "retl")
			_, err := h.Preview(context.Background(), "id", validData, 5)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "submitting preview request")
		})

		t.Run("Result API error", func(t *testing.T) {
			t.Parallel()
			mockClient := &mockRETLClient{}
			mockClient.submitSourcePreviewFunc = func(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error) {
				resp := &retlClient.PreviewSubmitResponse{}
				resp.ID = "req-1"
				return resp, nil
			}
			mockClient.getSourcePreviewResultFunc = func(ctx context.Context, resultID string) (*retlClient.PreviewResultResponse, error) {
				return nil, fmt.Errorf("network")
			}
			h := sqlmodel.NewHandler(mockClient, "retl")
			_, err := h.Preview(context.Background(), "id", validData, 5)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "getting preview results")
		})

		t.Run("Result failure with error message", func(t *testing.T) {
			t.Parallel()
			mockClient := &mockRETLClient{}
			mockClient.submitSourcePreviewFunc = func(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error) {
				resp := &retlClient.PreviewSubmitResponse{}
				resp.ID = "req-1"
				return resp, nil
			}
			mockClient.getSourcePreviewResultFunc = func(ctx context.Context, resultID string) (*retlClient.PreviewResultResponse, error) {
				resp := &retlClient.PreviewResultResponse{}
				resp.Status = retlClient.Failed
				resp.Error = "syntax error"
				return resp, nil
			}
			h := sqlmodel.NewHandler(mockClient, "retl")
			_, err := h.Preview(context.Background(), "id", validData, 5)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "preview request failed: syntax error")
		})

		t.Run("Missing SQL", func(t *testing.T) {
			t.Parallel()
			data := resources.ResourceData{
				sqlmodel.AccountIDKey:        "acc123",
				sqlmodel.SourceDefinitionKey: "postgres",
			}
			h := sqlmodel.NewHandler(&mockRETLClient{}, "retl")
			_, err := h.Preview(context.Background(), "id", data, 5)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "SQL not found")
		})

		t.Run("Missing AccountID", func(t *testing.T) {
			t.Parallel()
			data := resources.ResourceData{
				sqlmodel.SQLKey:              "SELECT 1",
				sqlmodel.SourceDefinitionKey: "postgres",
			}
			h := sqlmodel.NewHandler(&mockRETLClient{}, "retl")
			_, err := h.Preview(context.Background(), "id", data, 5)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "account ID not found")
		})
	})

	t.Run("LoadSpec with duplicate id", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockRETLClient{sourceID: "src123"}
		handler := sqlmodel.NewHandler(mockClient, "retl")

		err := handler.LoadSpec("test.yaml", createTestSpec("test-model", "Test Model", "Test description", "SELECT * FROM users"))
		require.NoError(t, err)

		err = handler.LoadSpec("test.yaml", createTestSpec("test-model", "Test Model", "Test description", "SELECT * FROM users"))

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sql model with id test-model already exists")
	})

	t.Run("LoadSpec with invalid spec structure", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockRETLClient{sourceID: "src123"}
		handler := sqlmodel.NewHandler(mockClient, "retl")

		// Create a spec with invalid structure that will cause mapstructure.Decode to fail
		invalidSpec := createTestSpecMap(map[string]interface{}{
			"id":      123,          // ID should be a string, not an int
			"enabled": "not-a-bool", // Enabled should be a bool
		})

		err := handler.LoadSpec("test.yaml", invalidSpec)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "converting spec")
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockRETLClient{sourceID: "src123"}
		handler := sqlmodel.NewHandler(mockClient, "retl")

		handler.LoadSpec("test.yaml", createTestSpec("test-model", "Test Model", "Test description", "SELECT * FROM users"))

		err := handler.Validate()

		assert.NoError(t, err)
	})

	t.Run("Validate with invalid resource", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockRETLClient{sourceID: "src123"}
		handler := sqlmodel.NewHandler(mockClient, "retl")

		err := handler.LoadSpec("test.yaml", createTestSpecMap(map[string]interface{}{
			"id":           "test-model",
			"display_name": "Test Model",
			"sql":          "SELECT * FROM users",
			// Missing description, account_id, primary_key, source_definition
		}))
		require.NoError(t, err)

		err = handler.Validate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validating sql model spec")
	})

	t.Run("GetResources", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockRETLClient{sourceID: "src123"}
		handler := sqlmodel.NewHandler(mockClient, "retl")
		handler.LoadSpec("test.yaml", createTestSpec("test-model", "Test Model", "Test description", "SELECT * FROM users"))
		handler.LoadSpec("test-2.yaml", createTestSpec("test-model-2", "Test Model 2", "Test description 2", "SELECT * FROM users"))

		resources, err := handler.GetResources()

		assert.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, sqlmodel.ResourceType, resources[0].Type())
		assert.Equal(t, sqlmodel.ResourceType, resources[1].Type())
		assert.Equal(t, true, resources[0].Data()[sqlmodel.EnabledKey])
		// Enabled should be true by default
		assert.Equal(t, true, resources[1].Data()[sqlmodel.EnabledKey])
	})

	t.Run("Create", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockRETLClient{sourceID: "src123"}
		handler := sqlmodel.NewHandler(mockClient, "retl")

		handler.LoadSpec("test.yaml", createTestSpec("test-model", "Test Model", "Test description", "SELECT * FROM users"))
		data := createTestResourceData("test-model", "Test Model", "Test description", "SELECT * FROM users")

		result, err := handler.Create(context.Background(), "test-model", data)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, mockClient.createCalled)
		assert.Equal(t, "src123", (*result)[sqlmodel.IDKey])
	})

	t.Run("Create with API error", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockRETLClient{
			sourceID: "src123",
			createRetlSourceFunc: func(ctx context.Context, req *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
				return nil, fmt.Errorf("API error creating source")
			},
		}
		handler := sqlmodel.NewHandler(mockClient, "retl")
		data := createTestResourceData("test-model", "Test Model", "Test description", "SELECT * FROM users")

		result, err := handler.Create(context.Background(), "test-model", data)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "creating RETL source")
	})

	t.Run("Create with timestamps", func(t *testing.T) {
		t.Parallel()

		createdAt := time.Now().Add(-24 * time.Hour)
		updatedAt := time.Now()

		mockClient := &mockRETLClient{
			sourceID: "src123",
			createRetlSourceFunc: func(ctx context.Context, req *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
				return &retlClient.RETLSource{
					ID:                   "src123",
					Name:                 req.Name,
					Config:               req.Config,
					SourceType:           req.SourceType,
					SourceDefinitionName: req.SourceDefinitionName,
					AccountID:            req.AccountID,
					IsEnabled:            true,
					CreatedAt:            &createdAt,
					UpdatedAt:            &updatedAt,
				}, nil
			},
		}
		handler := sqlmodel.NewHandler(mockClient, "retl")
		data := createTestResourceData("test-model", "Test Model", "Test description", "SELECT * FROM users")

		result, err := handler.Create(context.Background(), "test-model", data)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, &createdAt, (*result)[sqlmodel.CreatedAtKey])
		assert.Equal(t, &updatedAt, (*result)[sqlmodel.UpdatedAtKey])
	})

	t.Run("Update", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name          string
			data          resources.ResourceData
			state         resources.ResourceData
			expectedError bool
			errorMessage  string
			mockSetup     func() *mockRETLClient
		}{
			{
				name: "Valid update",
				data: resources.ResourceData{
					"display_name":      "Updated Model",
					"description":       "Updated description",
					"sql":               "SELECT id, name, timestamp FROM updated",
					"account_id":        "acc123",
					"primary_key":       "id",
					"source_definition": "postgres",
					"enabled":           true,
				},
				state: resources.ResourceData{
					sqlmodel.IDKey:               "src123",
					sqlmodel.EnabledKey:          true,
					sqlmodel.SourceDefinitionKey: "postgres",
					sqlmodel.PrimaryKeyKey:       "id",
					sqlmodel.AccountIDKey:        "acc123",
					sqlmodel.DisplayNameKey:      "Updated Model",
					sqlmodel.DescriptionKey:      "Updated description",
					sqlmodel.SQLKey:              "SELECT * FROM updated",
				},
				expectedError: false,
				mockSetup: func() *mockRETLClient {
					return &mockRETLClient{sourceID: "src123"}
				},
			},
			{
				name: "Missing source_id",
				data: resources.ResourceData{
					"display_name": "Updated Model",
					"description":  "Updated description",
					"sql":          "SELECT * FROM updated",
					"account_id":   "acc123",
					"primary_key":  "id",
					"enabled":      true,
				},
				state:         resources.ResourceData{},
				expectedError: true,
				errorMessage:  fmt.Sprintf("missing %s in resource state", sqlmodel.IDKey),
				mockSetup: func() *mockRETLClient {
					return &mockRETLClient{}
				},
			},
			{
				name: "API error",
				data: resources.ResourceData{
					"display_name": "Error Model",
					"description":  "Error description",
					"sql":          "SELECT * FROM error",
					"account_id":   "acc123",
					"primary_key":  "id",
					"enabled":      true,
				},
				state: resources.ResourceData{
					sqlmodel.IDKey: "error",
				},
				expectedError: true,
				errorMessage:  "updating RETL source",
				mockSetup: func() *mockRETLClient {
					return &mockRETLClient{sourceID: "error", updateError: true}
				},
			},
			{
				name: "Source definition name cannot be changed",
				data: resources.ResourceData{
					"source_definition": "redshift",
					"enabled":           true,
					"display_name":      "Updated Model",
					"description":       "Updated description",
					"sql":               "SELECT id, name, timestamp FROM updated",
					"account_id":        "acc123",
					"primary_key":       "id",
				},
				state: resources.ResourceData{
					sqlmodel.SourceDefinitionKey: "postgres",
					sqlmodel.IDKey:               "src123",
					sqlmodel.EnabledKey:          true,
					sqlmodel.DisplayNameKey:      "Updated Model",
					sqlmodel.DescriptionKey:      "Updated description",
					sqlmodel.SQLKey:              "SELECT id, name, timestamp FROM updated",
					sqlmodel.AccountIDKey:        "acc123",
					sqlmodel.PrimaryKeyKey:       "id",
				},
				expectedError: true,
				errorMessage:  "source definition name cannot be changed",
				mockSetup: func() *mockRETLClient {
					return &mockRETLClient{sourceID: "src123"}
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				mockClient := tc.mockSetup()
				handler := sqlmodel.NewHandler(mockClient, "retl")

				result, err := handler.Update(context.Background(), "test-model", tc.data, tc.state)

				if tc.expectedError {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.errorMessage)
					assert.Nil(t, result)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, result)
					assert.True(t, mockClient.updateCalled)
				}
			})
		}
	})

	t.Run("Update with timestamps", func(t *testing.T) {
		t.Parallel()

		createdAt := time.Now().Add(-24 * time.Hour)
		updatedAt := time.Now()

		mockClient := &mockRETLClient{
			sourceID: "src123",
			updateRetlSourceFunc: func(ctx context.Context, sourceID string, req *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error) {
				return &retlClient.RETLSource{
					ID:                   sourceID,
					Name:                 req.Name,
					Config:               req.Config,
					SourceType:           "model",
					SourceDefinitionName: "postgres",
					AccountID:            req.AccountID,
					IsEnabled:            req.IsEnabled,
					CreatedAt:            &createdAt,
					UpdatedAt:            &updatedAt,
				}, nil
			},
		}
		handler := sqlmodel.NewHandler(mockClient, "retl")

		data := resources.ResourceData{
			"display_name":      "Updated Model",
			"description":       "Updated description",
			"sql":               "SELECT id, name, timestamp FROM updated",
			"account_id":        "acc123",
			"primary_key":       "id",
			"source_definition": "postgres",
			"enabled":           true,
		}
		state := resources.ResourceData{
			sqlmodel.IDKey:               "src123",
			sqlmodel.EnabledKey:          true,
			sqlmodel.SourceDefinitionKey: "postgres",
			sqlmodel.PrimaryKeyKey:       "id",
			sqlmodel.AccountIDKey:        "acc123",
			sqlmodel.DisplayNameKey:      "Updated Model",
			sqlmodel.DescriptionKey:      "Updated description",
			sqlmodel.SQLKey:              "SELECT * FROM updated",
		}

		// Execute
		result, err := handler.Update(context.Background(), "test-model", data, state)

		// Verify
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, &createdAt, (*result)[sqlmodel.CreatedAtKey])
		assert.Equal(t, &updatedAt, (*result)[sqlmodel.UpdatedAtKey])
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name          string
			state         resources.ResourceData
			expectedError bool
			errorMessage  string
			mockSetup     func() *mockRETLClient
		}{
			{
				name: "Valid delete",
				state: resources.ResourceData{
					sqlmodel.IDKey: "src123",
				},
				expectedError: false,
				mockSetup: func() *mockRETLClient {
					return &mockRETLClient{sourceID: "src123"}
				},
			},
			{
				name:          "Missing source_id",
				state:         resources.ResourceData{},
				expectedError: true,
				errorMessage:  fmt.Sprintf("missing %s in resource state", sqlmodel.IDKey),
				mockSetup: func() *mockRETLClient {
					return &mockRETLClient{}
				},
			},
			{
				name: "API error",
				state: resources.ResourceData{
					sqlmodel.IDKey: "error",
				},
				expectedError: true,
				errorMessage:  "deleting RETL source",
				mockSetup: func() *mockRETLClient {
					return &mockRETLClient{sourceID: "error", deleteError: true}
				},
			},
		}

		for _, tc := range testCases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				mockClient := tc.mockSetup()
				handler := sqlmodel.NewHandler(mockClient, "retl")

				err := handler.Delete(context.Background(), "test-model", tc.state)

				if tc.expectedError {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.errorMessage)
				} else {
					assert.NoError(t, err)
					assert.True(t, mockClient.deleteCalled)
				}
			})
		}
	})

	t.Run("List", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			t.Parallel()

			createdAt := time.Now().Add(-24 * time.Hour)
			updatedAt := time.Now()

			source1 := createTestRETLSourceWithConfig("source-1", "Test Source 1", "postgres", "account-1", true,
				retlClient.RETLSQLModelConfig{
					Description: "Test description 1",
					PrimaryKey:  "id",
					Sql:         "SELECT * FROM table1",
				})
			source1.CreatedAt = &createdAt
			source1.UpdatedAt = &updatedAt

			source2 := createTestRETLSourceWithConfig("source-2", "Test Source 2", "mysql", "account-2", false,
				retlClient.RETLSQLModelConfig{
					Description: "Test description 2",
					PrimaryKey:  "id",
					Sql:         "SELECT * FROM table2",
				})

			mockClient := &mockRETLClient{
				sourceID:            "src123",
				listRetlSourcesFunc: mockListRetlSources(source1, source2),
			}

			handler := sqlmodel.NewHandler(mockClient, "retl")

			results, err := handler.List(context.Background(), nil)

			assert.NoError(t, err)
			assert.Len(t, results, 2)

			// Verify first source
			result1 := results[0]
			assert.Equal(t, "source-1", result1[sqlmodel.IDKey])
			assert.Equal(t, "Test Source 1", result1["name"]) // Handler uses "name" not DisplayNameKey
			// Note: EnabledKey and SourceTypeKey are not set by the current List implementation
			assert.Equal(t, "postgres", result1[sqlmodel.SourceDefinitionKey])
			assert.Equal(t, "account-1", result1[sqlmodel.AccountIDKey])

			// These fields are in the config sub-object
			config1, ok := result1["config"].(map[string]interface{})
			assert.True(t, ok, "config should be a map")
			assert.Equal(t, "Test description 1", config1[sqlmodel.DescriptionKey])
			assert.Equal(t, "id", config1[sqlmodel.PrimaryKeyKey])
			assert.Equal(t, "SELECT * FROM table1", config1[sqlmodel.SQLKey])

			assert.NotNil(t, result1[sqlmodel.CreatedAtKey])
			assert.NotNil(t, result1[sqlmodel.UpdatedAtKey])

			// Verify second source
			result2 := results[1]
			assert.Equal(t, "source-2", result2[sqlmodel.IDKey])
			assert.Equal(t, "Test Source 2", result2["name"]) // Handler uses "name" not DisplayNameKey
			// Note: EnabledKey and SourceTypeKey are not set by the current List implementation
			assert.Equal(t, "mysql", result2[sqlmodel.SourceDefinitionKey])
			assert.Equal(t, "account-2", result2[sqlmodel.AccountIDKey])
		})

		t.Run("Sources with external id", func(t *testing.T) {
			t.Parallel()

			source := createTestRETLSourceWithConfig("source-1", "Test Source 1", "postgres", "account-1", true,
				retlClient.RETLSQLModelConfig{
					Description: "desc",
					PrimaryKey:  "id",
					Sql:         "SELECT 1\nFROM dual",
				})

			mockClient := &mockRETLClient{
				sourceID:            "src123",
				listRetlSourcesFunc: mockListRetlSources(source),
			}

			handler := sqlmodel.NewHandler(mockClient, "retl")
			hasExternalID := true
			results, err := handler.List(context.Background(), &hasExternalID)
			assert.NoError(t, err)
			assert.Len(t, results, 1)
			// Basic field checks
			assert.Equal(t, "source-1", results[0][sqlmodel.IDKey])
			assert.Equal(t, "Test Source 1", results[0]["name"]) // Handler uses "name" not DisplayNameKey
			assert.Equal(t, "postgres", results[0][sqlmodel.SourceDefinitionKey])
			assert.Equal(t, "account-1", results[0][sqlmodel.AccountIDKey])
			// Config nested checks and SQL newline normalization
			cfg, ok := results[0]["config"].(map[string]any)
			require.True(t, ok, "config should be a map")
			assert.Equal(t, "id", cfg[sqlmodel.PrimaryKeyKey])
			assert.Equal(t, "desc", cfg[sqlmodel.DescriptionKey])
			// newlines should be collapsed to a single space
			assert.Equal(t, "SELECT 1 FROM dual", cfg[sqlmodel.SQLKey])
		})

		t.Run("EmptyList", func(t *testing.T) {
			t.Parallel()

			mockClient := &mockRETLClient{
				sourceID:            "src123",
				listRetlSourcesFunc: mockListRetlSources(),
			}

			handler := sqlmodel.NewHandler(mockClient, "retl")

			results, err := handler.List(context.Background(), nil)

			assert.NoError(t, err)
			assert.Len(t, results, 0)
		})

		t.Run("APIError", func(t *testing.T) {
			t.Parallel()

			mockClient := &mockRETLClient{
				sourceID: "src123",
				listRetlSourcesFunc: func(ctx context.Context) (*retlClient.RETLSources, error) {
					return nil, fmt.Errorf("API error")
				},
			}

			handler := sqlmodel.NewHandler(mockClient, "retl")

			results, err := handler.List(context.Background(), nil)

			assert.Error(t, err)
			assert.Nil(t, results)
			assert.Contains(t, err.Error(), "listing RETL sources")
		})
	})

	t.Run("ValidateSQLModelResource", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name          string
			resource      *sqlmodel.SQLModelResource
			expectedError string
		}{
			{
				name: "Valid resource",
				resource: &sqlmodel.SQLModelResource{
					ID:               "test-model",
					DisplayName:      "Test Model",
					Description:      "Test description",
					SQL:              "SELECT * FROM users",
					AccountID:        "acc123",
					PrimaryKey:       "id",
					SourceDefinition: "postgres",
					Enabled:          true,
				},
				expectedError: "",
			},
			{
				name: "Missing ID",
				resource: &sqlmodel.SQLModelResource{
					DisplayName:      "Test Model",
					Description:      "Test description",
					SQL:              "SELECT * FROM users",
					AccountID:        "acc123",
					PrimaryKey:       "id",
					SourceDefinition: "postgres",
					Enabled:          true,
				},
				expectedError: "id is required",
			},
			{
				name: "Missing DisplayName",
				resource: &sqlmodel.SQLModelResource{
					ID:               "test-model",
					Description:      "Test description",
					SQL:              "SELECT * FROM users",
					AccountID:        "acc123",
					PrimaryKey:       "id",
					SourceDefinition: "postgres",
					Enabled:          true,
				},
				expectedError: "display_name is required",
			},
			{
				name: "Missing SQL",
				resource: &sqlmodel.SQLModelResource{
					ID:               "test-model",
					DisplayName:      "Test Model",
					Description:      "Test description",
					AccountID:        "acc123",
					PrimaryKey:       "id",
					SourceDefinition: "postgres",
					Enabled:          true,
				},
				expectedError: "sql is required",
			},
			{
				name: "Missing AccountID",
				resource: &sqlmodel.SQLModelResource{
					ID:               "test-model",
					DisplayName:      "Test Model",
					Description:      "Test description",
					SQL:              "SELECT * FROM users",
					PrimaryKey:       "id",
					SourceDefinition: "postgres",
					Enabled:          true,
				},
				expectedError: "account_id is required",
			},
			{
				name: "Missing PrimaryKey",
				resource: &sqlmodel.SQLModelResource{
					ID:               "test-model",
					DisplayName:      "Test Model",
					Description:      "Test description",
					SQL:              "SELECT * FROM users",
					AccountID:        "acc123",
					SourceDefinition: "postgres",
					Enabled:          true,
				},
				expectedError: "primary_key is required",
			},
			{
				name: "Missing SourceDefinitionName",
				resource: &sqlmodel.SQLModelResource{
					ID:          "test-model",
					DisplayName: "Test Model",
					Description: "Test description",
					SQL:         "SELECT * FROM users",
					AccountID:   "acc123",
					PrimaryKey:  "id",
					Enabled:     true,
				},
				expectedError: "source_definition is required",
			},
			{
				name: "Invalid SourceDefinition",
				resource: &sqlmodel.SQLModelResource{
					ID:               "test-model",
					DisplayName:      "Test Model",
					Description:      "Test description",
					SQL:              "SELECT * FROM users",
					AccountID:        "acc123",
					PrimaryKey:       "id",
					SourceDefinition: "invalid-source",
					Enabled:          true,
				},
				expectedError: "source_definition 'invalid-source' is invalid, must be one of:",
			},
		}

		for _, tc := range testCases {
			tc := tc // capture range variable
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Execute
				err := sqlmodel.ValidateSQLModelResource(tc.resource)

				// Verify
				if tc.expectedError != "" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.expectedError)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("FetchImportData", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			mockClient := &mockRETLClient{}
			handler := sqlmodel.NewHandler(mockClient, "retl")

			// In Import tests, set mockClient.getRetlSourceFunc instead of mockClient.GetRetlSource
			mockClient.getRetlSourceFunc = func(ctx context.Context, sourceID string) (*retlClient.RETLSource, error) {
				return &retlClient.RETLSource{
					ID:                   "remote-id",
					Name:                 "Imported Model",
					Config:               retlClient.RETLSQLModelConfig{Description: "desc", PrimaryKey: "id", Sql: "SELECT * FROM t"},
					SourceType:           retlClient.ModelSourceType,
					SourceDefinitionName: "postgres",
					AccountID:            "acc123",
					IsEnabled:            true,
				}, nil
			}

			args := importremote.ImportArgs{
				RemoteID: "remote-id",
				LocalID:  "local-id",
			}
			results, err := handler.FetchImportData(context.Background(), args)
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

		t.Run("Non-SQL-model type", func(t *testing.T) {
			mockClient := &mockRETLClient{}
			handler := sqlmodel.NewHandler(mockClient, "retl")

			// Non-SQL-model type
			mockClient.getRetlSourceFunc = func(ctx context.Context, sourceID string) (*retlClient.RETLSource, error) {
				return &retlClient.RETLSource{
					ID:         "remote-id",
					SourceType: "not-model",
				}, nil
			}

			args := importremote.ImportArgs{RemoteID: "remote-id", LocalID: "local-id"}
			results, err := handler.FetchImportData(context.Background(), args)
			assert.Error(t, err)
			assert.Nil(t, results)
			assert.Contains(t, err.Error(), "is not a SQL model")
		})

		t.Run("API error", func(t *testing.T) {
			mockClient := &mockRETLClient{}
			handler := sqlmodel.NewHandler(mockClient, "retl")

			// API error
			mockClient.getRetlSourceFunc = func(ctx context.Context, sourceID string) (*retlClient.RETLSource, error) {
				return nil, fmt.Errorf("api error")
			}

			args := importremote.ImportArgs{RemoteID: "remote-id", LocalID: "local-id"}
			results, err := handler.FetchImportData(context.Background(), args)
			assert.Error(t, err)
			assert.Nil(t, results)
			assert.Contains(t, err.Error(), "getting RETL source for import")
		})

		t.Run("Missing local id", func(t *testing.T) {
			mockClient := &mockRETLClient{}
			handler := sqlmodel.NewHandler(mockClient, "retl")
			args := importremote.ImportArgs{RemoteID: "remote-id"}
			results, err := handler.FetchImportData(context.Background(), args)
			assert.Error(t, err)
			assert.Nil(t, results)
			assert.Contains(t, err.Error(), "local id is required")
		})

		t.Run("Missing remote id", func(t *testing.T) {
			mockClient := &mockRETLClient{}
			handler := sqlmodel.NewHandler(mockClient, "retl")
			args := importremote.ImportArgs{LocalID: "local-id"}
			results, err := handler.FetchImportData(context.Background(), args)
			assert.Error(t, err)
			assert.Nil(t, results)
			assert.Contains(t, err.Error(), "remote id is required")
		})
	})

	t.Run("LoadResourcesFromRemote", func(t *testing.T) {
		t.Run("Success with multiple sources", func(t *testing.T) {
			t.Parallel()

			createdAt := time.Now().Add(-24 * time.Hour)
			updatedAt := time.Now()

			source1 := createTestRETLSourceWithConfig("source-1", "Test Source 1", "postgres", "account-1", true,
				retlClient.RETLSQLModelConfig{
					Description: "Test description 1",
					PrimaryKey:  "id",
					Sql:         "SELECT * FROM table1",
				})
			source1.CreatedAt = &createdAt
			source1.UpdatedAt = &updatedAt
			source1.ExternalID = "ext-1"

			source2 := createTestRETLSourceWithConfig("source-2", "Test Source 2", "mysql", "account-2", false,
				retlClient.RETLSQLModelConfig{
					Description: "Test description 2",
					PrimaryKey:  "id",
					Sql:         "SELECT * FROM table2",
				})
			source2.ExternalID = "ext-2"

			mockClient := &mockRETLClient{
				listRetlSourcesFunc: mockListRetlSources(source1, source2),
			}

			handler := sqlmodel.NewHandler(mockClient, "retl")

			collection, err := handler.LoadResourcesFromRemote(context.Background())

			assert.NoError(t, err)
			assert.NotNil(t, collection)

			// Get all SQL model resources
			resources := collection.GetAll(sqlmodel.ResourceType)
			assert.Len(t, resources, 2)

			// Verify first resource
			resource1, exists := resources["source-1"]
			assert.True(t, exists)
			assert.Equal(t, "source-1", resource1.ID)
			assert.Equal(t, "ext-1", resource1.ExternalID)
			assert.NotNil(t, resource1.Data)

			// Verify second resource
			resource2, exists := resources["source-2"]
			assert.True(t, exists)
			assert.Equal(t, "source-2", resource2.ID)
			assert.Equal(t, "ext-2", resource2.ExternalID)
			assert.NotNil(t, resource2.Data)
		})

		t.Run("Success with single source", func(t *testing.T) {
			t.Parallel()

			source := createTestRETLSourceWithConfig("single-source", "Single Source", "postgres", "account-1", true,
				retlClient.RETLSQLModelConfig{
					Description: "Single source description",
					PrimaryKey:  "id",
					Sql:         "SELECT * FROM single_table",
				})
			source.ExternalID = "ext-single"

			mockClient := &mockRETLClient{
				listRetlSourcesFunc: mockListRetlSources(source),
			}

			handler := sqlmodel.NewHandler(mockClient, "retl")

			collection, err := handler.LoadResourcesFromRemote(context.Background())

			assert.NoError(t, err)
			assert.NotNil(t, collection)

			resources := collection.GetAll(sqlmodel.ResourceType)
			assert.Len(t, resources, 1)

			resource, exists := resources["single-source"]
			assert.True(t, exists)
			assert.Equal(t, "single-source", resource.ID)
			assert.Equal(t, "ext-single", resource.ExternalID)
		})

		t.Run("Success with empty list", func(t *testing.T) {
			t.Parallel()

			mockClient := &mockRETLClient{
				listRetlSourcesFunc: mockListRetlSources(),
			}

			handler := sqlmodel.NewHandler(mockClient, "retl")

			collection, err := handler.LoadResourcesFromRemote(context.Background())

			assert.NoError(t, err)
			assert.NotNil(t, collection)

			resources := collection.GetAll(sqlmodel.ResourceType)
			assert.Len(t, resources, 0)
		})

		t.Run("Success with mixed sources", func(t *testing.T) {
			t.Parallel()

			sourceWithExt := createTestRETLSourceWithConfig("source-with-ext", "Source With External ID", "postgres", "account-1", true,
				retlClient.RETLSQLModelConfig{
					Description: "With external ID",
					PrimaryKey:  "id",
					Sql:         "SELECT * FROM with_ext_table",
				})
			sourceWithExt.ExternalID = "ext-with"

			mockClient := &mockRETLClient{
				listRetlSourcesFunc: mockListRetlSources(sourceWithExt),
			}

			handler := sqlmodel.NewHandler(mockClient, "retl")

			collection, err := handler.LoadResourcesFromRemote(context.Background())

			assert.NoError(t, err)
			assert.NotNil(t, collection)

			resources := collection.GetAll(sqlmodel.ResourceType)
			assert.Len(t, resources, 1)

			resource, exists := resources["source-with-ext"]
			assert.True(t, exists)
			assert.Equal(t, "source-with-ext", resource.ID)
			assert.Equal(t, "ext-with", resource.ExternalID)
		})

		t.Run("API error", func(t *testing.T) {
			t.Parallel()

			mockClient := &mockRETLClient{
				listRetlSourcesFunc: func(ctx context.Context) (*retlClient.RETLSources, error) {
					return nil, fmt.Errorf("API error listing sources")
				},
			}

			handler := sqlmodel.NewHandler(mockClient, "retl")

			// Execute
			collection, err := handler.LoadResourcesFromRemote(context.Background())

			// Verify
			assert.Error(t, err)
			assert.Nil(t, collection)
			assert.Contains(t, err.Error(), "listing RETL sources")
		})
	})

	t.Run("LoadStateFromResources", func(t *testing.T) {
		t.Run("Success with multiple resources", func(t *testing.T) {
			t.Parallel()

			h := sqlmodel.NewHandler(&mockRETLClient{}, "retl")

			createdAt := time.Now().Add(-48 * time.Hour)
			updatedAt := time.Now()

			// Build a collection with two RETL sources having ExternalID set
			collection := resources.NewResourceCollection()
			resourceMap := map[string]*resources.RemoteResource{
				"remote-1": {
					ID:         "remote-1",
					ExternalID: "local-1",
					Data: retlClient.RETLSource{
						ID:                   "remote-1",
						Name:                 "Model One",
						AccountID:            "acc-1",
						SourceType:           retlClient.ModelSourceType,
						SourceDefinitionName: "postgres",
						IsEnabled:            true,
						WorkspaceID:          "ws-1",
						ExternalID:           "local-1",
						CreatedAt:            &createdAt,
						UpdatedAt:            &updatedAt,
						Config: retlClient.RETLSQLModelConfig{
							Description: "desc 1",
							PrimaryKey:  "id",
							Sql:         "SELECT * FROM one",
						},
					},
				},
				"remote-2": {
					ID:         "remote-2",
					ExternalID: "local-2",
					Data: retlClient.RETLSource{
						ID:                   "remote-2",
						Name:                 "Model Two",
						AccountID:            "acc-2",
						SourceType:           retlClient.ModelSourceType,
						SourceDefinitionName: "mysql",
						IsEnabled:            false,
						WorkspaceID:          "ws-2",
						ExternalID:           "local-2",
						Config: retlClient.RETLSQLModelConfig{
							Description: "desc 2",
							PrimaryKey:  "pk",
							Sql:         "SELECT * FROM two",
						},
					},
				},
			}
			collection.Set(sqlmodel.ResourceType, resourceMap)

			// Execute
			st, err := h.LoadStateFromResources(context.Background(), collection)

			// Verify
			require.NoError(t, err)
			require.NotNil(t, st)
			// Expect two resources keyed by URN(localID, type)
			urn1 := resources.URN("local-1", sqlmodel.ResourceType)
			urn2 := resources.URN("local-2", sqlmodel.ResourceType)
			r1 := st.GetResource(urn1)
			require.NotNil(t, r1)
			assert.Equal(t, "local-1", r1.ID)
			assert.Equal(t, sqlmodel.ResourceType, r1.Type)
			// Input fields
			assert.Equal(t, "Model One", r1.Input[sqlmodel.DisplayNameKey])
			assert.Equal(t, "desc 1", r1.Input[sqlmodel.DescriptionKey])
			assert.Equal(t, "acc-1", r1.Input[sqlmodel.AccountIDKey])
			assert.Equal(t, "id", r1.Input[sqlmodel.PrimaryKeyKey])
			assert.Equal(t, "SELECT * FROM one", r1.Input[sqlmodel.SQLKey])
			assert.Equal(t, true, r1.Input[sqlmodel.EnabledKey])
			assert.Equal(t, "postgres", r1.Input[sqlmodel.SourceDefinitionKey])
			assert.Equal(t, "local-1", r1.Input[sqlmodel.LocalIDKey])
			// Output timestamps should be present
			assert.Equal(t, &createdAt, r1.Output[sqlmodel.CreatedAtKey])
			assert.Equal(t, &updatedAt, r1.Output[sqlmodel.UpdatedAtKey])

			r2 := st.GetResource(urn2)
			require.NotNil(t, r2)
			assert.Equal(t, "Model Two", r2.Input[sqlmodel.DisplayNameKey])
			assert.Equal(t, "desc 2", r2.Input[sqlmodel.DescriptionKey])
			assert.Equal(t, "acc-2", r2.Input[sqlmodel.AccountIDKey])
			assert.Equal(t, "pk", r2.Input[sqlmodel.PrimaryKeyKey])
			assert.Equal(t, "SELECT * FROM two", r2.Input[sqlmodel.SQLKey])
			assert.Equal(t, false, r2.Input[sqlmodel.EnabledKey])
			assert.Equal(t, "mysql", r2.Input[sqlmodel.SourceDefinitionKey])
			assert.Equal(t, "local-2", r2.Input[sqlmodel.LocalIDKey])
		})

		t.Run("Error on invalid data type", func(t *testing.T) {
			t.Parallel()

			h := sqlmodel.NewHandler(&mockRETLClient{}, "retl")
			collection := resources.NewResourceCollection()
			collection.Set(sqlmodel.ResourceType, map[string]*resources.RemoteResource{
				"bad": {
					ID:         "bad",
					ExternalID: "local-bad",
					Data:       "not-a-retl-source",
				},
			})

			st, err := h.LoadStateFromResources(context.Background(), collection)
			assert.Error(t, err)
			assert.Nil(t, st)
			assert.Contains(t, err.Error(), "unable to cast resource to retl source")
		})
	})
}
