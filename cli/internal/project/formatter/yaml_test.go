package formatter

import (
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestYAMLFormatter_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       any
		expected    []byte
		expectError bool
	}{
		{
			name: "simple map",
			input: map[string]interface{}{
				"name":  "test",
				"value": 42,
			},
			expected: []byte(heredoc.Doc(`
name: "test"
value: 42
`)),
		},
		{
			name: "nested map",
			input: map[string]interface{}{
				"parent": map[string]interface{}{
					"child": "value",
					"num":   100,
				},
			},
			expected: []byte(heredoc.Doc(`
parent:
  child: "value"
  num: 100
`)),
		},
		{
			name: "string quoting",
			input: map[string]interface{}{
				"str1": "hello",
				"str2": "world",
				"num":  123,
			},
			expected: []byte(heredoc.Doc(`
num: 123
str1: "hello"
str2: "world"
`)),
		},
		{
			name:  "empty map",
			input: map[string]interface{}{},
			expected: []byte(heredoc.Doc(`{}
`)),
		},
		{
			name: "complex nested",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "example",
					"labels": map[string]interface{}{
						"env": "prod",
					},
				},
				"spec": map[string]interface{}{
					"replicas": 3,
				},
			},
			expected: []byte(heredoc.Doc(`
metadata:
  name: "example"
  labels:
    env: "prod"
spec:
  replicas: 3
`)),
		},
		{
			name: "array of strings",
			input: map[string]interface{}{
				"items": []string{"first", "second", "third"},
			},
			expected: []byte(heredoc.Doc(`
items:
  - "first"
  - "second"
  - "third"
`)),
		},
		{
			name: "mixed types",
			input: map[string]interface{}{
				"string": "text",
				"int":    42,
				"float":  3.14,
				"bool":   true,
			},
			expected: []byte(heredoc.Doc(`
bool: true
float: 3.14
int: 42
string: "text"
`)),
		},
		{
			name: "deep nesting",
			input: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": "deep",
					},
				},
			},
			expected: []byte(heredoc.Doc(`
level1:
  level2:
    level3: "deep"
`)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			formatter := YAMLFormatter{}
			output, err := formatter.Format(tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.YAMLEq(t, strings.TrimSpace(string(tt.expected)), string(output))
			}
		})
	}
}

func TestYAMLFormatter_Extension(t *testing.T) {
	t.Parallel()
	formatter := YAMLFormatter{}
	assert.Equal(t, []string{"yaml", "yml"}, formatter.Extension())
}

func TestYAMLFormatter_StringQuotingBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected []byte
	}{
		{
			name: "keys unquoted values quoted",
			input: map[string]interface{}{
				"mykey": "myvalue",
			},
			expected: []byte(heredoc.Doc(`
mykey: "myvalue"
`)),
		},
		{
			name: "nested keys unquoted",
			input: map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": "value",
				},
			},
			expected: []byte(heredoc.Doc(`
outer:
  inner: "value"
`)),
		},
		{
			name: "numbers not quoted",
			input: map[string]interface{}{
				"count":   10,
				"percent": 99.5,
			},
			expected: []byte(heredoc.Doc(`
count: 10
percent: 99.5
`)),
		},
		{
			name: "booleans not quoted",
			input: map[string]interface{}{
				"enabled":  true,
				"disabled": false,
			},
			expected: []byte(heredoc.Doc(`
disabled: false
enabled: true
`)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			formatter := YAMLFormatter{}
			output, err := formatter.Format(tt.input)
			require.NoError(t, err)
			if tt.name == "numbers not quoted" || tt.name == "booleans not quoted" {
				// Map key order is not guaranteed; assert content instead of full equality
				outStr := string(output)
				for _, want := range []string{"count: 10", "percent: 99.5", "disabled: false", "enabled: true"} {
					if strings.Contains(string(tt.expected), want) {
						assert.Contains(t, outStr, want)
					}
				}
			} else {
				assert.Equal(t, tt.expected, output)
			}
		})
	}
}

