package transformations

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ResourceMatchers overrides the EmptyProvider default to opt into import
// --merge smart linking. Libraries match on import name — the unique handle
// used in transformation code; two libraries can share a display name but not
// an import name. Transformations match on name. Libraries first, mirroring
// handler registration order (dependencies before dependents).
func (p *Provider) ResourceMatchers() []importmatcher.Matcher {
	return []importmatcher.Matcher{
		{ResourceType: ttypes.LibraryResourceType, Match: matchLibrary},
		{ResourceType: ttypes.TransformationResourceType, Match: matchTransformation},
	}
}

func matchLibrary(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	// Dispatched by resource type, so a wrong payload is a wiring bug — panic.
	remote := r.Data.(*model.RemoteLibrary)
	if remote.ImportName == "" {
		return nil
	}

	local, _ := importmatcher.ByRawData(scope.LocalGraph, ttypes.LibraryResourceType, func(raw any) bool {
		return raw.(*model.LibraryResource).ImportName == remote.ImportName
	})
	return local
}

func matchTransformation(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	remote := r.Data.(*model.RemoteTransformation)
	if remote.Name == "" {
		return nil
	}

	local, _ := importmatcher.ByRawData(scope.LocalGraph, ttypes.TransformationResourceType, func(raw any) bool {
		return raw.(*model.TransformationResource).Name == remote.Name
	})
	return local
}
