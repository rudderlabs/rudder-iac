package config

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// NullTypeConfig validates config for null type.
type NullTypeConfig struct{}

// ConfigAllowed returns false for null type (config not allowed per spec).
func (n *NullTypeConfig) ConfigAllowed() bool {
	return false
}

// ValidateField validates a single field for null type.
func (n *NullTypeConfig) ValidateField(_ ResolvedField) ([]rules.ValidationResult, error) {
	return nil, ErrFieldNotSupported
}

// ValidateCrossFields validates relationships between null config fields.
func (n *NullTypeConfig) ValidateCrossFields(_ map[ConfigKeyword]ResolvedField) []rules.ValidationResult {
	return nil
}
