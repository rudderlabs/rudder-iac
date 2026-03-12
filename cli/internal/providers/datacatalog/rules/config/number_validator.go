package config

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// NumberTypeConfig validates config for number type.
type NumberTypeConfig struct{}

var allowedNumberKeys = map[ConfigKeyword]bool{
	KeywordEnum:             true,
	KeywordMinimum:          true,
	KeywordMaximum:          true,
	KeywordExclusiveMinimum: true,
	KeywordExclusiveMaximum: true,
	KeywordMultipleOf:       true,
}

// ConfigAllowed returns true for number type.
func (n *NumberTypeConfig) ConfigAllowed() bool {
	return true
}

// ValidateField validates a single field for number type.
func (n *NumberTypeConfig) ValidateField(field ResolvedField) ([]rules.ValidationResult, error) {
	if !allowedNumberKeys[field.Keyword] {
		return nil, ErrFieldNotSupported
	}

	switch field.Keyword {
	case KeywordEnum:
		return validateEnum(field.RawKey, field.Value)

	case KeywordMultipleOf:
		if !isNumber(field.Value) {
			return []rules.ValidationResult{{
				Reference: field.RawKey,
				Message:   fmt.Sprintf("'%s' must be a number", field.RawKey),
			}}, nil
		}

		val, _ := toNumber(field.Value)
		if val <= 0 {
			return []rules.ValidationResult{{
				Reference: field.RawKey,
				Message:   fmt.Sprintf("'%s' must be > 0", field.RawKey),
			}}, nil
		}

		return nil, nil

	default:
		if !isNumber(field.Value) {
			return []rules.ValidationResult{{
				Reference: field.RawKey,
				Message:   fmt.Sprintf("'%s' must be a number", field.RawKey),
			}}, nil
		}

		return nil, nil
	}
}

// ValidateCrossFields validates relationships between number config fields.
func (n *NumberTypeConfig) ValidateCrossFields(config map[ConfigKeyword]ResolvedField) []rules.ValidationResult {
	var results []rules.ValidationResult

	minField, hasMin := config[KeywordMinimum]
	maxField, hasMax := config[KeywordMaximum]

	if hasMin && hasMax {
		minVal, minOk := toNumber(minField.Value)
		maxVal, maxOk := toNumber(maxField.Value)

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
		exMinVal, exMinOk := toNumber(exMinField.Value)
		exMaxVal, exMaxOk := toNumber(exMaxField.Value)

		if exMinOk && exMaxOk && exMinVal >= exMaxVal {
			results = append(results, rules.ValidationResult{
				Reference: "",
				Message:   fmt.Sprintf("%s must be less than %s", exMinField.RawKey, exMaxField.RawKey),
			})
		}
	}

	return results
}
