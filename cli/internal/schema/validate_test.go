package schema

import (
	"encoding/json"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// compileKind builds a runtime validator from the generated schema for a kind.
func compileKind(t *testing.T, kind string) *jsonschema.Schema {
	t.Helper()

	s, err := ForKind(kind)
	require.NoError(t, err)

	raw, err := json.Marshal(s)
	require.NoError(t, err)

	var doc any
	require.NoError(t, json.Unmarshal(raw, &doc))

	c := jsonschema.NewCompiler()
	require.NoError(t, c.AddResource("mem://schema.json", doc))
	compiled, err := c.Compile("mem://schema.json")
	require.NoError(t, err)
	return compiled
}

// validateYAML decodes a YAML document into the JSON data model and validates
// it, mirroring what yaml-language-server does in an editor.
func validateYAML(t *testing.T, schema *jsonschema.Schema, doc string) error {
	t.Helper()

	var data any
	require.NoError(t, yaml.Unmarshal([]byte(doc), &data))
	return schema.Validate(normalize(t, data))
}

// normalize converts yaml.v3's map[interface{}]interface{} into the
// map[string]interface{} shape the validator expects.
func normalize(t *testing.T, v any) any {
	t.Helper()
	raw, err := json.Marshal(convert(v))
	require.NoError(t, err)
	var out any
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

func convert(v any) any {
	switch tv := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(tv))
		for k, val := range tv {
			m[k] = convert(val)
		}
		return m
	case map[any]any:
		m := make(map[string]any, len(tv))
		for k, val := range tv {
			m[k.(string)] = convert(val)
		}
		return m
	case []any:
		for i := range tv {
			tv[i] = convert(tv[i])
		}
		return tv
	default:
		return v
	}
}

func TestGeneratedSchemaAcceptsGoodSpecAndRejectsBad(t *testing.T) {
	schema := compileKind(t, "transformation")

	good := `
version: rudder/v1
kind: transformation
metadata:
  name: my-transformations
spec:
  id: enrich_user
  name: Enrich User
  language: javascript
  code: "export function transformEvent(e){ return e }"
`
	require.NoError(t, validateYAML(t, schema, good), "known-good spec must validate")

	// language is constrained to javascript|python via a validate:"oneof" tag.
	badEnum := `
version: rudder/v1
kind: transformation
metadata:
  name: my-transformations
spec:
  id: enrich_user
  name: Enrich User
  language: cobol
  code: "noop"
`
	require.Error(t, validateYAML(t, schema, badEnum), "invalid language enum must be rejected")

	// name is required via validate:"required"; omitting it must fail.
	missingRequired := `
version: rudder/v1
kind: transformation
metadata:
  name: my-transformations
spec:
  id: enrich_user
  language: javascript
  code: "noop"
`
	require.Error(t, validateYAML(t, schema, missingRequired), "missing required field must be rejected")

	// A wrong field type (name as a number) must be flagged — this is the
	// acceptance-criteria case exercised by yaml-language-server in editors.
	wrongType := `
version: rudder/v1
kind: transformation
metadata:
  name: my-transformations
spec:
  id: enrich_user
  name: 12345
  language: javascript
  code: "noop"
`
	require.Error(t, validateYAML(t, schema, wrongType), "wrong field type must be rejected")
}
