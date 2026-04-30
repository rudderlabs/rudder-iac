package trackingplan

import (
	"fmt"
	"reflect"
	"regexp"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const maxNestingDepth = 3

const (
	eventRequiredMessage             = "'event' is required"
	eventOrIncludesRequiredMessage   = "event or includes is required"
	eventAndIncludesExclusiveMessage = "event and includes cannot be specified together"
	includesNotSupportedV0Message    = "'includes' is not supported"
	v1IncludesUnsupportedMessage     = "includes is not supported for tracking-plan v1 event rules"
	includesReferenceRequiredMessage = "'$ref' is required"
	includesReferencePatternMessage  = "'$ref' is not valid: must be of pattern #/tp/<group>/event_rule/<id-or-*>"
)

var tpIncludesReferenceRegexp = regexp.MustCompile(`^#/tp/[a-zA-Z0-9_-]+/event_rule/([a-zA-Z0-9_-]+|\*)$`)

var examples = rules.Examples{
	Valid: []string{
		`id: test_tp
display_name: Test Tracking Plan
rules:
  - id: signup_rule
    type: event_rule
    event:
      $ref: "#/events/user-events/signup"
    variants:
      - type: discriminator
        discriminator: "#/properties/signup-props/signup_method"
        cases:
          - display_name: "Email Signup"
            match: ["email"]
            description: "User signed up via email"
            properties:
              - $ref: "#/properties/signup-props/email_address"
                required: true
              - $ref: "#/properties/signup-props/email_verified"
                required: false
        default:
          - $ref: "#/properties/common/user_id"
            required: true`,
	},
	Invalid: []string{
		`id: test_tp
display_name: Test Tracking Plan
rules:
  - id: invalid_rule
    type: event_rule
    variants:
      - type: "wrong_type"  # Must be "discriminator"
        discriminator: "#/properties/props/field"
        cases:
          - display_name: "Case 1"
            properties:
              - $ref: "#/properties/props/prop1"`,
		`id: test_tp
display_name: Test Tracking Plan
rules:
  - id: invalid_rule
    type: event_rule
    variants:
      - type: discriminator
        discriminator: ""  # Cannot be empty
        cases: []  # Must have at least 1 case`,
	},
}

// validateTrackingPlanSpec validates the V0 tracking plan spec using struct tags
// and V0-specific follow-up checks.
var validateTrackingPlanSpec = func(
	Kind string,
	Version string,
	Metadata map[string]any,
	Spec localcatalog.TrackingPlan,
) []rules.ValidationResult {
	return validateTrackingPlanSpecWithEventRuleIncludes(Kind, Version, Metadata, Spec, false)
}

func validateTrackingPlanSpecWithEventRuleIncludes(
	_ string,
	_ string,
	_ map[string]any,
	spec localcatalog.TrackingPlan,
	eventRuleIncludesEnabled bool,
) []rules.ValidationResult {
	// validate the spec using struct tags through go-playground/validator
	// majority of the validation is done here, remaining spec validation
	// will be done in subsequent steps.
	validationErrors, err := rules.ValidateStruct(spec, "")

	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/rules",
				Message:   err.Error(),
			},
		}
	}

	results := funcs.ParseValidationErrors(validationErrors, reflect.TypeOf(spec))

	// validate the rules on the trackingplan spec
	results = append(results, validateRules(spec.Rules, eventRuleIncludesEnabled)...)

	return results
}

var validateTrackingPlanSpecV1 = func(
	_ string,
	_ string,
	_ map[string]any,
	spec localcatalog.TrackingPlanV1,
) []rules.ValidationResult {
	return validateTrackingPlanSpecV1WithEventRuleIncludes("", "", nil, spec, false)
}

func validateTrackingPlanSpecV1WithEventRuleIncludes(
	_ string,
	_ string,
	_ map[string]any,
	spec localcatalog.TrackingPlanV1,
	eventRuleIncludesEnabled bool,
) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/rules",
				Message:   err.Error(),
			},
		}
	}

	results := funcs.ParseValidationErrors(validationErrors, nil)
	results = append(results, validateRulesV1(spec.Rules, eventRuleIncludesEnabled)...)

	return results
}

