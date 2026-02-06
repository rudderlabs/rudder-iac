package config

import (
	"fmt"

	catalogRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// StringTypeConfig validates config for string type
type StringTypeConfig struct{}

var allowedStringKeys = map[string]bool{
	"enum":      true,
	"minLength": true,
	"maxLength": true,
	"pattern":   true,
	"format":    true,
}

// ConfigAllowed returns true for string type
func (s *StringTypeConfig) ConfigAllowed() bool {
	return true
}

// ValidateField validates a single field for string type
func (s *StringTypeConfig) ValidateField(fieldname string, fieldval any) ([]rules.ValidationResult, error) {
	// Check if field is allowed
	if !allowedStringKeys[fieldname] {
		return nil, ErrFieldNotSupported
	}

	// Validate field value based on field name
	switch fieldname {
	case "enum":
		enumArray, ok := fieldval.([]any)
		if !ok {
			return []rules.ValidationResult{{
				Reference: fieldname,
				Message:   "'enum' must be an array",
			}}, nil
		}
		// Check for duplicates and create result for each duplicate index
		duplicateIndices := findDuplicateIndices(enumArray)
		if len(duplicateIndices) > 0 {
			var results []rules.ValidationResult
			for _, idx := range duplicateIndices {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("%s/%d", fieldname, idx),
					Message:   fmt.Sprintf("'%v' is a duplicate value", enumArray[idx]),
				})
			}
			return results, nil
		}

	case "minLength", "maxLength":
		if !isInteger(fieldval) {
			return []rules.ValidationResult{{
				Reference: fieldname,
				Message:   fmt.Sprintf("'%s' must be an integer", fieldname),
			}}, nil
		}
		// Check if value is non-negative
		val, _ := toInteger(fieldval)
		if val < 0 {
			return []rules.ValidationResult{{
				Reference: fieldname,
				Message:   fmt.Sprintf("'%s' must be >= 0", fieldname),
			}}, nil
		}

	case "pattern":
		_, ok := fieldval.(string)
		if !ok {
			return []rules.ValidationResult{{
				Reference: fieldname,
				Message:   "'pattern' must be a string",
			}}, nil
		}

	case "format":
		formatStr, ok := fieldval.(string)
		if !ok {
			return []rules.ValidationResult{{
				Reference: fieldname,
				Message:   "'format' must be a string",
			}}, nil
		}
		if !isValidFormat(formatStr) {
			return []rules.ValidationResult{{
				Reference: fieldname,
				Message:   fmt.Sprintf("'format' must be one of: %v", catalogRules.ValidFormatValues),
			}}, nil
		}
	}

	return nil, nil
}

// ValidateCrossFields validates relationships between string config fields
func (s *StringTypeConfig) ValidateCrossFields(config map[string]any) []rules.ValidationResult {
	var results []rules.ValidationResult

	// Check minLength <= maxLength
	minLength, hasMin := config["minLength"]
	maxLength, hasMax := config["maxLength"]

	if hasMin && hasMax {
		// Both must be valid integers for cross-field check
		minVal, minOk := toInteger(minLength)
		maxVal, maxOk := toInteger(maxLength)

		if minOk && maxOk && minVal > maxVal {
			results = append(results, rules.ValidationResult{
				Reference: "",
				Message:   "minLength cannot be greater than maxLength",
			})
		}
	}

	return results
}
