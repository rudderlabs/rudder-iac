package customtype

import (
	"reflect"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const (
	customTypeNameRegexPattern = "^[A-Z][A-Za-z0-9_-]*$"
	customTypeNameRegexTag     = "custom_type_name"
	customTypeNameErrorMessage = "must start with uppercase and contain only alphanumeric, underscores, or hyphens"

	customTypeTypeRegexPattern = "^(string|number|integer|boolean|array|object|null)$"
	customTypeTypeRegexTag     = "primitive_type"
	customTypeTypeErrorMessage = "must be one of the following: string, number, integer, boolean, array, object, null"
)

func init() {
	// Register the custom type name pattern for use with validate:"pattern=custom_type_name"
	funcs.NewPattern(
		customTypeNameRegexTag,
		customTypeNameRegexPattern,
		customTypeNameErrorMessage,
	)

	// Register the custom type type pattern for use with validate:"pattern=primitive_type"
	funcs.NewPattern(
		customTypeTypeRegexTag,
		customTypeTypeRegexPattern,
		customTypeTypeErrorMessage,
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
// which delegates the validation to the go-validator
// through struct tags.
var validateCustomTypeSpec = func(Kind string, Version string, Metadata map[string]any, Spec localcatalog.CustomTypeSpec) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(Spec, "")

	// If any error on running the validate struct command
	// we report that at top `/types` layer
	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/types",
				Message:   err.Error(),
			},
		}
	}

	return funcs.ParseValidationErrors(
		validationErrors,
		reflect.TypeOf(Spec),
	)
}

func NewCustomTypeSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/custom-types/spec-syntax-valid",
		rules.Error,
		"custom type spec syntax must be valid",
		examples,
		prules.LegacyVersionPatterns("custom-types"),
		validateCustomTypeSpec,
	)
}
