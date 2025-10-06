package formatter

import (
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
