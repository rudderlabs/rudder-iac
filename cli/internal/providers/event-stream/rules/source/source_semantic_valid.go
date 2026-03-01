package source

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateSourceSemantic = func(
	_ string,
	_ string,
	_ map[string]any,
	spec esSource.SourceSpec,
	graph *resources.Graph,
) []rules.ValidationResult {
	var results []rules.ValidationResult

	results = append(results, validateTrackingPlanExists(spec, graph)...)
	results = append(results, validateSourceNameUniqueness(spec, graph)...)

	return results
}

// validateTrackingPlanExists checks that the referenced tracking plan exists in the graph.
// Ref format is already validated by syntactic rules, so we only need to verify existence.
func validateTrackingPlanExists(spec esSource.SourceSpec, graph *resources.Graph) []rules.ValidationResult {
	if spec.Governance == nil || spec.Governance.TrackingPlan == nil {
		return nil
	}

	matches := localcatalog.TrackingPlanRegex.FindStringSubmatch(spec.Governance.TrackingPlan.Ref)
	if len(matches) != 2 {
		return nil
	}

	trackingPlanID := matches[1]
	urn := resources.URN(trackingPlanID, types.TrackingPlanResourceType)

	if _, ok := graph.GetResource(urn); !ok {
		return []rules.ValidationResult{{
			Reference: "/governance/validations/tracking_plan",
			Message:   fmt.Sprintf("tracking plan '%s' not found in the project", trackingPlanID),
		}}
	}

	return nil
}

func validateSourceNameUniqueness(spec esSource.SourceSpec, graph *resources.Graph) []rules.ValidationResult {
	countMap := make(map[string]int)
	for _, resource := range graph.ResourcesByType(esSource.ResourceType) {
		data := resource.Data()
		name, _ := data["name"].(string)
		countMap[name]++
	}

	if countMap[spec.Name] > 1 {
		return []rules.ValidationResult{{
			Reference: "/name",
			Message:   fmt.Sprintf("duplicate name '%s' within kind 'event-stream-source'", spec.Name),
		}}
	}

	return nil
}

func NewSourceSemanticValidRule() rules.Rule {
	return prules.NewTypedRule(
		"event-stream/source/semantic-valid",
		rules.Error,
		"event stream source references must resolve to existing resources",
		rules.Examples{},
		prules.NewSemanticVariant(
			prules.LegacyVersionPatterns(esSource.ResourceKind),
			validateSourceSemantic,
		),
	)
}
