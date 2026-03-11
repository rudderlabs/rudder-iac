package trackingplan

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateTrackingPlanSemanticV1 = func(_ string, _ string, _ map[string]any, spec localcatalog.TrackingPlanV1, graph *resources.Graph) []rules.ValidationResult {
	results := funcs.ValidateReferences(spec, graph)
	results = append(results, validatePropertyNestingV1(spec, graph)...)
	results = append(results, validateTrackingPlanNameUniquenessV1(spec, graph)...)

	return results
}

func validateTrackingPlanNameUniquenessV1(spec localcatalog.TrackingPlanV1, graph *resources.Graph) []rules.ValidationResult {
	countMap := trackingPlanDisplayNameCountMap(graph)
	if countMap[spec.Name] > 1 {
		return []rules.ValidationResult{{
			Reference: "/display_name",
			Message:   fmt.Sprintf("duplicate display_name '%s' within kind 'tracking-plan'", spec.Name),
		}}
	}

	return nil
}

func validatePropertyNestingV1(spec localcatalog.TrackingPlanV1, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, rule := range spec.Rules {
		for j, prop := range rule.Properties {
			ref := fmt.Sprintf("/rules/%d/properties/%d", i, j)
			results = append(results, validatePropertyNestingAllowedV1(prop, ref, graph)...)
		}
	}

	return results
}

// validatePropertyNestingAllowed validates that property nesting is allowed for the property
// which internally can have nested properties, so it recursively checks on each
// nested property as well
func validatePropertyNestingAllowedV1(
	property *localcatalog.TPRulePropertyV1,
	baseRef string,
	graph *resources.Graph) []rules.ValidationResult {

	var (
		hasNested          = len(property.Properties) > 0
		hasAdditionalProps = property.AdditionalProperties != nil
	)

	if !hasNested && !hasAdditionalProps {
		return nil
	}

	// Resolve the property ref to get its type from the graph
	resourceType, localID, err := funcs.ParseURNRef(property.Property)
	if err != nil {
		// Ref parsing errors are already reported by
		// ValidateReferences as if we are unable to find them
		// here then simply return early.
		return nil
	}

	resource, exists := graph.GetResource(
		resources.URN(localID, resourceType),
	)
	if !exists {
		// Missing refs are already reported by ValidateReferences
		// so return early here
		return nil
	}

	var (
		data            = resource.Data()
		propertyType, _ = data["type"].(string)
		config, _       = data["config"].(map[string]any)
	)

	if nestingAllowed(propertyType, config) {
		// Type supports nesting — recurse into nested properties
		var results []rules.ValidationResult
		for k, nested := range property.Properties {
			nestedRef := fmt.Sprintf("%s/properties/%d", baseRef, k)
			results = append(results, validatePropertyNestingAllowedV1(
				nested,
				nestedRef,
				graph,
			)...)
		}
		return results
	}

	// Type does NOT support nesting at all
	// and if we reached this point, we should simply start
	// reporting errors
	var results []rules.ValidationResult
	if hasNested {
		results = append(results, rules.ValidationResult{
			Reference: baseRef,
			Message:   fmt.Sprintf("nested properties are not allowed for property '%s'", localID),
		})
	}
	if hasAdditionalProps {
		results = append(results, rules.ValidationResult{
			Reference: baseRef + "/additionalProperties",
			Message:   fmt.Sprintf("additional_properties is not allowed for property '%s'", localID),
		})
	}

	return results
}
