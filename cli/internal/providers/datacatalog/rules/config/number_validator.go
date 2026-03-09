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
func (n *NumberTypeConfig) ValidateField(rawKey string, keyword ConfigKeyword, fieldval any) ([]rules.ValidationResult, error) {
	if !allowedNumberKeys[keyword] {
		return nil, ErrFieldNotSupported
	}

	switch keyword {
	case KeywordEnum:
		return validateEnum(rawKey, fieldval)

	case KeywordMultipleOf:
		if !isNumber(fieldval) {
			return []rules.ValidationResult{{
				Reference: rawKey,
				Message:   fmt.Sprintf("'%s' must be a number", rawKey),
			}}, nil
		}

		val, _ := toNumber(fieldval)
		if val <= 0 {
			return []rules.ValidationResult{{
				Reference: rawKey,
				Message:   fmt.Sprintf("'%s' must be > 0", rawKey),
			}}, nil
		}

		return nil, nil

	default:
		if !isNumber(fieldval) {
			return []rules.ValidationResult{{
				Reference: rawKey,
				Message:   fmt.Sprintf("'%s' must be a number", rawKey),
			}}, nil
		}

		return nil, nil
	}
}

// ValidateCrossFields validates relationships between number config fields.
func (n *NumberTypeConfig) ValidateCrossFields(config map[ConfigKeyword]any) []rules.ValidationResult {
	var results []rules.ValidationResult

	minimum, hasMin := config[KeywordMinimum]
	maximum, hasMax := config[KeywordMaximum]

	if hasMin && hasMax {
		minVal, minOk := toNumber(minimum)
		maxVal, maxOk := toNumber(maximum)

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
		exMinVal, exMinOk := toNumber(exMinimum)
		exMaxVal, exMaxOk := toNumber(exMaximum)

		if exMinOk && exMaxOk && exMinVal >= exMaxVal {
			results = append(results, rules.ValidationResult{
				Reference: "",
				Message:   fmt.Sprintf("%s must be less than %s", KeywordExclusiveMinimum, KeywordExclusiveMaximum),
			})
		}
	}

	return results
}
