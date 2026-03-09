package config

import (
	"fmt"

	catalogRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// StringTypeConfig validates config for string type.
type StringTypeConfig struct{}

var allowedStringKeys = map[ConfigKeyword]bool{
	KeywordEnum:      true,
	KeywordMinLength: true,
	KeywordMaxLength: true,
	KeywordPattern:   true,
	KeywordFormat:    true,
}

// ConfigAllowed returns true for string type.
func (s *StringTypeConfig) ConfigAllowed() bool {
	return true
}

// ValidateField validates a single field for string type.
func (s *StringTypeConfig) ValidateField(rawKey string, keyword ConfigKeyword, fieldval any) ([]rules.ValidationResult, error) {
	if !allowedStringKeys[keyword] {
		return nil, ErrFieldNotSupported
	}

	switch keyword {
	case KeywordEnum:
		return validateEnum(rawKey, fieldval)

	case KeywordMinLength, KeywordMaxLength:
		if !isInteger(fieldval) {
			return []rules.ValidationResult{{
				Reference: rawKey,
				Message:   fmt.Sprintf("'%s' must be an integer", rawKey),
			}}, nil
		}
		val, _ := toInteger(fieldval)
		if val < 0 {
			return []rules.ValidationResult{{
				Reference: rawKey,
				Message:   fmt.Sprintf("'%s' must be >= 0", rawKey),
			}}, nil
		}

	case KeywordPattern:
		if _, ok := fieldval.(string); !ok {
			return []rules.ValidationResult{{
				Reference: rawKey,
				Message:   fmt.Sprintf("'%s' must be a string", rawKey),
			}}, nil
		}

	case KeywordFormat:
		formatStr, ok := fieldval.(string)
		if !ok {
			return []rules.ValidationResult{{
				Reference: rawKey,
				Message:   fmt.Sprintf("'%s' must be a string", rawKey),
			}}, nil
		}
		if !isValidFormat(formatStr) {
			return []rules.ValidationResult{{
				Reference: rawKey,
				Message:   fmt.Sprintf("'format' must be one of: %v", catalogRules.ValidFormatValues),
			}}, nil
		}
	}

	return nil, nil
}

// ValidateCrossFields validates relationships between string config fields.
func (s *StringTypeConfig) ValidateCrossFields(config map[ConfigKeyword]any) []rules.ValidationResult {
	var results []rules.ValidationResult

	minLength, hasMin := config[KeywordMinLength]
	maxLength, hasMax := config[KeywordMaxLength]

	if hasMin && hasMax {
		minVal, minOk := toInteger(minLength)
		maxVal, maxOk := toInteger(maxLength)

		if minOk && maxOk && minVal > maxVal {
			results = append(results, rules.ValidationResult{
				Reference: "",
				Message:   fmt.Sprintf("%s cannot be greater than %s", KeywordMinLength, KeywordMaxLength),
			})
		}
	}

	return results
}
