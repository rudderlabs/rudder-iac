package catalog

type ResourceCollection string

const (
	ResourceCollectionEvents        ResourceCollection = "events"
	ResourceCollectionProperties    ResourceCollection = "properties"
	ResourceCollectionTrackingPlans ResourceCollection = "tracking-plans"
	ResourceCollectionCustomTypes   ResourceCollection = "custom-types"
	ResourceCollectionCategories    ResourceCollection = "categories"
)
