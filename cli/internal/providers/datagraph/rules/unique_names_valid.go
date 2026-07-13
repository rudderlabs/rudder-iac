package rules

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	dgHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	modelHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	relHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateUniqueNames = func(_ string, _ string, _ map[string]any,
	spec dgModel.DataGraphSpec, graph *resources.Graph) []rules.ValidationResult {

	dataGraphURN := resources.URN(spec.ID, dgHandler.HandlerMetadata.ResourceType)

	var (
		modelDisplayNames = buildDisplayNameMap(graph, modelHandler.HandlerMetadata.ResourceType, dataGraphURN)
		relDisplayNames   = buildDisplayNameMap(graph, relHandler.HandlerMetadata.ResourceType, dataGraphURN)
		results           []rules.ValidationResult
	)

	for i, model := range spec.Models {
		modelURN := resources.URN(model.ID, modelHandler.HandlerMetadata.ResourceType)
		if hasDuplicate(modelDisplayNames, model.DisplayName, modelURN) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/models/%d/display_name", i),
				Message:   fmt.Sprintf("duplicate model display name %q", model.DisplayName),
			})
		}

		for j, rel := range model.Relationships {
			relURN := resources.URN(rel.ID, relHandler.HandlerMetadata.ResourceType)
			if hasDuplicate(relDisplayNames, rel.DisplayName, relURN) {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/models/%d/relationships/%d/display_name", i, j),
					Message:   fmt.Sprintf("duplicate relationship display name %q", rel.DisplayName),
				})
			}
		}
	}

	return results
}

// buildDisplayNameMap collects display_name → []URN for all resources of the given type
// that belong to the specified data graph.
func buildDisplayNameMap(graph *resources.Graph, resourceType, dataGraphURN string) map[string][]string {
	nameMap := make(map[string][]string)

	for _, res := range graph.ResourcesByType(resourceType) {
		displayName := extractDisplayName(res, resourceType)
		if displayName == "" {
			continue
		}
		if !belongsToDataGraph(res, resourceType, dataGraphURN) {
			continue
		}
		nameMap[displayName] = append(nameMap[displayName], res.URN())
	}

	return nameMap
}

func extractDisplayName(res *resources.Resource, resourceType string) string {
	switch resourceType {
	case modelHandler.HandlerMetadata.ResourceType:
		if m, ok := res.RawData().(*dgModel.ModelResource); ok {
			return m.DisplayName
		}
	case relHandler.HandlerMetadata.ResourceType:
		if r, ok := res.RawData().(*dgModel.RelationshipResource); ok {
			return r.DisplayName
		}
	}
	return ""
}

func belongsToDataGraph(res *resources.Resource, resourceType, dataGraphURN string) bool {
	switch resourceType {
	case modelHandler.HandlerMetadata.ResourceType:
		if m, ok := res.RawData().(*dgModel.ModelResource); ok {
			return m.DataGraphRef != nil && m.DataGraphRef.URN == dataGraphURN
		}
	case relHandler.HandlerMetadata.ResourceType:
		if r, ok := res.RawData().(*dgModel.RelationshipResource); ok {
			return r.DataGraphRef != nil && r.DataGraphRef.URN == dataGraphURN
		}
	}
	return false
}

// hasDuplicate returns true if the name maps to any URN other than the resource's own.
func hasDuplicate(nameMap map[string][]string, displayName, ownURN string) bool {
	for _, urn := range nameMap[displayName] {
		if urn != ownURN {
			return true
		}
	}
	return false
}

func NewUniqueNamesValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datagraph/data-graph/unique-names-valid",
		rules.Error,
		"model and relationship names must be unique within a data graph",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.V1VersionPatterns("data-graph"),
			validateUniqueNames,
		),
	)
}
