package datacatalog_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ catalog.DataCatalog = &MockCategoryCatalog{}

type MockCategoryCatalog struct {
	datacatalog.EmptyCatalog
	mockCategory        *catalog.Category
	err                 error
	updateCalled        bool
	setExternalIdCalled bool
}

func (m *MockCategoryCatalog) SetCategory(category *catalog.Category) {
	m.mockCategory = category
}

func (m *MockCategoryCatalog) SetError(err error) {
	m.err = err
}

func (m *MockCategoryCatalog) ResetSpies() {
	m.updateCalled = false
	m.setExternalIdCalled = false
}

func (m *MockCategoryCatalog) CreateCategory(ctx context.Context, categoryCreate catalog.CategoryCreate) (*catalog.Category, error) {
	return m.mockCategory, m.err
}

func (m *MockCategoryCatalog) UpdateCategory(ctx context.Context, id string, categoryUpdate catalog.CategoryUpdate) (*catalog.Category, error) {
	m.updateCalled = true
	if m.mockCategory != nil {
		m.mockCategory.Name = categoryUpdate.Name
	}
	return m.mockCategory, m.err
}

func (m *MockCategoryCatalog) DeleteCategory(ctx context.Context, categoryID string) error {
	return m.err
}

func (m *MockCategoryCatalog) GetCategory(ctx context.Context, id string) (*catalog.Category, error) {
	return m.mockCategory, m.err
}

func (m *MockCategoryCatalog) SetCategoryExternalId(ctx context.Context, categoryID, externalID string) error {
	m.setExternalIdCalled = true
	return m.err
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
				ExternalId:  "test-id",
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			mockCatalog.SetCategory(mockCategory)

			provider := datacatalog.NewCategoryProvider(mockCatalog, "test-import-dir")

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

			provider := datacatalog.NewCategoryProvider(mockCatalog, "test-import-dir")

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
				ExternalId:  "test-project-id",
				CreatedAt:   now.Add(-time.Hour),
				UpdatedAt:   now,
			}
			mockCatalog.SetCategory(updatedCategory)

			provider := datacatalog.NewCategoryProvider(mockCatalog, "test-import-dir")

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

			provider := datacatalog.NewCategoryProvider(mockCatalog, "test-import-dir")

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

			provider := datacatalog.NewCategoryProvider(mockCatalog, "test-import-dir")

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

			provider := datacatalog.NewCategoryProvider(mockCatalog, "test-import-dir")

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

			provider := datacatalog.NewCategoryProvider(mockCatalog, "test-import-dir")

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

	t.Run("Import", func(t *testing.T) {
		createdAt, _ := time.Parse(time.RFC3339, "2021-09-01T00:00:00Z")
		updatedAt, _ := time.Parse(time.RFC3339, "2021-09-02T00:00:00Z")
		mockCatalog := &MockCategoryCatalog{}
		categoryProvider := datacatalog.NewCategoryProvider(mockCatalog, "data-catalog")
		ctx := context.Background()

		tests := []struct {
			name           string
			localArgs      state.CategoryArgs
			remoteCategory *catalog.Category
			mockErr        error
			expectErr      bool
			expectUpdate   bool
			expectSetExtId bool
			expectResource *resources.ResourceData
		}{
			{
				name: "successful import no differences",
				localArgs: state.CategoryArgs{
					Name: "test-category",
				},
				remoteCategory: &catalog.Category{
					ID:          "remote-id",
					Name:        "test-category",
					WorkspaceID: "ws-id",
					CreatedAt:   createdAt,
					UpdatedAt:   updatedAt,
				},
				expectErr:      false,
				expectUpdate:   false,
				expectSetExtId: true,
				expectResource: &resources.ResourceData{
					"id":          "remote-id",
					"name":        "test-category",
					"workspaceId": "ws-id",
					"createdAt":   createdAt.String(),
					"updatedAt":   updatedAt.String(),
					"categoryArgs": map[string]interface{}{
						"name": "test-category",
					},
				},
			},
			{
				name: "successful import with differences",
				localArgs: state.CategoryArgs{
					Name: "new-category",
				},
				remoteCategory: &catalog.Category{
					ID:          "remote-id",
					Name:        "old-category",
					WorkspaceID: "ws-id",
					CreatedAt:   createdAt,
					UpdatedAt:   updatedAt,
				},
				expectErr:      false,
				expectUpdate:   true,
				expectSetExtId: true,
				expectResource: &resources.ResourceData{
					"id":          "remote-id",
					"name":        "new-category",
					"workspaceId": "ws-id",
					"createdAt":   createdAt.String(),
					"updatedAt":   updatedAt.String(),
					"categoryArgs": map[string]interface{}{
						"name": "new-category",
					},
				},
			},
			{
				name:           "error on get category",
				localArgs:      state.CategoryArgs{Name: "test-category"},
				remoteCategory: nil,
				mockErr:        fmt.Errorf("error getting category"),
				expectErr:      true,
			},
			{
				name: "error on update category",
				localArgs: state.CategoryArgs{
					Name: "new-category",
				},
				remoteCategory: &catalog.Category{
					ID:   "remote-id",
					Name: "old-category",
				},
				mockErr:      fmt.Errorf("error updating category"),
				expectErr:    true,
				expectUpdate: true,
			},
			{
				name: "error on set external ID",
				localArgs: state.CategoryArgs{
					Name: "test-category",
				},
				remoteCategory: &catalog.Category{
					ID:   "remote-id",
					Name: "test-category",
				},
				mockErr:        fmt.Errorf("error setting external ID"),
				expectErr:      true,
				expectSetExtId: true,
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				mockCatalog.ResetSpies()
				mockCatalog.SetCategory(tt.remoteCategory)
				mockCatalog.SetError(tt.mockErr)

				res, err := categoryProvider.Import(ctx, "local-id", tt.localArgs.ToResourceData(), "remote-id")

				if tt.expectErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.expectResource, res)
				assert.Equal(t, tt.expectUpdate, mockCatalog.updateCalled)
				assert.Equal(t, tt.expectSetExtId, mockCatalog.setExternalIdCalled)
			})
		}
	})
}
