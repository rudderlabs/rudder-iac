package registry

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules/datacatalog"
)

// NewDefaultRegistry creates a registry with all standard validation rules pre-registered
func NewDefaultRegistry() (*RuleRegistry, error) {
	registry := NewRegistry()

	// All available validation rules
	rules := []validation.Rule{
		&datacatalog.RequiredFieldsRule{},
		&datacatalog.CategoryNameRule{},
		&datacatalog.CustomTypeNameRule{},
	}

	for _, rule := range rules {
		if err := registry.Register(rule); err != nil {
			return nil, fmt.Errorf("registering rule %s: %w", rule.ID(), err)
		}
	}

	return registry, nil
}
