package schema

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/editor"
)

// FileName returns the conventional on-disk file name for a kind's schema.
func FileName(kind string) string {
	return editor.FileName(kind)
}

// RootFileName is the file name used for the combined, kind-discriminated root
// schema written alongside the per-kind schemas.
const RootFileName = "rudder-spec.schema.json"

// MarshalKind renders a kind's schema as indented JSON bytes.
func MarshalKind(schemas Set, kind string) ([]byte, error) {
	s, ok := schemas[kind]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
	}
	return marshalIndented(s)
}

// MarshalRoot renders the combined root schema as indented JSON bytes.
func MarshalRoot(schemas Set) ([]byte, error) {
	root, err := Root(schemas)
	if err != nil {
		return nil, err
	}
	return marshalIndented(root)
}

func marshalIndented(s *jsonschema.Schema) ([]byte, error) {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling schema: %w", err)
	}
	return b, nil
}
