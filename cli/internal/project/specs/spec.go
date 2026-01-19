package specs

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	SpecVersionV0_1        = "rudder/0.1"
	SpecVersionV0_1Variant = "rudder/v0.1" // Legacy variant with 'v' prefix
	SpecVersionV1          = "rudder/v1"
)

type Spec struct {
	Version  string         `yaml:"version"`
	Kind     string         `yaml:"kind"`
	Metadata map[string]any `yaml:"metadata"`
	Spec     map[string]any `yaml:"spec"`
}

// IsLegacyVersion returns true if the spec version is a legacy version (rudder/0.1 or rudder/v0.1)
func (s *Spec) IsLegacyVersion() bool {
	return s.Version == SpecVersionV0_1 || s.Version == SpecVersionV0_1Variant
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

type SpecReference struct {
	Kind  string
	Group string
	ID    string
	URN   string
}

// ParseSpecReference parses a spec reference string and returns a SpecReference
// which includes the kind, group, id, and URN. URNs are constructed using the provided kindMap.
// to associate kinds with the corresponding resource types.
func ParseSpecReference(ref string, kindMap map[string]string) (SpecReference, error) {
	parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
	if len(parts) != 3 {
		return SpecReference{}, fmt.Errorf("reference must have format #/{kind}/{group}/{id}, got: %s", ref)
	}

	kind, group, id := parts[0], parts[1], parts[2]

	if _, ok := kindMap[kind]; !ok {
		// TODO: this error could be ErrUnsupportedKind from composite provider, check for a better place.
		return SpecReference{}, fmt.Errorf("unkown kind: %s", kind)
	}

	urn := fmt.Sprintf("%s:%s", kindMap[kind], id)

	return SpecReference{
		Kind:  kind,
		Group: group,
		ID:    id,
		URN:   urn,
	}, nil
}
