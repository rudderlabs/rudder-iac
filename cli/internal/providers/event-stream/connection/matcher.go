package connection

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// Matcher returns the import --merge matcher for event stream connections. The
// backend allows one connection per source–destination pair, so a remote
// connection links to the local connection wired to the same endpoints. Listed
// after the source matcher so endpoint lookups can rely on source matches
// being recorded already.
func Matcher() importmatcher.Matcher {
	return importmatcher.Matcher{
		ResourceType: EventStreamConnectionResourceType,
		Match:        matchConnection,
	}
}

func matchConnection(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	// Dispatched by resource type, so a wrong payload is a wiring bug — panic.
	remote := r.Data.(*RemoteConnection)

	// Destination rows carry no marks today (the destination provider has no
	// matcher), so resolving them does not depend on cross-provider matcher
	// order.
	sourceURN, ok := importmatcher.ResolveLocalURN(scope, source.ResourceType, remote.SourceID)
	if !ok {
		return nil
	}
	destinationURN, ok := importmatcher.ResolveLocalURN(scope, destination.DestinationResourceType, remote.DestinationID)
	if !ok {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, EventStreamConnectionResourceType, func(data resources.ResourceData) bool {
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
