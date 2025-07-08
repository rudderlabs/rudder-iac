package retl

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

// mockHandler implements resourceHandler interface for testing
type mockHandler struct {
	loadSpecFunc     func(path string, s *specs.Spec) error
	validateFunc     func() error
	getResourcesFunc func() ([]*resources.Resource, error)
	createFunc       func(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error)
	updateFunc       func(ctx context.Context, ID string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error)
	deleteFunc       func(ctx context.Context, ID string, state resources.ResourceData) error
}

func (m *mockHandler) LoadSpec(path string, s *specs.Spec) error {
	if m.loadSpecFunc != nil {
		return m.loadSpecFunc(path, s)
	}
	return nil
}

func (m *mockHandler) Validate() error {
	if m.validateFunc != nil {
		return m.validateFunc()
	}
	return nil
}

func (m *mockHandler) GetResources() ([]*resources.Resource, error) {
	if m.getResourcesFunc != nil {
		return m.getResourcesFunc()
	}
	return nil, nil
}

func (m *mockHandler) Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, ID, data)
	}
	return nil, nil
}

func (m *mockHandler) Update(ctx context.Context, ID string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, ID, data, state)
	}
	return nil, nil
}

func (m *mockHandler) Delete(ctx context.Context, ID string, state resources.ResourceData) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, ID, state)
	}
	return nil
}

// TestHandlerInterface verifies that mockHandler implements resourceHandler
func TestHandlerInterface(t *testing.T) {
	t.Parallel()

	var _ resourceHandler = (*mockHandler)(nil)
}

// mockRETLSourceClient implements the RETLStore interface for testing
type mockRETLSourceClient struct {
	retlClient.RETLStore
	createRetlSourceFunc func(ctx context.Context, source *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error)
	updateRetlSourceFunc func(ctx context.Context, source *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error)
	deleteRetlSourceFunc func(ctx context.Context, id string) error
	getRetlSourceFunc    func(ctx context.Context, id string) (*retlClient.RETLSource, error)
	listRetlSourcesFunc  func(ctx context.Context) (*retlClient.RETLSources, error)
}

func (m *mockRETLSourceClient) CreateRetlSource(ctx context.Context, source *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
	if m.createRetlSourceFunc != nil {
		return m.createRetlSourceFunc(ctx, source)
	}
	return nil, nil
}

func (m *mockRETLSourceClient) UpdateRetlSource(ctx context.Context, source *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error) {
	if m.updateRetlSourceFunc != nil {
		return m.updateRetlSourceFunc(ctx, source)
	}
	return nil, nil
}

func (m *mockRETLSourceClient) DeleteRetlSource(ctx context.Context, id string) error {
	if m.deleteRetlSourceFunc != nil {
		return m.deleteRetlSourceFunc(ctx, id)
	}
	return nil
}

