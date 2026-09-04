package schema

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/invopop/jsonschema"
	orderedmap "github.com/pb33f/ordered-map/v2"
)

// ErrUnknownKind is returned when a schema is requested for a kind that is not
// in the registry.
var ErrUnknownKind = errors.New("unknown spec kind")

// ForKind generates the full Draft 2020-12 JSON Schema for a single spec kind.
// The schema covers the whole spec envelope (version/kind/metadata/spec) with
// the `spec` block reflected from the kind's Go struct.
func ForKind(kind string) (*jsonschema.Schema, error) {
	sample, ok := sampleFor(kind)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
	}
	return envelope(kind, sample), nil
}

// All generates schemas for every registered kind, keyed by kind.
func All() (map[string]*jsonschema.Schema, error) {
	out := make(map[string]*jsonschema.Schema, len(registry))
	for _, e := range registry {
		s, err := ForKind(e.kind)
		if err != nil {
			return nil, err
		}
		out[e.kind] = s
	}
	return out, nil
}

// Root generates a single root schema that discriminates on `kind`: a document
// matches exactly one branch, selected by its `kind` field. Editors can point a
// single `$schema` at this to validate any supported spec file.
func Root() (*jsonschema.Schema, error) {
	branches := make([]*jsonschema.Schema, 0, len(registry))
	for _, kind := range Kinds() {
		s, err := ForKind(kind)
		if err != nil {
			return nil, err
		}
		// Drop the per-branch dialect declaration; only the root carries it.
		s.Version = ""
		branches = append(branches, s)
	}

	return &jsonschema.Schema{
		Version: jsonschema.Version,
		Title:   "RudderStack CLI spec",
		OneOf:   branches,
	}, nil
}

// specBlock reflects a kind's `spec:` struct and enriches it with constraints
// carried on `validate` struct tags, which the reflector does not understand on
// its own. Nested and recursive types are represented via $defs/$ref (the
// reflector's default) so self-referential spec structs terminate cleanly.
func specBlock(sample any) *jsonschema.Schema {
	r := &jsonschema.Reflector{
		ExpandedStruct:            true,
		AllowAdditionalProperties: true,
	}

	t := reflect.TypeOf(sample)
	s := r.ReflectFromType(t)
	// The spec block lives inside the envelope; it declares its own dialect only
	// at the envelope root.
	s.Version = ""
	s.ID = ""

	enrich := &enricher{defs: s.Definitions, visited: map[reflect.Type]bool{}}
	enrich.walk(s, t)
	return s
}

// envelope wraps a reflected spec block in the standard rudder spec envelope.
func envelope(kind string, sample any) *jsonschema.Schema {
	props := orderedmap.New[string, *jsonschema.Schema]()
	props.Set("version", &jsonschema.Schema{
		Type:        "string",
		Description: "Spec format version, e.g. rudder/v1.",
	})
	props.Set("kind", &jsonschema.Schema{
		Type:  "string",
		Const: kind,
	})
	props.Set("metadata", &jsonschema.Schema{
		Type:        "object",
		Description: "Spec metadata (name and optional import block).",
	})
	spec := specBlock(sample)

	// $ref pointers ("#/$defs/...") resolve against the document root, so the
	// spec block's definitions must live at the envelope root, not nested under
	// properties/spec where the references would dangle.
	defs := spec.Definitions
	spec.Definitions = nil
	props.Set("spec", spec)

	return &jsonschema.Schema{
		Version:              jsonschema.Version,
		Title:                fmt.Sprintf("RudderStack %s spec", kind),
		Type:                 "object",
		Properties:           props,
		Required:             []string{"version", "kind", "spec"},
		AdditionalProperties: jsonschema.FalseSchema,
		Definitions:          defs,
	}
}
