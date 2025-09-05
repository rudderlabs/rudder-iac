package state_test

import (
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
)

func TestCategoryArgs_ResourceData(t *testing.T) {
	args := state.CategoryArgs{
		ProjectId: "test-category-id",
		Name:      "Marketing",
	}

	t.Run("to resource data", func(t *testing.T) {
		t.Parallel()

		resourceData := args.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"projectId": "test-category-id",
			"name":      "Marketing",
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
			ProjectId: "test-category-id",
			Name:      "Marketing",
		},
		ID:          "category-id",
		Name:        "Marketing",
		WorkspaceID: "workspace-id",
		CreatedAt:   "2021-09-01T00:00:00Z",
		UpdatedAt:   "2021-09-01T00:00:00Z",
	}

	t.Run("to resource data", func(t *testing.T) {
		t.Parallel()

		resourceData := categoryState.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"id":          "category-id",
			"name":        "Marketing",
			"workspaceId": "workspace-id",
			"createdAt":   "2021-09-01T00:00:00Z",
			"updatedAt":   "2021-09-01T00:00:00Z",
			"categoryArgs": map[string]interface{}{
				"projectId": "test-category-id",
				"name":      "Marketing",
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

func TestCategoryArgs_FromRemoteCategory(t *testing.T) {
	t.Parallel()

	now := time.Now()

	remoteCategory := &catalog.Category{
		ID:          "category-123",
		Name:        "Test Category",
		WorkspaceID: "workspace-456",
		ProjectId:   "category-123-local",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resourceCollection := resources.NewResourceCollection()
	resourceCollection.SetCategories([]*catalog.Category{
		remoteCategory,
	})

	args := &state.CategoryArgs{}
	args.FromRemoteCategory(remoteCategory, resourceCollection)

	assert.Equal(t, "category-123-local", args.ProjectId)
	assert.Equal(t, "Test Category", args.Name)
}
