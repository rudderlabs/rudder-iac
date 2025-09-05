package resources

import (
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
)

// ResourceCollection provides a generic container for mixed resource types
// with type-safe getters and efficient ID-based lookups
type ResourceCollection struct {
	resources map[string]map[string]interface{}
}

// NewResourceCollection creates a new empty ResourceCollection
func NewResourceCollection() *ResourceCollection {
	return &ResourceCollection{
		resources: make(map[string]map[string]interface{}),
	}
}

// ensureResourceType ensures the resource type map exists
func (rc *ResourceCollection) ensureResourceType(resourceType string) {
	if rc.resources[resourceType] == nil {
		rc.resources[resourceType] = make(map[string]interface{})
	}
}

// Type-safe getters for collections (collecting from maps)
func (rc *ResourceCollection) GetEvents() []*catalog.Event {
	resourceMap := rc.resources["events"]
	if resourceMap == nil {
		return nil
	}

	events := make([]*catalog.Event, 0, len(resourceMap))
	for _, resource := range resourceMap {
		events = append(events, resource.(*catalog.Event))
	}

	return events
}

func (rc *ResourceCollection) GetProperties() []*catalog.Property {
	resourceMap := rc.resources["properties"]
	if resourceMap == nil {
		return nil
	}

	properties := make([]*catalog.Property, 0, len(resourceMap))
	for _, resource := range resourceMap {
		properties = append(properties, resource.(*catalog.Property))
	}

	return properties
}

func (rc *ResourceCollection) GetCategories() []*catalog.Category {
	resourceMap := rc.resources["categories"]
	if resourceMap == nil {
		return nil
	}

	categories := make([]*catalog.Category, 0, len(resourceMap))
	for _, resource := range resourceMap {
		categories = append(categories, resource.(*catalog.Category))
	}

	return categories
}

func (rc *ResourceCollection) GetCustomTypes() []*catalog.CustomType {
	resourceMap := rc.resources["customTypes"]
	if resourceMap == nil {
		return nil
	}

	customTypes := make([]*catalog.CustomType, 0, len(resourceMap))
	for _, resource := range resourceMap {
		customTypes = append(customTypes, resource.(*catalog.CustomType))
	}

	return customTypes
}

func (rc *ResourceCollection) GetTrackingPlans() []*catalog.TrackingPlan {
	resourceMap := rc.resources["trackingPlans"]
	if resourceMap == nil {
		return nil
	}

	trackingPlans := make([]*catalog.TrackingPlan, 0, len(resourceMap))
	for _, resource := range resourceMap {
		trackingPlans = append(trackingPlans, resource.(*catalog.TrackingPlan))
	}

	return trackingPlans
}

// ID-based lookup methods
func (rc *ResourceCollection) GetEvent(id string) (*catalog.Event, bool) {
	resourceMap := rc.resources["events"]
	if resourceMap == nil {
		return nil, false
	}

	resource, exists := resourceMap[id]
	if !exists {
		return nil, false
	}

	return resource.(*catalog.Event), true
}

func (rc *ResourceCollection) GetProperty(id string) (*catalog.Property, bool) {
	resourceMap := rc.resources["properties"]
	if resourceMap == nil {
		return nil, false
	}

	resource, exists := resourceMap[id]
	if !exists {
		return nil, false
	}

	return resource.(*catalog.Property), true
}

func (rc *ResourceCollection) GetCategory(id string) (*catalog.Category, bool) {
	resourceMap := rc.resources["categories"]
	if resourceMap == nil {
		return nil, false
	}

	resource, exists := resourceMap[id]
	if !exists {
		return nil, false
	}

	return resource.(*catalog.Category), true
}

func (rc *ResourceCollection) GetCustomType(id string) (*catalog.CustomType, bool) {
	resourceMap := rc.resources["customTypes"]
	if resourceMap == nil {
		return nil, false
	}

	resource, exists := resourceMap[id]
	if !exists {
		return nil, false
	}

	return resource.(*catalog.CustomType), true
}

func (rc *ResourceCollection) GetTrackingPlan(id string) (*catalog.TrackingPlan, bool) {
	resourceMap := rc.resources["trackingPlans"]
	if resourceMap == nil {
		return nil, false
	}

	resource, exists := resourceMap[id]
	if !exists {
		return nil, false
	}

	return resource.(*catalog.TrackingPlan), true
}

// Convenience setter methods for each resource type
func (rc *ResourceCollection) SetEvents(events []*catalog.Event) {
	rc.ensureResourceType("events")
	for _, event := range events {
		rc.resources["events"][event.ID] = event
	}
}

func (rc *ResourceCollection) SetProperties(properties []*catalog.Property) {
	rc.ensureResourceType("properties")
	for _, property := range properties {
		rc.resources["properties"][property.ID] = property
	}
}

func (rc *ResourceCollection) SetCategories(categories []*catalog.Category) {
	rc.ensureResourceType("categories")
	for _, category := range categories {
		rc.resources["categories"][category.ID] = category
	}
}

func (rc *ResourceCollection) SetCustomTypes(customTypes []*catalog.CustomType) {
	rc.ensureResourceType("customTypes")
	for _, customType := range customTypes {
		rc.resources["customTypes"][customType.ID] = customType
	}
}

func (rc *ResourceCollection) SetTrackingPlans(trackingPlans []*catalog.TrackingPlan) {
	rc.ensureResourceType("trackingPlans")
	for _, trackingPlan := range trackingPlans {
		rc.resources["trackingPlans"][trackingPlan.ID] = trackingPlan
	}
}
