package config

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// IntegerTypeConfig validates config for integer type.
type IntegerTypeConfig struct{}

var allowedIntegerKeys = map[ConfigKeyword]bool{
	KeywordEnum:             true,
	KeywordMinimum:          true,
	KeywordMaximum:          true,
	KeywordExclusiveMinimum: true,
	KeywordExclusiveMaximum: true,
	KeywordMultipleOf:       true,
}

// ConfigAllowed returns true for integer type.
func (i *IntegerTypeConfig) ConfigAllowed() bool {
	return true
}

// ValidateField validates a single field for integer type.
func (i *IntegerTypeConfig) ValidateField(field ResolvedField) ([]rules.ValidationResult, error) {
	if !allowedIntegerKeys[field.Keyword] {
		return nil, ErrFieldNotSupported
	}

	switch field.Keyword {
	case KeywordEnum:
		return validateEnum(field.RawKey, field.Value)

	case KeywordMultipleOf:
		if !isInteger(field.Value) {
			return []rules.ValidationResult{{
				Reference: field.RawKey,
				Message:   fmt.Sprintf("'%s' must be an integer", field.RawKey),
			}}, nil
		}

		val, _ := toInteger(field.Value)
		if val <= 0 {
			return []rules.ValidationResult{{
				Reference: field.RawKey,
				Message:   fmt.Sprintf("'%s' must be > 0", field.RawKey),
			}}, nil
		}

		return nil, nil

	default:
		if !isInteger(field.Value) {
			return []rules.ValidationResult{{
				Reference: field.RawKey,
				Message:   fmt.Sprintf("'%s' must be an integer", field.RawKey),
			}}, nil
		}

		return nil, nil
	}
}

// ValidateCrossFields validates relationships between integer config fields.
func (i *IntegerTypeConfig) ValidateCrossFields(config map[ConfigKeyword]ResolvedField) []rules.ValidationResult {
	var results []rules.ValidationResult

	minField, hasMin := config[KeywordMinimum]
	maxField, hasMax := config[KeywordMaximum]

	if hasMin && hasMax {
		minVal, minOk := toInteger(minField.Value)
		maxVal, maxOk := toInteger(maxField.Value)

		if minOk && maxOk && minVal > maxVal {
			results = append(results, rules.ValidationResult{
				Reference: "",
				Message:   fmt.Sprintf("%s cannot be greater than %s", minField.RawKey, maxField.RawKey),
			})
		}
	}

	exMinField, hasExMin := config[KeywordExclusiveMinimum]
	exMaxField, hasExMax := config[KeywordExclusiveMaximum]

	if hasExMin && hasExMax {
		exMinVal, exMinOk := toInteger(exMinField.Value)
		exMaxVal, exMaxOk := toInteger(exMaxField.Value)

		if exMinOk && exMaxOk && exMinVal >= exMaxVal {
			results = append(results, rules.ValidationResult{
				Reference: "",
				Message:   fmt.Sprintf("%s must be less than %s", exMinField.RawKey, exMaxField.RawKey),
			})
		}
	}

	return results
}