func validateRules(tpRules []*localcatalog.TPRule, eventRuleIncludesEnabled bool) []rules.ValidationResult {
	var results []rules.ValidationResult

	results = append(
		results,
		validateDuplicateRuleIDs(tpRules, func(r *localcatalog.TPRule) string { return r.LocalID })...,
	)

	for i, rule := range tpRules {
		results = append(results, validateEventRuleShapeV0(rule, i, eventRuleIncludesEnabled)...)

		for j, prop := range rule.Properties {
			if len(prop.Properties) == 0 {
				continue
			}

			// recursively validate the nesting depth of the properties
			ref := fmt.Sprintf("/rules/%d/properties/%d", i, j)
			results = append(
				results,
				validateNestingDepth(prop.Properties, 1, ref)...,
			)
		}
	}

	return results
}

func validateRulesV1(tpRules []*localcatalog.TPRuleV1, eventRuleIncludesEnabled bool) []rules.ValidationResult {
	var results []rules.ValidationResult

	results = append(
		results,
		validateDuplicateRuleIDs(tpRules, func(r *localcatalog.TPRuleV1) string { return r.LocalID })...,
	)

	for i, rule := range tpRules {
		results = append(results, validateEventRuleIncludesUnsupportedV1(rule, i)...)

		for j, prop := range rule.Properties {
			ref := fmt.Sprintf("/rules/%d/properties/%d", i, j)
			if len(prop.Properties) > 0 {
				results = append(
					results,
					validateNestingDepthV1(prop.Properties, 1, ref)...,
				)
			}
			results = append(results, validateAdditionalPropertiesV1(prop, ref)...)
		}
	}

	return results
}

func validateEventRuleShapeV0(rule *localcatalog.TPRule, index int, eventRuleIncludesEnabled bool) []rules.ValidationResult {
	hasEvent := rule.Event != nil
	return validateEventRuleShape(hasEvent, rule.Includes, index, eventRuleIncludesEnabled)
}

func validateEventRuleIncludesUnsupportedV1(rule *localcatalog.TPRuleV1, index int) []rules.ValidationResult {
	if rule.Includes == nil {
		return nil
	}

	return []rules.ValidationResult{{
		Reference: fmt.Sprintf("/rules/%d/includes", index),
		Message:   v1IncludesUnsupportedMessage,
	}}
}

func validateEventRuleShape(
	hasEvent bool,
	includes *localcatalog.TPRuleIncludes,
	index int,
	eventRuleIncludesEnabled bool,
) []rules.ValidationResult {
	hasIncludes := includes != nil

	if !eventRuleIncludesEnabled {
		return validateEventRuleShapeWithoutIncludes(hasEvent, hasIncludes, index)
	}

	return validateEventRuleShapeWithIncludes(hasEvent, includes, index)
}

func validateEventRuleShapeWithoutIncludes(hasEvent, hasIncludes bool, index int) []rules.ValidationResult {
	var results []rules.ValidationResult
	if !hasEvent {
		results = append(results, rules.ValidationResult{
			Reference: fmt.Sprintf("/rules/%d/event", index),
			Message:   eventRequiredMessage,
		})
	}

	// When includes is present with a direct event, report a single clear error; when includes is
	// present without event, only the missing event error above is reported (no second error).
	if hasEvent && hasIncludes {
		results = append(results, rules.ValidationResult{
			Reference: fmt.Sprintf("/rules/%d/includes", index),
			Message:   includesNotSupportedV0Message,
		})
	}

	return results
}

func validateEventRuleShapeWithIncludes(
	hasEvent bool,
	includes *localcatalog.TPRuleIncludes,
	index int,
) []rules.ValidationResult {
	hasIncludes := includes != nil
	ruleRef := fmt.Sprintf("/rules/%d", index)

	switch {
	case !hasEvent && !hasIncludes:
		return []rules.ValidationResult{{
			Reference: ruleRef,
			Message:   eventOrIncludesRequiredMessage,
		}}
	case hasEvent && hasIncludes:
		return []rules.ValidationResult{{
			Reference: ruleRef,
			Message:   eventAndIncludesExclusiveMessage,
		}}
	case hasIncludes:
		return validateIncludesRef(includes.Ref, ruleRef+"/includes/$ref")
	default:
		return nil
	}
}

