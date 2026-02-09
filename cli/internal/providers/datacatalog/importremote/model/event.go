package model

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
)

type ImportableEvent struct {
	localcatalog.EventV1
}

// ForExport loads the event from the upstream and returns it in a format
// that can be exported to a file.
func (e *ImportableEvent) ForExport(
	externalID string,
	upstream *catalog.Event,
	resolver resolver.ReferenceResolver,
) (map[string]any, error) {
	if err := e.fromUpstream(externalID, upstream, resolver); err != nil {
		return nil, fmt.Errorf("loading event from upstream: %w", err)
	}

	toReturn := make(map[string]any)
	if err := mapstructure.Decode(e.EventV1, &toReturn); err != nil {
		return nil, fmt.Errorf("decoding event: %w", err)
	}

	return toReturn, nil
}

func (e *ImportableEvent) fromUpstream(externalID string, upstream *catalog.Event, resolver resolver.ReferenceResolver) error {
	e.EventV1.LocalID = externalID
	e.EventV1.Name = upstream.Name
	e.EventV1.Description = upstream.Description
	e.EventV1.Type = upstream.EventType

	// Resolve category reference if categoryId is set
	if upstream.CategoryId != nil {
		categoryRef, err := resolver.ResolveToReference(
			types.CategoryResourceType,
			*upstream.CategoryId,
		)
		if err != nil {
			return fmt.Errorf("category reference resolution for event %s: %w", e.EventV1.LocalID, err)
		}

		if categoryRef == "" {
			return fmt.Errorf("resolved category reference is empty for event %s", e.EventV1.LocalID)
		}

		e.EventV1.CategoryRef = &categoryRef
	}

	return nil
}

type ImportableEventV1 struct {
	localcatalog.EventV1
}

// ForExport loads the event from the upstream and returns it in a format
// that can be exported to a file.
func (e *ImportableEventV1) ForExport(
	externalID string,
	upstream *catalog.Event,
	resolver resolver.ReferenceResolver,
) (map[string]any, error) {
	if err := e.fromUpstream(externalID, upstream, resolver); err != nil {
		return nil, fmt.Errorf("loading event from upstream: %w", err)
	}

	toReturn := make(map[string]any)
	if err := mapstructure.Decode(e.EventV1, &toReturn); err != nil {
		return nil, fmt.Errorf("decoding event: %w", err)
	}

	return toReturn, nil
}

func (e *ImportableEventV1) fromUpstream(externalID string, upstream *catalog.Event, resolver resolver.ReferenceResolver) error {
	v0Event := ImportableEvent{}
	if err := v0Event.fromUpstream(externalID, upstream, resolver); err != nil {
		return fmt.Errorf("loading event from upstream: %w", err)
	}
	e.EventV1 = v0Event.EventV1
	return nil
}
