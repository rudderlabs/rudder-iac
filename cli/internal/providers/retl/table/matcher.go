package table

import (
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// Matcher returns the import --merge matcher for table sources. Mirrors the
// sqlmodel rule: a remote source links to a local one of the same display_name
// AND account_id. Matching on account_id too avoids falsely linking same-named
// tables across accounts — a false link is worse than falling back to the
// namer, which produces a new spec the user can reconcile.
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
			displayName = data[DisplayNameKey].(string)
			accountID   = data[AccountIDKey].(string)
		)
		return displayName == remote.Name && accountID == remote.AccountID
	})
	return local
}