func validateIncludesRef(ref, reference string) []rules.ValidationResult {
	if ref == "" {
		return []rules.ValidationResult{{
			Reference: reference,
			Message:   includesReferenceRequiredMessage,
		}}
	}

	if !tpIncludesReferenceRegexp.MatchString(ref) {
		return []rules.ValidationResult{{
			Reference: reference,
			Message:   includesReferencePatternMessage,
		}}
	}

	return nil
}

func validateDuplicateRuleIDs[T any](tpRules []T, idOf func(T) string) []rules.ValidationResult {
	counts := make(map[string]int)
	for _, rule := range tpRules {
		id := idOf(rule)
		if id == "" {
			continue
		}
		counts[id]++
	}

	var results []rules.ValidationResult
	for i, rule := range tpRules {
		if counts[idOf(rule)] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/rules/%d/id", i),
				Message:   "duplicate rule id in tracking plan rules",
			})
		}
	}
	return results
}

// validateNestingDepth walks nested properties recursively and reports an error
// at the root property reference when nesting exceeds maxNestingDepth (3 levels).
func validateNestingDepth(properties []*localcatalog.TPRuleProperty, currentDepth int, rootRef string) []rules.ValidationResult {
	if currentDepth > maxNestingDepth {
		return []rules.ValidationResult{{
			Reference: rootRef,
			Message:   fmt.Sprintf("maximum property nesting depth of %d levels exceeded", maxNestingDepth),
		}}
	}

	var results []rules.ValidationResult
	for _, prop := range properties {

		if len(prop.Properties) > 0 {
			results = append(
				results,
				validateNestingDepth(prop.Properties, currentDepth+1, rootRef)...)
		}
	}

	return results
}

func validateNestingDepthV1(properties []*localcatalog.TPRulePropertyV1, currentDepth int, rootRef string) []rules.ValidationResult {
	if currentDepth > maxNestingDepth {
		return []rules.ValidationResult{{
			Reference: rootRef,
			Message:   fmt.Sprintf("maximum property nesting depth of %d levels exceeded", maxNestingDepth),
		}}
	}

	var results []rules.ValidationResult
	for _, prop := range properties {
		if len(prop.Properties) == 0 {
			continue
		}

		results = append(
			results,
			validateNestingDepthV1(prop.Properties, currentDepth+1, rootRef)...,
		)
	}

	return results
}

func validateAdditionalPropertiesV1(prop *localcatalog.TPRulePropertyV1, ref string) []rules.ValidationResult {
	var results []rules.ValidationResult

	if prop.AdditionalProperties != nil && len(prop.Properties) == 0 {
		results = append(results, rules.ValidationResult{
			Reference: ref + "/additional_properties",
			Message:   "additional_properties is only allowed on properties with nested properties",
		})
	}

	for i, nested := range prop.Properties {
		nestedRef := fmt.Sprintf("%s/properties/%d", ref, i)
		results = append(results, validateAdditionalPropertiesV1(nested, nestedRef)...)
	}

	return results
}

// NewTrackingPlanSpecSyntaxValidRule creates a spec syntax validation rule
// for tracking plans across supported spec versions.
func NewTrackingPlanSpecSyntaxValidRule(eventRuleIncludesEnabled bool) rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/tracking-plans/spec-syntax-valid",
		rules.Error,
		"tracking plan spec syntax must be valid",
		examples,
		prules.NewPatternValidator(
			prules.LegacyVersionPatterns(localcatalog.KindTrackingPlans),
			func(kind string, version string, metadata map[string]any, spec localcatalog.TrackingPlan) []rules.ValidationResult {
				return validateTrackingPlanSpecWithEventRuleIncludes(kind, version, metadata, spec, eventRuleIncludesEnabled)
			},
		),
		prules.NewPatternValidator(
			prules.V1VersionPatterns(localcatalog.KindTrackingPlansV1),
			func(kind string, version string, metadata map[string]any, spec localcatalog.TrackingPlanV1) []rules.ValidationResult {
				return validateTrackingPlanSpecV1WithEventRuleIncludes(kind, version, metadata, spec, eventRuleIncludesEnabled)
			},
		),
	)
}
