package model

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
)

type ImportableEvent struct {
	localcatalog.Event
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
	if err := mapstructure.Decode(e.Event, &toReturn); err != nil {
		return nil, fmt.Errorf("decoding event: %w", err)
	}

	return toReturn, nil
}

func (e *ImportableEvent) fromUpstream(externalID string, upstream *catalog.Event, resolver resolver.ReferenceResolver) error {
	e.Event.LocalID = externalID
	e.Event.Name = upstream.Name
	e.Event.Description = upstream.Description
	e.Event.Type = upstream.EventType

	// Resolve category reference if categoryId is set
	if upstream.CategoryId != nil {
		categoryRef, err := resolver.ResolveToReference(
			state.CategoryResourceType,
			*upstream.CategoryId,
		)
		if err != nil {
			return fmt.Errorf("category reference resolution for event %s: %w", e.Event.LocalID, err)
		}

		if categoryRef == "" {
			return fmt.Errorf("resolved category reference is empty for event %s", e.Event.LocalID)
		}

		e.Event.CategoryRef = &categoryRef
	}

	return nil
}