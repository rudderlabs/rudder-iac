package source

import (
	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// Matcher returns the import --merge matcher for event stream sources. Sources
// are unique by name within a workspace (mirroring the source uniqueness
// rule), so a remote source links to a local source of the same name.
func Matcher() importmatcher.Matcher {
	return importmatcher.Matcher{
		ResourceType: ResourceType,
		Match:        matchSource,
	}
}

func matchSource(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	remote, ok := r.Data.(*sourceClient.EventStreamSource)
	if !ok || remote.Name == "" {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, ResourceType, func(data resources.ResourceData) bool {
		name, _ := data[NameKey].(string)
		return name == remote.Name
	})
	return local
}
