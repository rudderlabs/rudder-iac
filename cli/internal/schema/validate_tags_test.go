package schema

import (
	"encoding/json"
	"reflect"
	"testing"

	jsonschema "github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationBoundsFollowReflectedKind(t *testing.T) {
	tests := []struct {
		name   string
		sample any
		assert func(*testing.T, *jsonschema.Schema)
	}{
		{
			name: "string uses length",
			sample: struct {
				Value string `json:"value,omitempty" validate:"gte=2,lte=8"`
			}{},
			assert: func(t *testing.T, value *jsonschema.Schema) {
				assert.Equal(t, uint64(2), *value.MinLength)
				assert.Equal(t, uint64(8), *value.MaxLength)
				assert.Nil(t, value.MinItems)
				assert.Empty(t, value.Minimum)
			},
		},
		{
			name: "slice uses items",
			sample: struct {
				Value []string `json:"value,omitempty" validate:"min=1,max=3"`
			}{},
			assert: func(t *testing.T, value *jsonschema.Schema) {
				assert.Equal(t, uint64(1), *value.MinItems)
				assert.Equal(t, uint64(3), *value.MaxItems)
				assert.Nil(t, value.MinLength)
				assert.Empty(t, value.Minimum)
			},
		},
		{
			name: "array uses items",
			sample: struct {
				Value [2]int `json:"value,omitempty" validate:"gte=2,lte=2"`
			}{},
			assert: func(t *testing.T, value *jsonschema.Schema) {
				assert.Equal(t, uint64(2), *value.MinItems)
				assert.Equal(t, uint64(2), *value.MaxItems)
			},
		},
		{
			name: "integer uses numeric bounds",
			sample: struct {
				Value int `json:"value,omitempty" validate:"gte=-2,lte=10"`
			}{},
			assert: func(t *testing.T, value *jsonschema.Schema) {
				assert.Equal(t, json.Number("-2"), value.Minimum)
				assert.Equal(t, json.Number("10"), value.Maximum)
				assert.Nil(t, value.MinLength)
				assert.Nil(t, value.MinItems)
			},
		},
		{
			name: "float uses numeric bounds",
			sample: struct {
				Value float64 `json:"value,omitempty" validate:"min=0.25,max=0.75"`
			}{},
			assert: func(t *testing.T, value *jsonschema.Schema) {
				assert.Equal(t, json.Number("0.25"), value.Minimum)
				assert.Equal(t, json.Number("0.75"), value.Maximum)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assert(t, property(t, specBlock(tt.sample), "value"))
		})
	}
}

func TestValidationRequiredEnumAndConst(t *testing.T) {
	type constrained struct {
		Optional string `json:"optional"`
		Mode     string `json:"mode,omitempty" validate:"required,oneof=alpha beta 'with space'"`
		Retries  int    `json:"retries,omitempty" validate:"oneof=1 2 3"`
		Version  int    `json:"version,omitempty" validate:"eq=2"`
		Empty    string `json:"empty,omitempty" validate:"eq="`
	}

	s := specBlock(constrained{})
	assert.Equal(t, []string{"mode"}, s.Required)
	assert.Equal(t, []any{"alpha", "beta", "with space"}, property(t, s, "mode").Enum)
	assert.Equal(t, []any{json.Number("1"), json.Number("2"), json.Number("3")}, property(t, s, "retries").Enum)
	assert.Equal(t, json.Number("2"), property(t, s, "version").Const)
	assert.Equal(t, "", property(t, s, "empty").Const)
}

