package resources

import (
	"errors"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRemoteResources(t *testing.T) {
	t.Parallel()

	collection := NewRemoteResources()
	require.NotNil(t, collection)
	assert.NotNil(t, collection.resources)
}

func TestRemoteResources_BasicOperations(t *testing.T) {
	t.Parallel()

	now := time.Now()

	testCases := []struct {
		name          string
		resourceType  string
		testData      map[string]*RemoteResource
		expectedLen   int
		testKey       string
		expectedID    string
		expectedExtID string
	}{
		{
			name:         "Events",
			resourceType: "events",
			testData: map[string]*RemoteResource{
				"event-1": {
					ID:         "event-1",
					ExternalID: "ext-event-1",
					Data: &catalog.Event{
						ID:          "event-1",
						Name:        "Test Event 1",
						EventType:   "track",
						WorkspaceId: "workspace-1",
						CreatedAt:   now,
						UpdatedAt:   now,
					},
				},
				"event-2": {
					ID:         "event-2",
					ExternalID: "ext-event-2",
					Data: &catalog.Event{
						ID:          "event-2",
						Name:        "Test Event 2",
						EventType:   "identify",
						WorkspaceId: "workspace-1",
						CreatedAt:   now,
						UpdatedAt:   now,
					},
				},
			},
			expectedLen:   2,
			testKey:       "event-1",
			expectedID:    "event-1",
			expectedExtID: "ext-event-1",
		},
		{
			name:         "Properties",
			resourceType: "properties",
			testData: map[string]*RemoteResource{
				"prop-1": {
					ID:         "prop-1",
					ExternalID: "ext-prop-1",
					Data: &catalog.Property{
						ID:          "prop-1",
						Name:        "Test Property 1",
						Type:        "string",
						WorkspaceId: "workspace-1",
						CreatedAt:   now,
						UpdatedAt:   now,
					},
				},
			},
			expectedLen:   1,
			testKey:       "prop-1",
			expectedID:    "prop-1",
			expectedExtID: "ext-prop-1",
		},
		{
			name:         "Categories",
			resourceType: "categories",
			testData: map[string]*RemoteResource{
				"cat-1": {
					ID:         "cat-1",
					ExternalID: "ext-cat-1",
					Data: &catalog.Category{
						ID:          "cat-1",
						Name:        "Test Category 1",
						WorkspaceID: "workspace-1",
						CreatedAt:   now,
						UpdatedAt:   now,
					},
				},
			},
			expectedLen:   1,
			testKey:       "cat-1",
			expectedID:    "cat-1",
			expectedExtID: "ext-cat-1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			collection := NewRemoteResources()

			// Test with no resources
			assert.Nil(t, collection.GetAll(tc.resourceType))

			// Test GetById with no resources
			resource, found := collection.GetByID(tc.resourceType, "nonexistent")
			assert.Nil(t, resource)
			assert.False(t, found)

			// Set test data
			collection.Set(tc.resourceType, tc.testData)

			// Test GetAll
			retrieved := collection.GetAll(tc.resourceType)
			require.NotNil(t, retrieved)
			assert.Len(t, retrieved, tc.expectedLen)

			// Test GetById - found
			foundResource, exists := collection.GetByID(tc.resourceType, tc.testKey)
			require.True(t, exists)
			require.NotNil(t, foundResource)
			assert.Equal(t, tc.expectedID, foundResource.ID)
			assert.Equal(t, tc.expectedExtID, foundResource.ExternalID)
			assert.NotNil(t, foundResource.Data)

			// Test GetById - not found
			notFound, notExists := collection.GetByID(tc.resourceType, "nonexistent")
			assert.Nil(t, notFound)
			assert.False(t, notExists)
		})
	}
}

func TestRemoteResources_EmptyMaps(t *testing.T) {
	t.Parallel()

	collection := NewRemoteResources()

	// Set empty maps
	collection.Set("events", make(map[string]*RemoteResource))
	collection.Set("properties", make(map[string]*RemoteResource))

	// Should return nil for empty resource maps
	events := collection.GetAll("events")
	properties := collection.GetAll("properties")

	assert.Nil(t, events)
	assert.Nil(t, properties)

	// GetById lookups should return not found
	event, eventFound := collection.GetByID("events", "any-id")
	prop, propFound := collection.GetByID("properties", "any-id")

	assert.Nil(t, event)
	assert.False(t, eventFound)
	assert.Nil(t, prop)
	assert.False(t, propFound)
}

