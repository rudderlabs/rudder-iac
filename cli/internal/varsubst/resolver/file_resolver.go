package resolver

import (
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
	"gopkg.in/yaml.v3"
)

type fileResolver struct {
	vars map[string]string
}

func NewFileResolver(path string) (Resolver, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", varsubst.ErrVarFileNotFound, path)
		}
		return nil, fmt.Errorf("reading variable file %s: %w", path, err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: %s", varsubst.ErrVarFileParseFailed, path)
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
			vars[key] = ""
		default:
			return nil, fmt.Errorf("%w: key %q has non-scalar value in %s", varsubst.ErrVarFileParseFailed, key, path)
		}
	}

	return &fileResolver{vars: vars}, nil
}

func (r *fileResolver) Resolve(name string) (string, bool) {
	value, found := r.vars[name]
	return value, found
}
