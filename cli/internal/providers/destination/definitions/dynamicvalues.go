package definitions

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
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

// IsTemplateConfigValue reports whether value uses the RudderStack UI template
// syntax `{{ path || fallback }}` — the only dynamic form schema.json patterns
// declare, and the one rudder-transformer resolves on the event path.
//
// Deliberately narrower than IsDynamicConfigValue. `env.VAR` is excluded because
// it is deprecated and resolves only in rudder-server, behind an enterprise
// handler and a flag that may be off, so an accepted value could reach the
// destination unsubstituted. `{{ .VAR }}` is excluded because CLI var
// substitution resolves it before validation runs; one still present is a
// mistake, and lacking `||` upstream would reject it too.
func IsTemplateConfigValue(value string) bool {
	return uiTemplateValueRegex.MatchString(value)
}

func configValidateFuncs() []rules.CustomValidateFunc {
	return []rules.CustomValidateFunc{
		{
			Tag:  "dynamic_or_oneof",
			Func: dynamicOrOneOf,
		},
		{
			Tag:  "dynamic_or_pattern",
			Func: dynamicOrPattern,
		},
	}
}

// dynamicOrPattern accepts a UI template value, otherwise defers to the named
// pattern. It is a strict superset of `pattern=<name>`: an absent optional field
// passes, and every other value must be a template or match the pattern — empty
// strings included, so the pattern stays the authority on whether "" is legal.
func dynamicOrPattern(fl validator.FieldLevel) bool {
	if field := fl.Field(); field.Kind() == reflect.Pointer && field.IsNil() {
		return true
	}

	value, ok := stringFieldValue(fl)
	if !ok {
		return false
	}

	if IsTemplateConfigValue(value) {
		return true
	}

	return funcs.MatchPattern(fl.Param(), value)
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
