package provider

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// SpecValidationRule validates specs using go-validator tags.
// This generic rule works with any SpecFactory to perform struct validation
// on the spec's entities (properties, events, categories, etc.).
type SpecValidationRule struct {
	factory SpecFactory
}

// NewSpecValidationRule creates a new validation rule for the given spec factory.
func NewSpecValidationRule(factory SpecFactory) rules.Rule {
	return &SpecValidationRule{factory: factory}
}

func (r *SpecValidationRule) ID() string {
	return r.factory.Kind() + "/spec-syntax-valid"
}

func (r *SpecValidationRule) Severity() rules.Severity {
	return rules.Error
}

func (r *SpecValidationRule) Description() string {
	return "validates " + r.factory.Kind() + " spec structure"
}

func (r *SpecValidationRule) AppliesTo() []string {
	return []string{r.factory.Kind()}
}

func (r *SpecValidationRule) Examples() rules.Examples {
	return r.factory.Examples()
}

func (r *SpecValidationRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	spec := r.factory.NewSpec()

	// Decode the spec into the factory's struct type
	if err := mapstructure.Decode(ctx.Spec, spec); err != nil {
		fieldName := r.factory.SpecFieldName()

		return []rules.ValidationResult{{
			Message:   fmt.Sprintf("%s spec structure should be valid: %s", fieldName, err.Error()),
			Reference: fmt.Sprintf("/%s", fieldName),
		}}
	}

	// Validate the struct using go-validator tags
	results, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Reference: fmt.Sprintf("/%s", r.factory.SpecFieldName()),
			Message:   "validation error: " + err.Error(),
		}}
	}

	return results
}
