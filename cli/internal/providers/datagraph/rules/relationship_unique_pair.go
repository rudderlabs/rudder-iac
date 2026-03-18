package rules

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	modelHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	relationshipHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type modelPair struct {
	sourceURN string
	targetURN string
}

var validateRelationshipUniquePair = func(_ string, _ string, _ map[string]any, spec dgModel.DataGraphSpec, graph *resources.Graph) []rules.ValidationResult {
	seen := make(map[modelPair]bool)

	// Collect existing (source, target) pairs from the resource graph.
	for _, res := range graph.ResourcesByType(relationshipHandler.HandlerMetadata.ResourceType) {
		rel, ok := res.RawData().(*dgModel.RelationshipResource)
		if !ok || rel.SourceModelRef == nil || rel.TargetModelRef == nil {
			continue
		}
		seen[modelPair{rel.SourceModelRef.URN, rel.TargetModelRef.URN}] = true
	}

	var results []rules.ValidationResult

	for i, model := range spec.Models {
		sourceURN := resources.URN(model.ID, modelHandler.HandlerMetadata.ResourceType)

		for j, rel := range model.Relationships {
			targetID := parseTargetModelID(rel.Target)
			if targetID == "" {
				continue
			}
			targetURN := resources.URN(targetID, modelHandler.HandlerMetadata.ResourceType)

			p := modelPair{sourceURN, targetURN}
			if seen[p] {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/models/%d/relationships/%d", i, j),
					Message:   fmt.Sprintf("a relationship from model %q to model %q already exists; at most one relationship is allowed per source-target pair", model.ID, targetID),
				})
			} else {
				seen[p] = true
			}
		}
	}

	return results
}

func NewRelationshipUniquePairRule() rules.Rule {
	return prules.NewTypedRule(
		"datagraph/data-graph/relationship-unique-pair",
		rules.Error,
		"at most one relationship allowed from a source model to a target model",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.V1VersionPatterns("data-graph"),
			validateRelationshipUniquePair,
		),
	)
}
