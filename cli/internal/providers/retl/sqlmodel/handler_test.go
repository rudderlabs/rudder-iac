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
	"github.com/rudderlabs/rudder-iac/cli/internal/importutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

// mockRETLClient is a mock implementation of the RETL client
type mockRETLClient struct {
	createCalled         bool
	updateCalled         bool
	deleteCalled         bool
	sourceID             string
	deleteError          bool
	updateError          bool
	createRetlSourceFunc func(ctx context.Context, req *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error)
	updateRetlSourceFunc func(ctx context.Context, sourceID string, req *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error)
	listRetlSourcesFunc  func(ctx context.Context) (*retlClient.RETLSources, error)
	getRetlSourceFunc    func(ctx context.Context, sourceID string) (*retlClient.RETLSource, error) // <-- add this
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

func (m *mockRETLClient) ListRetlSources(ctx context.Context) (*retlClient.RETLSources, error) {
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
	return &retlClient.State{
		Resources: map[string]retlClient.ResourceState{},
	}, nil
}

func (m *mockRETLClient) PutResourceState(ctx context.Context, id string, req retlClient.PutStateRequest) error {
	return nil
}

func TestSQLModelHandler(t *testing.T) {
	t.Parallel()

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
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Setup
				mockClient := &mockRETLClient{sourceID: "src123"}
				handler := sqlmodel.NewHandler(mockClient)

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

	t.Run("LoadSpec with duplicate id", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockRETLClient{sourceID: "src123"}
		handler := sqlmodel.NewHandler(mockClient)

		err := handler.LoadSpec("test.yaml", &specs.Spec{
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
		})

		require.NoError(t, err)

		err = handler.LoadSpec("test.yaml", &specs.Spec{
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
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sql model with id test-model already exists")
	})

	t.Run("LoadSpec with invalid spec structure", func(t *testing.T) {
		t.Parallel()

		// Setup
		mockClient := &mockRETLClient{sourceID: "src123"}
		handler := sqlmodel.NewHandler(mockClient)

		// Create a spec with invalid structure that will cause mapstructure.Decode to fail
		invalidSpec := &specs.Spec{
			Version: "rudder/v0.1",
			Kind:    "retl-source-sql-model",
			Spec: map[string]interface{}{
				"id":      123,          // ID should be a string, not an int
				"enabled": "not-a-bool", // Enabled should be a bool
			},
		}

		// Execute
		err := handler.LoadSpec("test.yaml", invalidSpec)

		// Verify
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "converting spec")
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		// Setup
		mockClient := &mockRETLClient{sourceID: "src123"}
		handler := sqlmodel.NewHandler(mockClient)

		handler.LoadSpec("test.yaml", &specs.Spec{
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
		})

		// Execute
		err := handler.Validate()

		// Verify
		assert.NoError(t, err)
	})

	t.Run("Validate with invalid resource", func(t *testing.T) {
		t.Parallel()

		// Setup
		mockClient := &mockRETLClient{sourceID: "src123"}
		handler := sqlmodel.NewHandler(mockClient)

		// Load a spec with missing required fields to trigger validation error
		err := handler.LoadSpec("test.yaml", &specs.Spec{
			Version: "rudder/v0.1",
			Kind:    "retl-source-sql-model",
			Spec: map[string]interface{}{
				"id":           "test-model",
				"display_name": "Test Model",
				"sql":          "SELECT * FROM users",
				// Missing description, account_id, primary_key, source_definition
			},
		})
		require.NoError(t, err)

		// Execute
		err = handler.Validate()

		// Verify
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validating sql model spec")
	})

	t.Run("GetResources", func(t *testing.T) {
		t.Parallel()

		// Setup
		mockClient := &mockRETLClient{sourceID: "src123"}
		handler := sqlmodel.NewHandler(mockClient)
		handler.LoadSpec("test.yaml", &specs.Spec{
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
		})

		// Execute
		resources, err := handler.GetResources()

		// Verify
		assert.NoError(t, err)
		assert.Len(t, resources, 1)
		assert.Equal(t, "test-model", resources[0].ID())
		assert.Equal(t, sqlmodel.ResourceType, resources[0].Type())
	})

	t.Run("Create", func(t *testing.T) {
		t.Parallel()

		// Setup
		mockClient := &mockRETLClient{sourceID: "src123"}
		handler := sqlmodel.NewHandler(mockClient)

		handler.LoadSpec("test.yaml", &specs.Spec{
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
		})

		// Create resource data
		data := resources.ResourceData{
			"id":                "test-model",
			"display_name":      "Test Model",
			"description":       "Test description",
			"sql":               "SELECT * FROM users",
			"account_id":        "acc123",
			"primary_key":       "id",
			"source_definition": "postgres",
			"enabled":           true,
		}

		// Execute
		result, err := handler.Create(context.Background(), "test-model", data)

		// Verify
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, mockClient.createCalled)
		assert.Equal(t, "src123", (*result)[sqlmodel.IDKey])
	})

	t.Run("Create with API error", func(t *testing.T) {
		t.Parallel()

		// Setup
		mockClient := &mockRETLClient{
			sourceID: "src123",
			createRetlSourceFunc: func(ctx context.Context, req *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
				return nil, fmt.Errorf("API error creating source")
			},
		}
		handler := sqlmodel.NewHandler(mockClient)

		// Create resource data
		data := resources.ResourceData{
			"id":                "test-model",
			"display_name":      "Test Model",
			"description":       "Test description",
			"sql":               "SELECT * FROM users",
			"account_id":        "acc123",
			"primary_key":       "id",
			"source_definition": "postgres",
			"enabled":           true,
		}

		// Execute
		result, err := handler.Create(context.Background(), "test-model", data)

		// Verify
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
		handler := sqlmodel.NewHandler(mockClient)

		// Create resource data
		data := resources.ResourceData{
			"id":                "test-model",
			"display_name":      "Test Model",
			"description":       "Test description",
			"sql":               "SELECT * FROM users",
			"account_id":        "acc123",
			"primary_key":       "id",
			"source_definition": "postgres",
			"enabled":           true,
		}

		// Execute
		result, err := handler.Create(context.Background(), "test-model", data)

		// Verify
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
				handler := sqlmodel.NewHandler(mockClient)

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
		handler := sqlmodel.NewHandler(mockClient)

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
				handler := sqlmodel.NewHandler(mockClient)

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

			// Create mock with custom list function
			mockClient := &mockRETLClient{
				sourceID: "src123",
				listRetlSourcesFunc: func(ctx context.Context) (*retlClient.RETLSources, error) {
					createdAt := time.Now().Add(-24 * time.Hour)
					updatedAt := time.Now()

					return &retlClient.RETLSources{
						Data: []retlClient.RETLSource{
							{
								ID:                   "source-1",
								Name:                 "Test Source 1",
								IsEnabled:            true,
								SourceType:           retlClient.ModelSourceType,
								SourceDefinitionName: "postgres",
								AccountID:            "account-1",
								CreatedAt:            &createdAt,
								UpdatedAt:            &updatedAt,
								Config: retlClient.RETLSQLModelConfig{
									Description: "Test description 1",
									PrimaryKey:  "id",
									Sql:         "SELECT * FROM table1",
								},
							},
							{
								ID:                   "source-2",
								Name:                 "Test Source 2",
								IsEnabled:            false,
								SourceType:           retlClient.ModelSourceType,
								SourceDefinitionName: "mysql",
								AccountID:            "account-2",
								Config: retlClient.RETLSQLModelConfig{
									Description: "Test description 2",
									PrimaryKey:  "id",
									Sql:         "SELECT * FROM table2",
								},
							},
						},
					}, nil
				},
			}

			handler := sqlmodel.NewHandler(mockClient)

			// Execute
			results, err := handler.List(context.Background())

			// Verify
			assert.NoError(t, err)
			assert.Len(t, results, 2)

			// Verify first source
			source1 := results[0]
			assert.Equal(t, "source-1", source1[sqlmodel.IDKey])
			assert.Equal(t, "Test Source 1", source1["name"]) // Handler uses "name" not DisplayNameKey
			// Note: EnabledKey and SourceTypeKey are not set by the current List implementation
			assert.Equal(t, "postgres", source1[sqlmodel.SourceDefinitionKey])
			assert.Equal(t, "account-1", source1[sqlmodel.AccountIDKey])

			// These fields are in the config sub-object
			config1, ok := source1["config"].(map[string]interface{})
			assert.True(t, ok, "config should be a map")
			assert.Equal(t, "Test description 1", config1[sqlmodel.DescriptionKey])
			assert.Equal(t, "id", config1[sqlmodel.PrimaryKeyKey])
			assert.Equal(t, "SELECT * FROM table1", config1[sqlmodel.SQLKey])

			assert.NotNil(t, source1[sqlmodel.CreatedAtKey])
			assert.NotNil(t, source1[sqlmodel.UpdatedAtKey])

			// Verify second source
			source2 := results[1]
			assert.Equal(t, "source-2", source2[sqlmodel.IDKey])
			assert.Equal(t, "Test Source 2", source2["name"]) // Handler uses "name" not DisplayNameKey
			// Note: EnabledKey and SourceTypeKey are not set by the current List implementation
			assert.Equal(t, "mysql", source2[sqlmodel.SourceDefinitionKey])
			assert.Equal(t, "account-2", source2[sqlmodel.AccountIDKey])
		})

		t.Run("EmptyList", func(t *testing.T) {
			t.Parallel()

			mockClient := &mockRETLClient{
				sourceID: "src123",
				listRetlSourcesFunc: func(ctx context.Context) (*retlClient.RETLSources, error) {
					return &retlClient.RETLSources{
						Data: []retlClient.RETLSource{},
					}, nil
				},
			}

			handler := sqlmodel.NewHandler(mockClient)

			// Execute
			results, err := handler.List(context.Background())

			// Verify
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

			handler := sqlmodel.NewHandler(mockClient)

			// Execute
			results, err := handler.List(context.Background())

			// Verify
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

	t.Run("Import", func(t *testing.T) {
		t.Parallel()

		t.Run("Success", func(t *testing.T) {
			mockClient := &mockRETLClient{}
			handler := sqlmodel.NewHandler(mockClient)

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

			args := importutils.ImportArgs{
				RemoteID:    "remote-id",
				LocalID:     "local-id",
				WorkspaceID: "ws-1",
			}
			results, err := handler.Import(context.Background(), args)
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
			assert.Equal(t, "local-id", imported.Metadata["name"])
			assert.Equal(t, "ws-1", imported.Metadata["workspace"])
		})

		t.Run("Non-SQL-model type", func(t *testing.T) {
			mockClient := &mockRETLClient{}
			handler := sqlmodel.NewHandler(mockClient)

			// Non-SQL-model type
			mockClient.getRetlSourceFunc = func(ctx context.Context, sourceID string) (*retlClient.RETLSource, error) {
				return &retlClient.RETLSource{
					ID:         "remote-id",
					SourceType: "not-model",
				}, nil
			}

			args := importutils.ImportArgs{RemoteID: "remote-id", LocalID: "local-id", WorkspaceID: "ws-1"}
			results, err := handler.Import(context.Background(), args)
			assert.Error(t, err)
			assert.Nil(t, results)
			assert.Contains(t, err.Error(), "is not a SQL model")
		})

		t.Run("API error", func(t *testing.T) {
			mockClient := &mockRETLClient{}
			handler := sqlmodel.NewHandler(mockClient)

			// API error
			mockClient.getRetlSourceFunc = func(ctx context.Context, sourceID string) (*retlClient.RETLSource, error) {
				return nil, fmt.Errorf("api error")
			}

			args := importutils.ImportArgs{RemoteID: "remote-id", LocalID: "local-id", WorkspaceID: "ws-1"}
			results, err := handler.Import(context.Background(), args)
			assert.Error(t, err)
			assert.Nil(t, results)
			assert.Contains(t, err.Error(), "getting RETL source for import")
		})
	})
}
