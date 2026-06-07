package provider

import (
	"errors"
	"fmt"
)

// ErrUnsupportedType is the routing sentinel returned by CompositeProvider.ProviderForType
// when no registered provider handles the requested resource type. Verb-layer commands
// (get, delete, describe, set-external-id) use errors.Is against this value to detect
// an unrecognised type before dispatching. It is intentionally distinct from
// ErrUnsupportedResourceType: that is a per-operation struct error returned by handler
// lifecycle methods (Create/Update/Delete/Import) when a handler doesn't support a
// specific resource type within a provider.
var ErrUnsupportedType = errors.New("unsupported resource type")

type ErrUnsupportedSpecKind struct {
	Kind string
}

func (e *ErrUnsupportedSpecKind) Error() string {
	return fmt.Sprintf("unsupported spec kind '%s'", e.Kind)
}

// ErrUnsupportedResourceType is the per-operation struct error returned by handler
// lifecycle methods (Create/Update/Delete/Import) when a handler does not support
// the given resource type within a provider. It is intentionally distinct from
// ErrUnsupportedType: that is the routing sentinel returned at the composite-provider
// level when no provider is registered for a type at all.
type ErrUnsupportedResourceType struct {
	ResourceType string
}

func (e *ErrUnsupportedResourceType) Error() string {
	return fmt.Sprintf("unsupported resource type '%s'", e.ResourceType)
}
