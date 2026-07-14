package definitions

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var (
	uiEnvValueRegex      = regexp.MustCompile(`^env[.].+`)
	uiTemplateValueRegex = regexp.MustCompile(`^\{\{.*\|\|(.*)\}\}$`)
	varValueRegex        = regexp.MustCompile(`^\{\{\s*\.[A-Za-z_][A-Za-z0-9_]*(?:\s*\|\s*((?:[^}]|}[^}])*?))?\s*\}\}$`)
)

// IsDynamicConfigValue reports whether value uses RudderStack UI dynamic syntax
// (env.VAR or {{ path || fallback }}) or IaC variable substitution
// ({{ .VAR }} / {{ .VAR | default }}). These are accepted as opaque pass-through
// values, matching terraform-provider-rudderstack field validators.
func IsDynamicConfigValue(value string) bool {
	if value == "" {
		return false
	}

	return uiEnvValueRegex.MatchString(value) ||
		uiTemplateValueRegex.MatchString(value) ||
		varValueRegex.MatchString(value)
}

func configValidateFuncs() []rules.CustomValidateFunc {
	return []rules.CustomValidateFunc{
		{
			Tag:  "dynamic_or_oneof",
			Func: dynamicOrOneOf,
		},
	}
}

func dynamicOrOneOf(fl validator.FieldLevel) bool {
	value, ok := stringFieldValue(fl)
	if !ok || value == "" {
		return true
	}

	if IsDynamicConfigValue(value) {
		return true
	}

	for _, option := range strings.Fields(fl.Param()) {
		if value == option {
			return true
		}
	}

	return false
}

func stringFieldValue(fl validator.FieldLevel) (string, bool) {
	field := fl.Field()

	switch field.Kind() {
	case reflect.String:
		return field.String(), true

	case reflect.Pointer:
		if field.IsNil() {
			return "", true
		}
		if field.Elem().Kind() != reflect.String {
			return "", false
		}
		return field.Elem().String(), true

	default:
		return "", false
	}
}
