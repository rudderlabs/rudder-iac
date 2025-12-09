package specs

import (
	"bytes"
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
// It enforces strict validation and rejects unknown fields
func New(data []byte) (*Spec, error) {
	var spec Spec
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true) // Enable strict mode - reject unknown fields

	if err := decoder.Decode(&spec); err != nil {
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
