package configpath

import (
	"fmt"
	"strings"
	"unicode"
)

// Get returns the value at a dotted map path. Missing values are reported with
// ok=false; malformed paths return an error.
func Get(config map[string]any, path string) (any, bool, error) {
	segments, err := parse(path)
	if err != nil {
		return nil, false, err
	}
	if config == nil {
		return nil, false, nil
	}

	current := config
	for _, segment := range segments[:len(segments)-1] {
		next, ok := current[segment]
		if !ok {
			return nil, false, nil
		}
		nextMap, ok := next.(map[string]any)
		if !ok {
			return nil, false, nil
		}
		current = nextMap
	}

	value, ok := current[segments[len(segments)-1]]
	return value, ok, nil
}

// Set writes value at path, creating missing parent maps. Existing non-map
// parents reject the write so invalid dotted paths cannot silently reshape data.
func Set(config map[string]any, path string, value any) error {
	segments, err := parse(path)
	if err != nil {
		return err
	}
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	current := config
	for _, segment := range segments[:len(segments)-1] {
		next, ok := current[segment]
		if !ok {
			nextMap := map[string]any{}
			current[segment] = nextMap
			current = nextMap
			continue
		}
		nextMap, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("path %q parent %q is %T, not map[string]any", path, segment, next)
		}
		current = nextMap
	}

	current[segments[len(segments)-1]] = value
	return nil
}

// SetCopyOnWrite writes value at path on a root clone, cloning only ancestor maps
// along the edited path. Missing parent maps are created on the clone.
func SetCopyOnWrite(config map[string]any, path string, value any) (map[string]any, error) {
	segments, err := parse(path)
	if err != nil {
		return config, err
	}
	if config == nil {
		config = map[string]any{}
	}

	out := cloneMap(config)
	current := out
	for _, segment := range segments[:len(segments)-1] {
		next, ok := current[segment]
		if !ok {
			nextMap := map[string]any{}
			current[segment] = nextMap
			current = nextMap
			continue
		}
		nextMap, ok := next.(map[string]any)
		if !ok {
			return config, fmt.Errorf("path %q parent %q is %T, not map[string]any", path, segment, next)
		}
		cloned := cloneMap(nextMap)
		current[segment] = cloned
		current = cloned
	}

	current[segments[len(segments)-1]] = value
	return out, nil
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func parse(path string) ([]string, error) {
	segments := strings.Split(path, ".")
	for _, segment := range segments {
		if segment == "" {
			return nil, fmt.Errorf("path %q contains an empty segment", path)
		}
		if isNumeric(segment) {
			return nil, fmt.Errorf("path %q contains numeric segment %q; array indexes are not supported", path, segment)
		}
	}
	return segments, nil
}

func isNumeric(segment string) bool {
	for _, r := range segment {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
