package providers

type ErrUnsupportedResourceType struct {
	ResourceType string
}

func (e *ErrUnsupportedResourceType) Error() string {
	return "unsupported resource type: " + e.ResourceType
}

func NewErrUnsupportedResourceType(resourceType string) error {
	return &ErrUnsupportedResourceType{ResourceType: resourceType}
}

type ErrUnsupporterResourceAction struct {
	ResourceType string
	Action       string
}

func (e *ErrUnsupporterResourceAction) Error() string {
	return "unsupported resource action: " + e.Action + " for resource type: " + e.ResourceType
}

func NewErrUnsupporterResourceAction(resourceType, action string) error {
	return &ErrUnsupporterResourceAction{ResourceType: resourceType, Action: action}
}
