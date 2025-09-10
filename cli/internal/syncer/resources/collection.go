package resources

// ResourceCollection provides a generic container for mixed resource types
// with generic getters and efficient ID-based lookups
type ResourceCollection struct {
	resources map[string]map[string]interface{}
}

// NewResourceCollection creates a new empty ResourceCollection
func NewResourceCollection() *ResourceCollection {
	return &ResourceCollection{
		resources: make(map[string]map[string]interface{}),
	}
}

// Set stores a resource map for the given resource type
func (rc *ResourceCollection) Set(resourceType string, resourceMap map[string]interface{}) {
	rc.resources[resourceType] = resourceMap
}

// GetAll returns all resources of the given type as a slice
func (rc *ResourceCollection) GetAll(resourceType string) map[string]interface{} {
	resourceMap := rc.resources[resourceType]
	if resourceMap == nil || len(resourceMap) == 0 {
		return nil
	}

	return resourceMap
}

// GetById returns a specific resource by ID and type
func (rc *ResourceCollection) GetById(resourceType string, id string) (interface{}, bool) {
	resourceMap := rc.resources[resourceType]
	if resourceMap == nil {
		return nil, false
	}

	resource, exists := resourceMap[id]
	return resource, exists
}
