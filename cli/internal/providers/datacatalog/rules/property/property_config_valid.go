package property

import (
	"fmt"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	catalogRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

const (
	propConfigRuleID          = "datacatalog/properties/config-valid"
	propConfigRuleDescription = "property config must be valid for the given type"
)

var propConfigExamples = rules.Examples{
	Valid: []string{
		`properties:
  - id: user_email
    name: UserEmail
    type: string
    propConfig:
      format: "email"
      minLength: 5
      maxLength: 100`,
		`properties:
  - id: age
    name: Age
    type: integer
    propConfig:
      minimum: 0
      maximum: 120`,
		`properties:
  - id: tags
    name: Tags
    type: array
    propConfig:
      itemTypes: ["string"]
      minItems: 1
      maxItems: 10`,
	},
	Invalid: []string{
		`properties:
  - id: address
    name: Address
    type: object
    propConfig:
      # Config not allowed for object type
      properties: []`,
		`properties:
  - id: email
    name: Email
    type: string
    propConfig:
      # Invalid format value
      format: invalid`,
		`properties:
  - id: count
    name: Count
    type: integer
    propConfig:
      # minimum must be integer not float
      minimum: 1.5`,
	},
}

// Main validation function for property config validation
var validatePropertyConfig = func(Kind string, Version string, Metadata map[string]any, Spec localcatalog.PropertySpec) []rules.ValidationResult {
	var results []rules.ValidationResult

	// Validate each property's config
	for i, property := range Spec.Properties {
		if len(property.Config) == 0 {
			continue
		}

		reference := fmt.Sprintf("/properties/%d/propConfig", i)

		// Parse the type string to get individual types
		types := parsePropertyType(property.Type)

		// Use the shared config validation abstraction
		configResults := config.ValidateConfig(
			types,
			property.Config,
			reference,
		)

		results = append(results, configResults...)
	}

	return results
}

// parsePropertyType parses the property type string into individual types
func parsePropertyType(typeStr string) []string {
	// If empty, default to wildcard types
	if typeStr == "" {
		return catalogRules.ValidPrimitiveTypes
	}

	return lo.Map(strings.Split(typeStr, ","), func(t string, _ int) string {
		return strings.TrimSpace(t)
	})
}

// NewPropertyConfigValidRule creates a new property config validation rule using TypedRule pattern
func NewPropertyConfigValidRule() rules.Rule {
	return prules.NewTypedRule(
		propConfigRuleID,
		rules.Error,
		propConfigRuleDescription,
		propConfigExamples,
		prules.LegacyVersionPatterns("properties"),
		validatePropertyConfig,
	)
}
