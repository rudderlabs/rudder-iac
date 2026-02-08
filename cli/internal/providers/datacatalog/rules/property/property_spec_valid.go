package property

import (
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var examples = rules.Examples{
	Valid: []string{
		`properties:
  - id: user_id
    name: User ID
    description: Unique identifier for the user
    type: string
  - id: email
    name: Email
    type: string`,
	},
	Invalid: []string{
		`properties:
  - name: Missing ID
    type: string`,
		`properties:
  - id: user_id
    # Missing required name field
    type: string`,
		`properties:
  - id: user_id
    name: ""
    # Name cannot be empty
    type: string`,
		`properties:
  - id: user_id
    name: This is a very long name that exceeds the maximum allowed length of sixty five characters for a property name
    # Name exceeds 65 characters
    type: string`,
	},
}

// Main validation function for property spec
// which delegates the validation to the go-validator through struct tags.
var validatePropertySpec = func(Kind string, Version string, Metadata map[string]any, Spec localcatalog.PropertySpec) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(
		Spec,
		"",
		getValidateFuncs(Version)..., // attach custom validate funcs specially for property specs
	)

	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/properties",
				Message:   err.Error(),
			},
		}
	}

	return funcs.ParseValidationErrors(validationErrors)
}

func NewPropertySpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/properties/spec-syntax-valid",
		rules.Error,
		"property spec syntax must be valid",
		examples,
		[]string{"properties"},
		validatePropertySpec,
	)
}

// getValidateFuncs returns custom validate functions which the property spec employs
// by adding requisite tags to fields in the spec struct
func getValidateFuncs(_ string) []rules.CustomValidateFunc {
	// TODO: we would need to create a different regex for reference validation
	// based on the version of spec.
	return []rules.CustomValidateFunc{
		// primitive or reference validation on the type
		funcs.NewPrimitiveOrReference([]string{
			"string",
			"number",
			"integer",
			"boolean",
			"array",
			"object",
			"null",
		}, funcs.BuildLegacyReferenceRegex([]string{"custom-types"})),
	}
}
