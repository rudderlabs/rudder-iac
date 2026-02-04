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
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
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
				{ID: "cat2", Name: "E-commerce", WorkspaceID: "ws1", ExternalID: "ecommerce"},
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

		categories := collection.GetAll(types.CategoryResourceType)
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

		categories := collection.GetAll(types.CategoryResourceType)
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

	t.Run("creates v0 spec when v1 support disabled", func(t *testing.T) {
		mockResolver := &mockResolver{
			references: map[string]map[string]string{},
		}

		mockClient := &mockCategoryCatalog{
			categories: []*catalog.Category{
				{ID: "cat1", Name: "User Actions", WorkspaceID: "ws1"},
			},
		}

		provider := &CategoryImportProvider{
			client:        mockClient,
			log:           *logger.New("test"),
			filepath:      "data-catalog",
			v1SpecSupport: false,
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.NoError(t, err)

		result, err := provider.FormatForExport(collection, externalIdNamer, mockResolver)
		require.NoError(t, err)
		require.Len(t, result, 1)

		spec, ok := result[0].Content.(*specs.Spec)
		require.True(t, ok)

		categories, ok := spec.Spec["categories"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, categories, 1)
		assert.Contains(t, categories[0], "id")
		assert.Contains(t, categories[0], "name")
	})

	t.Run("creates v1 spec when v1 support enabled", func(t *testing.T) {
		mockResolver := &mockResolver{
			references: map[string]map[string]string{},
		}

		mockClient := &mockCategoryCatalog{
			categories: []*catalog.Category{
				{ID: "cat1", Name: "User Actions", WorkspaceID: "ws1"},
			},
		}

		provider := &CategoryImportProvider{
			client:        mockClient,
			log:           *logger.New("test"),
			filepath:      "data-catalog",
			v1SpecSupport: true,
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.NoError(t, err)

		result, err := provider.FormatForExport(collection, externalIdNamer, mockResolver)
		require.NoError(t, err)
		require.Len(t, result, 1)

		spec, ok := result[0].Content.(*specs.Spec)
		require.True(t, ok)

		categories, ok := spec.Spec["categories"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, categories, 1)
		assert.Contains(t, categories[0], "id")
		assert.Contains(t, categories[0], "name")
	})
}
