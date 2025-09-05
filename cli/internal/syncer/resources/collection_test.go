package resources

import (
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResourceCollection(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()
	require.NotNil(t, collection)
	// Test that collection is created successfully
	assert.NotNil(t, collection.resources)
}

func TestResourceCollection_Events(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Test with no events
	assert.Nil(t, collection.GetEvents())

	// Test GetEvent with no events
	event, found := collection.GetEvent("nonexistent")
	assert.Nil(t, event)
	assert.False(t, found)

	// Create test events
	now := time.Now()
	events := []*catalog.Event{
		{
			ID:          "event-1",
			Name:        "Test Event 1",
			Description: "First test event",
			EventType:   "track",
			WorkspaceId: "workspace-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "event-2",
			Name:        "Test Event 2",
			Description: "Second test event",
			EventType:   "identify",
			WorkspaceId: "workspace-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	// Set events
	collection.SetEvents(events)

	// Test GetEvents
	retrievedEvents := collection.GetEvents()
	require.NotNil(t, retrievedEvents)
	assert.Len(t, retrievedEvents, 2)
	assert.Equal(t, "Test Event 1", retrievedEvents[0].Name)
	assert.Equal(t, "Test Event 2", retrievedEvents[1].Name)

	// Test GetEvent by ID - found
	foundEvent, exists := collection.GetEvent("event-1")
	require.True(t, exists)
	require.NotNil(t, foundEvent)
	assert.Equal(t, "Test Event 1", foundEvent.Name)
	assert.Equal(t, "track", foundEvent.EventType)

	// Test GetEvent by ID - not found
	notFoundEvent, notExists := collection.GetEvent("nonexistent")
	assert.Nil(t, notFoundEvent)
	assert.False(t, notExists)
}

func TestResourceCollection_Properties(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Test with no properties
	assert.Nil(t, collection.GetProperties())

	// Test GetProperty with no properties
	prop, found := collection.GetProperty("nonexistent")
	assert.Nil(t, prop)
	assert.False(t, found)

	// Create test properties
	now := time.Now()
	properties := []*catalog.Property{
		{
			ID:          "prop-1",
			Name:        "Test Property 1",
			Description: "First test property",
			Type:        "string",
			WorkspaceId: "workspace-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "prop-2",
			Name:        "Test Property 2",
			Description: "Second test property",
			Type:        "integer",
			WorkspaceId: "workspace-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	// Set properties
	collection.SetProperties(properties)

	// Test GetProperties
	retrievedProps := collection.GetProperties()
	require.NotNil(t, retrievedProps)
	assert.Len(t, retrievedProps, 2)

	// Test GetProperty by ID - found
	foundProp, exists := collection.GetProperty("prop-1")
	require.True(t, exists)
	require.NotNil(t, foundProp)
	assert.Equal(t, "Test Property 1", foundProp.Name)
	assert.Equal(t, "string", foundProp.Type)

	// Test GetProperty by ID - not found
	notFoundProp, notExists := collection.GetProperty("nonexistent")
	assert.Nil(t, notFoundProp)
	assert.False(t, notExists)
}

func TestResourceCollection_Categories(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Test with no categories
	assert.Nil(t, collection.GetCategories())

	// Create test categories
	now := time.Now()
	categories := []*catalog.Category{
		{
			ID:          "cat-1",
			Name:        "Test Category 1",
			WorkspaceID: "workspace-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cat-2",
			Name:        "Test Category 2",
			WorkspaceID: "workspace-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	// Set categories
	collection.SetCategories(categories)

	// Test GetCategories
	retrievedCats := collection.GetCategories()
	require.NotNil(t, retrievedCats)
	assert.Len(t, retrievedCats, 2)

	// Test GetCategory by ID - found
	foundCat, exists := collection.GetCategory("cat-1")
	require.True(t, exists)
	require.NotNil(t, foundCat)
	assert.Equal(t, "Test Category 1", foundCat.Name)

	// Test GetCategory by ID - not found
	notFoundCat, notExists := collection.GetCategory("nonexistent")
	assert.Nil(t, notFoundCat)
	assert.False(t, notExists)
}

func TestResourceCollection_CustomTypes(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Test with no custom types
	assert.Nil(t, collection.GetCustomTypes())

	// Create test custom types
	now := time.Now()
	customTypes := []*catalog.CustomType{
		{
			ID:          "ct-1",
			Name:        "Test Custom Type 1",
			WorkspaceId: "workspace-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	// Set custom types
	collection.SetCustomTypes(customTypes)

	// Test GetCustomTypes
	retrievedTypes := collection.GetCustomTypes()
	require.NotNil(t, retrievedTypes)
	assert.Len(t, retrievedTypes, 1)

	// Test GetCustomType by ID - found
	foundType, exists := collection.GetCustomType("ct-1")
	require.True(t, exists)
	require.NotNil(t, foundType)
	assert.Equal(t, "Test Custom Type 1", foundType.Name)

	// Test GetCustomType by ID - not found
	notFoundType, notExists := collection.GetCustomType("nonexistent")
	assert.Nil(t, notFoundType)
	assert.False(t, notExists)
}

func TestResourceCollection_TrackingPlans(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Test with no tracking plans
	assert.Nil(t, collection.GetTrackingPlans())

	// Create test tracking plans
	now := time.Now()
	trackingPlans := []*catalog.TrackingPlan{
		{
			ID:          "tp-1",
			Name:        "Test Tracking Plan 1",
			WorkspaceID: "workspace-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	// Set tracking plans
	collection.SetTrackingPlans(trackingPlans)

	// Test GetTrackingPlans
	retrievedTPs := collection.GetTrackingPlans()
	require.NotNil(t, retrievedTPs)
	assert.Len(t, retrievedTPs, 1)

	// Test GetTrackingPlan by ID - found
	foundTP, exists := collection.GetTrackingPlan("tp-1")
	require.True(t, exists)
	require.NotNil(t, foundTP)
	assert.Equal(t, "Test Tracking Plan 1", foundTP.Name)

	// Test GetTrackingPlan by ID - not found
	notFoundTP, notExists := collection.GetTrackingPlan("nonexistent")
	assert.Nil(t, notFoundTP)
	assert.False(t, notExists)
}

func TestResourceCollection_TypeCastingEdgeCases(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Manually set wrong type in the map and test casting
	collection.resources["events"] = make(map[string]interface{})
	collection.resources["events"]["wrong-type"] = "not an event object"

	// Should return empty slice when casting fails
	events := collection.GetEvents()
	assert.Nil(t, events) // No valid events to return

	// ID lookup should return not found for wrong type
	event, found := collection.GetEvent("wrong-type")
	assert.Nil(t, event)
	assert.False(t, found)

	// Test with valid event mixed with invalid type
	now := time.Now()
	validEvent := &catalog.Event{ID: "valid-event", Name: "Valid Event", CreatedAt: now, UpdatedAt: now}
	collection.resources["events"]["valid-event"] = validEvent

	// Should return only valid events
	events = collection.GetEvents()
	require.NotNil(t, events)
	assert.Len(t, events, 1)
	assert.Equal(t, "Valid Event", events[0].Name)

	// Valid event should be found
	foundEvent, found := collection.GetEvent("valid-event")
	assert.True(t, found)
	assert.Equal(t, "Valid Event", foundEvent.Name)
}

func TestResourceCollection_EmptySlices(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Set empty slices
	collection.SetEvents([]*catalog.Event{})
	collection.SetProperties([]*catalog.Property{})

	// Should return nil for empty resource maps
	events := collection.GetEvents()
	properties := collection.GetProperties()

	assert.Nil(t, events)
	assert.Nil(t, properties)

	// ID lookups should return not found
	event, eventFound := collection.GetEvent("any-id")
	prop, propFound := collection.GetProperty("any-id")

	assert.Nil(t, event)
	assert.False(t, eventFound)
	assert.Nil(t, prop)
	assert.False(t, propFound)
}
