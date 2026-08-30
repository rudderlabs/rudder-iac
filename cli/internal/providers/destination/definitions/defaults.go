package definitions

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// configDefaultTag mirrors the `default` keyword in the destination's
// integrations-config schema.json.
const configDefaultTag = "default"

// buildConfigDefaults collects the declared defaults keyed by local config key,
// typed to match what that key carries once decoded from YAML or converted from
// an API response.
//
// Nested config blocks are walked recursively: a `default` tag on an inner
// field is collected as a nested map under the parent key. Nested defaults
// follow the backend's JSON-Schema semantics — they fill in only when the
// parent block itself is present, never materializing an absent block (see
// ApplyDefaults).
func buildConfigDefaults(configType reflect.Type) (map[string]any, error) {
	configType = derefType(configType)
	fields := structFieldsByMapstructureTag(configType)
	defaults := make(map[string]any)

	// Sorted so a definition with several bad tags always fails on the same one.
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		field := fields[key]
		raw, ok := field.Tag.Lookup(configDefaultTag)
		if !ok {
			if derefType(field.Type).Kind() != reflect.Struct {
				continue
			}
			nested, err := buildConfigDefaults(field.Type)
			if err != nil {
				return nil, fmt.Errorf("config key %q: %w", key, err)
			}
			if len(nested) > 0 {
				defaults[key] = nested
			}
			continue
		}

		// A required key is always present, so its default could never apply.
		if isRequiredField(field) {
			return nil, fmt.Errorf("config key %q is required and cannot declare a default", key)
		}

		value, err := parseDefaultValue(field, raw)
		if err != nil {
			return nil, fmt.Errorf("config key %q: %w", key, err)
		}
		defaults[key] = value
	}

	return defaults, nil
}

// isRequiredField matches the bare `required` rule only: conditional forms
// (required_if, required_with) leave the field optional, so they may carry a
// default.
func isRequiredField(field reflect.StructField) bool {
	return slices.Contains(strings.Split(field.Tag.Get("validate"), ","), "required")
}

// parseDefaultValue converts the tag's string form to the field's type.
//
// String, bool and integer are the only JSON types upstream schemas default
// today. Any other kind is rejected at registration rather than guessed at, so
// widening this switch stays a deliberate act.
func parseDefaultValue(field reflect.StructField, raw string) (any, error) {
	switch kind := derefType(field.Type).Kind(); kind {
	case reflect.String:
		return raw, nil

	case reflect.Bool:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid bool default %q", raw)
		}
		return value, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Parsed as an integer to reject "1.5", stored as float64 so it compares
		// equal to the same number decoded from a JSON API response.
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid integer default %q", raw)
		}
		return float64(value), nil

	default:
		return nil, fmt.Errorf("unsupported kind %s for a default tag", kind)
	}
}

// ApplyDefaults returns a copy of config with the declared defaults filled in
// for omitted keys. A present key is never overwritten, so an explicit value
// wins even when it equals the zero value. Nested defaults fill in only inside
// a block the spec already carries — an absent block stays absent, matching
// how the backend's schema validation applies nested defaults.
//
// Local specs are enriched before they enter the resource graph: the backend
// applies these same defaults when it persists a destination, so a spec that
// omits one would otherwise diff forever against the remote state carrying it.
func (d *RegisteredDefinition) ApplyDefaults(config map[string]any) map[string]any {
	return applyDefaults(config, d.configDefaults)
}

func applyDefaults(config, defaults map[string]any) map[string]any {
	enriched := make(map[string]any, len(config)+len(defaults))
	maps.Copy(enriched, config)

	for key, value := range defaults {
		nested, isNested := value.(map[string]any)
		if !isNested {
			if _, ok := enriched[key]; !ok {
				enriched[key] = value
			}
			continue
		}

		current, ok := enriched[key].(map[string]any)
		if !ok {
			continue
		}
		enriched[key] = applyDefaults(current, nested)
	}

	return enriched
}

// ConfigDefaults returns the declared defaults keyed by local config key,
// deep-copied so callers cannot mutate the registered nested defaults.
func (d *RegisteredDefinition) ConfigDefaults() map[string]any {
	return cloneDefaults(d.configDefaults)
}

func cloneDefaults(defaults map[string]any) map[string]any {
	out := make(map[string]any, len(defaults))
	for key, value := range defaults {
		if nested, ok := value.(map[string]any); ok {
			out[key] = cloneDefaults(nested)
			continue
		}
		out[key] = value
	}
	return out
}
