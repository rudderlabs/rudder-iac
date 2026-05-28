package resolver

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type fileResolver struct {
	vars map[string]string
}

// NewFileResolver loads variables from a flat YAML file whose top-level keys
// map to scalar values (string, int, float, bool).
//
// Nil values are rejected. A YAML key with no value (`KEY:`) or an explicit
// null (`KEY: null`) returns ErrIllegalArgument because the intent is
// ambiguous. To represent an empty string, use explicit quotes: `KEY: ""` or
// `KEY: ''`.
func NewFileResolver(path string) (Resolver, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: variable file %s", ErrNotFound, path)
		}
		return nil, fmt.Errorf("reading variable file %s: %w", path, err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: parsing variable file %s", ErrIllegalArgument, path)
	}

	vars := make(map[string]string, len(raw))
	for key, val := range raw {
		switch v := val.(type) {
		case string:
			vars[key] = v
		case int:
			vars[key] = fmt.Sprint(v)
		case float64:
			vars[key] = fmt.Sprint(v)
		case bool:
			vars[key] = fmt.Sprint(v)
		case nil:
			return nil, fmt.Errorf("%w: key %q has nil value in %s (use \"\" or '' for empty strings)", ErrIllegalArgument, key, path)
		default:
			return nil, fmt.Errorf("%w: key %q has non-scalar value in %s", ErrIllegalArgument, key, path)
		}
	}

	return &fileResolver{vars: vars}, nil
}

func (r *fileResolver) Resolve(name string) (string, bool) {
	value, found := r.vars[name]
	return value, found
}
