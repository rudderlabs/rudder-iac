package datagraph

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	dghandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	modelhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	relhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ResourceMatchers overrides the EmptyProvider default to opt into import
// --merge smart linking. A workspace holds at most one data graph per account,
// so the parent matches on account_id alone. Children (models, relationships)
// match by display_name — unique within a graph, enforced by the
// unique-names rule — and only within a matched parent, so the parent matcher
// is registered first and child matchers consult its match via the importable
// collection.
func (p *Provider) ResourceMatchers() []importmatcher.Matcher {
	return []importmatcher.Matcher{
		{ResourceType: dghandler.HandlerMetadata.ResourceType, Match: matchDataGraph},
		{ResourceType: modelhandler.HandlerMetadata.ResourceType, Match: matchModel},
		{ResourceType: relhandler.HandlerMetadata.ResourceType, Match: matchRelationship},
	}
}

func matchDataGraph(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	// Dispatched by resource type, so a wrong payload is a wiring bug — panic.
	remote := r.Data.(*dgModel.RemoteDataGraph)
	if remote.AccountID == "" {
		return nil
	}

	local, _ := importmatcher.ByRawData(scope.LocalGraph, dghandler.HandlerMetadata.ResourceType, func(raw any) bool {
		return raw.(*dgModel.DataGraphResource).AccountID == remote.AccountID
	})
	return local
}

func matchModel(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	remote := r.Data.(*dgModel.RemoteModel)
	if remote.Name == "" {
		return nil
	}

	parentURN, ok := matchedParentURN(scope, remote.DataGraphID)
	if !ok {
		return nil
	}

	local, _ := importmatcher.ByRawData(scope.LocalGraph, modelhandler.HandlerMetadata.ResourceType, func(raw any) bool {
		m := raw.(*dgModel.ModelResource)
		return m.DisplayName == remote.Name && m.DataGraphRef != nil && m.DataGraphRef.URN == parentURN
	})
	return local
}

func matchRelationship(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	remote := r.Data.(*dgModel.RemoteRelationship)
	if remote.Name == "" {
		return nil
	}

	parentURN, ok := matchedParentURN(scope, remote.DataGraphID)
	if !ok {
		return nil
	}

	local, _ := importmatcher.ByRawData(scope.LocalGraph, relhandler.HandlerMetadata.ResourceType, func(raw any) bool {
		rel := raw.(*dgModel.RelationshipResource)
		return rel.DisplayName == remote.Name && rel.DataGraphRef != nil && rel.DataGraphRef.URN == parentURN
	})
	return local
}

// matchedParentURN resolves the remote parent data graph to its matched local
// URN. Children only link within a matched parent — an unmatched or
// non-importable parent means the child never matches.
func matchedParentURN(scope importmatcher.Scope, dataGraphID string) (string, bool) {
	parent, ok := scope.Importable.GetByID(dghandler.HandlerMetadata.ResourceType, dataGraphID)
	if !ok || parent.MatchedWith == nil {
		return "", false
	}
	return parent.MatchedWith.URN(), true
}
