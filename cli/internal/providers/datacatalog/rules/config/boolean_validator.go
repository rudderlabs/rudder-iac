package config

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// BooleanTypeConfig validates config for boolean type.
type BooleanTypeConfig struct{}

var allowedBooleanKeys = map[ConfigKeyword]bool{
	KeywordEnum: true,
}

// ConfigAllowed returns true for boolean type.
func (b *BooleanTypeConfig) ConfigAllowed() bool {
	return true
}

// ValidateField validates a single field for boolean type.
func (b *BooleanTypeConfig) ValidateField(field ResolvedField) ([]rules.ValidationResult, error) {
	if !allowedBooleanKeys[field.Keyword] {
		return nil, ErrFieldNotSupported
	}

	if field.Keyword == KeywordEnum {
		return validateEnum(field.RawKey, field.Value)
	}

	return nil, ErrFieldNotSupported
}

// ValidateCrossFields validates relationships between boolean config fields.
func (b *BooleanTypeConfig) ValidateCrossFields(_ map[ConfigKeyword]ResolvedField) []rules.ValidationResult {
	return nil
}
