package definitions

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// findNullValues reports every null in a spec's config. No destination
// schema.json admits null, so the backend rejects one, and dropping it instead
// leaves the spec diffing forever against a remote that never stores the key.
// consent_management is skipped because validateConsentManagement has its own
// null checks.
func findNullValues(value any, basePath string) []ConfigError {
	var errors []ConfigError
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			if basePath == "" && key == "consent_management" {
				continue
			}
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

// errorsOutsideNullPaths keeps the explicit null error authoritative for that
// value. Decoding null into a Go zero value can otherwise make model validation
// report a second, misleading error for the same path or one of its children.
func errorsOutsideNullPaths(errors, nullErrors []ConfigError) []ConfigError {
	return slices.DeleteFunc(errors, func(configError ConfigError) bool {
		return slices.ContainsFunc(nullErrors, func(nullError ConfigError) bool {
			return configError.Path == nullError.Path || strings.HasPrefix(configError.Path, nullError.Path+"/")
		})
	})
}
