package config

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// ObjectTypeConfig validates config for object type
type ObjectTypeConfig struct{}

var allowedObjectKeys = map[string]bool{}

// ConfigAllowed returns false for object type (config not allowed per spec)
func (o *ObjectTypeConfig) ConfigAllowed() bool {
	return false
}

// ValidateField validates a single field for object type
func (o *ObjectTypeConfig) ValidateField(fieldname string, fieldval any) ([]rules.ValidationResult, error) {
	if !allowedObjectKeys[fieldname] {
		return nil, ErrFieldNotSupported
	}

	return nil, nil
}

// ValidateCrossFields validates relationships between object config fields
func (o *ObjectTypeConfig) ValidateCrossFields(config map[string]any) []rules.ValidationResult {
	// No cross-field validation for object type
	return nil
}
