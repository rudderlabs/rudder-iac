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
	// Index all graph relationships by their (source, target) model pair.
	graphPairs := make(map[modelPair][]string)
	for _, res := range graph.ResourcesByType(relationshipHandler.HandlerMetadata.ResourceType) {
		rel, ok := res.RawData().(*dgModel.RelationshipResource)
		if !ok || rel.SourceModelRef == nil || rel.TargetModelRef == nil {
			continue
		}
		p := modelPair{rel.SourceModelRef.URN, rel.TargetModelRef.URN}
		graphPairs[p] = append(graphPairs[p], res.URN())
	}

	var results []rules.ValidationResult

	for i, model := range spec.Models {
		for j, rel := range model.Relationships {
			relURN := resources.URN(rel.ID, relationshipHandler.HandlerMetadata.ResourceType)
			res, exists := graph.GetResource(relURN)
			if !exists {
				continue
			}
			relRes, ok := res.RawData().(*dgModel.RelationshipResource)
			if !ok || relRes.SourceModelRef == nil || relRes.TargetModelRef == nil {
				continue
			}

			p := modelPair{relRes.SourceModelRef.URN, relRes.TargetModelRef.URN}
			for _, urn := range graphPairs[p] {
				if urn != relURN {
					results = append(results, rules.ValidationResult{
						Reference: fmt.Sprintf("/models/%d/relationships/%d", i, j),
						Message:   fmt.Sprintf("a relationship from model %q to model %q already exists; at most one relationship is allowed per source-target pair", model.ID, relRes.TargetModelRef.URN),
					})
					break
				}
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