func (m *mockRETLSourceClient) GetRetlSource(ctx context.Context, id string) (*retlClient.RETLSource, error) {
	if m.getRetlSourceFunc != nil {
		return m.getRetlSourceFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockRETLSourceClient) ListRetlSources(ctx context.Context) (*retlClient.RETLSources, error) {
	if m.listRetlSourcesFunc != nil {
		return m.listRetlSourcesFunc(ctx)
	}
	return nil, nil
}

// TestSQLModelHandler tests the SQLModelHandler implementation
func TestSQLModelHandler(t *testing.T) {
	t.Parallel()

	// Setup common test data
	now := time.Now()
	mockClient := &mockRETLSourceClient{}

	t.Run("LoadSpec", func(t *testing.T) {
		t.Parallel()

		t.Run("ValidSpec", func(t *testing.T) {
			t.Parallel()

			// Create a test handler
			handler := NewSQLModelHandler(mockClient)

			// Create a valid spec
			spec := &specs.Spec{
				Kind: "retl-source-sql-model",
				Spec: map[string]interface{}{
					"id":           "test-model",
					"display_name": "Test Model",
					"description":  "Test Description",
					"account_id":   "test-account",
					"primary_key":  "id",
					"sql":          "SELECT * FROM users",
				},
			}

			// Test loading the spec
			err := handler.LoadSpec("test.yaml", spec)
			require.NoError(t, err)
			assert.Len(t, handler.specs, 1)

			// Verify the loaded spec
			loadedSpec := handler.specs[0]
			assert.Equal(t, "test-model", loadedSpec.ID)
			assert.Equal(t, "Test Model", loadedSpec.DisplayName)
			assert.Equal(t, "Test Description", loadedSpec.Description)
			assert.Equal(t, "test-account", loadedSpec.AccountID)
			assert.Equal(t, "id", loadedSpec.PrimaryKey)
			assert.NotNil(t, loadedSpec.SQL)
			assert.Equal(t, "SELECT * FROM users", *loadedSpec.SQL)
		})

		t.Run("InvalidSpec", func(t *testing.T) {
			t.Parallel()

			// Create a test handler
			handler := NewSQLModelHandler(mockClient)

			// Create an invalid spec (missing required fields)
			spec := &specs.Spec{
				Kind: "retl-source-sql-model",
				Spec: map[string]interface{}{
					"id": "test-model",
				},
			}

			// Test loading the spec
			err := handler.LoadSpec("test.yaml", spec)
			require.NoError(t, err) // LoadSpec should not validate, just load
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Parallel()

		t.Run("ValidSpecs", func(t *testing.T) {
			t.Parallel()

			// Create a test handler with valid specs
			handler := NewSQLModelHandler(mockClient)
			sql := "SELECT * FROM users"
			handler.specs = []*SQLModelSpec{
				{
					ID:          "test-model",
					DisplayName: "Test Model",
					Description: "Test Description",
					AccountID:   "test-account",
					PrimaryKey:  "id",
					SQL:         &sql,
				},
			}

			// Test validation
			err := handler.Validate()
			assert.NoError(t, err)
		})

		t.Run("InvalidSpecs", func(t *testing.T) {
			t.Parallel()

			// Create a test handler with invalid specs
			handler := NewSQLModelHandler(mockClient)
			handler.specs = []*SQLModelSpec{
				{
					ID: "test-model",
					// Missing required fields
				},
			}

			// Test validation
			err := handler.Validate()
			assert.Error(t, err)
		})
	})

	t.Run("GetResources", func(t *testing.T) {
		t.Parallel()

		// Create a test handler with specs
		handler := NewSQLModelHandler(mockClient)
		sql := "SELECT * FROM users"
		handler.specs = []*SQLModelSpec{
			{
				ID:          "test-model",
				DisplayName: "Test Model",
				Description: "Test Description",
				AccountID:   "test-account",
				PrimaryKey:  "id",
				SQL:         &sql,
			},
		}

		// Test getting resources
		resources, err := handler.GetResources()
		require.NoError(t, err)
		require.Len(t, resources, 1)

		// Verify the resource
		resource := resources[0]
		assert.Equal(t, "test-model", resource.ID())
		assert.Equal(t, "sql-model", resource.Type())

		// Verify resource data
		data := resource.Data()
		assert.Equal(t, "test-model", data["id"])
		assert.Equal(t, "Test Model", data["display_name"])
		assert.Equal(t, "Test Description", data["description"])
		assert.Equal(t, "test-account", data["account_id"])
		assert.Equal(t, "id", data["primary_key"])
		assert.Equal(t, "SELECT * FROM users", data["sql"])
	})

	t.Run("Create", func(t *testing.T) {
		t.Parallel()

		// Setup mock client
		mockClient := &mockRETLSourceClient{}
		mockClient.createRetlSourceFunc = func(ctx context.Context, source *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
			// Verify request
			assert.Equal(t, "Test Model", source.Name)
			assert.Equal(t, "Test Description", source.Config.Description)
			assert.Equal(t, "id", source.Config.PrimaryKey)
			assert.Equal(t, "SELECT * FROM users", source.Config.Sql)
			assert.Equal(t, "test-account", source.AccountID)
			assert.Equal(t, "sql-model", source.SourceType)

			// Return mock response
			return &retlClient.RETLSource{
				ID:                   "remote-id",
				Name:                 source.Name,
				Config:               source.Config,
				SourceType:           source.SourceType,
				SourceDefinitionName: source.SourceDefinitionName,
				AccountID:            source.AccountID,
				CreatedAt:            &now,
			}, nil
		}

		// Create a test handler
		handler := NewSQLModelHandler(mockClient)

		// Test data
		data := resources.ResourceData{
			"id":           "test-model",
			"display_name": "Test Model",
			"description":  "Test Description",
			"account_id":   "test-account",
			"primary_key":  "id",
			"sql":          "SELECT * FROM users",
		}

		// Test creating a resource
		result, err := handler.Create(context.Background(), "test-model", data)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify the result
		assert.Equal(t, "test-model", (*result)["id"])
		assert.Equal(t, "Test Model", (*result)["display_name"])
		assert.Equal(t, "Test Description", (*result)["description"])
		assert.Equal(t, "test-account", (*result)["account_id"])
		assert.Equal(t, "id", (*result)["primary_key"])
		assert.Equal(t, "SELECT * FROM users", (*result)["sql"])
		assert.Equal(t, "remote-id", (*result)["source_id"])
		assert.Equal(t, "sql-model", (*result)["source_type"])
		assert.True(t, (*result)["enabled"].(bool))
		assert.Equal(t, &now, (*result)["created_at"])
	})

	t.Run("Update", func(t *testing.T) {
		t.Parallel()

		// Setup mock client
		mockClient := &mockRETLSourceClient{}
		mockClient.updateRetlSourceFunc = func(ctx context.Context, source *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error) {
			// Verify request
			assert.Equal(t, "Updated Model", source.Name)
			assert.Equal(t, "Updated Description", source.Config.Description)
			assert.Equal(t, "updated_id", source.Config.PrimaryKey)
			assert.Equal(t, "SELECT * FROM updated_users", source.Config.Sql)
			assert.Equal(t, "test-account", source.AccountID)

			// Return mock response
			updatedAt := now.Add(time.Hour)
			return &retlClient.RETLSource{
				ID:                   source.ID,
				Name:                 source.Name,
				Config:               source.Config,
				SourceType:           "model",
				SourceDefinitionName: "postgres",
				AccountID:            source.AccountID,
				CreatedAt:            &now,
				UpdatedAt:            &updatedAt,
			}, nil
		}

		// Create a test handler
		handler := NewSQLModelHandler(mockClient)

		// Test data
		data := resources.ResourceData{
			"id":           "test-model",
			"display_name": "Updated Model",
			"description":  "Updated Description",
			"account_id":   "test-account",
			"primary_key":  "updated_id",
			"sql":          "SELECT * FROM updated_users",
		}

		state := resources.ResourceData{
			"source_id": "remote-id",
		}

		// Test updating a resource
		result, err := handler.Update(context.Background(), "test-model", data, state)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify the result
		assert.Equal(t, "test-model", (*result)["id"])
		assert.Equal(t, "Updated Model", (*result)["display_name"])
		assert.Equal(t, "Updated Description", (*result)["description"])
		assert.Equal(t, "test-account", (*result)["account_id"])
		assert.Equal(t, "updated_id", (*result)["primary_key"])
		assert.Equal(t, "SELECT * FROM updated_users", (*result)["sql"])
		assert.Equal(t, "remote-id", (*result)["source_id"])
		assert.Equal(t, "model", (*result)["source_type"])
		assert.True(t, (*result)["enabled"].(bool))
		assert.Equal(t, &now, (*result)["created_at"])
		assert.NotNil(t, (*result)["updated_at"])
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()

		// Setup mock client
		mockClient := &mockRETLSourceClient{}
		deleted := false
		mockClient.deleteRetlSourceFunc = func(ctx context.Context, id string) error {
			// Verify request
			assert.Equal(t, "remote-id", id)
			deleted = true
			return nil
		}

		// Create a test handler
		handler := NewSQLModelHandler(mockClient)

		// Test data
		state := resources.ResourceData{
			"source_id": "remote-id",
		}

		// Test deleting a resource
		err := handler.Delete(context.Background(), "test-model", state)
		require.NoError(t, err)
		assert.True(t, deleted)
	})
}
