package sqlmodel

import (
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// Matcher returns the import --merge matcher for SQL models. A remote model
// links to a local model of the same display_name AND account_id. The local
// uniqueness rule keys on display_name alone, but a SQL model belongs to an
// account, so matching on account_id too avoids falsely linking same-named
// models across accounts — a false link is worse than falling back to the
// namer (which then produces a new spec the user can reconcile).
func Matcher() importmatcher.Matcher {
	return importmatcher.Matcher{
		ResourceType: ResourceType,
		Match:        matchSQLModel,
	}
}

func matchSQLModel(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	remote, ok := r.Data.(*retlClient.RETLSource)
	if !ok || remote.Name == "" {
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
