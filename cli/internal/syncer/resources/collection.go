package resources

import (
	"errors"
	"fmt"
)

// RemoteResource represents a resource that is fetched from the remote catalog
// It contains the ID, ExternalID, and Data of the resource
// ResourceCollection is type-agnostic and uses ID/ExternalID for framework operations like building the URN given the remoteID
type RemoteResource struct {
	ID         string
	ExternalID string
	Reference  string
	Data       interface{}
}

// ResourceCollection provides a generic container for mixed resource types
// with generic getters and efficient ID-based lookups
type ResourceCollection struct {
	resources map[string]map[string]*RemoteResource
}

var (
	ErrDuplicateResource                = errors.New("duplicate resource detected")
	ErrRemoteResourceNotFound           = errors.New("remote resource not found")
	ErrRemoteResourceExternalIdNotFound = errors.New("remote resource does not have externalId")
)

// NewResourceCollection creates a new empty ResourceCollection
func NewResourceCollection() *ResourceCollection {
	return &ResourceCollection{
		resources: make(map[string]map[string]*RemoteResource),
	}
}

func (rc *ResourceCollection) Len() int {
	count := 0
	for _, resources := range rc.resources {
		count += len(resources)
	}
	return count
}

// Set stores a resource map for the given resource type
func (rc *ResourceCollection) Set(resourceType string, resourceMap map[string]*RemoteResource) {
	rc.resources[resourceType] = resourceMap
}

// GetAll returns all resources of the given type as a slice
func (rc *ResourceCollection) GetAll(resourceType string) map[string]*RemoteResource {
	resourceMap := rc.resources[resourceType]
	if len(resourceMap) == 0 {
		return nil
	}

	return resourceMap
}

// GetByID returns a specific resource by ID and type
func (rc *ResourceCollection) GetByID(resourceType string, id string) (*RemoteResource, bool) {
	resourceMap := rc.resources[resourceType]
	if resourceMap == nil {
		return nil, false
	}

	resource, exists := resourceMap[id]
	return resource, exists
}

func (rc *ResourceCollection) GetURNByID(resourceType string, id string) (string, error) {
	resource, exists := rc.GetByID(resourceType, id)
	if !exists {
		return "", ErrRemoteResourceNotFound
	}

	if resource.ExternalID == "" {
		return "", ErrRemoteResourceExternalIdNotFound
	}

	return URN(resource.ExternalID, resourceType), nil
}

// Merge merges resources from another ResourceCollection into a new collection
// Returns a new ResourceCollection or an error if there are any overlapping keys
func (rc *ResourceCollection) Merge(other *ResourceCollection) (*ResourceCollection, error) {
	if other == nil {
		return rc, nil
	}
	newCollection := NewResourceCollection()

	// First, copy all resources from current collection
	for k1, v1 := range rc.resources {
		newMap := make(map[string]*RemoteResource)
		for k2, v2 := range v1 {
			newMap[k2] = v2
		}
		newCollection.resources[k1] = newMap
	}

	// Check for overlaps and merge from other collection
	for k1, v1 := range other.resources {
		// Check if the key already exists - error out immediately
		if existingMap := newCollection.resources[k1]; existingMap != nil {
			return nil, fmt.Errorf("%w at %s", ErrDuplicateResource, k1)
		}

		// Initialize map and copy all resources
		newCollection.resources[k1] = make(map[string]*RemoteResource)
		for k2, v2 := range v1 {
			newCollection.resources[k1][k2] = v2
		}
	}

	return newCollection, nil
}
