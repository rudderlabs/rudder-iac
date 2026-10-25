package definitions

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
)

// findNullValues reports every null in a spec's config. No destination
// schema.json admits null, so the backend rejects one, and dropping it instead
// leaves the spec diffing forever against a remote that never stores the key.
// consent_management is skipped because validateConsentManagement has its own
// null checks.
func findNullValues(value any, basePath string) []ConfigError {
	switch v := value.(type) {
	case map[string]any:
		var errors []ConfigError
		for _, key := range slices.Sorted(maps.Keys(v)) {
			if basePath == "" && key == "consent_management" {
				continue
			}
			path := joinConfigPath(basePath, key)
			if v[key] == nil {
				errors = append(errors, ConfigError{
					Path:    path,
					Message: fmt.Sprintf("'%s' must not be null; set a value or remove the key", key),
				})
				continue
			}
			errors = append(errors, findNullValues(v[key], path)...)
		}
		return errors
	case []any:
		var errors []ConfigError
		for i, item := range v {
			path := joinConfigPath(basePath, strconv.Itoa(i))
			if item == nil {
				errors = append(errors, ConfigError{Path: path, Message: "list item must not be null; remove it"})
				continue
			}
			errors = append(errors, findNullValues(item, path)...)
		}
		return errors
	default:
		return nil
	}
}
