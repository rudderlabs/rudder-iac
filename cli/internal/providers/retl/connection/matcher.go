package connection

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// Matcher returns the import --merge matcher for rETL connections. The backend
// allows one connection per source–destination pair, so a remote connection
// links to the local connection wired to the same endpoints. Listed after the
// source matchers so endpoint lookups can rely on source matches being recorded
// already.
func Matcher() importmatcher.Matcher {
	return importmatcher.Matcher{
		ResourceType: ResourceType,
		Match:        matchConnection,
	}
}

func matchConnection(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	// Dispatched by resource type, so a wrong payload is a wiring bug — panic.
	remote := r.Data.(*RemoteConnection)

	sourceURN, ok := importmatcher.EndpointURN(scope, remote.SourceKind.ResourceType, remote.SourceID, remote.SourceExternalID)
	if !ok {
		return nil
	}
	destinationURN, ok := importmatcher.EndpointURN(scope, destination.DestinationResourceType, remote.DestinationID, remote.DestinationExternalID)
	if !ok {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, ResourceType, func(data resources.ResourceData) bool {
		sourceRef, ok := data[SourceKey].(*resources.PropertyRef)
		if !ok {
			return false
		}
		destinationRef, ok := data[DestinationKey].(*resources.PropertyRef)
		if !ok {
			return false
		}
		return sourceRef.URN == sourceURN && destinationRef.URN == destinationURN
	})
	return local
}
