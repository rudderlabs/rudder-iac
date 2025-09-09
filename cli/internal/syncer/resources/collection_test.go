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
	assert.NotNil(t, collection.resources)
}

func TestResourceCollection_Generic_Events(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Test with no events
	assert.Nil(t, collection.GetAll("events"))

	// Test GetById with no events
	event, found := collection.GetById("events", "nonexistent")
	assert.Nil(t, event)
	assert.False(t, found)

	// Create test events map
	now := time.Now()
	eventsMap := map[string]interface{}{
		"event-1": &catalog.Event{
			ID:          "event-1",
			Name:        "Test Event 1",
			Description: "First test event",
			EventType:   "track",
			WorkspaceId: "workspace-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		"event-2": &catalog.Event{
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
	collection.Set("events", eventsMap)

	// Test GetAll
	retrievedEvents := collection.GetAll("events")
	require.NotNil(t, retrievedEvents)
	assert.Len(t, retrievedEvents, 2)

	// Verify events are correct (need to convert and check)
	eventFound1 := false
	eventFound2 := false
	for _, eventInterface := range retrievedEvents {
		event := eventInterface.(*catalog.Event)
		if event.Name == "Test Event 1" {
			eventFound1 = true
			assert.Equal(t, "track", event.EventType)
		}
		if event.Name == "Test Event 2" {
			eventFound2 = true
			assert.Equal(t, "identify", event.EventType)
		}
	}
	assert.True(t, eventFound1, "Test Event 1 should be found")
	assert.True(t, eventFound2, "Test Event 2 should be found")

	// Test GetById - found
	foundEvent, exists := collection.GetById("events", "event-1")
	require.True(t, exists)
	require.NotNil(t, foundEvent)
	event1 := foundEvent.(*catalog.Event)
	assert.Equal(t, "Test Event 1", event1.Name)
	assert.Equal(t, "track", event1.EventType)

	// Test GetById - not found
	notFoundEvent, notExists := collection.GetById("events", "nonexistent")
	assert.Nil(t, notFoundEvent)
	assert.False(t, notExists)
}

func TestResourceCollection_Generic_Properties(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Test with no properties
	assert.Nil(t, collection.GetAll("properties"))

	// Test GetById with no properties
	prop, found := collection.GetById("properties", "nonexistent")
	assert.Nil(t, prop)
	assert.False(t, found)

	// Create test properties map
	now := time.Now()
	propertiesMap := map[string]interface{}{
		"prop-1": &catalog.Property{
			ID:          "prop-1",
			Name:        "Test Property 1",
			Description: "First test property",
			Type:        "string",
			WorkspaceId: "workspace-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		"prop-2": &catalog.Property{
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
	collection.Set("properties", propertiesMap)

	// Test GetAll
	retrievedProps := collection.GetAll("properties")
	require.NotNil(t, retrievedProps)
	assert.Len(t, retrievedProps, 2)

	// Test GetById - found
	foundProp, exists := collection.GetById("properties", "prop-1")
	require.True(t, exists)
	require.NotNil(t, foundProp)
	prop1 := foundProp.(*catalog.Property)
	assert.Equal(t, "Test Property 1", prop1.Name)
	assert.Equal(t, "string", prop1.Type)

	// Test GetById - not found
	notFoundProp, notExists := collection.GetById("properties", "nonexistent")
	assert.Nil(t, notFoundProp)
	assert.False(t, notExists)
}

func TestResourceCollection_Generic_Categories(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Test with no categories
	assert.Nil(t, collection.GetAll("categories"))

	// Create test categories map
	now := time.Now()
	categoriesMap := map[string]interface{}{
		"cat-1": &catalog.Category{
			ID:          "cat-1",
			Name:        "Test Category 1",
			WorkspaceID: "workspace-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		"cat-2": &catalog.Category{
			ID:          "cat-2",
			Name:        "Test Category 2",
			WorkspaceID: "workspace-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	// Set categories
	collection.Set("categories", categoriesMap)

	// Test GetAll
	retrievedCats := collection.GetAll("categories")
	require.NotNil(t, retrievedCats)
	assert.Len(t, retrievedCats, 2)

	// Test GetById - found
	foundCat, exists := collection.GetById("categories", "cat-1")
	require.True(t, exists)
	require.NotNil(t, foundCat)
	cat1 := foundCat.(*catalog.Category)
	assert.Equal(t, "Test Category 1", cat1.Name)

	// Test GetById - not found
	notFoundCat, notExists := collection.GetById("categories", "nonexistent")
	assert.Nil(t, notFoundCat)
	assert.False(t, notExists)
}

func TestResourceCollection_Generic_CustomTypes(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Test with no custom types
	assert.Nil(t, collection.GetAll("customTypes"))

	// Create test custom types map
	now := time.Now()
	customTypesMap := map[string]interface{}{
		"ct-1": &catalog.CustomType{
			ID:          "ct-1",
			Name:        "Test Custom Type 1",
			WorkspaceId: "workspace-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	// Set custom types
	collection.Set("customTypes", customTypesMap)

	// Test GetAll
	retrievedTypes := collection.GetAll("customTypes")
	require.NotNil(t, retrievedTypes)
	assert.Len(t, retrievedTypes, 1)

	// Test GetById - found
	foundType, exists := collection.GetById("customTypes", "ct-1")
	require.True(t, exists)
	require.NotNil(t, foundType)
	ct1 := foundType.(*catalog.CustomType)
	assert.Equal(t, "Test Custom Type 1", ct1.Name)

	// Test GetById - not found
	notFoundType, notExists := collection.GetById("customTypes", "nonexistent")
	assert.Nil(t, notFoundType)
	assert.False(t, notExists)
}

func TestResourceCollection_Generic_TrackingPlans(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Test with no tracking plans
	assert.Nil(t, collection.GetAll("trackingPlans"))

	// Create test tracking plans map
	now := time.Now()
	trackingPlansMap := map[string]interface{}{
		"tp-1": &catalog.TrackingPlan{
			ID:          "tp-1",
			Name:        "Test Tracking Plan 1",
			WorkspaceID: "workspace-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	// Set tracking plans
	collection.Set("trackingPlans", trackingPlansMap)

	// Test GetAll
	retrievedTPs := collection.GetAll("trackingPlans")
	require.NotNil(t, retrievedTPs)
	assert.Len(t, retrievedTPs, 1)

	// Test GetById - found
	foundTP, exists := collection.GetById("trackingPlans", "tp-1")
	require.True(t, exists)
	require.NotNil(t, foundTP)
	tp1 := foundTP.(*catalog.TrackingPlan)
	assert.Equal(t, "Test Tracking Plan 1", tp1.Name)

	// Test GetById - not found
	notFoundTP, notExists := collection.GetById("trackingPlans", "nonexistent")
	assert.Nil(t, notFoundTP)
	assert.False(t, notExists)
}

func TestResourceCollection_EmptyMaps(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Set empty maps
	collection.Set("events", make(map[string]interface{}))
	collection.Set("properties", make(map[string]interface{}))

	// Should return nil for empty resource maps
	events := collection.GetAll("events")
	properties := collection.GetAll("properties")

	assert.Nil(t, events)
	assert.Nil(t, properties)

	// GetById lookups should return not found
	event, eventFound := collection.GetById("events", "any-id")
	prop, propFound := collection.GetById("properties", "any-id")

	assert.Nil(t, event)
	assert.False(t, eventFound)
	assert.Nil(t, prop)
	assert.False(t, propFound)
}

func TestResourceCollection_NonExistentResourceType(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Test GetAll with non-existent resource type
	resources := collection.GetAll("nonexistent")
	assert.Nil(t, resources)

	// Test GetById with non-existent resource type
	resource, found := collection.GetById("nonexistent", "some-id")
	assert.Nil(t, resource)
	assert.False(t, found)
}

func TestResourceCollection_OverwriteResourceType(t *testing.T) {
	t.Parallel()

	collection := NewResourceCollection()

	// Set initial map
	initialMap := map[string]interface{}{
		"test-1": "initial value",
	}
	collection.Set("test-type", initialMap)

	// Verify initial value
	resource, found := collection.GetById("test-type", "test-1")
	assert.True(t, found)
	assert.Equal(t, "initial value", resource)

	// Overwrite with new map
	newMap := map[string]interface{}{
		"test-2": "new value",
	}
	collection.Set("test-type", newMap)

	// Verify old value is gone, new value exists
	oldResource, oldFound := collection.GetById("test-type", "test-1")
	assert.Nil(t, oldResource)
	assert.False(t, oldFound)

	newResource, newFound := collection.GetById("test-type", "test-2")
	assert.True(t, newFound)
	assert.Equal(t, "new value", newResource)

	// Verify GetAll returns only new resources
	allResources := collection.GetAll("test-type")
	assert.Len(t, allResources, 1)
	assert.Equal(t, "new value", allResources["test-2"])
}
