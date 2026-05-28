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
// Null/empty values are rejected. A YAML key with no value (`KEY:`) or an
// explicit null (`KEY: null`) returns ErrVarFileParseFailed — setting null
// values is not supported. To represent an empty value, use empty quotes:
// `KEY: ""`.
func NewFileResolver(path string) (Resolver, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: variable file %s", ErrVarFileNotFound, path)
		}
		return nil, fmt.Errorf("reading variable file %s: %w", path, err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: parsing variable file %s", ErrVarFileParseFailed, path)
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
			return nil, fmt.Errorf("%w: key %q has a null or empty value in %s. Setting null values is not supported; to set an empty value use empty quotes \"\"", ErrVarFileParseFailed, key, path)
		default:
			return nil, fmt.Errorf("%w: key %q has non-scalar value in %s", ErrVarFileParseFailed, key, path)
		}
	}

	return &fileResolver{vars: vars}, nil
}

func (r *fileResolver) Resolve(name string) (string, bool) {
	value, found := r.vars[name]
	return value, found
}
