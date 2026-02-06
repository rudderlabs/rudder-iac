package config

import (
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
		return validateEnum(fieldname, fieldval)
	}

	return nil, ErrFieldNotSupported
}

// ValidateCrossFields validates relationships between boolean config fields
func (b *BooleanTypeConfig) ValidateCrossFields(config map[string]any) []rules.ValidationResult {
	// No cross-field validation for boolean type
	return nil
}
