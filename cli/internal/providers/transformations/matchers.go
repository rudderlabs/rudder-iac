package transformations

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/resolve"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ResourceMatchers implements resolve.MatcherProvider for import --merge
// conflict detection. Libraries match on import name — the unique handle used
// in transformation code; two libraries can share a display name but not an
// import name. Transformations match on name. Libraries first, mirroring
// handler registration order (dependencies before dependents).
func (p *Provider) ResourceMatchers() []resolve.Matcher {
	return []resolve.Matcher{
		{ResourceType: ttypes.LibraryResourceType, Match: matchLibrary},
		{ResourceType: ttypes.TransformationResourceType, Match: matchTransformation},
	}
}

func matchLibrary(ctx resolve.MatchContext, r *resources.RemoteResource) *resources.Resource {
	remote, ok := r.Data.(*model.RemoteLibrary)
	if !ok || remote.ImportName == "" {
		return nil
	}

	local, _ := resolve.MatchByRawData(ctx.LocalGraph, ttypes.LibraryResourceType, func(raw any) bool {
		library, ok := raw.(*model.LibraryResource)
		return ok && library.ImportName == remote.ImportName
	})
	return local
}

func matchTransformation(ctx resolve.MatchContext, r *resources.RemoteResource) *resources.Resource {
	remote, ok := r.Data.(*model.RemoteTransformation)
	if !ok || remote.Name == "" {
		return nil
	}

	local, _ := resolve.MatchByRawData(ctx.LocalGraph, ttypes.TransformationResourceType, func(raw any) bool {
		transformation, ok := raw.(*model.TransformationResource)
		return ok && transformation.Name == remote.Name
	})
	return local
}
