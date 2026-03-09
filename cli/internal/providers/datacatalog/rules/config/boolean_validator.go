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
func (b *BooleanTypeConfig) ValidateField(rawKey string, keyword ConfigKeyword, fieldval any) ([]rules.ValidationResult, error) {
	if !allowedBooleanKeys[keyword] {
		return nil, ErrFieldNotSupported
	}

	if keyword == KeywordEnum {
		return validateEnum(rawKey, fieldval)
	}

	return nil, ErrFieldNotSupported
}

// ValidateCrossFields validates relationships between boolean config fields.
func (b *BooleanTypeConfig) ValidateCrossFields(_ map[ConfigKeyword]any) []rules.ValidationResult {
	return nil
}
