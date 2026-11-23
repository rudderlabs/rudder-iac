package definitions

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// findNullValues reports every null in a spec's config. No destination
// schema.json admits null, so the backend rejects one, and dropping it instead
// leaves the spec diffing forever against a remote that never stores the key.
// consent_management and connection_mode are skipped because their validators
// have their own null checks, which also leave a null under an unsupported
// source type to be reported as unsupported.
func findNullValues(value any, basePath string) []ConfigError {
	var errors []ConfigError
	switch v := value.(type) {
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(v)) {
			if basePath == "" && (key == "consent_management" || key == "connection_mode") {
				continue
			}
			item := v[key]
			path := joinConfigPath(basePath, key)
			if item == nil {
				errors = append(errors, ConfigError{
					Path:    path,
					Message: fmt.Sprintf("'%s' must not be null; set a value or remove the key", key),
				})
				continue
			}
			errors = append(errors, findNullValues(item, path)...)
		}
	case []any:
		for i, item := range v {
			path := joinConfigPath(basePath, strconv.Itoa(i))
			if item == nil {
				errors = append(errors, ConfigError{Path: path, Message: "list item must not be null; remove it"})
				continue
			}
			errors = append(errors, findNullValues(item, path)...)
		}
	}
	return errors
}

// errorsOutside drops the errors at or under an owner error's path, so the
// owner is the one error reported there. A null or an empty enum value decodes
// to a Go zero value, so model validation can otherwise report a second,
// misleading error for the same path or one of its children.
func errorsOutside(errors, owners []ConfigError) []ConfigError {
	return slices.DeleteFunc(errors, func(configError ConfigError) bool {
		return slices.ContainsFunc(owners, func(owner ConfigError) bool {
			return configError.Path == owner.Path || strings.HasPrefix(configError.Path, owner.Path+"/")
		})
	})
}