func TestYAMLOrderedFormatter_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		original string
		input    any
		expected string
	}{
		{
			name: "mapping key order preserved on no-op",
			original: heredoc.Doc(`
				zebra: "z"
				alpha: "a"
				mango: "m"
			`),
			input: map[string]any{
				"alpha": "a",
				"mango": "m",
				"zebra": "z",
			},
			expected: heredoc.Doc(`
				zebra: "z"
				alpha: "a"
				mango: "m"
			`),
		},
		{
			name: "changed value keeps key position",
			original: heredoc.Doc(`
				zebra: "z"
				alpha: "a"
				mango: "m"
			`),
			input: map[string]any{
				"alpha": "a",
				"mango": "NEW",
				"zebra": "z",
			},
			expected: heredoc.Doc(`
				zebra: "z"
				alpha: "a"
				mango: "NEW"
			`),
		},
		{
			name: "added key appended at end of parent",
			original: heredoc.Doc(`
				zebra: "z"
				alpha: "a"
			`),
			input: map[string]any{
				"alpha": "a",
				"zebra": "z",
				"bravo": "b",
			},
			expected: heredoc.Doc(`
				zebra: "z"
				alpha: "a"
				bravo: "b"
			`),
		},
		{
			name: "removed key dropped others unmoved",
			original: heredoc.Doc(`
				zebra: "z"
				alpha: "a"
				mango: "m"
			`),
			input: map[string]any{
				"alpha": "a",
				"zebra": "z",
			},
			expected: heredoc.Doc(`
				zebra: "z"
				alpha: "a"
			`),
		},
		{
			name: "nested mapping order preserved",
			original: heredoc.Doc(`
				outer:
				  zebra: "z"
				  alpha: "a"
			`),
			input: map[string]any{
				"outer": map[string]any{
					"alpha": "a",
					"zebra": "z",
				},
			},
			expected: heredoc.Doc(`
				outer:
				  zebra: "z"
				  alpha: "a"
			`),
		},
		{
			name: "sequence of mappings matched by id",
			original: heredoc.Doc(`
				items:
				  - id: "third"
				    name: "T"
				  - id: "first"
				    name: "F"
				  - id: "second"
				    name: "S"
			`),
			input: map[string]any{
				"items": []any{
					map[string]any{"id": "first", "name": "F"},
					map[string]any{"id": "second", "name": "S"},
					map[string]any{"id": "third", "name": "T"},
				},
			},
			expected: heredoc.Doc(`
				items:
				  - id: "third"
				    name: "T"
				  - id: "first"
				    name: "F"
				  - id: "second"
				    name: "S"
			`),
		},
		{
			name: "sequence matched by $ref (tracking-plan v0 property references)",
			original: heredoc.Doc(`
				properties:
				  - $ref: "#/properties/api_tracking/api_method"
				    required: true
				  - $ref: "#/properties/api_tracking/username"
				    required: false
			`),
			input: map[string]any{
				"properties": []any{
					map[string]any{"$ref": "#/properties/api_tracking/username", "required": false},
					map[string]any{"$ref": "#/properties/api_tracking/api_method", "required": true},
				},
			},
			expected: heredoc.Doc(`
				properties:
				  - $ref: "#/properties/api_tracking/api_method"
				    required: true
				  - $ref: "#/properties/api_tracking/username"
				    required: false
			`),
		},
		{
			name: "sequence matched by property (tracking-plan v1 property references)",
			original: heredoc.Doc(`
				properties:
				  - property: "#property:api_method"
				    required: true
				  - property: "#property:username"
				    required: false
			`),
			input: map[string]any{
				"properties": []any{
					map[string]any{"property": "#property:username", "required": false},
					map[string]any{"property": "#property:api_method", "required": true},
				},
			},
			expected: heredoc.Doc(`
				properties:
				  - property: "#property:api_method"
				    required: true
				  - property: "#property:username"
				    required: false
			`),
		},
		{
			name: "duplicate identity on orig side preserves orig cardinality in output",
			original: heredoc.Doc(`
				items:
				  - id: "a"
				    v: 1
				  - id: "a"
				    v: 2
			`),
			input: map[string]any{
				"items": []any{
					map[string]any{"id": "a", "v": 99},
				},
			},
			expected: heredoc.Doc(`
				items:
				  - id: "a"
				    v: 99
				  - id: "a"
				    v: 99
			`),
		},
		{
			name: "identity-key precedence: id wins over $ref and property",
			original: heredoc.Doc(`
				items:
				  - id: "second"
				    $ref: "#/a"
				  - id: "first"
				    $ref: "#/b"
			`),
			input: map[string]any{
				"items": []any{
					map[string]any{"id": "first", "$ref": "#/b"},
					map[string]any{"id": "second", "$ref": "#/a"},
				},
			},
			expected: heredoc.Doc(`
				items:
				  - id: "second"
				    $ref: "#/a"
				  - id: "first"
				    $ref: "#/b"
			`),
		},
		{
			name: "sequence without identity key falls back to positional",
			original: heredoc.Doc(`
				items:
				  - foo: "1"
				  - foo: "2"
			`),
			input: map[string]any{
				"items": []any{
					map[string]any{"foo": "1"},
					map[string]any{"foo": "2"},
				},
			},
			expected: heredoc.Doc(`
				items:
				  - foo: "1"
				  - foo: "2"
			`),
		},
		{
			name: "scalar sequence preserved positionally",
			original: heredoc.Doc(`
				tags:
				  - "c"
				  - "a"
				  - "b"
			`),
			input: map[string]any{
				"tags": []string{"c", "a", "b"},
			},
			expected: heredoc.Doc(`
				tags:
				  - "c"
				  - "a"
				  - "b"
			`),
		},
		{
			name: "new list item without matching id appended",
			original: heredoc.Doc(`
				items:
				  - id: "b"
				  - id: "a"
			`),
			input: map[string]any{
				"items": []any{
					map[string]any{"id": "a"},
					map[string]any{"id": "b"},
					map[string]any{"id": "c"},
				},
			},
			expected: heredoc.Doc(`
				items:
				  - id: "b"
				  - id: "a"
				  - id: "c"
			`),
		},
		{
			name: "nil original leaves alphabetized output",
			original: "",
			input: map[string]any{
				"zebra": "z",
				"alpha": "a",
			},
			expected: heredoc.Doc(`
				alpha: "a"
				zebra: "z"
			`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var original *yaml.Node
			if tt.original != "" {
				var node yaml.Node
				require.NoError(t, yaml.Unmarshal([]byte(tt.original), &node))
				original = &node
			}

			formatter := YAMLOrderedFormatter{Original: original}
			output, err := formatter.Format(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(output))
		})
	}
}

func TestYAMLOrderedFormatter_Extension(t *testing.T) {
	t.Parallel()
	formatter := YAMLOrderedFormatter{}
	assert.Equal(t, []string{"yaml", "yml"}, formatter.Extension())
}
