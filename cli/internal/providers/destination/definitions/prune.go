package definitions

import (
	"strings"
)

// PruneEmptyOptional returns a shallow copy of config with keys removed when
// their value is empty AND the field is not required given the config's other
// values. It exists for the export/import path: the backend returns the full
// config schema with every field populated (empty strings, empty arrays,
// arrays of empty objects), which otherwise clutters imported YAML with fields
// irrelevant to the destination's actual configuration — e.g. a noAuth HTTP
// destination importing empty username/password/token fields.
//
// Requiredness is evaluated with the same validator project validation uses, so
// conditionally-required fields (required_if) are honored: an empty field that
// is required given the current sibling values (e.g. password when
// auth=basicAuth) is kept, while an empty field that is not required
// (password when auth=noAuth) is dropped. Non-empty values are always kept, so
// user-set optional fields survive.
//
// This never runs in the apply/converter path — pruning empty values there would
// cause phantom, non-converging diffs. Export only.
func (d *RegisteredDefinition) PruneEmptyOptional(config map[string]any) map[string]any {
	if len(config) == 0 {
		return config
	}

	pruned := make(map[string]any, len(config))
	var dropped []string
	for key, value := range config {
		if isEmptyConfigValue(value) {
			dropped = append(dropped, key)
			continue
		}
		pruned[key] = value
	}

	if len(dropped) == 0 {
		return pruned
	}

	// Restore any dropped key the validator now flags as required. Restoring a
	// field only satisfies requiredness, so a single pass reaches a fixed point.
	requiredPaths := requiredErrorPaths(d.ValidateConfig(pruned))
	for _, key := range dropped {
		if requiredPaths["/"+key] {
			pruned[key] = config[key]
		}
	}

	return pruned
}

// requiredErrorPaths collects the JSON-pointer paths of required / required_if
// validation failures. Required messages are the only ones containing the word
// "required" (pattern/oneof/type errors use distinct wording).
func requiredErrorPaths(errors []ConfigError) map[string]bool {
	paths := make(map[string]bool)
	for _, e := range errors {
		if strings.Contains(e.Message, "required") {
			paths[e.Path] = true
		}
	}
	return paths
}

// isEmptyConfigValue reports whether a decoded config value carries no
// meaningful data: the zero string, nil, an empty/all-empty slice, or an
// empty/all-empty map. Arrays of empty objects (e.g. the backend's
// [{"eventName":""}] event filter) are treated as empty.
func isEmptyConfigValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return v == ""
	case bool:
		// A bool carries meaning (false is a real setting), never prune it.
		return false
	case []any:
		for _, item := range v {
			if !isEmptyConfigValue(item) {
				return false
			}
		}
		return true
	case map[string]any:
		for _, item := range v {
			if !isEmptyConfigValue(item) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
