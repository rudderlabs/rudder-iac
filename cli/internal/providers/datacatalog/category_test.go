package datacatalog_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ catalog.DataCatalog = &MockCategoryCatalog{}

type MockCategoryCatalog struct {
	datacatalog.EmptyCatalog
	mockCategory *catalog.Category
	err          error
}

func (m *MockCategoryCatalog) SetCategory(category *catalog.Category) {
	m.mockCategory = category
}

func (m *MockCategoryCatalog) SetError(err error) {
	m.err = err
}

func (m *MockCategoryCatalog) CreateCategory(ctx context.Context, categoryCreate catalog.CategoryCreate) (*catalog.Category, error) {
	return m.mockCategory, m.err
}

func (m *MockCategoryCatalog) UpdateCategory(ctx context.Context, id string, categoryUpdate catalog.CategoryUpdate) (*catalog.Category, error) {
	return m.mockCategory, m.err
}

func (m *MockCategoryCatalog) DeleteCategory(ctx context.Context, categoryID string) error {
	return m.err
}

func (m *MockCategoryCatalog) GetCategory(ctx context.Context, id string) (*catalog.Category, error) {
	return m.mockCategory, m.err
}

func TestCategoryProviderOperations(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		t.Run("successful creation", func(t *testing.T) {
			t.Parallel()

			mockCatalog := &MockCategoryCatalog{}
			now := time.Now()
			mockCategory := &catalog.Category{
				ID:          "cat-123",
				Name:        "User Actions",
				WorkspaceID: "ws-456",
				ProjectId:   "test-id",
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			mockCatalog.SetCategory(mockCategory)

			provider := datacatalog.NewCategoryProvider(mockCatalog)

			inputData := resources.ResourceData{
				"name": "User Actions",
			}

			result, err := provider.Create(context.Background(), "test-id", inputData)

			require.NoError(t, err)
			require.NotNil(t, result)

			resultData := *result
			assert.Equal(t, "cat-123", resultData["id"])
			assert.Equal(t, "User Actions", resultData["name"])
			assert.Equal(t, "ws-456", resultData["workspaceId"])
			assert.NotEmpty(t, resultData["createdAt"])
			assert.NotEmpty(t, resultData["updatedAt"])

			// Verify args are properly embedded
			categoryArgs, ok := resultData["categoryArgs"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "User Actions", categoryArgs["name"])
		})

		t.Run("creation error", func(t *testing.T) {
			t.Parallel()

			mockCatalog := &MockCategoryCatalog{}
			mockCatalog.SetError(errors.New("creation failed"))

			provider := datacatalog.NewCategoryProvider(mockCatalog)

			inputData := resources.ResourceData{
				"name": "User Actions",
			}

			result, err := provider.Create(context.Background(), "test-id", inputData)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "creating category resource in upstream catalog")
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("successful update", func(t *testing.T) {
			t.Parallel()

			mockCatalog := &MockCategoryCatalog{}
			now := time.Now()
			updatedCategory := &catalog.Category{
				ID:          "cat-123",
				Name:        "Updated User Actions",
				WorkspaceID: "ws-456",
				ProjectId:   "test-project-id",
				CreatedAt:   now.Add(-time.Hour),
				UpdatedAt:   now,
			}
			mockCatalog.SetCategory(updatedCategory)

			provider := datacatalog.NewCategoryProvider(mockCatalog)

			inputData := resources.ResourceData{
				"name": "Updated User Actions",
			}

			oldStateData := resources.ResourceData{
				"id":          "cat-123",
				"name":        "User Actions",
				"workspaceId": "ws-456",
				"createdAt":   now.Add(-time.Hour).String(),
				"updatedAt":   now.Add(-time.Minute).String(),
				"categoryArgs": map[string]interface{}{
					"name": "User Actions",
				},
			}

			result, err := provider.Update(context.Background(), "test-id", inputData, oldStateData)

			require.NoError(t, err)
			require.NotNil(t, result)

			resultData := *result
			assert.Equal(t, "cat-123", resultData["id"])
			assert.Equal(t, "Updated User Actions", resultData["name"])
			assert.Equal(t, "ws-456", resultData["workspaceId"])
			assert.NotEmpty(t, resultData["createdAt"])
			assert.NotEmpty(t, resultData["updatedAt"])

			// Verify args are properly embedded
			categoryArgs, ok := resultData["categoryArgs"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "Updated User Actions", categoryArgs["name"])
		})

		t.Run("update error", func(t *testing.T) {
			t.Parallel()

			mockCatalog := &MockCategoryCatalog{}
			mockCatalog.SetError(errors.New("update failed"))

			provider := datacatalog.NewCategoryProvider(mockCatalog)

			inputData := resources.ResourceData{
				"name": "Updated User Actions",
			}

			oldStateData := resources.ResourceData{
				"id":          "cat-123",
				"name":        "User Actions",
				"workspaceId": "ws-456",
				"createdAt":   time.Now().Add(-time.Hour).String(),
				"updatedAt":   time.Now().Add(-time.Minute).String(),
				"categoryArgs": map[string]interface{}{
					"name": "User Actions",
				},
			}

			result, err := provider.Update(context.Background(), "test-id", inputData, oldStateData)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "updating category resource in upstream catalog")
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("successful deletion", func(t *testing.T) {
			t.Parallel()

			mockCatalog := &MockCategoryCatalog{}
			// No error set, should succeed

			provider := datacatalog.NewCategoryProvider(mockCatalog)

			stateData := resources.ResourceData{
				"id":          "cat-123",
				"name":        "User Actions",
				"workspaceId": "ws-456",
			}

			err := provider.Delete(context.Background(), "test-id", stateData)

			assert.NoError(t, err)
		})

		t.Run("not found error (should not fail)", func(t *testing.T) {
			t.Parallel()

			mockCatalog := &MockCategoryCatalog{}
			notFoundErr := &client.APIError{
				HTTPStatusCode: 400,
				Message:        "Category not found",
			}
			mockCatalog.SetError(notFoundErr)

			provider := datacatalog.NewCategoryProvider(mockCatalog)

			stateData := resources.ResourceData{
				"id":          "cat-123",
				"name":        "User Actions",
				"workspaceId": "ws-456",
			}

			err := provider.Delete(context.Background(), "test-id", stateData)

			assert.NoError(t, err) // Should not fail for not found errors
		})

		t.Run("other deletion error", func(t *testing.T) {
			t.Parallel()

			mockCatalog := &MockCategoryCatalog{}
			mockCatalog.SetError(errors.New("deletion failed"))

			provider := datacatalog.NewCategoryProvider(mockCatalog)

			stateData := resources.ResourceData{
				"id":          "cat-123",
				"name":        "User Actions",
				"workspaceId": "ws-456",
			}

			err := provider.Delete(context.Background(), "test-id", stateData)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "deleting category resource in upstream catalog")
		})
	})
}
