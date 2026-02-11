package trackingplan

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateTrackingPlanSemantic = func(_ string, _ string, _ map[string]any, spec localcatalog.TrackingPlan, graph *resources.Graph) []rules.ValidationResult {
	results := funcs.ValidateReferences(spec, graph)

	// Variant discriminator validation and name uniqueness checks
	results = append(results, validateTrackingPlanVariants(spec, graph)...)
	results = append(results, validateTrackingPlanNameUniqueness(spec, graph)...)

	return results
}

func validateTrackingPlanVariants(spec localcatalog.TrackingPlan, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult
	for i, rule := range spec.Rules {
		if len(rule.Variants) == 0 {
			continue
		}
		ownRefs := make([]string, 0, len(rule.Properties))
		for _, prop := range rule.Properties {
			ownRefs = append(ownRefs, prop.Ref)
		}
		results = append(results, funcs.ValidateVariantDiscriminators(
			rule.Variants, ownRefs, fmt.Sprintf("/rules/%d", i), graph,
		)...)
	}

	return results
}

func validateTrackingPlanNameUniqueness(spec localcatalog.TrackingPlan, graph *resources.Graph) []rules.ValidationResult {
	countMap := make(map[string]int)
	for _, resource := range graph.ResourcesByType(types.TrackingPlanResourceType) {
		data := resource.Data()
		name, _ := data["name"].(string)
		countMap[name]++
	}

	if countMap[spec.Name] > 1 {
		return []rules.ValidationResult{{
			Reference: "/display_name",
			Message:   fmt.Sprintf("tracking plan with name '%s' is not unique across the project", spec.Name),
		}}
	}

	return nil
}

func NewTrackingPlanSemanticValidRule() rules.Rule {
	return prules.NewSemanticTypedRule(
		"datacatalog/tracking-plans/semantic-valid",
		rules.Error,
		"tracking plan references must resolve to existing resources",
		rules.Examples{},
		[]string{localcatalog.KindTrackingPlans},
		validateTrackingPlanSemantic,
	)
}
