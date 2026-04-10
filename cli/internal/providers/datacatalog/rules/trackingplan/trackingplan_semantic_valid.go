package trackingplan

import (
	"fmt"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules/variant"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateTrackingPlanSemantic = func(_ string, _ string, _ map[string]any, spec localcatalog.TrackingPlan, graph *resources.Graph) []rules.ValidationResult {
	results := funcs.ValidateReferences(spec, graph)

	// Variant discriminator validation, property nesting, and name uniqueness checks
	results = append(results, validateTrackingPlanVariants(spec, graph)...)
	results = append(results, validatePropertyNestingV0(spec, graph)...)
	results = append(results, validateTrackingPlanNameUniquenessV0(spec, graph)...)
	results = append(results, validateDuplicateEventsV0(spec)...)
	results = append(results, validateDuplicatePropertiesV0(spec)...)

	return results
}

// validateDuplicateEventsV0 emits one error per occurrence of any event ref
// that appears more than once across rules. Comparison is raw-string equality.
func validateDuplicateEventsV0(spec localcatalog.TrackingPlan) []rules.ValidationResult {
	counts := make(map[string]int)
	for _, rule := range spec.Rules {
		if rule.Event == nil {
			continue
		}
		counts[rule.Event.Ref]++
	}

	var results []rules.ValidationResult
	for i, rule := range spec.Rules {
		if rule.Event == nil {
			continue
		}
		ref := rule.Event.Ref
		if counts[ref] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/rules/%d/event/$ref", i),
				Message:   fmt.Sprintf("duplicate event reference '%s' (appears %d times)", ref, counts[ref]),
			})
		}
	}
	return results
}

// validateDuplicatePropertiesV0 walks each rule and dedupes property refs at
// every sibling scope: rule.Properties (and recursively), variants[*].cases[*].properties,
// and variants[*].default. Each sibling list is an independent scope.
func validateDuplicatePropertiesV0(spec localcatalog.TrackingPlan) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, rule := range spec.Rules {
		ruleRef := fmt.Sprintf("/rules/%d", i)

		results = append(results, checkDuplicateSiblingPropsV0(rule.Properties, ruleRef+"/properties")...)

		for v, variant := range rule.Variants {
			for c, kase := range variant.Cases {
				caseRef := fmt.Sprintf("%s/variants/%d/cases/%d/properties", ruleRef, v, c)
				results = append(results, checkDuplicateVariantPropRefsV0(kase.Properties, caseRef)...)
			}
			defaultRef := fmt.Sprintf("%s/variants/%d/default", ruleRef, v)
			results = append(results, checkDuplicateVariantPropRefsV0(variant.Default, defaultRef)...)
		}
	}

	return results
}

// checkDuplicateSiblingPropsV0 dedupes one list of TPRuleProperty and recurses
// into nested Properties. Each nested list is its own sibling scope.
func checkDuplicateSiblingPropsV0(props []*localcatalog.TPRuleProperty, parentRef string) []rules.ValidationResult {
	counts := make(map[string]int)
	for _, prop := range props {
		counts[prop.Ref]++
	}

	var results []rules.ValidationResult
	for i, prop := range props {
		if counts[prop.Ref] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("%s/%d/$ref", parentRef, i),
				Message:   fmt.Sprintf("duplicate property reference '%s' (appears %d times)", prop.Ref, counts[prop.Ref]),
			})
		}
		if len(prop.Properties) > 0 {
			nestedRef := fmt.Sprintf("%s/%d/properties", parentRef, i)
			results = append(results, checkDuplicateSiblingPropsV0(prop.Properties, nestedRef)...)
		}
	}
	return results
}

// checkDuplicateVariantPropRefsV0 dedupes a flat list of variant property refs.
// Variant property refs use PropertyReference and do not recurse.
func checkDuplicateVariantPropRefsV0(props []localcatalog.PropertyReference, parentRef string) []rules.ValidationResult {
	counts := make(map[string]int)
	for _, prop := range props {
		counts[prop.Ref]++
	}

	var results []rules.ValidationResult
	for i, prop := range props {
		if counts[prop.Ref] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("%s/%d/$ref", parentRef, i),
				Message:   fmt.Sprintf("duplicate property reference '%s' (appears %d times)", prop.Ref, counts[prop.Ref]),
			})
		}
	}
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
		results = append(results, variant.ValidateVariantDiscriminatorsV0(
			rule.Variants, ownRefs, fmt.Sprintf("/rules/%d", i), graph,
		)...)
	}

	return results
}

func validateTrackingPlanNameUniquenessV0(spec localcatalog.TrackingPlan, graph *resources.Graph) []rules.ValidationResult {
	countMap := trackingPlanDisplayNameCountMap(graph)
	if countMap[spec.Name] > 1 {
		return []rules.ValidationResult{{
			Reference: "/display_name",
			Message:   fmt.Sprintf("duplicate display_name '%s' within kind 'tp'", spec.Name),
		}}
	}

	return nil
}

func trackingPlanDisplayNameCountMap(graph *resources.Graph) map[string]int {
	countMap := make(map[string]int)
	for _, resource := range graph.ResourcesByType(types.TrackingPlanResourceType) {
		data := resource.Data()
		name, _ := data["name"].(string)
		countMap[name]++
	}

	return countMap
}

// validatePropertyNesting checks that each rule property's referenced type supports
// nesting before allowing nested Properties or AdditionalProperties.
func validatePropertyNestingV0(spec localcatalog.TrackingPlan, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, rule := range spec.Rules {
		for j, prop := range rule.Properties {
			ref := fmt.Sprintf("/rules/%d/properties/%d", i, j)
			results = append(results, validatePropertyNestingAllowedV0(prop, ref, graph)...)
		}
	}

	return results
}

// validatePropertyNestingAllowed validates that property nesting is allowed for the property
// which internally can have nested properties.
func validatePropertyNestingAllowedV0(
	property *localcatalog.TPRuleProperty,
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
	resourceType, localID, err := funcs.ParseURNRef(property.Ref)
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
			results = append(results, validatePropertyNestingAllowedV0(nested, nestedRef, graph)...)
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
	return prules.NewTypedRule(
		"datacatalog/tracking-plans/semantic-valid",
		rules.Error,
		"tracking plan references must resolve to existing resources",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.LegacyVersionPatterns(localcatalog.KindTrackingPlans),
			validateTrackingPlanSemantic,
		),
		prules.NewSemanticPatternValidator(
			prules.V1VersionPatterns(localcatalog.KindTrackingPlansV1),
			validateTrackingPlanSemanticV1,
		),
	)
}
