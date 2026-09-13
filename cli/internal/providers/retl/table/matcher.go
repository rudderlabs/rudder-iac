package table

import (
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// Matcher returns the import --merge matcher for table sources. It mirrors the
// sqlmodel rule: a remote source links to a local one with the same
// display_name AND account_id, since a false link across accounts is worse than
// falling back to the namer and generating a new spec.
func Matcher() importmatcher.Matcher {
	return importmatcher.Matcher{
		ResourceType: ResourceType,
		Match:        matchTable,
	}
}

func matchTable(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	// Dispatched by resource type, so a wrong payload is a wiring bug — panic.
	remote := r.Data.(*retlClient.RETLSource)
	if remote.Name == "" {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, ResourceType, func(data resources.ResourceData) bool {
		var (
			displayName, _ = data[DisplayNameKey].(string)
			accountID, _   = data[AccountIDKey].(string)
		)
		return displayName == remote.Name && accountID == remote.AccountID
	})
	return local
}
