package funcs

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

func NewPrimitiveOrReference(
	primitives []string,
	refRegex *regexp.Regexp) rules.CustomValidateFunc {

	var fn validator.Func = func(fl validator.FieldLevel) bool {
		value := fl.Field().String()

		var (
			isPrimitive = isPrimitive(value, primitives)
			isReference = refRegex.MatchString(value)
		)

		// return true if it's either a primitive or a reference
		return isPrimitive || isReference
	}

	return rules.CustomValidateFunc{
		Tag:  "primitive_or_reference",
		Func: fn,
	}
}

func isPrimitive(value string, primitives []string) bool {
	typs := strings.Split(value, ",")
	typs = lo.Map(typs, func(item string, _ int) string {
		return strings.TrimSpace(item)
	})

	return lo.Every(primitives, typs)
}
