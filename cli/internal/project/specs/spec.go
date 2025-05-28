package specs

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Spec struct {
	Version  string                 `yaml:"version"`
	Kind     string                 `yaml:"kind"`
	Metadata map[string]interface{} `yaml:"metadata"`
	Spec     map[string]interface{} `yaml:"spec"`
}

// New creates and validates a Spec from YAML data
func New(data []byte) (*Spec, error) {
	var spec Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("unmarshaling yaml: %w", err)
	}

	if spec.Version == "" {
		return nil, fmt.Errorf("missing required field 'version'")
	}
	if spec.Kind == "" {
		return nil, fmt.Errorf("missing required field 'kind'")
	}
	if spec.Metadata == nil {
		return nil, fmt.Errorf("missing required field 'metadata'")
	}
	if spec.Spec == nil {
		return nil, fmt.Errorf("missing required field 'spec'")
	}

	return &spec, nil
}

type ErrUnsupportedKind struct {
	Kind string
}

func (e ErrUnsupportedKind) Error() string {
	return fmt.Sprintf("unsupported kind: %s", e.Kind)
}
