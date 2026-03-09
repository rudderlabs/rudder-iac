package config

import (
	"fmt"
	"strings"

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
func (s *StringTypeConfig) ValidateField(field ResolvedField) ([]rules.ValidationResult, error) {
	if !allowedStringKeys[field.Keyword] {
		return nil, ErrFieldNotSupported
	}

	switch field.Keyword {
	case KeywordEnum:
		return validateEnum(field.RawKey, field.Value)

	case KeywordMinLength, KeywordMaxLength:
		if !isInteger(field.Value) {
			return []rules.ValidationResult{{
				Reference: field.RawKey,
				Message:   fmt.Sprintf("'%s' must be an integer", field.RawKey),
			}}, nil
		}
		val, _ := toInteger(field.Value)
		if val < 0 {
			return []rules.ValidationResult{{
				Reference: field.RawKey,
				Message:   fmt.Sprintf("'%s' must be >= 0", field.RawKey),
			}}, nil
		}

	case KeywordPattern:
		if _, ok := field.Value.(string); !ok {
			return []rules.ValidationResult{{
				Reference: field.RawKey,
				Message:   fmt.Sprintf("'%s' must be a string", field.RawKey),
			}}, nil
		}

	case KeywordFormat:
		formatStr, ok := field.Value.(string)
		if !ok {
			return []rules.ValidationResult{{
				Reference: field.RawKey,
				Message:   fmt.Sprintf("'%s' must be a string", field.RawKey),
			}}, nil
		}
		if !isValidFormat(formatStr) {
			return []rules.ValidationResult{{
				Reference: field.RawKey,
				Message: fmt.Sprintf("'%s' must be one of: [%s]",
					field.RawKey,
					strings.Join(catalogRules.ValidFormatValues, ", "),
				),
			}}, nil
		}
	}

	return nil, nil
}

// ValidateCrossFields validates relationships between string config fields.
func (s *StringTypeConfig) ValidateCrossFields(config map[ConfigKeyword]ResolvedField) []rules.ValidationResult {
	var results []rules.ValidationResult

	minField, hasMin := config[KeywordMinLength]
	maxField, hasMax := config[KeywordMaxLength]

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

	return results
}
