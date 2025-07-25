package state_test

import (
	"testing"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
)

func TestCategoryArgs_ResourceData(t *testing.T) {
	args := state.CategoryArgs{
		Name: "Marketing",
	}

	t.Run("to resource data", func(t *testing.T) {
		t.Parallel()

		resourceData := args.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"name": "Marketing",
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
			Name: "Marketing",
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
				"name": "Marketing",
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
