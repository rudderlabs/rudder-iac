package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
)

// categoryResource creates a category resource with name in its data,
// matching the shape produced by CategoryArgs.ToResourceData().
func categoryResource(id, name string) *resources.Resource {
	data := resources.ResourceData{
		"name": name,
	}
	return resources.NewResource(id, "category", data, nil)
}

func TestCategorySemanticValid_Uniqueness(t *testing.T) {
	t.Parallel()

	t.Run("no duplicate — unique categories", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(categoryResource("user_actions", "User Actions"))
		graph.AddResource(categoryResource("system_events", "System Events"))

		spec := localcatalog.CategorySpec{
			Categories: []localcatalog.Category{
				{LocalID: "user_actions", Name: "User Actions"},
			},
		}

		results := validateCategorySemantic(localcatalog.KindCategories, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "unique category names should not trigger errors")
	})

	t.Run("duplicate category detected", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(categoryResource("user_actions_v1", "User Actions"))
		graph.AddResource(categoryResource("user_actions_v2", "User Actions"))

		spec := localcatalog.CategorySpec{
			Categories: []localcatalog.Category{
				{LocalID: "user_actions_v1", Name: "User Actions"},
			},
		}

		results := validateCategorySemantic(localcatalog.KindCategories, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/categories/0/name", results[0].Reference)
		assert.Contains(t, results[0].Message, "not unique across the catalog")
	})

	t.Run("single category in graph — no false positive", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(categoryResource("user_actions", "User Actions"))

		spec := localcatalog.CategorySpec{
			Categories: []localcatalog.Category{
				{LocalID: "user_actions", Name: "User Actions"},
			},
		}

		results := validateCategorySemantic(localcatalog.KindCategories, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "single category should not be flagged")
	})

	t.Run("multiple duplicates reported independently", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(categoryResource("ua_v1", "User Actions"))
		graph.AddResource(categoryResource("ua_v2", "User Actions"))
		graph.AddResource(categoryResource("se_v1", "System Events"))
		graph.AddResource(categoryResource("se_v2", "System Events"))

		spec := localcatalog.CategorySpec{
			Categories: []localcatalog.Category{
				{LocalID: "ua_v1", Name: "User Actions"},
				{LocalID: "se_v1", Name: "System Events"},
			},
		}

		results := validateCategorySemantic(localcatalog.KindCategories, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 2)
		assert.Equal(t, "/categories/0/name", results[0].Reference)
		assert.Contains(t, results[0].Message, "not unique across the catalog")
		assert.Equal(t, "/categories/1/name", results[1].Reference)
		assert.Contains(t, results[1].Message, "not unique across the catalog")
	})

	t.Run("mixed unique and duplicate — only duplicate flagged", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(categoryResource("ua_v1", "User Actions"))
		graph.AddResource(categoryResource("ua_v2", "User Actions"))
		graph.AddResource(categoryResource("analytics", "Analytics"))

		spec := localcatalog.CategorySpec{
			Categories: []localcatalog.Category{
				{LocalID: "ua_v1", Name: "User Actions"},
				{LocalID: "analytics", Name: "Analytics"},
			},
		}

		results := validateCategorySemantic(localcatalog.KindCategories, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/categories/0/name", results[0].Reference)
		assert.Contains(t, results[0].Message, "category with name 'User Actions' is not unique")
	})
}
