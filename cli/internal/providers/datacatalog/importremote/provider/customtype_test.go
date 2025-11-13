package provider

import (
	"context"
	"path/filepath"
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

type mockCustomTypeDataCatalog struct {
	catalog.DataCatalog
	customTypes []*catalog.CustomType
	err         error
}

func (m *mockCustomTypeDataCatalog) GetCustomTypes(ctx context.Context, options catalog.ListOptions) ([]*catalog.CustomType, error) {
	return m.customTypes, m.err
}

func TestCustomTypeLoadImportable(t *testing.T) {
	t.Run("filters custom types with ExternalId set", func(t *testing.T) {
		mockClient := &mockCustomTypeDataCatalog{
			customTypes: []*catalog.CustomType{
				{ID: "ct1", Name: "Custom Type 1", Type: "object", WorkspaceId: "ws1"},
				{ID: "ct2", Name: "Custom Type 2", Type: "object", WorkspaceId: "ws1", ExternalID: "custom-type-2"},
				{ID: "ct3", Name: "Custom Type 3", Type: "object", WorkspaceId: "ws1"},
			},
		}

		provider := &CustomTypeImportProvider{
			client:   mockClient,
			log:      *logger.New("test"),
			filepath: "custom-types",
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		customTypes := collection.GetAll(state.CustomTypeResourceType)
		assert.Equal(t, 2, len(customTypes))

		resourceIDs := make([]string, 0, len(customTypes))
		for _, ct := range customTypes {
			resourceIDs = append(resourceIDs, ct.ID)
		}

		assert.True(t, lo.Every(resourceIDs, []string{"ct1", "ct3"}))
		assert.False(t, lo.Contains(resourceIDs, "ct2"))
	})

	t.Run("correctly assigns externalId and reference after namer is loaded", func(t *testing.T) {
		mockClient := &mockCustomTypeDataCatalog{
			customTypes: []*catalog.CustomType{
				{ID: "ct1", Name: "Address Type", Type: "object", WorkspaceId: "ws1"},
				{ID: "ct2", Name: "Product Type", Type: "object", WorkspaceId: "ws1"},
			},
		}

		provider := &CustomTypeImportProvider{
			client:   mockClient,
			log:      *logger.New("test"),
			filepath: "custom-types",
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		customTypes := collection.GetAll(state.CustomTypeResourceType)
		require.Equal(t, 2, len(customTypes))

		ct1, ok := customTypes["ct1"]
		require.True(t, ok)
		assert.NotEmpty(t, ct1.ExternalID)
		assert.NotEmpty(t, ct1.Reference)

		ct2, ok := customTypes["ct2"]
		require.True(t, ok)
		assert.NotEmpty(t, ct2.ExternalID)
		assert.NotEmpty(t, ct2.Reference)
	})
}

func TestCustomTypeFormatForExport(t *testing.T) {
	t.Run("generates spec with correct relativePath and content structure", func(t *testing.T) {

		mockResolver := &mockResolver{
			references: map[string]map[string]string{},
		}

		mockClient := &mockCustomTypeDataCatalog{
			customTypes: []*catalog.CustomType{
				{ID: "ct1", Name: "Address Type", Type: "object", WorkspaceId: "ws1"},
				{ID: "ct2", Name: "Product Type", Type: "object", WorkspaceId: "ws1"},
			},
		}

		provider := NewCustomTypeImportProvider(
			mockClient,
			*logger.New("test"),
			"data-catalog",
		)

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
		assert.Equal(t, entity.RelativePath, filepath.Join(
			"data-catalog",
			"custom-types/custom-types.yaml",
		))
		spec, ok := entity.Content.(*specs.Spec)
		require.True(t, ok)

		assert.Equal(t, "custom-types", spec.Kind)
		assert.Equal(t, "custom-types", spec.Metadata["name"])
		assert.NotNil(t, spec.Metadata["import"])

		customTypes, ok := spec.Spec["types"].([]map[string]any)
		require.True(t, ok)
		assert.Equal(t, 2, len(customTypes))
	})
}
