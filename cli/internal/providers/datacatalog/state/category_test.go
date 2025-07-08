package state_test

import (
	"testing"
	"time"

	catalog "github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
)

func TestCategoryArgs_ResourceData(t *testing.T) {
	args := state.CategoryArgs{
		Name: "Marketing Categories",
	}

	t.Run("to resource data", func(t *testing.T) {
		t.Parallel()

		resourceData := args.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"name": "Marketing Categories",
		}, resourceData)
	})

	t.Run("from resource data", func(t *testing.T) {
		t.Parallel()

		loopback := state.CategoryArgs{}
		loopback.FromResourceData(args.ToResourceData())
		assert.Equal(t, args, loopback)
	})
}

func TestCategoryState_ResourceData(t *testing.T) {
	categoryState := state.CategoryState{
		CategoryArgs: state.CategoryArgs{
			Name: "Marketing Categories",
		},
		ID:          "upstream-category-catalog-id",
		Name:        "Marketing Categories",
		WorkspaceID: "workspace-id",
		CreatedAt:   "2021-09-01T00:00:00Z",
		UpdatedAt:   "2021-09-01T00:00:00Z",
	}

	t.Run("to resource data", func(t *testing.T) {
		t.Parallel()

		resourceData := categoryState.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"id":          "upstream-category-catalog-id",
			"name":        "Marketing Categories",
			"workspaceId": "workspace-id",
			"createdAt":   "2021-09-01T00:00:00Z",
			"updatedAt":   "2021-09-01T00:00:00Z",
			"categoryArgs": map[string]interface{}{
				"name": "Marketing Categories",
			},
		}, resourceData)
	})

	t.Run("from resource data", func(t *testing.T) {
		t.Parallel()

		loopback := state.CategoryState{}
		loopback.FromResourceData(categoryState.ToResourceData())
		assert.Equal(t, categoryState, loopback)
	})
}

func TestCategoryStateFromAPI(t *testing.T) {
	createdTime := time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC)
	updatedTime := time.Date(2021, 9, 2, 0, 0, 0, 0, time.UTC)

	apiCategory := &catalog.Category{
		ID:          "api-category-id",
		Name:        "Test Category",
		WorkspaceID: "workspace-id",
		CreatedAt:   createdTime,
		UpdatedAt:   updatedTime,
	}

	t.Run("converts API category to state correctly", func(t *testing.T) {
		t.Parallel()

		categoryState := state.CategoryStateFromAPI(apiCategory)

		expectedState := &state.CategoryState{
			CategoryArgs: state.CategoryArgs{
				Name: "Test Category",
			},
			ID:          "api-category-id",
			Name:        "Test Category",
			WorkspaceID: "workspace-id",
			CreatedAt:   "2021-09-01T00:00:00Z",
			UpdatedAt:   "2021-09-02T00:00:00Z",
		}

		assert.Equal(t, expectedState, categoryState)
	})
}

func TestCategoryStateToAPI(t *testing.T) {
	categoryState := &state.CategoryState{
		CategoryArgs: state.CategoryArgs{
			Name: "Test Category",
		},
		ID:          "state-category-id",
		Name:        "Test Category",
		WorkspaceID: "workspace-id",
		CreatedAt:   "2021-09-01T00:00:00Z",
		UpdatedAt:   "2021-09-02T00:00:00Z",
	}

	t.Run("converts state to API category correctly", func(t *testing.T) {
		t.Parallel()

		apiCategory := state.CategoryStateToAPI(categoryState)

		expectedCategory := &catalog.Category{
			ID:          "state-category-id",
			Name:        "Test Category",
			WorkspaceID: "workspace-id",
			CreatedAt:   time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2021, 9, 2, 0, 0, 0, 0, time.UTC),
		}

		assert.Equal(t, expectedCategory, apiCategory)
	})
}

func TestCategoryStateCollection(t *testing.T) {
	category1 := &state.CategoryState{
		ID:   "category-1",
		Name: "Marketing",
	}

	category2 := &state.CategoryState{
		ID:   "category-2",
		Name: "Engineering",
	}

	category3 := &state.CategoryState{
		ID:   "category-3",
		Name: "Sales",
	}

	t.Run("GetByID", func(t *testing.T) {
		t.Parallel()

		collection := state.CategoryStateCollection{category1, category2, category3}

		// Test finding existing category
		found := collection.GetByID("category-2")
		assert.Equal(t, category2, found)

		// Test not finding category
		notFound := collection.GetByID("non-existent")
		assert.Nil(t, notFound)
	})

	t.Run("GetByName", func(t *testing.T) {
		t.Parallel()

		collection := state.CategoryStateCollection{category1, category2, category3}

		// Test finding existing category
		found := collection.GetByName("Engineering")
		assert.Equal(t, category2, found)

		// Test not finding category
		notFound := collection.GetByName("Non-existent")
		assert.Nil(t, notFound)
	})

	t.Run("Add", func(t *testing.T) {
		t.Parallel()

		collection := state.CategoryStateCollection{category1}
		collection.Add(category2)

		assert.Equal(t, 2, len(collection))
		assert.Equal(t, category1, collection[0])
		assert.Equal(t, category2, collection[1])
	})

	t.Run("Update", func(t *testing.T) {
		t.Parallel()

		collection := state.CategoryStateCollection{category1, category2}

		updatedCategory := &state.CategoryState{
			ID:   "category-2",
			Name: "Updated Engineering",
		}

		// Test successful update
		updated := collection.Update(updatedCategory)
		assert.True(t, updated)
		assert.Equal(t, "Updated Engineering", collection[1].Name)

		// Test update of non-existent category
		nonExistentCategory := &state.CategoryState{
			ID:   "non-existent",
			Name: "Non-existent",
		}
		notUpdated := collection.Update(nonExistentCategory)
		assert.False(t, notUpdated)
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()

		collection := state.CategoryStateCollection{category1, category2, category3}

		// Test successful deletion
		deleted := collection.Delete("category-2")
		assert.True(t, deleted)
		assert.Equal(t, 2, len(collection))
		assert.Equal(t, category1, collection[0])
		assert.Equal(t, category3, collection[1])

		// Test deletion of non-existent category
		notDeleted := collection.Delete("non-existent")
		assert.False(t, notDeleted)
		assert.Equal(t, 2, len(collection))
	})

	t.Run("All", func(t *testing.T) {
		t.Parallel()

		collection := state.CategoryStateCollection{category1, category2, category3}

		all := collection.All()
		assert.Equal(t, 3, len(all))
		assert.Equal(t, category1, all[0])
		assert.Equal(t, category2, all[1])
		assert.Equal(t, category3, all[2])
	})
}
