package provider

type ErrUnsupportedSpecKind struct {
	Kind string
}

func (e *ErrUnsupportedSpecKind) Error() string {
	return "unsupported spec kind: " + e.Kind
}

type ErrUnsupportedResourceType struct {
	ResourceType string
}

func (e *ErrUnsupportedResourceType) Error() string {
	return "unsupported resource type: " + e.ResourceType
}
