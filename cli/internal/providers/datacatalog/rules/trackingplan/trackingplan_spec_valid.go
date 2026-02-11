package trackingplan

import (
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

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

// validateTrackingPlanSpec validates the trackingplan spec including variants
// using struct tags and go-playground/validator
var validateTrackingPlanSpec = func(
	Kind string,
	Version string,
	Metadata map[string]any,
	Spec localcatalog.TrackingPlan,
) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(Spec, "")

	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/rules",
				Message:   err.Error(),
			},
		}
	}

	return funcs.ParseValidationErrors(validationErrors)
}

// NewTrackingPlanSpecSyntaxValidRule creates a spec syntax validation rule
// for trackingplan. This rule validates the syntactic correctness of the trackingplan spec,
// including variants in trackingplan rules (required fields, types, format) while catalog
// validators handle semantic checks (reference existence, discriminator in properties).
func NewTrackingPlanSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/tracking-plans/spec-syntax-valid",
		rules.Error,
		"tracking plan spec syntax must be valid",
		examples,
		[]string{"tp"},
		validateTrackingPlanSpec,
	)
}
