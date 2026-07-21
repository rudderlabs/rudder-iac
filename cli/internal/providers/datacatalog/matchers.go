package datacatalog

import (
	"slices"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ResourceMatchers overrides the EmptyProvider default to opt into import
// --merge smart linking. Match keys mirror the provider's own uniqueness
// rules: category, custom-type and tracking-plan are unique by name; events
// by name+eventType (non-track events carry empty names, enforced by syntax
// validation, so eventType alone distinguishes them); properties by
// name+type+item_types. Registered in dependency order.
func (p *Provider) ResourceMatchers() []importmatcher.Matcher {
	return []importmatcher.Matcher{
		{ResourceType: types.CategoryResourceType, Match: matchCategory},
		{ResourceType: types.CustomTypeResourceType, Match: matchCustomType},
		{ResourceType: types.PropertyResourceType, Match: matchProperty},
		{ResourceType: types.EventResourceType, Match: matchEvent},
		{ResourceType: types.TrackingPlanResourceType, Match: matchTrackingPlan},
	}
}

func matchCategory(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	// Dispatched by resource type, so a wrong payload is a wiring bug — panic.
	remote := r.Data.(*catalog.Category)
	if remote.Name == "" {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, types.CategoryResourceType, func(data resources.ResourceData) bool {
		return data["name"].(string) == remote.Name
	})
	return local
}

func matchCustomType(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	remote := r.Data.(*catalog.CustomType)
	if remote.Name == "" {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, types.CustomTypeResourceType, func(data resources.ResourceData) bool {
		return data["name"].(string) == remote.Name
	})
	return local
}

func matchTrackingPlan(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	remote := r.Data.(*catalog.TrackingPlanWithIdentifiers)
	if remote.Name == "" {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, types.TrackingPlanResourceType, func(data resources.ResourceData) bool {
		return data["name"].(string) == remote.Name
	})
	return local
}

func matchEvent(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	remote := r.Data.(*catalog.Event)
	if remote.EventType == "" {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, types.EventResourceType, func(data resources.ResourceData) bool {
		var (
			name      = data["name"].(string)
			eventType = data["eventType"].(string)
		)
		return name == remote.Name && eventType == remote.EventType
	})
	return local
}

func matchProperty(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	remote := r.Data.(*catalog.Property)
	if remote.Name == "" {
		return nil
	}

	// A remote property whose type (or array item type) is a custom type
	// references it by DefinitionId/ItemDefinitionId. Resolve each to the local
	// custom-type URN so it can be compared against the local property's
	// PropertyRef. A DefinitionId that resolves to nothing (the custom type is
	// neither imported now nor already managed) makes the property unmatchable.
	typeRef, ok := resolveTypeRef(scope, remote.DefinitionId)
	if !ok {
		return nil
	}
	itemRef, ok := resolveTypeRef(scope, remote.ItemDefinitionId)
	if !ok {
		return nil
	}

	// Remote config uses camelCase keys; local config is snake_case.
	remoteItems, ok := stringItemTypes(remote.Config, "itemTypes")
	if !ok {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, types.PropertyResourceType, func(data resources.ResourceData) bool {
		if data["name"].(string) != remote.Name {
			return false
		}

		if !typeMatches(data["type"], remote.Type, typeRef) {
			return false
		}

		return itemTypesMatch(data["config"], remoteItems, itemRef)
	})
	return local
}

// resolveTypeRef maps a remote custom-type ID to the local custom-type URN it
// corresponds to: either the URN of the custom type matched in this import, or
// of one already managed locally (found by its import metadata's remote ID).
// An empty definitionID means the property has no custom-type ref at that slot,
// which resolves to the empty URN. ok is false only when a non-empty ID cannot
// be resolved — the property then stays unmatched (namer fallback).
func resolveTypeRef(scope importmatcher.Scope, definitionID string) (urn string, ok bool) {
	if definitionID == "" {
		return "", true
	}

	if scope.Importable != nil {
		if ct, found := scope.Importable.GetByID(types.CustomTypeResourceType, definitionID); found {
			if ct.MatchedWith != nil {
				return ct.MatchedWith.URN(), true
			}
			return "", false
		}
	}

	for _, ct := range scope.LocalGraph.ResourcesByType(types.CustomTypeResourceType) {
		if meta := ct.ImportMetadata(); meta != nil && meta.RemoteId == definitionID {
			return ct.URN(), true
		}
	}
	return "", false
}

// typeMatches compares a local property's type against the remote's. When the
// remote type is a custom type (typeRef non-empty), the local type must be the
// PropertyRef pointing at the same custom-type URN. Otherwise both are plain
// strings, compared as the sorted comma-joined form they normalize to.
func typeMatches(localType any, remoteType, typeRef string) bool {
	if typeRef != "" {
		ref, ok := localType.(resources.PropertyRef)
		return ok && ref.URN == typeRef
	}

	s, ok := localType.(string)
	return ok && s == remoteType
}

// itemTypesMatch compares array item types. When the remote item type is a
// custom type (itemRef non-empty), the local item_types must be a single
// PropertyRef pointing at the same custom-type URN. Otherwise both are string
// lists compared order-insensitively.
func itemTypesMatch(localConfig any, remoteItems []string, itemRef string) bool {
	config, _ := localConfig.(map[string]any)

	if itemRef != "" {
		items, _ := config["item_types"].([]any)
		if len(items) != 1 {
			return false
		}
		ref, ok := items[0].(resources.PropertyRef)
		return ok && ref.URN == itemRef
	}

	localItems, ok := stringItemTypes(config, "item_types")
	if !ok {
		return false
	}
	return itemTypesEqual(localItems, remoteItems)
}

// stringItemTypes extracts the item types under key as strings. ok is false
// when the entry exists but contains non-string elements (custom-type item
// references) — those are matched separately in itemTypesMatch, not here. An
// absent entry is ok with nil items.
func stringItemTypes(config map[string]any, key string) ([]string, bool) {
	if config == nil {
		return nil, true
	}
	raw, exists := config[key]
	if !exists {
		return nil, true
	}

	items, ok := raw.([]any)
	if !ok {
		return nil, false
	}

	result := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			return nil, false
		}
		result = append(result, s)
	}
	return result, true
}

// itemTypesEqual compares item types order-insensitively — the local side is
// sorted at spec load, the remote side is not guaranteed to be.
func itemTypesEqual(local, remote []string) bool {
	if len(local) != len(remote) {
		return false
	}

	var (
		l = slices.Clone(local)
		r = slices.Clone(remote)
	)
	slices.Sort(l)
	slices.Sort(r)
	return slices.Equal(l, r)
}
