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

type mockDataCatalog struct {
	catalog.DataCatalog
	properties []*catalog.Property
	err        error
}

func (m *mockDataCatalog) GetProperties(ctx context.Context) ([]*catalog.Property, error) {
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
				{ID: "prop2", Name: "Property 2", Type: "number", WorkspaceId: "ws1", ExternalId: "property-2"},
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

		properties := collection.GetAll(state.PropertyResourceType)
		assert.Equal(t, 2, len(properties))

		resourceIDs := make([]string, 0, len(properties))
		for _, prop := range properties {
			resourceIDs = append(resourceIDs, prop.ID)
		}

		assert.True(t, lo.Every(resourceIDs, []string{"prop1", "prop3"}))
		assert.False(t, lo.Contains(resourceIDs, "prop2"))
	})

	t.Run("correctly assigns externalId and reference after namer is loaded", func(t *testing.T) {
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

		properties := collection.GetAll(state.PropertyResourceType)
		require.Equal(t, 2, len(properties))

		prop1, ok := properties["prop1"]
		require.True(t, ok)
		assert.NotEmpty(t, prop1.ExternalID)
		assert.NotEmpty(t, prop1.Reference)

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

		provider := &PropertyImportProvider{
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

		assert.Equal(t, "properties", spec.Kind)
		assert.Equal(t, "properties", spec.Metadata["name"])
		assert.NotNil(t, spec.Metadata["import"])

		properties, ok := spec.Spec["properties"].([]map[string]any)
		require.True(t, ok)
		assert.Equal(t, 2, len(properties))
	})
}