func TestRemoteResources_NonExistentResourceType(t *testing.T) {
	t.Parallel()

	collection := NewRemoteResources()

	// Test GetAll with non-existent resource type
	resources := collection.GetAll("nonexistent")
	assert.Nil(t, resources)

	// Test GetById with non-existent resource type
	resource, found := collection.GetByID("nonexistent", "some-id")
	assert.Nil(t, resource)
	assert.False(t, found)
}

func TestRemoteResources_OverwriteResourceType(t *testing.T) {
	t.Parallel()

	collection := NewRemoteResources()

	// Set initial map
	initialMap := map[string]*RemoteResource{
		"test-1": {
			ID:         "test-1",
			ExternalID: "ext-test-1",
			Data:       "initial value",
		},
	}
	collection.Set("test-type", initialMap)

	// Verify initial value
	resource, found := collection.GetByID("test-type", "test-1")
	assert.True(t, found)
	assert.Equal(t, "initial value", resource.Data)
	assert.Equal(t, "test-1", resource.ID)

	// Overwrite with new map
	newMap := map[string]*RemoteResource{
		"test-2": {
			ID:         "test-2",
			ExternalID: "ext-test-2",
			Data:       "new value",
		},
	}
	collection.Set("test-type", newMap)

	// Verify old value is gone, new value exists
	oldResource, oldFound := collection.GetByID("test-type", "test-1")
	assert.Nil(t, oldResource)
	assert.False(t, oldFound)

	newResource, newFound := collection.GetByID("test-type", "test-2")
	assert.True(t, newFound)
	assert.Equal(t, "new value", newResource.Data)
	assert.Equal(t, "test-2", newResource.ID)

	// Verify GetAll returns only new resources
	allResources := collection.GetAll("test-type")
	assert.Len(t, allResources, 1)
	assert.Equal(t, "new value", allResources["test-2"].Data)
}

func TestRemoteResources_Merge(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		collection1    map[string]map[string]*RemoteResource
		collection2    map[string]map[string]*RemoteResource
		expectError    bool
		errorContains  string
		expectedResult map[string]int // resourceType -> expected count
	}{
		{
			name: "SuccessfulMerge",
			collection1: map[string]map[string]*RemoteResource{
				"events": {
					"event-1": {
						ID:         "event-1",
						ExternalID: "ext-event-1",
						Data:       "test-event",
					},
				},
			},
			collection2: map[string]map[string]*RemoteResource{
				"properties": {
					"prop-1": {
						ID:         "prop-1",
						ExternalID: "ext-prop-1",
						Data:       "test-prop",
					},
				},
			},
			expectError: false,
			expectedResult: map[string]int{
				"events":     1,
				"properties": 1,
			},
		},
		{
			name: "ResourceTypeOverlap",
			collection1: map[string]map[string]*RemoteResource{
				"events": {
					"event-1": {
						ID:         "event-1",
						ExternalID: "ext-event-1",
						Data:       "test-event-1",
					},
				},
			},
			collection2: map[string]map[string]*RemoteResource{
				"events": {
					"event-2": {
						ID:         "event-2",
						ExternalID: "ext-event-2",
						Data:       "test-event-2",
					},
				},
			},
			expectError:   true,
			errorContains: "at events",
		},
		{
			name: "MergeWithEmpty",
			collection1: map[string]map[string]*RemoteResource{
				"events": {
					"event-1": {
						ID:         "event-1",
						ExternalID: "ext-event-1",
						Data:       "test-event",
					},
				},
			},
			collection2: map[string]map[string]*RemoteResource{},
			expectError: false,
			expectedResult: map[string]int{
				"events": 1,
			},
		},
		{
			name:        "MergeEmptyIntoPopulated",
			collection1: map[string]map[string]*RemoteResource{},
			collection2: map[string]map[string]*RemoteResource{
				"properties": {
					"prop-1": {
						ID:         "prop-1",
						ExternalID: "ext-prop-1",
						Data:       "test-prop",
					},
				},
			},
			expectError: false,
			expectedResult: map[string]int{
				"properties": 1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Setup collections
			collection1 := NewRemoteResources()
			for resourceType, resources := range tc.collection1 {
				collection1.Set(resourceType, resources)
			}

			collection2 := NewRemoteResources()
			for resourceType, resources := range tc.collection2 {
				collection2.Set(resourceType, resources)
			}

			// Perform merge
			result, err := collection1.Merge(collection2)

			if tc.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.True(t, errors.Is(err, ErrDuplicateResource))
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				// Verify it's a different object reference (not same pointer)
				assert.NotSame(t, collection1, result) // Should be a new collection object

				// Verify expected results
				for resourceType, expectedCount := range tc.expectedResult {
					resources := result.GetAll(resourceType)
					require.NotNil(t, resources, "resourceType %s should exist", resourceType)
					assert.Len(t, resources, expectedCount, "resourceType %s should have %d resources", resourceType, expectedCount)

					// Verify each resource has proper RemoteResource structure
					for _, resource := range resources {
						assert.NotEmpty(t, resource.ID)
						assert.NotEmpty(t, resource.ExternalID)
						assert.NotNil(t, resource.Data)
					}
				}
			}
		})
	}

	// Test merge with nil (separate test since it returns the same collection)
	t.Run("MergeWithNil", func(t *testing.T) {
		t.Parallel()

		collection := NewRemoteResources()
		collection.Set("events", map[string]*RemoteResource{
			"event-1": {
				ID:         "event-1",
				ExternalID: "ext-event-1",
				Data:       "test",
			},
		})

		result, err := collection.Merge(nil)
		require.NoError(t, err)
		assert.Same(t, collection, result) // Should return same collection object when merging with nil
	})
}

