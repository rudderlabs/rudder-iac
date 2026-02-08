package config

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// ArrayTypeConfig validates config for array type
type ArrayTypeConfig struct{}

var allowedArrayKeys = map[string]bool{
	"itemTypes":   true,
	"minItems":    true,
	"maxItems":    true,
	"uniqueItems": true,
}

// ConfigAllowed returns true for array type
func (a *ArrayTypeConfig) ConfigAllowed() bool {
	return true
}

// ValidateField validates a single field for array type
func (a *ArrayTypeConfig) ValidateField(fieldname string, fieldval any) ([]rules.ValidationResult, error) {
	if !allowedArrayKeys[fieldname] {
		return nil, ErrFieldNotSupported
	}

	switch fieldname {
	case "minItems", "maxItems":
		if !isInteger(fieldval) {
			return []rules.ValidationResult{{
				Reference: fieldname,
				Message:   fmt.Sprintf("'%s' must be an integer", fieldname),
			}}, nil
		}
		val, _ := toInteger(fieldval)
		if val < 0 {
			return []rules.ValidationResult{{
				Reference: fieldname,
				Message:   fmt.Sprintf("'%s' must be >= 0", fieldname),
			}}, nil
		}

	case "uniqueItems":
		if _, ok := fieldval.(bool); !ok {
			return []rules.ValidationResult{{
				Reference: fieldname,
				Message:   "'uniqueItems' must be a boolean",
			}}, nil
		}

	case "itemTypes":
		itemTypesArray, ok := fieldval.([]any)
		if !ok {
			return []rules.ValidationResult{{
				Reference: fieldname,
				Message:   "'itemTypes' must be an array",
			}}, nil
		}

		if len(itemTypesArray) == 0 {
			return []rules.ValidationResult{{
				Reference: fieldname,
				Message:   "'itemTypes' must contain at least one type",
			}}, nil
		}

		for idx, itemType := range itemTypesArray {
			typeStr, ok := itemType.(string)
			if !ok {
				return []rules.ValidationResult{{
					Reference: fmt.Sprintf("%s/%d", fieldname, idx),
					Message:   fmt.Sprintf("'%v' must be a string value", itemType),
				}}, nil
			}

			// Check if it's a custom type reference (legacy format)
			if customTypeLegacyReferenceRegex.MatchString(typeStr) {
				// Custom type reference cannot be paired with other types
				if len(itemTypesArray) > 1 {
					return []rules.ValidationResult{{
						Reference: fmt.Sprintf("%s/%d", fieldname, idx),
						Message:   fmt.Sprintf("'%v' custom type reference cannot be paired with other types", itemType),
					}}, nil
				}
				// Valid custom type reference - skip primitive type check
				continue
			}

			// Must be a valid primitive type
			if !isValidPrimitiveType(typeStr) {
				return []rules.ValidationResult{{
					Reference: fmt.Sprintf("%s/%d", fieldname, idx),
					Message:   fmt.Sprintf("invalid type '%s' in itemTypes, must be a primitive type or custom type reference", typeStr),
				}}, nil
			}
		}
	}

	return nil, nil
}

// ValidateCrossFields validates relationships between array config fields
func (a *ArrayTypeConfig) ValidateCrossFields(config map[string]any) []rules.ValidationResult {
	var results []rules.ValidationResult

	// Check minItems <= maxItems
	minItems, hasMin := config["minItems"]
	maxItems, hasMax := config["maxItems"]

	if hasMin && hasMax {
		minVal, minOk := toInteger(minItems)
		maxVal, maxOk := toInteger(maxItems)

		if minOk && maxOk && minVal > maxVal {
			results = append(results, rules.ValidationResult{
				Reference: "",
				Message:   "minItems cannot be greater than maxItems",
			})
		}
	}

	return results
}
