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

type RawSpec struct {
	Data    []byte
	parsed  *Spec
	errored error
}

func (r *RawSpec) Parse() (*Spec, error) {
	if r.parsed != nil || r.errored != nil {
		return r.parsed, r.errored
	}

	s, err := New(r.Data)
	if err != nil {
		r.errored = fmt.Errorf("parsing spec: %w", err)
	}

	r.parsed = s
	return r.parsed, r.errored
}

func (r *RawSpec) Parsed() *Spec {
	return r.parsed
}

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

func (s *Spec) Validate() error {

	if s.Version == "" {
		return fmt.Errorf("missing required field 'version'")
	}
	if s.Kind == "" {
		return fmt.Errorf("missing required field 'kind'")
	}
	if s.Metadata == nil {
		return fmt.Errorf("missing required field 'metadata'")
	}
	if s.Spec == nil {
		return fmt.Errorf("missing required field 'spec'")
	}

	return nil
}

// LocalID represents a local identifier extracted from a spec, paired with its
// JSON Pointer path for precise error reporting during validation.
type LocalID struct {
	ID              string // The local ID value (e.g., "user_email")
	JSONPointerPath string // JSON Pointer path (e.g., "/spec/properties/2/id")
}

type ParsedSpec struct {
	URNs               []string
	LegacyResourceType string // For backward compatibility with local_id imports
	LocalIDs           []LocalID
}

// New creates and validates a Spec from YAML data
// It enforces strict validation and rejects unknown fields
func New(data []byte) (*Spec, error) {
	var spec Spec
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true) // Enable strict mode - reject unknown fields
	// /spec, /kind, /version, /abc( unknown )

	if err := decoder.Decode(&spec); err != nil {
		return nil, fmt.Errorf("unmarshaling yaml: %w", err)
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
