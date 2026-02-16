package trackingplan

import (
	"fmt"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateTrackingPlanSemantic = func(_ string, _ string, _ map[string]any, spec localcatalog.TrackingPlan, graph *resources.Graph) []rules.ValidationResult {
	results := funcs.ValidateReferences(spec, graph)

	// Variant discriminator validation, property nesting, and name uniqueness checks
	results = append(results, validateTrackingPlanVariants(spec, graph)...)
	results = append(results, validatePropertyNesting(spec, graph)...)
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

// validatePropertyNesting checks that each rule property's referenced type supports
// nesting before allowing nested Properties or AdditionalProperties.
func validatePropertyNesting(spec localcatalog.TrackingPlan, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, rule := range spec.Rules {
		for j, prop := range rule.Properties {
			ref := fmt.Sprintf("/rules/%d/properties/%d", i, j)
			results = append(results, validatePropertyNestingAllowed(prop, ref, graph)...)
		}
	}

	return results
}

func validatePropertyNestingAllowed(prop *localcatalog.TPRuleProperty, ref string, graph *resources.Graph) []rules.ValidationResult {
	hasNested := len(prop.Properties) > 0
	hasAdditionalProps := prop.AdditionalProperties != nil

	if !hasNested && !hasAdditionalProps {
		return nil
	}

	// Resolve the property ref to get its type from the graph
	resourceType, localID, err := funcs.ParseURNRef(prop.Ref)
	if err != nil {
		// Ref parsing errors are already reported by ValidateReferences
		return nil
	}

	urn := resources.URN(localID, resourceType)
	resource, exists := graph.GetResource(urn)
	if !exists {
		// Missing refs are already reported by ValidateReferences
		return nil
	}

	data := resource.Data()
	propertyType, _ := data["type"].(string)

	config, _ := data["config"].(map[string]any)

	if nestingAllowed(propertyType, config) {
		// Type supports nesting — recurse into nested properties
		var results []rules.ValidationResult
		for k, nested := range prop.Properties {
			nestedRef := fmt.Sprintf("%s/properties/%d", ref, k)
			results = append(results, validatePropertyNestingAllowed(nested, nestedRef, graph)...)
		}
		return results
	}

	// Type does NOT support nesting
	var results []rules.ValidationResult
	if hasNested {
		results = append(results, rules.ValidationResult{
			Reference: ref,
			Message:   fmt.Sprintf("nested properties are not allowed for property '%s'", localID),
		})
	}
	if hasAdditionalProps {
		results = append(results, rules.ValidationResult{
			Reference: ref + "/additionalProperties",
			Message:   fmt.Sprintf("additional_properties is not allowed for property '%s'", localID),
		})
	}

	return results
}

// nestingAllowed determines if a property type supports nested properties.
// Object types always allow nesting. Array types allow nesting if their
// item types include "object".
func nestingAllowed(propertyType string, config map[string]any) bool {
	hasObject := strings.Contains(propertyType, "object")
	hasArray := strings.Contains(propertyType, "array")

	// Object+array together cannot support nesting
	if hasObject && hasArray {
		return false
	}

	if hasObject {
		return true
	}

	if !hasArray {
		return false
	}

	// Array type — check if item types include "object"
	if config == nil {
		return true
	}

	itemTypes, ok := config["item_types"].([]any)
	if !ok {
		return true
	}

	for _, item := range itemTypes {
		if s, ok := item.(string); ok && s == "object" {
			return true
		}
	}

	return false
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
