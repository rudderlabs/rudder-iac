package destination

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ResourceMatchers overrides the EmptyProvider default to opt into import
// --merge smart linking: a remote destination links to the local destination
// with the same display name, which is expected to be unique per workspace and
// is compared case-sensitively. Type stays out of the key, so a same-named
// destination of another type surfaces as the immutable-type error at apply
// instead of as a duplicate spec.
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
