package datacatalog

import (
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ImportableRefs overrides the EmptyProvider default to declare the
// cross-resource references catalog importables hold, so import --merge can
// detect pending-delete conflicts up front. These mirror the top-level
// references each resource's export resolves: events → category, properties →
// custom-type definitions, custom types → properties and nested custom types,
// tracking plans → events and their properties. Deeper references (e.g. variant
// properties) fall back to the export-time resolver.
func (p *Provider) ImportableRefs() []importmatcher.RefLister {
	return []importmatcher.RefLister{
		{ResourceType: types.EventResourceType, Refs: eventRefs},
		{ResourceType: types.PropertyResourceType, Refs: propertyRefs},
		{ResourceType: types.CustomTypeResourceType, Refs: customTypeRefs},
		{ResourceType: types.TrackingPlanResourceType, Refs: trackingPlanRefs},
	}
}

func eventRefs(r *resources.RemoteResource) []importmatcher.Ref {
	e := r.Data.(*catalog.Event)
	if e.CategoryId == nil || *e.CategoryId == "" {
		return nil
	}
	return []importmatcher.Ref{{EntityType: types.CategoryResourceType, RemoteID: *e.CategoryId}}
}

func propertyRefs(r *resources.RemoteResource) []importmatcher.Ref {
	p := r.Data.(*catalog.Property)
	var refs []importmatcher.Ref
	for _, id := range []string{p.DefinitionId, p.ItemDefinitionId} {
		if id != "" {
			refs = append(refs, importmatcher.Ref{EntityType: types.CustomTypeResourceType, RemoteID: id})
		}
	}
	return refs
}

func customTypeRefs(r *resources.RemoteResource) []importmatcher.Ref {
	ct := r.Data.(*catalog.CustomType)
	var refs []importmatcher.Ref
	for _, prop := range ct.Properties {
		if prop.ID != "" {
			refs = append(refs, importmatcher.Ref{EntityType: types.PropertyResourceType, RemoteID: prop.ID})
		}
	}
	// ItemDefinitions are untyped; each is a map carrying a nested custom-type
	// id — read exactly as the custom-type export does.
	for _, item := range ct.ItemDefinitions {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, ok := itemMap["id"].(string)
		if !ok || id == "" {
			continue
		}
		refs = append(refs, importmatcher.Ref{EntityType: types.CustomTypeResourceType, RemoteID: id})
	}
	return refs
}

func trackingPlanRefs(r *resources.RemoteResource) []importmatcher.Ref {
	tp := r.Data.(*catalog.TrackingPlanWithIdentifiers)
	var refs []importmatcher.Ref
	for _, event := range tp.Events {
		if event == nil {
			continue
		}
		if event.ID != "" {
			refs = append(refs, importmatcher.Ref{EntityType: types.EventResourceType, RemoteID: event.ID})
		}
		for _, prop := range event.Properties {
			if prop != nil && prop.ID != "" {
				refs = append(refs, importmatcher.Ref{EntityType: types.PropertyResourceType, RemoteID: prop.ID})
			}
		}
	}
	return refs
}
