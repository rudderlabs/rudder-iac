package rules

import (
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var examples = rules.Examples{
	Valid: []string{
		`events:
  - id: page_viewed
    event_type: track
    description: User viewed a page
  - id: product_clicked
    name: Product Clicked
    event_type: track`,
	},
	Invalid: []string{
		`events:
  - name: Missing ID
    event_type: track`,
		`events:
  - id: missing_type
    # Missing required event_type field`,
	},
}

// Main validation function for event spec
// which delegates the validation to the go-validator through struct tags.
var validateEventSpec = func(Kind string, Version string, Metadata map[string]any, Spec localcatalog.EventSpec) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(
		Spec,
		"",
		getValidateFuncs(Version)..., // attach custom validate funcs specially for event specs
	)

	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/events",
				Message:   err.Error(),
			},
		}
	}

	return funcs.ParseValidationErrors(validationErrors)
}

func NewEventSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/events/spec-syntax-valid",
		rules.Error,
		"event spec syntax must be valid",
		examples,
		[]string{"events"},
		validateEventSpec,
	)
}

// getValidateFuncs returns custom validate functions which the event spec employs
// by adding requisite tags to fields in the spec struct
func getValidateFuncs(_ string) []rules.CustomValidateFunc {
	// TODO: we would need to create a different regex for reference validation
	// based on the version of spec.
	return []rules.CustomValidateFunc{
		// reference validation for categories
		funcs.NewLegacyReferenceValidateFunc([]string{"categories"}),
	}
}
