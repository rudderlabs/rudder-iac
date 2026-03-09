package customtype

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const (
	ruleID          = "datacatalog/custom-types/config-valid"
	ruleDescription = "custom type config must be valid for the given type"
)

var configExamples = rules.Examples{
	Valid: []string{
		`types:
  - id: user_status
    name: UserStatus
    type: string
    config:
      enum: ["active", "inactive"]
      pattern: "^[a-z]+$"`,
		`types:
  - id: age
    name: Age
    type: integer
    config:
      minimum: 0
      maximum: 120`,
		`types:
  - id: tags
    name: Tags
    type: array
    config:
      itemTypes: ["string"]
      minItems: 1`,
	},
	Invalid: []string{
		`types:
  - id: address
    name: Address
    type: object
    config:
      # Config not allowed for object type
      properties: []`,
		`types:
  - id: status
    name: Status
    type: string
    config:
      # Invalid format value
      format: invalid`,
		`types:
  - id: count
    name: Count
    type: integer
    config:
      # enum values must be integers
      enum: [1.5, 2.5]`,
	},
}

// customTypeObjectConfig is a local validator for a custom type of type object.
// Config is allowed, but only 'additionalProperties' (boolean) is a recognised field.
// The keyword is matched via ConfigKeyword so both V0 ("additionalProperties") and
// V1 ("additional_properties") raw keys resolve correctly through the alias layer.
type customTypeObjectConfig struct{}

var allowedCustomTypeObjectKeys = map[config.ConfigKeyword]bool{
	config.KeywordAdditionalProperties: true,
}

func (c *customTypeObjectConfig) ConfigAllowed() bool { return true }

func (c *customTypeObjectConfig) ValidateField(rawKey string, keyword config.ConfigKeyword, fieldval any) ([]rules.ValidationResult, error) {
	if !allowedCustomTypeObjectKeys[keyword] {
		return nil, config.ErrFieldNotSupported
	}
	switch keyword {
	case config.KeywordAdditionalProperties:
		if _, ok := fieldval.(bool); !ok {
			return []rules.ValidationResult{{
				Reference: rawKey,
				Message:   fmt.Sprintf("'%s' must be a boolean", rawKey),
			}}, nil
		}
	}
	return nil, nil
}

func (c *customTypeObjectConfig) ValidateCrossFields(_ map[config.ConfigKeyword]any) []rules.ValidationResult {
	return nil
}

// customTypeValidatorOverrides injects context-specific validators for the custom-type context.
var customTypeValidatorOverrides = map[string]config.TypeConfigValidator{
	"object": &customTypeObjectConfig{},
}

// validateCustomTypeConfig is the main validation function for custom type config.
var validateCustomTypeConfig = func(Kind string, Version string, Metadata map[string]any, Spec localcatalog.CustomTypeSpec) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, customType := range Spec.Types {
		if len(customType.Config) == 0 {
			continue
		}

		reference := fmt.Sprintf("/types/%d/config", i)

		configResults := config.ValidateConfig(
			[]string{customType.Type},
			customType.Config,
			reference,
			customTypeValidatorOverrides,
		)

		results = append(results, configResults...)
	}

	return results
}

// NewCustomTypeConfigValidRule creates a new custom type config validation rule using TypedRule pattern.
func NewCustomTypeConfigValidRule() rules.Rule {
	return prules.NewTypedRule(
		ruleID,
		rules.Error,
		ruleDescription,
		configExamples,
		prules.NewPatternValidator(
			prules.LegacyVersionPatterns(localcatalog.KindCustomTypes),
			validateCustomTypeConfig,
		),
	)
}
