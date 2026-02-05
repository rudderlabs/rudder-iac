package config

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// BooleanTypeConfig validates config for boolean type
type BooleanTypeConfig struct{}

var allowedBooleanKeys = map[string]bool{
	"enum": true,
}

// ConfigAllowed returns true for boolean type
func (b *BooleanTypeConfig) ConfigAllowed() bool {
	return true
}

// ValidateField validates a single field for boolean type
func (b *BooleanTypeConfig) ValidateField(fieldname string, fieldval any) ([]rules.ValidationResult, error) {
	// Check if field is allowed
	if !allowedBooleanKeys[fieldname] {
		return nil, ErrFieldNotSupported
	}

	// Only enum is allowed for boolean type
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

	return nil, ErrFieldNotSupported
}

// ValidateCrossFields validates relationships between boolean config fields
func (b *BooleanTypeConfig) ValidateCrossFields(config map[string]any) []rules.ValidationResult {
	// No cross-field validation for boolean type
	return nil
}
