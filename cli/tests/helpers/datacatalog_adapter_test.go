package helpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDataCatalog implements catalog.DataCatalog for testing
type mockDataCatalog struct {
	getCategoriesFunc                   func(ctx context.Context, options catalog.ListOptions) ([]*catalog.Category, error)
	getEventsFunc                       func(ctx context.Context, options catalog.ListOptions) ([]*catalog.Event, error)
	getPropertiesFunc                   func(ctx context.Context, options catalog.ListOptions) ([]*catalog.Property, error)
	getCustomTypesFunc                  func(ctx context.Context, options catalog.ListOptions) ([]*catalog.CustomType, error)
	getTrackingPlansFunc                func(ctx context.Context, options catalog.ListOptions) ([]*catalog.TrackingPlan, error)
	getCategoryFunc                     func(ctx context.Context, id string) (*catalog.Category, error)
	getEventFunc                        func(ctx context.Context, id string) (*catalog.Event, error)
	getPropertyFunc                     func(ctx context.Context, id string) (*catalog.Property, error)
	getCustomTypeFunc                   func(ctx context.Context, id string) (*catalog.CustomType, error)
	getTrackingPlanWithIdentifiersFunc  func(ctx context.Context, id string, rebuildSchemas bool) (*catalog.TrackingPlanWithIdentifiers, error)
}

