package rules

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	modelHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateRelationshipRefs = func(_ string, _ string, _ map[string]any, spec dgModel.DataGraphSpec, graph *resources.Graph) []rules.ValidationResult {
	// Build set of model IDs defined in this spec for local resolution
	localModelIDs := make(map[string]bool, len(spec.Models))
	for _, m := range spec.Models {
		localModelIDs[m.ID] = true
	}

	var results []rules.ValidationResult
	for i, model := range spec.Models {
		for j, rel := range model.Relationships {
			targetModelID := parseTargetModelID(rel.Target)
			if targetModelID == "" {
				// Invalid format — caught by syntactic rule's required tag
				continue
			}

			// Check if target exists locally or in the graph
			if localModelIDs[targetModelID] {
				continue
			}
			urn := resources.URN(targetModelID, modelHandler.HandlerMetadata.ResourceType)
			if _, exists := graph.GetResource(urn); exists {
				continue
			}

			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/models/%d/relationships/%d/target", i, j),
				Message:   fmt.Sprintf("target model %q does not exist", targetModelID),
			})
		}
	}

	return results
}

func NewRelationshipRefsValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datagraph/data-graph/relationship-refs-valid",
		rules.Error,
		"relationship target references must resolve to existing models",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.LegacyVersionPatterns("data-graph"),
			validateRelationshipRefs,
		),
		prules.NewSemanticPatternValidator(
			prules.V1VersionPatterns("data-graph"),
			validateRelationshipRefs,
		),
	)
}
