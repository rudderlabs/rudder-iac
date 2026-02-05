package config

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// EmptyTypeConfig validates config for empty type
type EmptyTypeConfig struct{}

func (e *EmptyTypeConfig) ConfigAllowed() bool {
	return false
}

// ValidateField validates a single field for custom type
func (e *EmptyTypeConfig) ValidateField(fieldname string, fieldval any) ([]rules.ValidationResult, error) {
	return nil, ErrFieldNotSupported
}

func (e *EmptyTypeConfig) ValidateCrossFields(config map[string]any) []rules.ValidationResult {
	return nil
}
