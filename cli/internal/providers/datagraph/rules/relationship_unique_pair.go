package rules

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
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
	// Build set of relationship IDs from this spec so we can exclude them
	// when seeding from the graph (the graph already contains them).
	specRelIDs := make(map[string]bool)
	for _, model := range spec.Models {
		for _, rel := range model.Relationships {
			specRelIDs[rel.ID] = true
		}
	}

	seen := make(map[modelPair]bool)

	// Seed with existing pairs from other specs in the graph.
	for _, res := range graph.ResourcesByType(relationshipHandler.HandlerMetadata.ResourceType) {
		if specRelIDs[res.ID()] {
			continue
		}
		rel, ok := res.RawData().(*dgModel.RelationshipResource)
		if !ok || rel.SourceModelRef == nil || rel.TargetModelRef == nil {
			continue
		}
		seen[modelPair{rel.SourceModelRef.URN, rel.TargetModelRef.URN}] = true
	}

	var results []rules.ValidationResult

	for i, model := range spec.Models {
		for j, rel := range model.Relationships {
			relRes := lookupRelationship(rel.ID, graph)
			if relRes == nil || relRes.SourceModelRef == nil || relRes.TargetModelRef == nil {
				continue
			}

			p := modelPair{relRes.SourceModelRef.URN, relRes.TargetModelRef.URN}
			if seen[p] {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/models/%d/relationships/%d", i, j),
					Message:   fmt.Sprintf("a relationship from model %q to model %q already exists; at most one relationship is allowed per source-target pair", model.ID, relRes.TargetModelRef.URN),
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
