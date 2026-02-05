package config

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// CustomTypeConfig validates config for custom types (types starting with uppercase)
type CustomTypeConfig struct{}

// ConfigAllowed returns true for custom types
func (c *CustomTypeConfig) ConfigAllowed() bool {
	return false
}

// ValidateField validates a single field for custom type
func (c *CustomTypeConfig) ValidateField(fieldname string, fieldval any) ([]rules.ValidationResult, error) {
	// Custom types don't have specific type-level keywords
	// Per requirements, we don't validate $ref or other custom type specific fields
	return nil, ErrFieldNotSupported
}

// ValidateCrossFields validates relationships between custom type config fields
func (c *CustomTypeConfig) ValidateCrossFields(config map[string]any) []rules.ValidationResult {
	// No cross-field validation for custom types
	return nil
}
