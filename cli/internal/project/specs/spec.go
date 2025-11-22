package specs

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

const (
	SpecVersion = "rudder/v0.1"
)

type Spec struct {
	Version  string         `yaml:"version"`
	Kind     string         `yaml:"kind"`
	Metadata map[string]any `yaml:"metadata"`
	Spec     map[string]any `yaml:"spec"`
}

type ParsedSpec struct {
	ExternalIDs []string
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
