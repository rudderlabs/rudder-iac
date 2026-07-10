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
	remote, ok := r.Data.(*catalog.Category)
	if !ok || remote.Name == "" {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, types.CategoryResourceType, func(data resources.ResourceData) bool {
		name, _ := data["name"].(string)
		return name == remote.Name
	})
	return local
}

func matchCustomType(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	remote, ok := r.Data.(*catalog.CustomType)
	if !ok || remote.Name == "" {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, types.CustomTypeResourceType, func(data resources.ResourceData) bool {
		name, _ := data["name"].(string)
		return name == remote.Name
	})
	return local
}

func matchTrackingPlan(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	remote, ok := r.Data.(*catalog.TrackingPlanWithIdentifiers)
	if !ok || remote.Name == "" {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, types.TrackingPlanResourceType, func(data resources.ResourceData) bool {
		name, _ := data["name"].(string)
		return name == remote.Name
	})
	return local
}

func matchEvent(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	remote, ok := r.Data.(*catalog.Event)
	if !ok || remote.EventType == "" {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, types.EventResourceType, func(data resources.ResourceData) bool {
		var (
			name, _      = data["name"].(string)
			eventType, _ = data["eventType"].(string)
		)
		return name == remote.Name && eventType == remote.EventType
	})
	return local
}

func matchProperty(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
	remote, ok := r.Data.(*catalog.Property)
	// DefinitionId links the remote property to a custom type; those are
	// imported normally via the namer, never smart-linked.
	if !ok || remote.Name == "" || remote.DefinitionId != "" {
		return nil
	}

	// Remote config uses camelCase keys; local config is snake_case.
	remoteItems, ok := stringItemTypes(remote.Config, "itemTypes")
	if !ok {
		return nil
	}

	local, _ := importmatcher.ByData(scope.LocalGraph, types.PropertyResourceType, func(data resources.ResourceData) bool {
		name, _ := data["name"].(string)
		if name != remote.Name {
			return false
		}

		// A non-string local type is a resources.PropertyRef — a custom-type
		// reference — which never smart-links. Multi-types compare as the
		// sorted comma-joined string both sides already normalize to.
		localType, ok := data["type"].(string)
		if !ok || localType != remote.Type {
			return false
		}

		localConfig, _ := data["config"].(map[string]any)
		localItems, ok := stringItemTypes(localConfig, "item_types")
		if !ok {
			return false
		}
		return itemTypesEqual(localItems, remoteItems)
	})
	return local
}

// stringItemTypes extracts the item types under key as strings. ok is false
// when the entry exists but contains non-string elements (custom-type item
// references), which never smart-link. An absent entry is ok with nil items.
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
