package funcs

import (
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

// NewPrimitive creates a validator that checks if a field contains a single primitive type
// from the allowed list of primitives
func NewPrimitive(primitives []string) rules.CustomValidateFunc {
	var fn validator.Func = func(fl validator.FieldLevel) bool {
		value := strings.TrimSpace(fl.Field().String())
		// Check if the value is one of the allowed primitives
		return lo.Contains(primitives, value)
	}

	return rules.CustomValidateFunc{
		Tag:  "primitive",
		Func: fn,
	}
}
