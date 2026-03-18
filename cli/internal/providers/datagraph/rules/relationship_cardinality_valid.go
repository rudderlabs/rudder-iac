package rules

import (
	"fmt"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	modelHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateRelationshipCardinality = func(_ string, _ string, _ map[string]any, spec dgModel.DataGraphSpec, graph *resources.Graph) []rules.ValidationResult {
	// Build a map of model ID -> type from the spec for source model lookup
	modelTypes := make(map[string]string, len(spec.Models))
	for _, m := range spec.Models {
		modelTypes[m.ID] = m.Type
	}

	var results []rules.ValidationResult
	for i, model := range spec.Models {
		for j, rel := range model.Relationships {
			targetModelID := parseTargetModelID(rel.Target)
			if targetModelID == "" {
				continue
			}

			sourceType := model.Type
			targetType := resolveModelType(targetModelID, modelTypes, graph)
			if targetType == "" {
				// Target not found — handled by refs_valid rule
				continue
			}

			if err := checkCardinality(sourceType, targetType, rel.Cardinality); err != "" {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/models/%d/relationships/%d/cardinality", i, j),
					Message:   err,
				})
			}
		}
	}

	return results
}

// parseTargetModelID extracts the model ID from a target reference like "#data-graph-model:user"
func parseTargetModelID(target string) string {
	const prefix = "#data-graph-model:"
	if !strings.HasPrefix(target, prefix) {
		return ""
	}
	return strings.TrimPrefix(target, prefix)
}

// resolveModelType looks up a model's type first from the local spec, then from the graph
func resolveModelType(modelID string, localTypes map[string]string, graph *resources.Graph) string {
	if t, ok := localTypes[modelID]; ok {
		return t
	}

	urn := resources.URN(modelID, modelHandler.HandlerMetadata.ResourceType)
	res, exists := graph.GetResource(urn)
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