func TestValidationTagsEnrichEmbeddedAndNestedStructs(t *testing.T) {
	type Shared struct {
		EmbeddedName string `json:"embedded_name,omitempty" validate:"required,min=2"`
	}
	type child struct {
		Code string `json:"code,omitempty" validate:"required,oneof=x y"`
	}
	type spec struct {
		Shared
		Nested child            `json:"nested,omitempty"`
		Other  Shared           `json:"other,omitempty"`
		List   []child          `json:"list,omitempty"`
		ByName map[string]child `json:"by_name,omitempty"`
	}

	s := specBlock(spec{})
	assert.Equal(t, []string{"embedded_name"}, s.Required)
	assert.Equal(t, uint64(2), *property(t, s, "embedded_name").MinLength)

	childDef := referencedDefinition(t, s, property(t, s, "nested"))
	assert.Equal(t, []string{"code"}, childDef.Required)
	assert.Equal(t, []any{"x", "y"}, property(t, childDef, "code").Enum)

	sharedDef := referencedDefinition(t, s, property(t, s, "other"))
	assert.Equal(t, []string{"embedded_name"}, sharedDef.Required)
	assert.Equal(t, uint64(2), *property(t, sharedDef, "embedded_name").MinLength)

	list := property(t, s, "list")
	assert.NotEmpty(t, list.Items.Ref)
	byName := property(t, s, "by_name")
	assert.NotNil(t, byName.AdditionalProperties)
	assert.NotEmpty(t, byName.AdditionalProperties.Ref)
}

func TestValidationTagsApplyDiveRulesToArrayItems(t *testing.T) {
	type catalogShape struct {
		Types []string `json:"types,omitempty" validate:"dive,oneof=string number integer boolean null array object"`
	}

	s := specBlock(catalogShape{})
	items := property(t, s, "types").Items
	require.NotNil(t, items)
	assert.Equal(t, []any{"string", "number", "integer", "boolean", "null", "array", "object"}, items.Enum)
	assert.Empty(t, property(t, s, "types").Enum)
}

func TestValidationTagsAddRequiredWithoutAnyOf(t *testing.T) {
	type spec struct {
		Code string `json:"code,omitempty" validate:"required_without=File"`
		File string `json:"file,omitempty" validate:"required_without=Code"`
	}

	s := specBlock(spec{})
	require.Len(t, s.AllOf, 2)
	assert.Equal(t, []string{"code"}, s.AllOf[0].AnyOf[0].Required)
	assert.Equal(t, []string{"file"}, s.AllOf[0].AnyOf[1].Required)
	assert.Equal(t, []string{"file"}, s.AllOf[1].AnyOf[0].Required)
	assert.Equal(t, []string{"code"}, s.AllOf[1].AnyOf[1].Required)
}

func TestValidationTagsAddRequiredIfAndUnless(t *testing.T) {
	type spec struct {
		SourceDefinition string `json:"source_definition,omitempty"`
		PrimaryKey       string `json:"primary_key,omitempty" validate:"required_unless=SourceDefinition s3"`
		BucketName       string `json:"bucket_name,omitempty" validate:"required_if=SourceDefinition s3"`
	}

	s := specBlock(spec{})
	require.Len(t, s.AllOf, 2)
	assert.Equal(t, []string{"primary_key"}, s.AllOf[0].Then.Required)
	assert.Equal(t, []string{"bucket_name"}, s.AllOf[1].Then.Required)
}

func TestValidationTagsUseMapstructureFallback(t *testing.T) {
	type mapstructureOnly struct {
		APIKey string `mapstructure:"api_key" validate:"required,min=4"`
	}

	typeOfSample := reflect.TypeOf(mapstructureOnly{})
	properties := jsonschema.NewProperties()
	properties.Set("api_key", &jsonschema.Schema{Type: "string"})
	s := &jsonschema.Schema{Type: "object", Properties: properties}

	e := &enricher{}
	e.walk(s, typeOfSample)

	apiKey := property(t, s, "api_key")
	assert.Equal(t, []string{"api_key"}, s.Required)
	assert.Equal(t, uint64(4), *apiKey.MinLength)
}

