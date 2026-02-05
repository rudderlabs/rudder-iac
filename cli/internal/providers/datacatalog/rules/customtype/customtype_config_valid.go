package customtype

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

const (
	ruleID          = "datacatalog/custom-types/config-valid"
	ruleDescription = "custom type config must be valid for the given type"
)

var (
	validFormatValues = []string{
		"date-time", "date", "time", "email", "uuid", "hostname", "ipv4", "ipv6",
	}

	validPrimitiveTypes = []string{
		"string", "number", "integer", "boolean", "null", "array", "object",
	}

	// Allowed config keys per type
	allowedStringConfigKeys = map[string]bool{
		"enum": true, "min_length": true, "max_length": true, "pattern": true, "format": true,
	}

	allowedNumberConfigKeys = map[string]bool{
		"enum": true, "minimum": true, "maximum": true,
		"exclusive_minimum": true, "exclusive_maximum": true, "multiple_of": true,
	}

	allowedArrayConfigKeys = map[string]bool{
		"item_types": true, "minItems": true, "maxItems": true, "uniqueItems": true,
	}

	// Legacy custom type reference pattern
	customTypeLegacyReferenceRegex = regexp.MustCompile(`^#/custom-types/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)$`)
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
      item_types: ["string"]
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

// Main validation function for custom type config validation
var validateCustomTypeConfig = func(Kind string, Version string, Metadata map[string]any, Spec localcatalog.CustomTypeSpec) []rules.ValidationResult {
	var results []rules.ValidationResult

	// Validate each custom type's config
	for i, customType := range Spec.Types {
		// Skip if no config
		if len(customType.Config) == 0 {
			continue
		}

		// Validate config based on type
		switch customType.Type {
		case "object":
			results = append(results, validateObjectConfig(customType, i)...)
		case "string":
			results = append(results, validateStringConfig(customType.Config, i)...)
		case "number", "integer":
			results = append(results, validateNumberConfig(customType.Config, i, customType.Type)...)
		case "array":
			results = append(results, validateArrayConfig(customType.Config, i)...)
		case "boolean", "null":
			// No validation required for boolean/null types
			continue
		default:
			// Unknown type - skip validation
			continue
		}
	}

	return results
}

// NewCustomTypeConfigValidRule creates a new custom type config validation rule using TypedRule pattern
func NewCustomTypeConfigValidRule() rules.Rule {
	return prules.NewTypedRule(
		ruleID,
		rules.Error,
		ruleDescription,
		configExamples,
		[]string{"custom-types"},
		validateCustomTypeConfig,
	)
}

// validateObjectConfig validates that object types don't have config
func validateObjectConfig(customType localcatalog.CustomType, typeIndex int) []rules.ValidationResult {
	if len(customType.Config) > 0 {
		return []rules.ValidationResult{
			{
				Reference: fmt.Sprintf("/types/%d/config", typeIndex),
				Message:   "'config' is not allowed for custom type of type 'object'",
			},
		}
	}
	return nil
}

// validateStringConfig validates config for string type
func validateStringConfig(config map[string]any, typeIndex int) []rules.ValidationResult {
	var results []rules.ValidationResult

	// Check for unknown config keys
	for key := range config {
		if !allowedStringConfigKeys[key] {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/config/%s", typeIndex, key),
				Message:   fmt.Sprintf("'%s' is not a valid config key for type 'string', allowed keys are: %s", key, strings.Join(lo.Keys(allowedStringConfigKeys), ", ")),
			})
		}
	}

	// Validate enum - must be array of strings
	if enum, ok := config["enum"]; ok {
		enumArray, ok := enum.([]any)
		if !ok {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/config/enum", typeIndex),
				Message:   "'enum' must be an array",
			})
		} else {
			// Check each enum value is a string
			for i, val := range enumArray {
				if _, ok := val.(string); !ok {
					results = append(results, rules.ValidationResult{
						Reference: fmt.Sprintf("/types/%d/config/enum/%d", typeIndex, i),
						Message:   fmt.Sprintf("'enum[%d]' must be a string", i),
					})
				}
			}
		}
	}

	// Validate min_length
	if minLength, ok := config["min_length"]; ok {
		if !isInteger(minLength) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/config/min_length", typeIndex),
				Message:   "'min_length' must be a number",
			})
		}
	}

	// Validate max_length
	if maxLength, ok := config["max_length"]; ok {
		if !isInteger(maxLength) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/config/max_length", typeIndex),
				Message:   "'max_length' must be a number",
			})
		}
	}

	// Validate pattern
	if pattern, ok := config["pattern"]; ok {
		if _, ok := pattern.(string); !ok {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/config/pattern", typeIndex),
				Message:   "'pattern' must be a string",
			})
		}
	}

	// Validate format
	if format, ok := config["format"]; ok {
		formatStr, ok := format.(string)
		if !ok {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/config/format", typeIndex),
				Message:   "'format' must be a string",
			})
		} else if !slices.Contains(validFormatValues, formatStr) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/config/format", typeIndex),
				Message:   fmt.Sprintf("'format' must be one of: %s", strings.Join(validFormatValues, ", ")),
			})
		}
	}

	return results
}

