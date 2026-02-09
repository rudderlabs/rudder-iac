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

type mockEventCatalog struct {
	catalog.DataCatalog
	events []*catalog.Event
	err    error
}

func (m *mockEventCatalog) GetEvents(ctx context.Context, options catalog.ListOptions) ([]*catalog.Event, error) {
	return m.events, m.err
}

func TestEventLoadImportable(t *testing.T) {
	t.Run("filters events with ExternalId set", func(t *testing.T) {
		mockClient := &mockEventCatalog{
			events: []*catalog.Event{
				{ID: "evt1", Name: "Page Viewed", EventType: "track", WorkspaceId: "ws1"},
				{ID: "evt2", Name: "Button Clicked", EventType: "track", WorkspaceId: "ws1", ExternalID: "button_clicked"},
				{ID: "evt3", Name: "Product Purchased", EventType: "track", WorkspaceId: "ws1"},
			},
		}

		provider := &EventImportProvider{
			client:   mockClient,
			log:      *logger.New("test"),
			filepath: "data-catalog",
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		events := collection.GetAll(types.EventResourceType)
		assert.Equal(t, 2, len(events))

		resourceIDs := make([]string, 0, len(events))
		for _, evt := range events {
			resourceIDs = append(resourceIDs, evt.ID)
		}

		assert.True(t, lo.Every(resourceIDs, []string{"evt1", "evt3"}))
		assert.False(t, lo.Contains(resourceIDs, "evt2"))
	})

	t.Run("correctly assigns externalId and reference", func(t *testing.T) {
		mockClient := &mockEventCatalog{
			events: []*catalog.Event{
				{ID: "evt1", Name: "Page Viewed", EventType: "track", WorkspaceId: "ws1"},
				{ID: "evt2", Name: "Product Purchased", EventType: "track", WorkspaceId: "ws1"},
			},
		}

		provider := &EventImportProvider{
			client:   mockClient,
			log:      *logger.New("test"),
			filepath: "data-catalog",
		}

		externalIdNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
		collection, err := provider.LoadImportable(context.Background(), externalIdNamer)
		require.Nil(t, err)

		events := collection.GetAll(types.EventResourceType)
		require.Equal(t, 2, len(events))

		evt1, ok := events["evt1"]
		require.True(t, ok)
		assert.NotEmpty(t, evt1.ExternalID)
		assert.NotEmpty(t, evt1.Reference)

		evt2, ok := events["evt2"]
		require.True(t, ok)
		assert.NotEmpty(t, evt2.ExternalID)
		assert.NotEmpty(t, evt2.Reference)
	})
}

func TestEventFormatForExport(t *testing.T) {
	t.Run("generates spec with correct structure", func(t *testing.T) {
		mockResolver := &mockResolver{
			references: map[string]map[string]string{},
		}

		mockClient := &mockEventCatalog{
			events: []*catalog.Event{
				{ID: "evt1", Name: "Page Viewed", EventType: "track", WorkspaceId: "ws1"},
				{ID: "evt2", Name: "Product Purchased", EventType: "track", WorkspaceId: "ws1"},
			},
		}

		provider := &EventImportProvider{
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

		assert.Equal(t, "events", spec.Kind)
		assert.Equal(t, "events", spec.Metadata["name"])
		assert.NotNil(t, spec.Metadata["import"])

		events, ok := spec.Spec["events"].([]map[string]any)
		require.True(t, ok)
		assert.Equal(t, 2, len(events))
	})

	t.Run("creates v0 spec when v1 support disabled", func(t *testing.T) {
		categoryID := "cat1"
		mockResolver := &mockResolver{
			references: map[string]map[string]string{
				types.CategoryResourceType: {
					"cat1": "#/categories/default/ecommerce_category",
				},
			},
		}

		mockClient := &mockEventCatalog{
			events: []*catalog.Event{
				{
					ID:          "evt1",
					Name:        "Product Purchased",
					EventType:   "track",
					WorkspaceId: "ws1",
					CategoryId:  &categoryID,
				},
			},
		}

		provider := &EventImportProvider{
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

		events, ok := spec.Spec["events"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, events, 1)
		require.NotNil(t, events[0]["category"])
		assert.Contains(t, *events[0]["category"].(*string), "#/categories/")
	})

	t.Run("creates v1 spec when v1 support enabled", func(t *testing.T) {
		categoryID := "cat1"
		mockResolver := &mockResolver{
			references: map[string]map[string]string{
				types.CategoryResourceType: {
					"cat1": "#category:ecommerce_category",
				},
			},
		}

		mockClient := &mockEventCatalog{
			events: []*catalog.Event{
				{
					ID:          "evt1",
					Name:        "Product Purchased",
					EventType:   "track",
					WorkspaceId: "ws1",
					CategoryId:  &categoryID,
				},
			},
		}

		provider := &EventImportProvider{
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

		events, ok := spec.Spec["events"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, events, 1)
		require.NotNil(t, events[0]["category"])
		assert.Contains(t, *events[0]["category"].(*string), "#category:")
	})
}
