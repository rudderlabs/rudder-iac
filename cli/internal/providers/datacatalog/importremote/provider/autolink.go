package provider

import (
	"reflect"
	"sort"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// findLocalCategoryMatch searches the local graph for a category with the given name.
// Returns the local ID and true if found, or empty string and false if not.
func findLocalCategoryMatch(graph *resources.Graph, remoteName string) (string, bool) {
	for _, r := range graph.ResourcesByType(types.CategoryResourceType) {
		data := r.Data()
		if name, ok := data["name"].(string); ok && name == remoteName {
			return r.ID(), true
		}
	}
	return "", false
}

// findLocalEventMatch searches the local graph for an event matching name+eventType.
// Non-track events have empty names and match on eventType alone.
func findLocalEventMatch(graph *resources.Graph, remoteName string, remoteEventType string) (string, bool) {
	for _, r := range graph.ResourcesByType(types.EventResourceType) {
		data := r.Data()
		localName, _ := data["name"].(string)
		localEventType, _ := data["eventType"].(string)

		if localName == remoteName && localEventType == remoteEventType {
			return r.ID(), true
		}
	}
	return "", false
}

// findLocalPropertyMatch searches the local graph for a property matching name+type+itemTypes.
// Properties with custom type references (type is not a string) are skipped.
func findLocalPropertyMatch(graph *resources.Graph, remoteName string, remoteType string, remoteConfig map[string]interface{}) (string, bool) {
	for _, r := range graph.ResourcesByType(types.PropertyResourceType) {
		data := r.Data()

		localName, _ := data["name"].(string)
		if localName != remoteName {
			continue
		}

		// Skip if local type is not a simple string (custom type ref)
		localType, ok := data["type"].(string)
		if !ok {
			continue
		}
		if localType != remoteType {
			continue
		}

		localConfig, _ := data["config"].(map[string]interface{})
		if !itemTypesMatch(localConfig, remoteConfig) {
			continue
		}

		return r.ID(), true
	}
	return "", false
}

// findLocalTrackingPlanMatch searches the local graph for a tracking plan with the given name.
func findLocalTrackingPlanMatch(graph *resources.Graph, remoteName string) (string, bool) {
	for _, r := range graph.ResourcesByType(types.TrackingPlanResourceType) {
		data := r.Data()
		if name, ok := data["name"].(string); ok && name == remoteName {
			return r.ID(), true
		}
	}
	return "", false
}

// itemTypesMatch compares the item_types field from two config maps.
func itemTypesMatch(localConfig, remoteConfig map[string]interface{}) bool {
	localItems := extractItemTypes(localConfig)
	remoteItems := extractItemTypes(remoteConfig)
	sort.Strings(localItems)
	sort.Strings(remoteItems)
	return reflect.DeepEqual(localItems, remoteItems)
}

// extractItemTypes pulls string item_types from a config map.
func extractItemTypes(config map[string]interface{}) []string {
	if config == nil {
		return nil
	}
	raw, ok := config["item_types"]
	if !ok {
		return nil
	}
	items, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