// CategoryStore methods
func (m *mockDataCatalog) CreateCategory(ctx context.Context, input catalog.CategoryCreate) (*catalog.Category, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) UpdateCategory(ctx context.Context, id string, input catalog.CategoryUpdate) (*catalog.Category, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) DeleteCategory(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) GetCategory(ctx context.Context, id string) (*catalog.Category, error) {
	if m.getCategoryFunc != nil {
		return m.getCategoryFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) GetCategories(ctx context.Context, options catalog.ListOptions) ([]*catalog.Category, error) {
	if m.getCategoriesFunc != nil {
		return m.getCategoriesFunc(ctx, options)
	}
	return []*catalog.Category{}, nil
}

func (m *mockDataCatalog) SetCategoryExternalId(ctx context.Context, id string, externalId string) error {
	return fmt.Errorf("not implemented")
}

// EventStore methods
func (m *mockDataCatalog) CreateEvent(ctx context.Context, input catalog.EventCreate) (*catalog.Event, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) UpdateEvent(ctx context.Context, id string, input *catalog.EventUpdate) (*catalog.Event, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) DeleteEvent(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) GetEvent(ctx context.Context, id string) (*catalog.Event, error) {
	if m.getEventFunc != nil {
		return m.getEventFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) GetEvents(ctx context.Context, options catalog.ListOptions) ([]*catalog.Event, error) {
	if m.getEventsFunc != nil {
		return m.getEventsFunc(ctx, options)
	}
	return []*catalog.Event{}, nil
}

func (m *mockDataCatalog) SetEventExternalId(ctx context.Context, id string, externalId string) error {
	return fmt.Errorf("not implemented")
}

// PropertyStore methods
func (m *mockDataCatalog) CreateProperty(ctx context.Context, input catalog.PropertyCreate) (*catalog.Property, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) UpdateProperty(ctx context.Context, id string, input *catalog.PropertyUpdate) (*catalog.Property, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) DeleteProperty(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) GetProperty(ctx context.Context, id string) (*catalog.Property, error) {
	if m.getPropertyFunc != nil {
		return m.getPropertyFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) GetProperties(ctx context.Context, options catalog.ListOptions) ([]*catalog.Property, error) {
	if m.getPropertiesFunc != nil {
		return m.getPropertiesFunc(ctx, options)
	}
	return []*catalog.Property{}, nil
}

func (m *mockDataCatalog) SetPropertyExternalId(ctx context.Context, id string, externalId string) error {
	return fmt.Errorf("not implemented")
}

// CustomTypeStore methods
func (m *mockDataCatalog) CreateCustomType(ctx context.Context, input catalog.CustomTypeCreate) (*catalog.CustomType, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) UpdateCustomType(ctx context.Context, id string, input *catalog.CustomTypeUpdate) (*catalog.CustomType, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) DeleteCustomType(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) GetCustomType(ctx context.Context, id string) (*catalog.CustomType, error) {
	if m.getCustomTypeFunc != nil {
		return m.getCustomTypeFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) GetCustomTypes(ctx context.Context, options catalog.ListOptions) ([]*catalog.CustomType, error) {
	if m.getCustomTypesFunc != nil {
		return m.getCustomTypesFunc(ctx, options)
	}
	return []*catalog.CustomType{}, nil
}

func (m *mockDataCatalog) SetCustomTypeExternalId(ctx context.Context, id string, externalId string) error {
	return fmt.Errorf("not implemented")
}

// TrackingPlanStore methods
func (m *mockDataCatalog) CreateTrackingPlan(ctx context.Context, input catalog.TrackingPlanCreate) (*catalog.TrackingPlan, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) UpsertTrackingPlan(ctx context.Context, id string, input catalog.TrackingPlanUpsertEvent) (*catalog.TrackingPlan, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) UpdateTrackingPlan(ctx context.Context, id string, name, description string) (*catalog.TrackingPlan, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) DeleteTrackingPlan(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) DeleteTrackingPlanEvent(ctx context.Context, trackingPlanId string, eventId string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) GetTrackingPlanWithSchemas(ctx context.Context, id string) (*catalog.TrackingPlanWithSchemas, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) GetTrackingPlan(ctx context.Context, id string) (*catalog.TrackingPlan, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) GetTrackingPlans(ctx context.Context, options catalog.ListOptions) ([]*catalog.TrackingPlan, error) {
	if m.getTrackingPlansFunc != nil {
		return m.getTrackingPlansFunc(ctx, options)
	}
	return []*catalog.TrackingPlan{}, nil
}

func (m *mockDataCatalog) GetTrackingPlanWithIdentifiers(ctx context.Context, id string, rebuildSchemas bool) (*catalog.TrackingPlanWithIdentifiers, error) {
	if m.getTrackingPlanWithIdentifiersFunc != nil {
		return m.getTrackingPlanWithIdentifiersFunc(ctx, id, rebuildSchemas)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) GetTrackingPlansWithIdentifiers(ctx context.Context, options catalog.ListOptions) ([]*catalog.TrackingPlanWithIdentifiers, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) GetTrackingPlanEventSchema(ctx context.Context, id string, eventId string) (*catalog.TrackingPlanEventSchema, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) UpdateTrackingPlanEvents(ctx context.Context, id string, input []catalog.EventIdentifierDetail, rebuildSchemas bool) error {
	return fmt.Errorf("not implemented")
}

func (m *mockDataCatalog) SetTrackingPlanExternalId(ctx context.Context, id string, externalId string) error {
	return fmt.Errorf("not implemented")
}

func TestDataCatalogAdapter_RemoteIDs(t *testing.T) {
	t.Parallel()

	t.Run("returns all resources with external IDs", func(t *testing.T) {
		t.Parallel()

		client := &mockDataCatalog{
			getCategoriesFunc: func(ctx context.Context, options catalog.ListOptions) ([]*catalog.Category, error) {
				// API filters by HasExternalID, so only return items with external IDs
				return []*catalog.Category{
					{ID: "cat-1", ExternalID: "ext-cat-1", Name: "Category 1"},
				}, nil
			},
			getEventsFunc: func(ctx context.Context, options catalog.ListOptions) ([]*catalog.Event, error) {
				return []*catalog.Event{
					{ID: "evt-1", ExternalID: "ext-evt-1", Name: "Event 1"},
				}, nil
			},
			getPropertiesFunc: func(ctx context.Context, options catalog.ListOptions) ([]*catalog.Property, error) {
				return []*catalog.Property{
					{ID: "prop-1", ExternalID: "ext-prop-1", Name: "Property 1"},
				}, nil
			},
			getCustomTypesFunc: func(ctx context.Context, options catalog.ListOptions) ([]*catalog.CustomType, error) {
				return []*catalog.CustomType{
					{ID: "ct-1", ExternalID: "ext-ct-1", Name: "CustomType 1"},
				}, nil
			},
			getTrackingPlansFunc: func(ctx context.Context, options catalog.ListOptions) ([]*catalog.TrackingPlan, error) {
				return []*catalog.TrackingPlan{
					{ID: "tp-1", ExternalID: "ext-tp-1", Name: "Tracking Plan 1"},
				}, nil
			},
		}

		adapter := NewDataCatalogAdapter(client)
		ids, err := adapter.RemoteIDs(context.Background())

		require.NoError(t, err)
		assert.Len(t, ids, 5)

		assert.Equal(t, "cat-1", ids["category:ext-cat-1"])
		assert.Equal(t, "evt-1", ids["event:ext-evt-1"])
		assert.Equal(t, "prop-1", ids["property:ext-prop-1"])
		assert.Equal(t, "ct-1", ids["custom-type:ext-ct-1"])
		assert.Equal(t, "tp-1", ids["tracking-plan:ext-tp-1"])
	})

	t.Run("returns empty map when no resources exist", func(t *testing.T) {
		t.Parallel()

		client := &mockDataCatalog{}
		adapter := NewDataCatalogAdapter(client)
		ids, err := adapter.RemoteIDs(context.Background())

		require.NoError(t, err)
		assert.Len(t, ids, 0)
	})

	t.Run("returns error when listing categories fails", func(t *testing.T) {
		t.Parallel()

		client := &mockDataCatalog{
			getCategoriesFunc: func(ctx context.Context, options catalog.ListOptions) ([]*catalog.Category, error) {
				return nil, fmt.Errorf("API error")
			},
		}

		adapter := NewDataCatalogAdapter(client)
		ids, err := adapter.RemoteIDs(context.Background())

		require.Error(t, err)
		assert.Nil(t, ids)
	})

	t.Run("returns error when listing events fails", func(t *testing.T) {
		t.Parallel()

		client := &mockDataCatalog{
			getCategoriesFunc: func(ctx context.Context, options catalog.ListOptions) ([]*catalog.Category, error) {
				return []*catalog.Category{}, nil
			},
			getEventsFunc: func(ctx context.Context, options catalog.ListOptions) ([]*catalog.Event, error) {
				return nil, fmt.Errorf("API error")
			},
		}

		adapter := NewDataCatalogAdapter(client)
		ids, err := adapter.RemoteIDs(context.Background())

		require.Error(t, err)
		assert.Nil(t, ids)
	})
}

func TestDataCatalogAdapter_FetchResource(t *testing.T) {
	t.Parallel()

	t.Run("fetches event by ID", func(t *testing.T) {
		t.Parallel()

		client := &mockDataCatalog{
			getEventFunc: func(ctx context.Context, id string) (*catalog.Event, error) {
				return &catalog.Event{
					ID:          id,
					Name:        "Test Event",
					Description: "Test Description",
					ExternalID:  "ext-123",
				}, nil
			},
		}

		adapter := NewDataCatalogAdapter(client)
		result, err := adapter.FetchResource(context.Background(), state.EventResourceType, "evt-123")

		require.NoError(t, err)
		event := result.(*catalog.Event)
		assert.Equal(t, "evt-123", event.ID)
		assert.Equal(t, "Test Event", event.Name)
	})

	t.Run("fetches category by ID", func(t *testing.T) {
		t.Parallel()

		client := &mockDataCatalog{
			getCategoryFunc: func(ctx context.Context, id string) (*catalog.Category, error) {
				return &catalog.Category{
					ID:         id,
					Name:       "Test Category",
					ExternalID: "ext-cat-123",
				}, nil
			},
		}

		adapter := NewDataCatalogAdapter(client)
		result, err := adapter.FetchResource(context.Background(), state.CategoryResourceType, "cat-123")

		require.NoError(t, err)
		category := result.(*catalog.Category)
		assert.Equal(t, "cat-123", category.ID)
		assert.Equal(t, "Test Category", category.Name)
	})

	t.Run("fetches property by ID", func(t *testing.T) {
		t.Parallel()

		client := &mockDataCatalog{
			getPropertyFunc: func(ctx context.Context, id string) (*catalog.Property, error) {
				return &catalog.Property{
					ID:         id,
					Name:       "Test Property",
					ExternalID: "ext-prop-123",
				}, nil
			},
		}

		adapter := NewDataCatalogAdapter(client)
		result, err := adapter.FetchResource(context.Background(), state.PropertyResourceType, "prop-123")

		require.NoError(t, err)
		property := result.(*catalog.Property)
		assert.Equal(t, "prop-123", property.ID)
		assert.Equal(t, "Test Property", property.Name)
	})

	t.Run("fetches custom type by ID", func(t *testing.T) {
		t.Parallel()

		client := &mockDataCatalog{
			getCustomTypeFunc: func(ctx context.Context, id string) (*catalog.CustomType, error) {
				return &catalog.CustomType{
					ID:         id,
					Name:       "Test CustomType",
					ExternalID: "ext-ct-123",
				}, nil
			},
		}

		adapter := NewDataCatalogAdapter(client)
		result, err := adapter.FetchResource(context.Background(), state.CustomTypeResourceType, "ct-123")

		require.NoError(t, err)
		customType := result.(*catalog.CustomType)
		assert.Equal(t, "ct-123", customType.ID)
		assert.Equal(t, "Test CustomType", customType.Name)
	})

	t.Run("fetches tracking plan by ID", func(t *testing.T) {
		t.Parallel()

		client := &mockDataCatalog{
			getTrackingPlanWithIdentifiersFunc: func(ctx context.Context, id string, rebuildSchemas bool) (*catalog.TrackingPlanWithIdentifiers, error) {
				return &catalog.TrackingPlanWithIdentifiers{
					TrackingPlan: catalog.TrackingPlan{
						ID:         id,
						Name:       "Test Tracking Plan",
						ExternalID: "ext-tp-123",
					},
				}, nil
			},
		}

		adapter := NewDataCatalogAdapter(client)
		result, err := adapter.FetchResource(context.Background(), state.TrackingPlanResourceType, "tp-123")

		require.NoError(t, err)
		tp := result.(*catalog.TrackingPlanWithIdentifiers)
		assert.Equal(t, "tp-123", tp.ID)
		assert.Equal(t, "Test Tracking Plan", tp.Name)
	})

	t.Run("returns error for unsupported resource type", func(t *testing.T) {
		t.Parallel()

		client := &mockDataCatalog{}
		adapter := NewDataCatalogAdapter(client)

		_, err := adapter.FetchResource(context.Background(), "unknown-type", "id-123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported resource type")
	})
}
