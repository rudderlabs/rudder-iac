package destination

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const SemanticValidRuleID = "destination/semantic-valid"

type semanticValidRule struct{}

// NewSemanticValidRule validates cross-resource concerns: the referenced
// transformation exists in the project and display_name is unique across all
// destinations. Envelope/config shape errors are owned by the syntactic rule.
func NewSemanticValidRule() vrules.Rule {
	return &semanticValidRule{}
}

func (r *semanticValidRule) ID() string {
	return SemanticValidRuleID
}

func (r *semanticValidRule) Severity() vrules.Severity {
	return vrules.Error
}

func (r *semanticValidRule) Description() string {
	return "destination transformation reference must resolve to a project transformation and display_name must be unique across destinations"
}

func (r *semanticValidRule) AppliesTo() []vrules.MatchPattern {
	return prules.V1VersionPatterns(DestinationSpecKind)
}

func (r *semanticValidRule) Examples() vrules.Examples {
	return vrules.Examples{}
}

func (r *semanticValidRule) Validate(ctx *vrules.ValidationContext) []vrules.ValidationResult {
	if !ctx.HasGraph() {
		return nil
	}

	// Decode/format failures are reported by the syntactic rule; skip quietly.
	spec, _, err := decodeDestinationSpec(ctx.Spec)
	if err != nil {
		return nil
	}

	var results []vrules.ValidationResult

	if matches := transformationRefRegex.FindStringSubmatch(spec.Transformation); len(matches) == 2 {
		transformationID := matches[1]
		urn := resources.URN(transformationID, ttypes.TransformationResourceType)
		if _, exists := ctx.Graph.GetResource(urn); !exists {
			results = append(results, vrules.ValidationResult{
				Reference: "/transformation",
				Message:   fmt.Sprintf("transformation '%s' not found in the project", transformationID),
			})
		}
	}

	if spec.DisplayName != "" && countDisplayName(ctx.Graph, spec.DisplayName) > 1 {
		results = append(results, vrules.ValidationResult{
			Reference: "/display_name",
			Message:   fmt.Sprintf("duplicate display_name '%s' within kind '%s'", spec.DisplayName, DestinationSpecKind),
		})
	}

	return prefixSpecReferences(results)
}

func countDisplayName(graph *resources.Graph, displayName string) int {
	count := 0
	for _, res := range graph.ResourcesByType(DestinationResourceType) {
		destination, ok := res.RawData().(*DestinationResource)
		if !ok {
			continue
		}
		if destination.DisplayName == displayName {
			count++
		}
	}
	return count
}
