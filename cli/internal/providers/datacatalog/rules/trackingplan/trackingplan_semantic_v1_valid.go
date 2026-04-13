package trackingplan

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules/variant"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateTrackingPlanSemanticV1 = func(_ string, _ string, _ map[string]any, spec localcatalog.TrackingPlanV1, graph *resources.Graph) []rules.ValidationResult {
	results := funcs.ValidateReferences(spec, graph)
	results = append(results, validateTrackingPlanVariantsV1(spec, graph)...)
	results = append(results, validatePropertyNestingV1(spec, graph)...)
	results = append(results, validateTrackingPlanNameUniquenessV1(spec, graph)...)
	results = append(results, validateDuplicateEventsV1(spec)...)
	results = append(results, validateDuplicatePropertiesV1(spec)...)

	return results
}

func validateDuplicateEventsV1(spec localcatalog.TrackingPlanV1) []rules.ValidationResult {
	counts := make(map[string]int)
	for _, rule := range spec.Rules {
		counts[rule.Event]++
	}

	var results []rules.ValidationResult
	for i, rule := range spec.Rules {
		if counts[rule.Event] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/rules/%d/event", i),
				Message:   "duplicate event reference in tracking plan rules",
			})
		}
	}
	return results
}

func validateDuplicatePropertiesV1(spec localcatalog.TrackingPlanV1) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, rule := range spec.Rules {
		ruleRef := fmt.Sprintf("/rules/%d", i)

		results = append(results, checkDuplicateSiblingPropsV1(rule.Properties, ruleRef+"/properties")...)

		for v, variant := range rule.Variants {
			for c, vCase := range variant.Cases {
				caseRef := fmt.Sprintf("%s/variants/%d/cases/%d/properties", ruleRef, v, c)
				results = append(results, checkDuplicateVariantPropRefsV1(vCase.Properties, caseRef)...)
			}
			defaultRef := fmt.Sprintf("%s/variants/%d/default/properties", ruleRef, v)
			results = append(results, checkDuplicateVariantPropRefsV1(variant.Default.Properties, defaultRef)...)
		}
	}

	return results
}

func checkDuplicateSiblingPropsV1(props []*localcatalog.TPRulePropertyV1, parentRef string) []rules.ValidationResult {
	counts := make(map[string]int)
	for _, prop := range props {
		counts[prop.Property]++
	}

	var results []rules.ValidationResult
	for i, prop := range props {
		if counts[prop.Property] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("%s/%d/property", parentRef, i),
				Message:   "duplicate property reference in tracking plan event rule",
			})
		}
		if len(prop.Properties) > 0 {
			nestedRef := fmt.Sprintf("%s/%d/properties", parentRef, i)
			results = append(results, checkDuplicateSiblingPropsV1(prop.Properties, nestedRef)...)
		}
	}
	return results
}

func checkDuplicateVariantPropRefsV1(props []localcatalog.PropertyReferenceV1, parentRef string) []rules.ValidationResult {
	counts := make(map[string]int)
	for _, prop := range props {
		counts[prop.Property]++
	}

	var results []rules.ValidationResult
	for i, prop := range props {
		if counts[prop.Property] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("%s/%d/property", parentRef, i),
				Message:   "duplicate property reference in tracking plan event rule",
			})
		}
	}
	return results
}

func validateTrackingPlanVariantsV1(spec localcatalog.TrackingPlanV1, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult
	for i, rule := range spec.Rules {
		if len(rule.Variants) == 0 {
			continue
		}
		ownRefs := make([]string, 0, len(rule.Properties))
		for _, prop := range rule.Properties {
			ownRefs = append(ownRefs, prop.Property)
		}
		results = append(results, variant.ValidateVariantDiscriminatorsV1(
			rule.Variants, ownRefs, fmt.Sprintf("/rules/%d", i), graph,
		)...)
	}
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
			Reference: baseRef + "/additional_properties",
			Message:   fmt.Sprintf("additional_properties is not allowed for property '%s'", localID),
		})
	}

	return results
}
