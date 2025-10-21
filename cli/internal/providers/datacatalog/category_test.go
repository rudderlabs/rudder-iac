package datacatalog_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockCategoryCatalog struct {
	datacatalog.EmptyCatalog
	category            *catalog.Category
	err                 error
	updateCalled        bool
	setExternalIdCalled bool
}

func (m *MockCategoryCatalog) CreateCategory(ctx context.Context, categoryCreate catalog.CategoryCreate) (*catalog.Category, error) {
	return m.category, m.err
}

func (m *MockCategoryCatalog) UpdateCategory(ctx context.Context, id string, categoryUpdate catalog.CategoryUpdate) (*catalog.Category, error) {
	m.updateCalled = true
	m.category.Name = categoryUpdate.Name
	return m.category, m.err
}

func (m *MockCategoryCatalog) DeleteCategory(ctx context.Context, categoryID string) error {
	return m.err
}

func (m *MockCategoryCatalog) GetCategory(ctx context.Context, id string) (*catalog.Category, error) {
	return m.category, m.err
}

func (m *MockCategoryCatalog) SetCategoryExternalId(ctx context.Context, categoryID, externalID string) error {
	m.setExternalIdCalled = true
	return m.err
}

func (m *MockCategoryCatalog) SetCategory(category *catalog.Category) {
	m.category = category
}

func (m *MockCategoryCatalog) SetError(err error) {
	m.err = err
}

func (m *MockCategoryCatalog) ResetSpies() {
	m.updateCalled = false
	m.setExternalIdCalled = false
}

func TestCategoryProviderOperations(t *testing.T) {
	var (
		ctx              = context.Background()
		mockCatalog      = &MockCategoryCatalog{}
		categoryProvider = datacatalog.NewCategoryProvider(mockCatalog, "data-catalog")
		createdAt, _     = time.Parse(time.RFC3339, "2021-09-01T00:00:00Z")
		updatedAt, _     = time.Parse(time.RFC3339, "2021-09-02T00:00:00Z")
	)

	t.Run("Create", func(t *testing.T) {
		mockCatalog.SetCategory(&catalog.Category{
			ID:          "upstream-catalog-id",
			Name:        "category",
			WorkspaceID: "workspace-id",
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})

		toArgs := state.CategoryArgs{
			Name: "category",
		}

		resourceData, err := categoryProvider.Create(ctx, "category-id", toArgs.ToResourceData())
		require.NoError(t, err)
		assert.Equal(t, resources.ResourceData{
			"id":          "upstream-catalog-id",
			"name":        "category",
			"workspaceId": "workspace-id",
			"createdAt":   "2021-09-01 00:00:00 +0000 UTC",
			"updatedAt":   "2021-09-02 00:00:00 +0000 UTC",
			"categoryArgs": map[string]interface{}{
				"name": "category",
			},
		}, *resourceData)
	})

	t.Run("Update", func(t *testing.T) {
		prevState := state.CategoryState{
			CategoryArgs: state.CategoryArgs{
				Name: "old-category",
			},
			ID:          "upstream-catalog-id",
			Name:        "old-category",
			WorkspaceID: "workspace-id",
			CreatedAt:   "2021-09-01 00:00:00 +0000 UTC",
			UpdatedAt:   "2021-09-02 00:00:00 +0000 UTC",
		}

		toArgs := state.CategoryArgs{
			Name: "new-category",
		}

		mockCatalog.SetCategory(&catalog.Category{
			ID:          "upstream-catalog-id",
			Name:        "new-category",
			WorkspaceID: "workspace-id",
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})

		updatedResource, err := categoryProvider.Update(ctx, "category-id", toArgs.ToResourceData(), prevState.ToResourceData())
		require.NoError(t, err)
		assert.Equal(t, resources.ResourceData{
			"id":          "upstream-catalog-id",
			"name":        "new-category",
			"workspaceId": "workspace-id",
			"createdAt":   "2021-09-01 00:00:00 +0000 UTC",
			"updatedAt":   "2021-09-02 00:00:00 +0000 UTC",
			"categoryArgs": map[string]interface{}{
				"name": "new-category",
			},
		}, *updatedResource)
	})

	t.Run("Delete", func(t *testing.T) {
		prevState := state.CategoryState{
			ID: "upstream-catalog-id",
		}
		mockCatalog.SetError(nil)

		err := categoryProvider.Delete(ctx, "category-id", prevState.ToResourceData())
		require.NoError(t, err)
	})

	t.Run("Import", func(t *testing.T) {
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
