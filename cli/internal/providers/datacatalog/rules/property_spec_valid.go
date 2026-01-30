package rules

import (
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
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
	},
}

func validatePropertySpec(
	Kind string,
	Version string,
	Metadata map[string]any,
	Spec localcatalog.PropertySpec,
) []rules.ValidationResult {

	result, err := rules.ValidateStruct(Spec, "")

	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/properties",
				Message:   err.Error(),
			},
		}
	}

	return result
}

func NewPropertySpecSyntaxValidRule() prules.TypedRule[localcatalog.PropertySpec] {
	return prules.NewTypedRule(
		"datacatalog/properties/spec-syntax-valid",
		rules.Error,
		"property spec syntax must be valid",
		examples,
		[]string{"properties"},
		validatePropertySpec,
	)
}
