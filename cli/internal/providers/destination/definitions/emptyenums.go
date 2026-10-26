package definitions

import (
	"fmt"
	"slices"
	"strings"
)

// findEmptyEnumValues reports an explicit "" on a string enum unless the enum
// lists the empty string as a member, as schema.json does where the backend
// accepts it. Model validation can't tell: an omitted key decodes to the same
// "", and omitempty skips both.
func findEmptyEnumValues(config map[string]any, fields []Field) []ConfigError {
	var errors []ConfigError
	for _, field := range fields {
		if field.Type != "string" || len(field.Enum) == 0 || slices.Contains(field.Enum, any("")) {
			continue
		}
		walkConfigPath(config, strings.Split(field.Key, "."), "", func(value any, path string) {
			if value != "" {
				return
			}
			errors = append(errors, ConfigError{
				Path:    path,
				Message: enumErrorMessage(field),
			})
		})
	}
	return errors
}

func walkConfigPath(value any, segments []string, path string, visit func(any, string)) {
	if len(segments) == 0 {
		visit(value, path)
		return
	}

	name, list := strings.CutSuffix(segments[0], "[]")
	object, ok := value.(map[string]any)
	if !ok {
		return
	}
	child, present := object[name]
	if !present {
		return
	}

	childPath := joinConfigPath(path, name)
	if !list {
		walkConfigPath(child, segments[1:], childPath, visit)
		return
	}

	items, ok := child.([]any)
	if !ok {
		return
	}
	for index, item := range items {
		walkConfigPath(item, segments[1:], joinConfigPath(childPath, fmt.Sprint(index)), visit)
	}
}

func enumErrorMessage(field Field) string {
	key := strings.TrimPrefix(field.Key, parentKey(field.Key)+".")
	message := fmt.Sprintf("'%s' must be one of %v", strings.TrimSuffix(key, "[]"), field.Enum)
	if field.OpenEnum {
		message += " or a dynamic config value (env.VAR, {{ path || fallback }}, or {{ .VAR }})"
	}
	return message
}
