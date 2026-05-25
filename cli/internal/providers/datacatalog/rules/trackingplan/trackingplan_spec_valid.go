package trackingplan

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const maxNestingDepth = 3

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
	// validate the spec using struct tags through go-playground/validator
	// majority of the validation is done here, remaining spec validation
	// will be done in subsequent steps.
	validationErrors, err := rules.ValidateStruct(Spec, "")

	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/rules",
				Message:   err.Error(),
			},
		}
	}

	results := funcs.ParseValidationErrors(validationErrors, nil)

	// validate the rules on the trackingplan spec
	results = append(results, validateRules(Spec.Rules)...)

	return results
}

var validateTrackingPlanSpecV1 = func(
	_ string,
	_ string,
	_ map[string]any,
	spec localcatalog.TrackingPlanV1,
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
	results = append(results, validateRulesV1(spec.Rules)...)

	return results
}

func validateRules(tpRules []*localcatalog.TPRule) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, rule := range tpRules {
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

func validateRulesV1(tpRules []*localcatalog.TPRuleV1) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, rule := range tpRules {
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
func NewTrackingPlanSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/tracking-plans/spec-syntax-valid",
		rules.Error,
		"tracking plan spec syntax must be valid",
		examples,
		prules.NewPatternValidator(
			prules.LegacyVersionPatterns(localcatalog.KindTrackingPlans),
			validateTrackingPlanSpec,
		),
		prules.NewPatternValidator(
			prules.V1VersionPatterns(localcatalog.KindTrackingPlansV1),
			validateTrackingPlanSpecV1,
		),
	)
}
