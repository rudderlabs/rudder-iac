package table

import (
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// Matcher returns the import --merge matcher for table sources. It mirrors the
// sqlmodel rule: a remote source links to a local one whose display_name folds
// to the same value. See sqlmodel.Matcher for why account_id is no longer a
// second conjunct.
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
		displayName, _ := data[sqlmodel.DisplayNameKey].(string)
		return importmatcher.SameName(displayName, remote.Name)
	})
	return local
}
