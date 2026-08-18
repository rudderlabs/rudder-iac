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

	sourceURN, ok := resolveEndpoint(scope, source.ResourceType, remote.SourceID)
	if !ok {
		return nil
	}
	destinationURN, ok := resolveEndpoint(scope, destination.DestinationResourceType, remote.DestinationID)
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

// resolveEndpoint maps a remote endpoint ID to the local resource URN it
// corresponds to: either the URN of the endpoint matched in this import, or of
// one already managed locally (found by its import metadata's remote ID),
// mirroring the datacatalog matchers' resolveTypeRef. ok is false when the
// endpoint has no local counterpart — the connection then stays unmatched.
// Destination rows carry no marks today (the destination provider has no
// matcher), so consulting them does not depend on cross-provider matcher
// order.
func resolveEndpoint(scope importmatcher.Scope, resourceType string, remoteID string) (urn string, ok bool) {
	if scope.Importable != nil {
		if endpoint, found := scope.Importable.GetByID(resourceType, remoteID); found {
			if endpoint.MatchedWith != nil {
				return endpoint.MatchedWith.URN(), true
			}
			return "", false
		}
	}
	for _, endpoint := range scope.LocalGraph.ResourcesByType(resourceType) {
		if meta := endpoint.ImportMetadata(); meta != nil && meta.RemoteId == remoteID {
			return endpoint.URN(), true
		}
	}
	return "", false
}
