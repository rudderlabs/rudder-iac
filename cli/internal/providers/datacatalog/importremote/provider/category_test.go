package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
)

type mockCategoryCatalog struct {
	catalog.DataCatalog
	categories []*catalog.Category
	err        error
}

func (m *mockCategoryCatalog) GetCategories(ctx context.Context, options catalog.ListOptions) ([]*catalog.Category, error) {
	return m.categories, m.err
}

func TestCategoryLoadImportable(t *testing.T) {
	t.Run("filters categories with ExternalId set", func(t *testing.T) {
		mockClient := &mockCategoryCatalog{
			categories: []*catalog.Category{
				{ID: "cat1", Name: "User Actions", WorkspaceID: "ws1"},
				{ID: "cat2", Name: "E-commerce", WorkspaceID: "ws1", ExternalId: "ecommerce"},
				{ID: "cat3", Name: "Analytics", WorkspaceID: "ws1"},
			},
		}

		provider := &CategoryImportProvider{
			client:   mockClient,
			log:      *logger.New("test"),
			filepath: "data-catalog",
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		categories := collection.GetAll(state.CategoryResourceType)
		assert.Equal(t, 2, len(categories))

		resourceIDs := make([]string, 0, len(categories))
		for _, cat := range categories {
			resourceIDs = append(resourceIDs, cat.ID)
		}

		assert.True(t, lo.Every(resourceIDs, []string{"cat1", "cat3"}))
		assert.False(t, lo.Contains(resourceIDs, "cat2"))
	})

	t.Run("correctly assigns externalId and reference", func(t *testing.T) {
		mockClient := &mockCategoryCatalog{
			categories: []*catalog.Category{
				{ID: "cat1", Name: "User Actions", WorkspaceID: "ws1"},
				{ID: "cat2", Name: "E-commerce", WorkspaceID: "ws1"},
			},
		}

		provider := &CategoryImportProvider{
			client:   mockClient,
			log:      *logger.New("test"),
			filepath: "data-catalog",
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		categories := collection.GetAll(state.CategoryResourceType)
		require.Equal(t, 2, len(categories))

		cat1, ok := categories["cat1"]
		require.True(t, ok)
		assert.NotEmpty(t, cat1.ExternalID)
		assert.NotEmpty(t, cat1.Reference)

		cat2, ok := categories["cat2"]
		require.True(t, ok)
		assert.NotEmpty(t, cat2.ExternalID)
		assert.NotEmpty(t, cat2.Reference)
	})
}

func TestCategoryFormatForExport(t *testing.T) {
	t.Run("generates spec with correct structure", func(t *testing.T) {
		mockResolver := &mockResolver{
			references: map[string]map[string]string{},
		}

		mockClient := &mockCategoryCatalog{
			categories: []*catalog.Category{
				{ID: "cat1", Name: "User Actions", WorkspaceID: "ws1"},
				{ID: "cat2", Name: "E-commerce", WorkspaceID: "ws1"},
			},
		}

		provider := &CategoryImportProvider{
			client:   mockClient,
			log:      *logger.New("test"),
			filepath: "data-catalog",
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		result, err := provider.FormatForExport(
			context.Background(),
			collection,
			externalIdNamer,
			mockResolver,
		)
		require.Nil(t, err)
		require.Equal(t, 1, len(result))

		entity := result[0]
		assert.True(t, strings.HasSuffix(entity.RelativePath, "data-catalog"))

		spec, ok := entity.Content.(*specs.Spec)
		require.True(t, ok)

		assert.Equal(t, "categories", spec.Kind)
		assert.Equal(t, "categories", spec.Metadata["name"])
		assert.NotNil(t, spec.Metadata["import"])

		categories, ok := spec.Spec["categories"].([]map[string]any)
		require.True(t, ok)
		assert.Equal(t, 2, len(categories))
	})
}
