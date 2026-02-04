package customtype

import (
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const (
	customTypeNameRegexPattern = "^[A-Z][A-Za-z0-9_-]*$"
	customTypeNameRegexTag     = "custom_type_name"
	customTypeNameErrorMessage = "must start with uppercase and contain only alphanumeric, underscores, or hyphens"
)

func init() {
	// Register the custom type name pattern for use with validate:"pattern=custom_type_name"
	funcs.NewPattern(
		customTypeNameRegexTag,
		customTypeNameRegexPattern,
		customTypeNameErrorMessage,
	)
}

var examples = rules.Examples{
	Valid: []string{
		`types:
  - id: address
    name: Address
    description: Physical address structure
    type: object
  - id: user_status
    name: User Status
    type: string`,
	},
	Invalid: []string{
		`types:
  - name: Missing ID
    type: string`,
		`types:
  - id: user_status
    # Missing required name field
    type: string`,
		`types:
  - id: status
    name: Status
    # Missing required type field`,
	},
}

// Main validation function for custom type spec
// which delegates the validation to the go-validator through struct tags.
var validateCustomTypeSpec = func(Kind string, Version string, Metadata map[string]any, Spec localcatalog.CustomTypeSpec) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(
		Spec,
		"",
		getValidateFuncs(Version)..., // attach custom validate funcs specially for custom type specs
	)

	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/types",
				Message:   err.Error(),
			},
		}
	}

	return funcs.ParseValidationErrors(validationErrors)
}

func NewCustomTypeSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/custom-types/spec-syntax-valid",
		rules.Error,
		"custom type spec syntax must be valid",
		examples,
		[]string{"custom-types"},
		validateCustomTypeSpec,
	)
}

// getValidateFuncs returns custom validate functions which the custom type spec employs
// by adding requisite tags to fields in the spec struct
func getValidateFuncs(_ string) []rules.CustomValidateFunc {
	// TODO: we would need to create a different regex for reference validation
	// based on the version of spec.
	return []rules.CustomValidateFunc{
		// primitive validation on the type field
		funcs.NewPrimitive([]string{
			"string",
			"number",
			"integer",
			"boolean",
			"array",
			"object",
			"null",
		}),
		// reference validation for the $ref field in properties
		funcs.NewLegacyReferenceValidateFunc([]string{"properties"}),
		// Note: Pattern validation is now handled by the global "pattern" validator
		// registered via init(). Use validate:"pattern:pattern_custom_type_name" in struct tags.
	}
}
