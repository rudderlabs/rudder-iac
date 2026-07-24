package datagraph

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	dghandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	modelhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	relhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ImportableRefs overrides the EmptyProvider default to declare the references
// data graph importables hold. These references resolve through GetURNByID at
// export, which never consults the local graph, so without this precheck a
// child pointing at a pending-deleted parent would emit a dangling URN caught
// only at apply. A model references its parent data graph; a relationship
// references its parent data graph and both endpoint models.
func (p *Provider) ImportableRefs() []importmatcher.RefLister {
	return []importmatcher.RefLister{
		{ResourceType: modelhandler.HandlerMetadata.ResourceType, Refs: modelRefs},
		{ResourceType: relhandler.HandlerMetadata.ResourceType, Refs: relationshipRefs},
	}
}

func modelRefs(r *resources.RemoteResource) []importmatcher.Ref {
	m := r.Data.(*dgModel.RemoteModel)
	if m.DataGraphID == "" {
		return nil
	}
	return []importmatcher.Ref{{EntityType: dghandler.HandlerMetadata.ResourceType, RemoteID: m.DataGraphID}}
}

func relationshipRefs(r *resources.RemoteResource) []importmatcher.Ref {
	rel := r.Data.(*dgModel.RemoteRelationship)
	var refs []importmatcher.Ref
	if rel.DataGraphID != "" {
		refs = append(refs, importmatcher.Ref{EntityType: dghandler.HandlerMetadata.ResourceType, RemoteID: rel.DataGraphID})
	}
	for _, modelID := range []string{rel.SourceModelID, rel.TargetModelID} {
		if modelID != "" {
			refs = append(refs, importmatcher.Ref{EntityType: modelhandler.HandlerMetadata.ResourceType, RemoteID: modelID})
		}
	}
	return refs
}