func TestRemoteResources_Types(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice for empty collection", func(t *testing.T) {
		collection := NewRemoteResources()
		types := collection.Types()
		assert.Empty(t, types)
	})

	t.Run("returns single type", func(t *testing.T) {
		collection := NewRemoteResources()
		collection.Set("events", map[string]*RemoteResource{
			"event-1": {ID: "event-1", ExternalID: "ext-1", Data: "test"},
		})

		types := collection.Types()
		assert.ElementsMatch(t, []string{"events"}, types)
	})

	t.Run("returns multiple types", func(t *testing.T) {
		collection := NewRemoteResources()
		collection.Set("events", map[string]*RemoteResource{
			"event-1": {ID: "event-1", ExternalID: "ext-1", Data: "test"},
		})
		collection.Set("properties", map[string]*RemoteResource{
			"prop-1": {ID: "prop-1", ExternalID: "ext-2", Data: "test"},
		})
		collection.Set("categories", map[string]*RemoteResource{
			"cat-1": {ID: "cat-1", ExternalID: "ext-3", Data: "test"},
		})

		types := collection.Types()
		assert.ElementsMatch(t, []string{"events", "properties", "categories"}, types)
	})

	t.Run("returns types matching what was Set", func(t *testing.T) {
		collection := NewRemoteResources()
		expectedTypes := []string{"type-a", "type-b", "type-c", "type-d"}

		for _, typ := range expectedTypes {
			collection.Set(typ, map[string]*RemoteResource{
				"id-1": {ID: "id-1", ExternalID: "ext-1", Data: "test"},
			})
		}

		types := collection.Types()
		assert.ElementsMatch(t, expectedTypes, types)
	})
}

func TestRemoteResources_GetURNByID(t *testing.T) {
	t.Parallel()

	const (
		resourceType = "event"
		resourceID   = "resource-1"
		externalID   = "ext-resource-1"
	)

	t.Run("resource does not exist", func(t *testing.T) {
		collection := NewRemoteResources()

		urn, err := collection.GetURNByID(resourceType, "non-existent-id")
		assert.Empty(t, urn)
		assert.ErrorIs(t, err, ErrRemoteResourceNotFound)
	})

	t.Run("resource exists but externalID missing", func(t *testing.T) {
		collection := NewRemoteResources()
		resourceMap := map[string]*RemoteResource{
			resourceID: {
				ID:         resourceID,
				ExternalID: "",
				Data:       struct{}{},
			},
		}
		collection.Set(resourceType, resourceMap)

		urn, err := collection.GetURNByID(resourceType, resourceID)
		assert.Empty(t, urn)
		assert.ErrorIs(t, err, ErrRemoteResourceExternalIdNotFound)
	})

	t.Run("resource exists and has externalID", func(t *testing.T) {
		collection := NewRemoteResources()
		resourceMap := map[string]*RemoteResource{
			resourceID: {
				ID:         resourceID,
				ExternalID: externalID,
				Data:       struct{}{},
			},
		}
		collection.Set(resourceType, resourceMap)

		urn, err := collection.GetURNByID(resourceType, resourceID)
		require.NoError(t, err)
		assert.Equal(t, URN(externalID, resourceType), urn)
	})
}
