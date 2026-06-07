package provider

import (
	"errors"
	"fmt"
)

// ErrUnsupportedType is the sentinel returned when no provider handles the requested resource type.
// Callers can use errors.Is(err, provider.ErrUnsupportedType) for type-safe error checking.
var ErrUnsupportedType = errors.New("unsupported resource type")

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

// Unwrap allows errors.Is(err, ErrUnsupportedType) to work on ErrUnsupportedResourceType values.
func (e *ErrUnsupportedResourceType) Unwrap() error {
	return ErrUnsupportedType
}
