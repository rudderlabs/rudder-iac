package sqlmodel

import (
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// Matcher returns the import --merge matcher for SQL models. A remote model
// links to a local model whose display_name folds to the same value.
//
// account_id used to be a second conjunct, guarding against falsely linking
// same-named models across accounts. That case can no longer arise: source
// names are unique case-insensitively across the whole workspace regardless of
// account or category, so display_name alone already identifies the source
// upstream. Keeping the conjunct was actively harmful — a spec naming its
// account by reference holds a PropertyRef whose remote id is unknown until
// apply, so it never matched, and import wrote a second spec for a source that
// already existed.
func Matcher() importmatcher.Matcher {
	return importmatcher.Matcher{
		ResourceType: ResourceType,
		Match:        matchSQLModel,
	}
}

func matchSQLModel(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	// Dispatched by resource type, so a wrong payload is a wiring bug — panic.
	remote := r.Data.(*retlClient.RETLSource)
	if remote.Name == "" {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, ResourceType, func(data resources.ResourceData) bool {
		displayName, _ := data[DisplayNameKey].(string)
		return importmatcher.SameName(displayName, remote.Name)
	})
	return local
}
