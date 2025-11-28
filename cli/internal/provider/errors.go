package provider

import "fmt"

type ErrUnsupportedSpecKind struct {
	Kind string
}

func (e *ErrUnsupportedSpecKind) Error() string {
	return fmt.Sprintf("unsupported spec kind '%s'", e.Kind)
}

type ErrUnsupportedResourceType struct {
	ResourceType string
}

func (e *ErrUnsupportedResourceType) Error() string {
	return fmt.Sprintf("unsupported resource type '%s'", e.ResourceType)
}
