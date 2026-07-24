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

	in, ok := preparePropertyMatch(scope, remote)
	if !ok {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, types.PropertyResourceType, func(data resources.ResourceData) bool {
		return propertyDataMatches(data, in)
	})
	return local
}

// propertyMatchInputs is the remote property distilled into what a local
// candidate must match: its name, its type (with any custom-type ref already
// resolved to a local URN), and its item types (likewise). The *URN fields hold
// resolved local custom-type URNs, not remote IDs.
type propertyMatchInputs struct {
	name        string
	remoteType  string
	typeRefURN  string
	remoteItems []string
	itemRefURN  string
}

// preparePropertyMatch resolves the remote property's references up front. ok
// is false when the property cannot match at all: it has no name, or one of its
// custom-type refs resolves to nothing (the custom type is neither imported now
// nor already managed), or its item types are unreadable.
func preparePropertyMatch(scope importmatcher.Scope, remote *catalog.Property) (propertyMatchInputs, bool) {
	if remote.Name == "" {
		return propertyMatchInputs{}, false
	}

	// A remote property whose type (or array item type) is a custom type
	// references it by DefinitionId/ItemDefinitionId; resolve each to the local
	// custom-type URN so it can be compared against the local PropertyRef.
	typeRefURN, ok := resolveTypeRef(scope, remote.DefinitionId)
	if !ok {
		return propertyMatchInputs{}, false
	}
	itemRefURN, ok := resolveTypeRef(scope, remote.ItemDefinitionId)
	if !ok {
		return propertyMatchInputs{}, false
	}

	// Remote config uses camelCase keys; local config is snake_case.
	remoteItems, ok := stringItemTypes(remote.Config, "itemTypes")
	if !ok {
		return propertyMatchInputs{}, false
	}

	return propertyMatchInputs{
		name:        remote.Name,
		remoteType:  remote.Type,
		typeRefURN:  typeRefURN,
		remoteItems: remoteItems,
		itemRefURN:  itemRefURN,
	}, true
}

// propertyDataMatches reports whether a local property (name, type, itemTypes)
// equals the prepared remote inputs. Order matters only for short-circuiting.
func propertyDataMatches(data resources.ResourceData, in propertyMatchInputs) bool {
	return data["name"].(string) == in.name &&
		typeMatches(data["type"], in.remoteType, in.typeRefURN) &&
		itemTypesMatch(data["config"], in.remoteItems, in.itemRefURN)
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
// remote type is a custom type (typeRefURN non-empty), the local type must be
// the PropertyRef pointing at the same custom-type URN. Otherwise both are
// plain strings, compared as the sorted comma-joined form they normalize to.
func typeMatches(localType any, remoteType, typeRefURN string) bool {
	if typeRefURN != "" {
		ref, ok := localType.(resources.PropertyRef)
		return ok && ref.URN == typeRefURN
	}

	s, ok := localType.(string)
	return ok && s == remoteType
}

// itemTypesMatch compares array item types. When the remote item type is a
// custom type (itemRefURN non-empty), the local item_types must be a single
// PropertyRef pointing at the same custom-type URN. Otherwise both are string
// lists compared order-insensitively.
func itemTypesMatch(localConfig any, remoteItems []string, itemRefURN string) bool {
	config, _ := localConfig.(map[string]any)

	if itemRefURN != "" {
		items, _ := config["item_types"].([]any)
		if len(items) != 1 {
			return false
		}
		ref, ok := items[0].(resources.PropertyRef)
		return ok && ref.URN == itemRefURN
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
