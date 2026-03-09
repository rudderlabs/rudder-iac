package config

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// ObjectTypeConfig validates config for object type.
type ObjectTypeConfig struct{}

// ConfigAllowed returns false for object type (config not allowed per spec).
func (o *ObjectTypeConfig) ConfigAllowed() bool {
	return false
}

// ValidateField validates a single field for object type.
func (o *ObjectTypeConfig) ValidateField(_ string, _ ConfigKeyword, _ any) ([]rules.ValidationResult, error) {
	return nil, ErrFieldNotSupported
}

// ValidateCrossFields validates relationships between object config fields.
func (o *ObjectTypeConfig) ValidateCrossFields(_ map[ConfigKeyword]any) []rules.ValidationResult {
	return nil
}
