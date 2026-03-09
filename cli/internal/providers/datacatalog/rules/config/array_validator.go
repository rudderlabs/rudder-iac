package config

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// ArrayTypeConfig validates config for array type.
type ArrayTypeConfig struct {
	// isCustomTypeRef is the caller-supplied matcher for custom type references.
	// Used for both top-level type resolution and itemTypes entry validation.
	isCustomTypeRef func(string) bool
}

var allowedArrayKeys = map[ConfigKeyword]bool{
	KeywordItemTypes:   true,
	KeywordMinItems:    true,
	KeywordMaxItems:    true,
	KeywordUniqueItems: true,
}

// ConfigAllowed returns true for array type.
func (a *ArrayTypeConfig) ConfigAllowed() bool {
	return true
}

// ValidateField validates a single field for array type.
func (a *ArrayTypeConfig) ValidateField(field ResolvedField) ([]rules.ValidationResult, error) {
	if !allowedArrayKeys[field.Keyword] {
		return nil, ErrFieldNotSupported
	}

	switch field.Keyword {
	case KeywordMinItems, KeywordMaxItems:
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

	case KeywordUniqueItems:
		if _, ok := field.Value.(bool); !ok {
			return []rules.ValidationResult{{
				Reference: field.RawKey,
				Message:   fmt.Sprintf("'%s' must be a boolean", field.RawKey),
			}}, nil
		}

	case KeywordItemTypes:
		return a.validateItemTypes(field.RawKey, field.Value)
	}

	return nil, nil
}

func (a *ArrayTypeConfig) validateItemTypes(rawKey string, fieldval any) ([]rules.ValidationResult, error) {
	itemTypesArray, ok := fieldval.([]any)
	if !ok {
		return []rules.ValidationResult{{
			Reference: rawKey,
			Message:   fmt.Sprintf("'%s' must be an array", rawKey),
		}}, nil
	}

	if len(itemTypesArray) == 0 {
		return []rules.ValidationResult{{
			Reference: rawKey,
			Message:   fmt.Sprintf("'%s' must contain at least one type", rawKey),
		}}, nil
	}

	for idx, itemType := range itemTypesArray {
		typeStr, ok := itemType.(string)
		if !ok {
			return []rules.ValidationResult{{
				Reference: fmt.Sprintf("%s/%d", rawKey, idx),
				Message:   fmt.Sprintf("'%v' must be a string value", itemType),
			}}, nil
		}

		if a.isCustomTypeRef != nil && a.isCustomTypeRef(typeStr) {
			// Custom type reference cannot be paired with other types.
			if len(itemTypesArray) > 1 {
				return []rules.ValidationResult{{
					Reference: fmt.Sprintf("%s/%d", rawKey, idx),
					Message:   fmt.Sprintf("'%v' custom type reference cannot be paired with other types", itemType),
				}}, nil
			}
			continue
		}

		if !isValidPrimitiveType(typeStr) {
			return []rules.ValidationResult{{
				Reference: fmt.Sprintf("%s/%d", rawKey, idx),
				Message:   fmt.Sprintf("invalid type '%s' in %s, must be a primitive type or custom type reference", typeStr, rawKey),
			}}, nil
		}
	}

	return nil, nil
}

// ValidateCrossFields validates relationships between array config fields.
func (a *ArrayTypeConfig) ValidateCrossFields(config map[ConfigKeyword]ResolvedField) []rules.ValidationResult {
	var results []rules.ValidationResult

	minField, hasMin := config[KeywordMinItems]
	maxField, hasMax := config[KeywordMaxItems]

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
