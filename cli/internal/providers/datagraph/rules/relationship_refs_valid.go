package rules

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateRelationshipRefs = func(_ string, _ string, _ map[string]any, spec dgModel.DataGraphSpec, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult
	for i, model := range spec.Models {
		for j, rel := range model.Relationships {
			relRes := lookupRelationship(rel.ID, graph)
			if relRes == nil || relRes.TargetModelRef == nil {
				continue
			}

			if _, exists := graph.GetResource(relRes.TargetModelRef.URN); exists {
				continue
			}

			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/models/%d/relationships/%d/target", i, j),
				Message:   fmt.Sprintf("target model %q does not exist", relRes.TargetModelRef.URN),
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
			prules.V1VersionPatterns("data-graph"),
			validateRelationshipRefs,
		),
	)
}
