package catalog

type State struct {
	Version   string                   `json:"version"`
	Resources map[string]ResourceState `json:"resources"`
}

type ResourceCollection string

const (
	ResourceCollectionEvents        ResourceCollection = "events"
	ResourceCollectionProperties    ResourceCollection = "properties"
	ResourceCollectionTrackingPlans ResourceCollection = "tracking-plans"
	ResourceCollectionCustomTypes   ResourceCollection = "custom-types"
	ResourceCollectionCategories    ResourceCollection = "categories"
)

type ResourceState struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Input        map[string]interface{} `json:"input"`
	Output       map[string]interface{} `json:"output"`
	Dependencies []string               `json:"dependencies"`
}

type PutStateRequest struct {
	Collection ResourceCollection
	ID         string
	URN        string
	State      ResourceState
}

type DeleteStateRequest struct {
	Collection ResourceCollection
	ID         string
}

type EventStateArgs struct {
	PutStateRequest
	EventID string
}