func TestValidationTagsFollowLocalRefsIntoDefinitions(t *testing.T) {
	type nested struct {
		Code string `json:"code,omitempty" validate:"required,min=2"`
	}
	type root struct {
		Nested nested `json:"nested,omitempty"`
	}

	typeOfRoot := reflect.TypeOf(root{})
	rootProperties := jsonschema.NewProperties()
	rootProperties.Set("nested", &jsonschema.Schema{Ref: "#/$defs/nested-alias"})
	nestedProperties := jsonschema.NewProperties()
	nestedProperties.Set("code", &jsonschema.Schema{Type: "string"})
	nestedSchema := &jsonschema.Schema{Type: "object", Properties: nestedProperties}
	rootSchema := &jsonschema.Schema{Type: "object", Properties: rootProperties}

	e := &enricher{
		defs: jsonschema.Definitions{
			"nested-alias":  {Ref: "#/$defs/custom-nested"},
			"custom-nested": nestedSchema,
		},
		visitedNodes: map[enrichmentVisit]bool{},
	}
	e.walk(rootSchema, typeOfRoot)

	assert.Equal(t, []string{"code"}, nestedSchema.Required)
	assert.Equal(t, uint64(2), *property(t, nestedSchema, "code").MinLength)
}

func TestValidationEnrichmentIsRefCycleSafeAndDeterministic(t *testing.T) {
	type node struct {
		Name     string  `json:"name,omitempty" validate:"required,min=1"`
		Next     *node   `json:"next,omitempty"`
		Children []*node `json:"children,omitempty" validate:"min=1"`
	}

	first := specBlock(node{})
	second := specBlock(node{})

	assert.Equal(t, []string{"name"}, first.Required)
	assert.Equal(t, uint64(1), *property(t, first, "name").MinLength)
	assert.Equal(t, "#/$defs/node", property(t, first, "next").Ref)
	children := property(t, first, "children")
	assert.Equal(t, uint64(1), *children.MinItems)
	assert.Equal(t, "#/$defs/node", children.Items.Ref)

	firstJSON, err := json.Marshal(first)
	require.NoError(t, err)
	secondJSON, err := json.Marshal(second)
	require.NoError(t, err)
	assert.Equal(t, string(firstJSON), string(secondJSON))
}

func TestValidationEnrichmentHandlesDefinitionRefCycles(t *testing.T) {
	type recursive struct {
		Name string `json:"name,omitempty" validate:"required"`
	}

	defs := jsonschema.Definitions{}
	defs["a"] = &jsonschema.Schema{Ref: "#/$defs/b"}
	defs["b"] = &jsonschema.Schema{Ref: "#/$defs/a"}
	e := &enricher{defs: defs}

	assert.NotPanics(t, func() {
		e.walk(defs["a"], reflect.TypeOf(recursive{}))
	})
}

func TestUnsupportedOrInvalidRulesAreIgnored(t *testing.T) {
	type permissive struct {
		Text   string   `json:"text,omitempty" validate:"min=-1,max=not-a-number"`
		Items  []string `json:"items,omitempty" validate:"oneof=a b,eq=a,gte=-1"`
		Count  int      `json:"count,omitempty" validate:"gte=not-a-number,oneof=1 nope"`
		Object struct{} `json:"object,omitempty" validate:"gte=1,lte=2,eq=x,oneof=x y"`
	}

	s := specBlock(permissive{})
	text := property(t, s, "text")
	assert.Nil(t, text.MinLength)
	assert.Nil(t, text.MaxLength)

	items := property(t, s, "items")
	assert.Nil(t, items.MinItems)
	assert.Empty(t, items.Enum)
	assert.Nil(t, items.Const)

	count := property(t, s, "count")
	assert.Empty(t, count.Minimum)
	assert.Empty(t, count.Enum)

	object := property(t, s, "object")
	assert.Empty(t, object.Minimum)
	assert.Nil(t, object.MinLength)
	assert.Nil(t, object.MinItems)
	assert.Empty(t, object.Enum)
	assert.Nil(t, object.Const)
}

func property(t *testing.T, s *jsonschema.Schema, name string) *jsonschema.Schema {
	t.Helper()
	require.NotNil(t, s.Properties)
	prop, ok := s.Properties.Get(name)
	require.True(t, ok, "property %q not found", name)
	return prop
}

func referencedDefinition(t *testing.T, root, ref *jsonschema.Schema) *jsonschema.Schema {
	t.Helper()
	const prefix = "#/$defs/"
	require.Contains(t, ref.Ref, prefix)
	definition, ok := root.Definitions[ref.Ref[len(prefix):]]
	require.True(t, ok, "definition for %q not found", ref.Ref)
	return definition
}
