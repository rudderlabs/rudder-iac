package rules

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	relationshipHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateRelationshipCardinality = func(_ string, _ string, _ map[string]any, spec dgModel.DataGraphSpec, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult
	for i, model := range spec.Models {
		for j, rel := range model.Relationships {
			relRes := lookupRelationship(rel.ID, graph)
			if relRes == nil || relRes.TargetModelRef == nil {
				continue
			}

			targetType := resolveModelType(relRes.TargetModelRef.URN, graph)
			if targetType == "" {
				// Target not found — handled by refs_valid rule
				continue
			}

			if err := checkCardinality(model.Type, targetType, rel.Cardinality); err != "" {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/models/%d/relationships/%d/cardinality", i, j),
					Message:   err,
				})
			}
		}
	}

	return results
}

// lookupRelationship finds a relationship resource in the graph by its spec ID
func lookupRelationship(id string, graph *resources.Graph) *dgModel.RelationshipResource {
	urn := resources.URN(id, relationshipHandler.HandlerMetadata.ResourceType)
	res, exists := graph.GetResource(urn)
	if !exists {
		return nil
	}
	rel, ok := res.RawData().(*dgModel.RelationshipResource)
	if !ok {
		return nil
	}
	return rel
}

// resolveModelType looks up a model's type from the graph by URN
func resolveModelType(modelURN string, graph *resources.Graph) string {
	res, exists := graph.GetResource(modelURN)
	if !exists {
		return ""
	}

	modelResource, ok := res.RawData().(*dgModel.ModelResource)
	if !ok {
		return ""
	}
	return modelResource.Type
}

// checkCardinality validates cardinality constraints based on source and target model types.
// Returns an error message if the cardinality is invalid, or empty string if valid.
func checkCardinality(sourceType, targetType, cardinality string) string {
	switch {
	case sourceType == "event" && targetType == "event":
		return "event models cannot be connected to other event models"
	case sourceType == "event" && targetType == "entity":
		if cardinality != "many-to-one" {
			return fmt.Sprintf("relationships from event models must have cardinality 'many-to-one', got %q", cardinality)
		}
	case sourceType == "entity" && targetType == "event":
		if cardinality != "one-to-many" {
			return fmt.Sprintf("relationships from entity models to event models must have cardinality 'one-to-many', got %q", cardinality)
		}
	}
	return ""
}

func NewRelationshipCardinalityValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datagraph/data-graph/relationship-cardinality-valid",
		rules.Error,
		"relationship cardinality must be valid for the source and target model types",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.V1VersionPatterns("data-graph"),
			validateRelationshipCardinality,
		),
	)
}
