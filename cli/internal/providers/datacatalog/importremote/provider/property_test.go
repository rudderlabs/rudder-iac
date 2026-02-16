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

type mockDataCatalog struct {
	catalog.DataCatalog
	properties []*catalog.Property
	err        error
}

func (m *mockDataCatalog) GetProperties(ctx context.Context, options catalog.ListOptions) ([]*catalog.Property, error) {
	return m.properties, m.err
}

type mockResolver struct {
	references map[string]map[string]string
}

func (m *mockResolver) ResolveToReference(entityType string, remoteID string) (string, error) {
	if typeMap, ok := m.references[entityType]; ok {
		if ref, ok := typeMap[remoteID]; ok {
			return ref, nil
		}
	}
	return "", nil
}

func TestLoadImportable(t *testing.T) {
	t.Run("filters properties with ExternalId set", func(t *testing.T) {
		mockClient := &mockDataCatalog{
			properties: []*catalog.Property{
				{ID: "prop1", Name: "Property 1", Type: "string", WorkspaceId: "ws1"},
				{ID: "prop2", Name: "Property 2", Type: "number", WorkspaceId: "ws1", ExternalID: "property-2"},
				{ID: "prop3", Name: "Property 3", Type: "boolean", WorkspaceId: "ws1"},
			},
		}

		provider := &PropertyImportProvider{
			client:   mockClient,
			log:      *logger.New("test"),
			filepath: "data-catalog",
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		properties := collection.GetAll(types.PropertyResourceType)
		assert.Equal(t, 2, len(properties))

		resourceIDs := make([]string, 0, len(properties))
		for _, prop := range properties {
			resourceIDs = append(resourceIDs, prop.ID)
		}

		assert.True(t, lo.Every(resourceIDs, []string{"prop1", "prop3"}))
		assert.False(t, lo.Contains(resourceIDs, "prop2"))
	})

	t.Run("correctly assigns externalId and old path based reference after namer is loaded", func(t *testing.T) {
		mockClient := &mockDataCatalog{
			properties: []*catalog.Property{
				{ID: "prop1", Name: "User Email", Type: "string", WorkspaceId: "ws1"},
				{ID: "prop2", Name: "User Age", Type: "number", WorkspaceId: "ws1"},
			},
		}

		provider := &PropertyImportProvider{
			client:   mockClient,
			log:      *logger.New("test"),
			filepath: "data-catalog",
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		properties := collection.GetAll(types.PropertyResourceType)
		require.Equal(t, 2, len(properties))

		prop1, ok := properties["prop1"]
		require.True(t, ok)
		assert.NotEmpty(t, prop1.ExternalID)
		assert.Equal(t, prop1.Reference, fmt.Sprintf("#/%s/%s/%s", localcatalog.KindProperties, MetadataNameProperties, prop1.ExternalID))

		prop2, ok := properties["prop2"]
		require.True(t, ok)
		assert.NotEmpty(t, prop2.ExternalID)
		assert.Equal(t, prop2.Reference, fmt.Sprintf("#/%s/%s/%s", localcatalog.KindProperties, MetadataNameProperties, prop2.ExternalID))
	})

	t.Run("correctly assigns externalId and new URN based reference after namer is loaded", func(t *testing.T) {
		mockClient := &mockDataCatalog{
			properties: []*catalog.Property{
				{ID: "prop1", Name: "User Email", Type: "string", WorkspaceId: "ws1"},
				{ID: "prop2", Name: "User Age", Type: "number", WorkspaceId: "ws1"},
			},
		}

		provider := &PropertyImportProvider{
			client:        mockClient,
			log:           *logger.New("test"),
			filepath:      "data-catalog",
			v1SpecSupport: true,
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		properties := collection.GetAll(types.PropertyResourceType)
		require.Equal(t, 2, len(properties))

		prop1, ok := properties["prop1"]
		require.True(t, ok)
		assert.NotEmpty(t, prop1.ExternalID)
		assert.Equal(t, prop1.Reference, fmt.Sprintf("#%s:%s", types.PropertyResourceType, prop1.ExternalID))

		prop2, ok := properties["prop2"]
		require.True(t, ok)
		assert.NotEmpty(t, prop2.ExternalID)
		assert.NotEmpty(t, prop2.Reference)
	})
}

func TestFormatForExport(t *testing.T) {
	t.Run("generates spec with correct relativePath and content structure", func(t *testing.T) {

		mockResolver := &mockResolver{
			references: map[string]map[string]string{},
		}

		mockClient := &mockDataCatalog{
			properties: []*catalog.Property{
				{ID: "prop1", Name: "User Email", Type: "string", WorkspaceId: "ws1"},
				{ID: "prop2", Name: "User Age", Type: "number", WorkspaceId: "ws1"},
			},
		}

		provider := NewPropertyImportProvider(
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
			"properties/properties.yaml",
		))

		spec, ok := entity.Content.(*specs.Spec)
		require.True(t, ok)

		assert.Equal(t, "properties", spec.Kind)
		assert.Equal(t, "properties", spec.Metadata["name"])
		assert.NotNil(t, spec.Metadata["import"])

		properties, ok := spec.Spec["properties"].([]map[string]any)
		require.True(t, ok)
		assert.Equal(t, 2, len(properties))
	})

	t.Run("creates v0 spec when v1 support disabled", func(t *testing.T) {
		mockResolver := &mockResolver{
			references: map[string]map[string]string{},
		}

		mockClient := &mockDataCatalog{
			properties: []*catalog.Property{
				{
					ID:          "prop1",
					Name:        "User Email",
					Type:        "string",
					WorkspaceId: "ws1",
					Config: map[string]interface{}{
						"minLength": float64(5),
					},
				},
			},
		}

		provider := &PropertyImportProvider{
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

		properties, ok := spec.Spec["properties"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, properties, 1)

		config, ok := properties[0]["propConfig"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, config, "minLength")
	})

	t.Run("creates v1 spec when v1 support enabled", func(t *testing.T) {
		mockResolver := &mockResolver{
			references: map[string]map[string]string{},
		}

		mockClient := &mockDataCatalog{
			properties: []*catalog.Property{
				{
					ID:          "prop1",
					Name:        "User Email",
					Type:        "string",
					WorkspaceId: "ws1",
					Config: map[string]interface{}{
						"minLength": float64(5),
					},
				},
			},
		}

		provider := &PropertyImportProvider{
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

		properties, ok := spec.Spec["properties"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, properties, 1)

		config, ok := properties[0]["config"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, config, "min_length")
	})
}
