package schema

import (
	"encoding/json"
	"testing"

	tekuri "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
)

type schemaProviderStub struct {
	schemas map[string][]provider.SchemaVariant
}

func (s schemaProviderStub) SpecSchemas() map[string][]provider.SchemaVariant { return s.schemas }

type recursiveSpec struct {
	Name     string          `json:"name" validate:"required,gte=2,lte=10"`
	Mode     string          `json:"mode" validate:"oneof=one two"`
	Count    int             `json:"count" validate:"gte=1,lte=5"`
	Children []recursiveSpec `json:"children" validate:"max=2"`
	Tags     []string        `json:"tags" validate:"dive,oneof=a b"`
}

func TestCatalogCompilesAndValidates(t *testing.T) {
	catalog, err := New(schemaProviderStub{schemas: map[string][]provider.SchemaVariant{
		"example": {{Spec: recursiveSpec{}}},
	}})
	require.NoError(t, err)

	raw, err := catalog.MarshalKind("example")
	require.NoError(t, err)
	var document any
	require.NoError(t, json.Unmarshal(raw, &document))
	compiler := tekuri.NewCompiler()
	require.NoError(t, compiler.AddResource("mem://example.json", document))
	compiled, err := compiler.Compile("mem://example.json")
	require.NoError(t, err)

	var valid any
	require.NoError(t, yaml.Unmarshal([]byte("version: rudder/v1\nkind: example\nmetadata:\n  name: fixture\nspec:\n  name: good\n  mode: one\n  count: 2\n  tags: [a]\n"), &valid))
	require.NoError(t, compiled.Validate(valid))

	var invalid any
	require.NoError(t, yaml.Unmarshal([]byte("version: rudder/v1\nkind: example\nmetadata:\n  name: fixture\nspec:\n  name: x\n  mode: invalid\n  count: 9\n  tags: [invalid]\n"), &invalid))
	assert.Error(t, compiled.Validate(invalid))
}

func TestCatalogRootCompilesWithConflictingDefinitionNames(t *testing.T) {
	catalog, err := New(schemaProviderStub{schemas: map[string][]provider.SchemaVariant{
		"first":  {{Spec: recursiveSpec{}}},
		"second": {{Spec: recursiveSpec{}}},
	}})
	require.NoError(t, err)
	raw, err := catalog.MarshalRoot()
	require.NoError(t, err)
	var document any
	require.NoError(t, json.Unmarshal(raw, &document))
	compiler := tekuri.NewCompiler()
	require.NoError(t, compiler.AddResource("mem://root.json", document))
	_, err = compiler.Compile("mem://root.json")
	require.NoError(t, err)
}

func TestCatalogKindsAreSorted(t *testing.T) {
	catalog, err := New(schemaProviderStub{schemas: map[string][]provider.SchemaVariant{
		"z": {{Spec: struct{}{}}},
		"a": {{Spec: struct{}{}}},
	}})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "z"}, catalog.Kinds())
}
