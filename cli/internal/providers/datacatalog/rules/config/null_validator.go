package config

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// NullTypeConfig validates config for null type
type NullTypeConfig struct{}

// ConfigAllowed returns false for null type (config not allowed per spec)
func (n *NullTypeConfig) ConfigAllowed() bool {
	return false
}

// ValidateField validates a single field for null type
func (n *NullTypeConfig) ValidateField(fieldname string, fieldval any) ([]rules.ValidationResult, error) {
	// Null type has no allowed keywords
	return nil, ErrFieldNotSupported
}

// ValidateCrossFields validates relationships between null config fields
func (n *NullTypeConfig) ValidateCrossFields(config map[string]any) []rules.ValidationResult {
	// No cross-field validation for null type
	return nil
}