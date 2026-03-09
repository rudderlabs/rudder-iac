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
func (i *IntegerTypeConfig) ValidateField(rawKey string, keyword ConfigKeyword, fieldval any) ([]rules.ValidationResult, error) {
	if !allowedIntegerKeys[keyword] {
		return nil, ErrFieldNotSupported
	}

	switch keyword {
	case KeywordEnum:
		return validateEnum(rawKey, fieldval)

	case KeywordMultipleOf:
		if !isInteger(fieldval) {
			return []rules.ValidationResult{{
				Reference: rawKey,
				Message:   fmt.Sprintf("'%s' must be an integer", rawKey),
			}}, nil
		}

		val, _ := toInteger(fieldval)
		if val <= 0 {
			return []rules.ValidationResult{{
				Reference: rawKey,
				Message:   fmt.Sprintf("'%s' must be > 0", rawKey),
			}}, nil
		}

		return nil, nil

	default:
		if !isInteger(fieldval) {
			return []rules.ValidationResult{{
				Reference: rawKey,
				Message:   fmt.Sprintf("'%s' must be an integer", rawKey),
			}}, nil
		}

		return nil, nil
	}
}

// ValidateCrossFields validates relationships between integer config fields.
func (i *IntegerTypeConfig) ValidateCrossFields(config map[ConfigKeyword]any) []rules.ValidationResult {
	var results []rules.ValidationResult

	minimum, hasMin := config[KeywordMinimum]
	maximum, hasMax := config[KeywordMaximum]

	if hasMin && hasMax {
		minVal, minOk := toInteger(minimum)
		maxVal, maxOk := toInteger(maximum)

		if minOk && maxOk && minVal > maxVal {
			results = append(results, rules.ValidationResult{
				Reference: "",
				Message:   fmt.Sprintf("%s cannot be greater than %s", KeywordMinimum, KeywordMaximum),
			})
		}
	}

	exMinimum, hasExMin := config[KeywordExclusiveMinimum]
	exMaximum, hasExMax := config[KeywordExclusiveMaximum]

	if hasExMin && hasExMax {
		exMinVal, exMinOk := toInteger(exMinimum)
		exMaxVal, exMaxOk := toInteger(exMaximum)

		if exMinOk && exMaxOk && exMinVal >= exMaxVal {
			results = append(results, rules.ValidationResult{
				Reference: "",
				Message:   fmt.Sprintf("%s must be less than %s", KeywordExclusiveMinimum, KeywordExclusiveMaximum),
			})
		}
	}

	return results
}
