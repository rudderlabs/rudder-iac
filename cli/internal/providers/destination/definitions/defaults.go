package definitions

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// configDefaultTag declares a config field's default value on the definition's
// config struct, mirroring the `default` keyword in the destination's
// integrations-config schema.json.
const configDefaultTag = "default"

// buildConfigDefaults reads `default:"..."` tags off the config model and
// returns them keyed by local config key, typed to match what the same key
// carries once decoded from YAML or converted from an API response.
//
// Only top-level fields are considered: schema.json declares its defaults at
// the top level, and a nested default would have no unambiguous home when the
// parent block itself is absent.
func buildConfigDefaults(configType reflect.Type) (map[string]any, error) {
	configType = derefType(configType)
	fields := structFieldsByMapstructureTag(configType)
	defaults := make(map[string]any)

	// Sorted so a definition with more than one bad tag always fails on the
	// same key.
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		field := fields[key]
		raw, ok := field.Tag.Lookup(configDefaultTag)
		if !ok {
			continue
		}

		// A required field is always present, so a default for it is dead
		// config that would silently never apply.
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

// isRequiredField reports whether the field carries the unconditional
// `required` validation rule. Conditional rules (required_if, required_with)
// leave the field genuinely optional, so they do not conflict with a default.
func isRequiredField(field reflect.StructField) bool {
	return slices.Contains(strings.Split(field.Tag.Get("validate"), ","), "required")
}

// parseDefaultValue converts the tag's string form to the field's type.
// Numbers become float64 to match JSON decoding, so a defaulted value compares
// equal to the same value read back from the API.
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

	// Integer kinds are range-checked against their own parser first — float
	// parsing alone would accept "1.5" for an int or "-1" for a uint.
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if _, err := strconv.ParseInt(raw, 10, 64); err != nil {
			return nil, fmt.Errorf("invalid integer default %q", raw)
		}
		return parseNumericDefault(raw)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if _, err := strconv.ParseUint(raw, 10, 64); err != nil {
			return nil, fmt.Errorf("invalid unsigned integer default %q", raw)
		}
		return parseNumericDefault(raw)

	case reflect.Float32, reflect.Float64:
		return parseNumericDefault(raw)

	default:
		return nil, fmt.Errorf("unsupported kind %s for a default tag", kind)
	}
}

// parseNumericDefault returns the default as float64 — the type JSON decoding
// yields — so a defaulted number compares equal to the same value read back
// from the API.
func parseNumericDefault(raw string) (any, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid numeric default %q", raw)
	}
	return value, nil
}

// ApplyDefaults returns a copy of config with every declared default filled in
// for keys the caller omitted. Present keys are never overwritten, so an
// explicit value — including one equal to the zero value — always wins.
//
// Local specs are enriched with this before they enter the resource graph: the
// backend applies the same schema defaults when it persists a destination, so
// without it a spec that omits a defaulted key would diff forever against the
// remote state that carries it.
func (d *RegisteredDefinition) ApplyDefaults(config map[string]any) map[string]any {
	enriched := make(map[string]any, len(config)+len(d.configDefaults))
	maps.Copy(enriched, config)

	for key, value := range d.configDefaults {
		if _, ok := enriched[key]; !ok {
			enriched[key] = value
		}
	}

	return enriched
}

// ConfigDefaults returns the declared defaults keyed by local config key.
// Values are scalars, so the copy shields the definition from callers.
func (d *RegisteredDefinition) ConfigDefaults() map[string]any {
	out := make(map[string]any, len(d.configDefaults))
	maps.Copy(out, d.configDefaults)
	return out
}
