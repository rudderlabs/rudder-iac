package config

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// CustomTypeConfig validates config for custom type references.
// Config is not allowed for custom types at the top-level type position.
type CustomTypeConfig struct{}

// ConfigAllowed returns false for custom types.
func (c *CustomTypeConfig) ConfigAllowed() bool {
	return false
}

// ValidateField validates a single field for custom type.
func (c *CustomTypeConfig) ValidateField(_ string, _ ConfigKeyword, _ any) ([]rules.ValidationResult, error) {
	return nil, ErrFieldNotSupported
}

// ValidateCrossFields validates relationships between custom type config fields.
func (c *CustomTypeConfig) ValidateCrossFields(_ map[ConfigKeyword]any) []rules.ValidationResult {
	return nil
}
