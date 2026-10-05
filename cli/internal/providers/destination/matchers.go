package destination

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ResourceMatchers overrides the EmptyProvider default to opt into import
// --merge smart linking. Destinations are unique by exact name within a
// workspace (case-sensitive, unlike sources) — the same uniqueness the semantic
// rule enforces on display_name locally — so a remote destination links to a
// local destination of the same display name.
// Type is deliberately not part of the match: a same-named destination of a
// different type is an upstream collision either way, and linking it surfaces
// that as the immutable-type error rather than as a duplicate spec.
func (p *Provider) ResourceMatchers() []importmatcher.Matcher {
	return []importmatcher.Matcher{
		{ResourceType: DestinationResourceType, Match: matchDestination},
	}
}

func matchDestination(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	// Dispatched by resource type, so a wrong payload is a wiring bug — panic.
	remote := r.Data.(*RemoteDestination)
	if remote.Name == "" {
		return nil
	}

	local, _ := importmatcher.ByRawData(scope.LocalGraph, DestinationResourceType, func(raw any) bool {
		return raw.(*DestinationResource).DisplayName == remote.Name
	})
	return local
}