// validateNumberConfig validates config for number/integer type
func validateNumberConfig(config map[string]any, typeIndex int, typeName string) []rules.ValidationResult {
	var results []rules.ValidationResult

	// Determine type checker based on type (integer is stricter)
	typeCheck := isNumber
	if typeName == "integer" {
		typeCheck = isInteger
	}

	// Check for unknown config keys
	for key := range config {
		if !allowedNumberConfigKeys[key] {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/config/%s", typeIndex, key),
				Message:   fmt.Sprintf("'%s' is not a valid config key for type '%s', allowed keys are: %s", key, typeName, strings.Join(lo.Keys(allowedNumberConfigKeys), ", ")),
			})
		}
	}

	// Validate enum
	if enum, ok := config["enum"]; ok {
		enumArray, ok := enum.([]any)
		if !ok {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/config/enum", typeIndex),
				Message:   "'enum' must be an array",
			})
		} else {
			for i, val := range enumArray {
				if !typeCheck(val) {
					results = append(results, rules.ValidationResult{
						Reference: fmt.Sprintf("/types/%d/config/enum/%d", typeIndex, i),
						Message:   fmt.Sprintf("'enum[%d]' must be a %s", i, typeName),
					})
				}
			}
		}
	}

	// Validate numeric fields
	numericFields := []string{"minimum", "maximum", "exclusive_minimum", "exclusive_maximum", "multiple_of"}
	for _, field := range numericFields {
		if val, ok := config[field]; ok {
			if !typeCheck(val) {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/types/%d/config/%s", typeIndex, field),
					Message:   fmt.Sprintf("'%s' must be a %s", field, typeName),
				})
			}
		}
	}

	return results
}

// validateArrayConfig validates config for array type
func validateArrayConfig(config map[string]any, typeIndex int) []rules.ValidationResult {
	var results []rules.ValidationResult

	// Check for unknown config keys
	for key := range config {
		if !allowedArrayConfigKeys[key] {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/config/%s", typeIndex, key),
				Message:   fmt.Sprintf("'%s' is not a valid config key for type 'array', allowed keys are: %s", key, strings.Join(lo.Keys(allowedArrayConfigKeys), ", ")),
			})
		}
	}

	// Validate item_types
	if itemTypes, ok := config["item_types"]; ok {
		itemTypesArray, ok := itemTypes.([]any)
		if !ok {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/config/item_types", typeIndex),
				Message:   "'item_types' must be an array",
			})
		} else {
			for idx, itemType := range itemTypesArray {
				val, ok := itemType.(string)
				if !ok {
					results = append(results, rules.ValidationResult{
						Reference: fmt.Sprintf("/types/%d/config/item_types/%d", typeIndex, idx),
						Message:   fmt.Sprintf("'item_types[%d]' must be a string value", idx),
					})
					continue
				}

				// Check if it's a custom type reference
				if customTypeLegacyReferenceRegex.MatchString(val) {
					// Custom type reference must be the only item
					if len(itemTypesArray) != 1 {
						results = append(results, rules.ValidationResult{
							Reference: fmt.Sprintf("/types/%d/config/item_types/%d", typeIndex, idx),
							Message:   "'item_types' containing custom type reference cannot be paired with other types",
						})
					}
					continue
				}

				// Must be a valid primitive type
				if !slices.Contains(validPrimitiveTypes, val) {
					results = append(results, rules.ValidationResult{
						Reference: fmt.Sprintf("/types/%d/config/item_types/%d", typeIndex, idx),
						Message:   fmt.Sprintf("'item_types[%d]' is invalid, valid type values are: %s", idx, strings.Join(validPrimitiveTypes, ", ")),
					})
				}
			}
		}
	}

	// Validate minItems
	if minItems, ok := config["minItems"]; ok {
		if !isInteger(minItems) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/config/minItems", typeIndex),
				Message:   "'minItems' must be a number",
			})
		}
	}

	// Validate maxItems
	if maxItems, ok := config["maxItems"]; ok {
		if !isInteger(maxItems) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/config/maxItems", typeIndex),
				Message:   "'maxItems' must be a number",
			})
		}
	}

	// Validate uniqueItems
	if uniqueItems, ok := config["uniqueItems"]; ok {
		if _, ok := uniqueItems.(bool); !ok {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/config/uniqueItems", typeIndex),
				Message:   "'uniqueItems' must be a boolean",
			})
		}
	}

	return results
}

// isNumber checks if value is any numeric type
func isNumber(val any) bool {
	switch val.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	}
	return false
}

// isInteger checks if value is an integer (includes float64 that are whole numbers)
func isInteger(val any) bool {
	switch v := val.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32:
		return float32(int(v)) == v
	case float64:
		return float64(int64(v)) == v
	}
	return false
}
