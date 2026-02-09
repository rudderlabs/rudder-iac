package provider

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
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

		customTypes := collection.GetAll(types.CustomTypeResourceType)
		assert.Equal(t, 2, len(customTypes))

		resourceIDs := make([]string, 0, len(customTypes))
		for _, ct := range customTypes {
			resourceIDs = append(resourceIDs, ct.ID)
		}

		assert.True(t, lo.Every(resourceIDs, []string{"ct1", "ct3"}))
		assert.False(t, lo.Contains(resourceIDs, "ct2"))
	})

	t.Run("correctly assigns externalId and old path based reference after namer is loaded", func(t *testing.T) {
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

		customTypes := collection.GetAll(types.CustomTypeResourceType)
		require.Equal(t, 2, len(customTypes))

		ct1, ok := customTypes["ct1"]
		require.True(t, ok)
		assert.NotEmpty(t, ct1.ExternalID)
		assert.Equal(t, ct1.Reference, fmt.Sprintf("#/%s/%s/%s", localcatalog.KindCustomTypes, MetadataNameCustomTypes, ct1.ExternalID))

		ct2, ok := customTypes["ct2"]
		require.True(t, ok)
		assert.NotEmpty(t, ct2.ExternalID)
		assert.Equal(t, ct2.Reference, fmt.Sprintf("#/%s/%s/%s", localcatalog.KindCustomTypes, MetadataNameCustomTypes, ct2.ExternalID))
	})

	t.Run("correctly assigns externalId and new URN based reference after namer is loaded", func(t *testing.T) {
		mockClient := &mockCustomTypeDataCatalog{
			customTypes: []*catalog.CustomType{
				{ID: "ct1", Name: "Address Type", Type: "object", WorkspaceId: "ws1"},
				{ID: "ct2", Name: "Product Type", Type: "object", WorkspaceId: "ws1"},
			},
		}

		provider := &CustomTypeImportProvider{
			client:        mockClient,
			log:           *logger.New("test"),
			filepath:      "custom-types",
			v1SpecSupport: true,
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		customTypes := collection.GetAll(types.CustomTypeResourceType)
		require.Equal(t, 2, len(customTypes))

		ct1, ok := customTypes["ct1"]
		require.True(t, ok)
		assert.NotEmpty(t, ct1.ExternalID)
		assert.Equal(t, ct1.Reference, fmt.Sprintf("#%s:%s", types.CustomTypeResourceType, ct1.ExternalID))

		ct2, ok := customTypes["ct2"]
		require.True(t, ok)
		assert.NotEmpty(t, ct2.ExternalID)
		assert.Equal(t, ct2.Reference, fmt.Sprintf("#%s:%s", types.CustomTypeResourceType, ct2.ExternalID))
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

	t.Run("creates v0 spec when v1 support disabled", func(t *testing.T) {
		mockResolver := &mockResolver{
			references: map[string]map[string]string{
				types.PropertyResourceType: {
					"prop1": "#property:street",
				},
			},
		}

		mockClient := &mockCustomTypeDataCatalog{
			customTypes: []*catalog.CustomType{
				{
					ID:          "ct1",
					Name:        "Address",
					Type:        "object",
					WorkspaceId: "ws1",
					Properties: []catalog.CustomTypeProperty{
						{ID: "prop1", Required: true},
					},
				},
			},
		}

		provider := &CustomTypeImportProvider{
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
		assert.Equal(t, specs.SpecVersionV0_1, spec.Version)

		customTypes, ok := spec.Spec["types"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, customTypes, 1)

		properties, ok := customTypes[0]["properties"].([]any)
		require.True(t, ok)
		require.Len(t, properties, 1)

		propMap, ok := properties[0].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, propMap, "$ref")
	})

	t.Run("creates v1 spec when v1 support enabled", func(t *testing.T) {
		mockResolver := &mockResolver{
			references: map[string]map[string]string{
				types.PropertyResourceType: {
					"prop1": "#property:street",
				},
			},
		}

		mockClient := &mockCustomTypeDataCatalog{
			customTypes: []*catalog.CustomType{
				{
					ID:          "ct1",
					Name:        "Address",
					Type:        "object",
					WorkspaceId: "ws1",
					Properties: []catalog.CustomTypeProperty{
						{ID: "prop1", Required: true},
					},
				},
			},
		}

		provider := &CustomTypeImportProvider{
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
		assert.Equal(t, specs.SpecVersionV1, spec.Version)

		customTypes, ok := spec.Spec["types"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, customTypes, 1)

		properties, ok := customTypes[0]["properties"].([]any)
		require.True(t, ok)
		require.Len(t, properties, 1)

		propMap, ok := properties[0].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, propMap, "property")
	})
}
