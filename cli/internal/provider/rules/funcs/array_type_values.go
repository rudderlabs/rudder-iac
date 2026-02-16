package funcs

import (
	"math"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// GetArrayItemTypesValidator returns a default validator for the "array_item_types" tag.
// It validates that each element in a []any slice matches one of the allowed types
// specified as space-separated params.
//
// Usage: validate:"array_item_types=string bool integer"
//
// Supported type names:
//   - "string"  — Go string
//   - "bool"    — Go bool
//   - "integer" — Go float64 with no fractional part (JSON/YAML numbers are float64)
func GetArrayItemTypesValidator() rules.CustomValidateFunc {
	fn := validator.Func(func(fl validator.FieldLevel) bool {
		field := fl.Field()
		if field.Kind() != reflect.Slice {
			return false
		}

		allowedTypes := strings.Fields(fl.Param())

		for i := 0; i < field.Len(); i++ {
			elem := field.Index(i).Interface()
			if !matchesAllowedType(elem, allowedTypes) {
				return false
			}
		}

		return true
	})

	return rules.CustomValidateFunc{
		Tag:  "array_item_types",
		Func: fn,
	}
}

func matchesAllowedType(v any, allowedTypes []string) bool {
	for _, allowed := range allowedTypes {
		switch allowed {
		case "string":
			if _, ok := v.(string); ok {
				return true
			}
		case "bool":
			if _, ok := v.(bool); ok {
				return true
			}
		case "integer":
			if val, ok := v.(float64); ok && val == math.Trunc(val) {
				return true
			}
		}
	}

	return false
}
