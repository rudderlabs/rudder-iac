package config

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// NumberTypeConfig validates config for number type
type NumberTypeConfig struct{}

var allowedNumberKeys = map[string]bool{
	"enum":             true,
	"minimum":          true,
	"maximum":          true,
	"exclusiveMinimum": true,
	"exclusiveMaximum": true,
	"multipleOf":       true,
}

// ConfigAllowed returns true for number type
func (n *NumberTypeConfig) ConfigAllowed() bool {
	return true
}

// ValidateField validates a single field for number type
func (n *NumberTypeConfig) ValidateField(fieldname string, fieldval any) ([]rules.ValidationResult, error) {
	if !allowedNumberKeys[fieldname] {
		return nil, ErrFieldNotSupported
	}

	// Special handling for enum
	if fieldname == "enum" {
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

		return nil, nil
	}

	// All other number fields must be numbers
	if !isNumber(fieldval) {
		return []rules.ValidationResult{{
			Reference: fieldname,
			Message:   fmt.Sprintf("'%s' must be a number", fieldname),
		}}, nil
	}

	if fieldname == "multipleOf" {
		val, _ := toNumber(fieldval)
		if val <= 0 {
			return []rules.ValidationResult{{
				Reference: fieldname,
				Message:   "'multipleOf' must be > 0",
			}}, nil
		}
	}

	return nil, nil
}

// ValidateCrossFields validates relationships between number config fields
func (n *NumberTypeConfig) ValidateCrossFields(config map[string]any) []rules.ValidationResult {
	var results []rules.ValidationResult

	// Check minimum <= maximum
	minimum, hasMin := config["minimum"]
	maximum, hasMax := config["maximum"]

	if hasMin && hasMax {
		minVal, minOk := toNumber(minimum)
		maxVal, maxOk := toNumber(maximum)

		if minOk && maxOk && minVal > maxVal {
			results = append(results, rules.ValidationResult{
				Reference: "",
				Message:   "minimum cannot be greater than maximum",
			})
		}
	}

	// Check exclusiveMinimum < exclusiveMaximum
	exMinimum, hasExMin := config["exclusiveMinimum"]
	exMaximum, hasExMax := config["exclusiveMaximum"]

	if hasExMin && hasExMax {
		exMinVal, exMinOk := toNumber(exMinimum)
		exMaxVal, exMaxOk := toNumber(exMaximum)

		if exMinOk && exMaxOk && exMinVal >= exMaxVal {
			results = append(results, rules.ValidationResult{
				Reference: "",
				Message:   "exclusiveMinimum must be less than exclusiveMaximum",
			})
		}
	}

	return results
}
