package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToSnakeCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple camelCase",
			input:    "minLength",
			expected: "min_length",
		},
		{
			name:     "camelCase with multiple words",
			input:    "maxLength",
			expected: "max_length",
		},
		{
			name:     "camelCase with Of",
			input:    "multipleOf",
			expected: "multiple_of",
		},
		{
			name:     "camelCase with Types",
			input:    "itemTypes",
			expected: "item_types",
		},
		{
			name:     "lowercase only",
			input:    "enum",
			expected: "enum",
		},
		{
			name:     "lowercase only multiple words",
			input:    "minimum",
			expected: "minimum",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single uppercase letter",
			input:    "A",
			expected: "a",
		},
		{
			name:     "single lowercase letter",
			input:    "a",
			expected: "a",
		},
		{
			name:     "PascalCase",
			input:    "ExclusiveMinimum",
			expected: "exclusive_minimum",
		},
		{
			name:     "PascalCase with multiple capitals",
			input:    "ExclusiveMaximum",
			expected: "exclusive_maximum",
		},
		{
			name:     "already snake_case",
			input:    "min_length",
			expected: "min_length",
		},
		{
			name:     "mixed snake_case with camelCase",
			input:    "some_varName",
			expected: "some_var_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ToSnakeCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToCamelCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "simple snake_case", input: "min_length", expected: "minLength"},
		{name: "multiple words", input: "max_length", expected: "maxLength"},
		{name: "exclusive maximum", input: "exclusive_maximum", expected: "exclusiveMaximum"},
		{name: "already lowercase", input: "enum", expected: "enum"},
		{name: "empty string", input: "", expected: ""},
		{name: "single word", input: "name", expected: "name"},
		{name: "underscore before word", input: "_name", expected: "Name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, ToCamelCase(tt.input))
		})
	}
}

func TestSplitMultiTypeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{name: "single type", input: "string", expected: []string{"string"}},
		{name: "comma separated", input: "string,number", expected: []string{"string", "number"}},
		{name: "trims whitespace", input: " string , number , boolean ", expected: []string{"string", "number", "boolean"}},
		{name: "empty string", input: "", expected: []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, SplitMultiTypeString(tt.input))
		})
	}
}

type sortableResource struct {
	localID string
}

func (r sortableResource) GetLocalID() string {
	return r.localID
}

func TestSortByLocalID(t *testing.T) {
	t.Parallel()

	items := []sortableResource{
		{localID: "zebra"},
		{localID: "alpha"},
		{localID: "middle"},
	}

	SortByLocalID(items)

	assert.Equal(t, []sortableResource{
		{localID: "alpha"},
		{localID: "middle"},
		{localID: "zebra"},
	}, items)
}

func TestSortLexicographically(t *testing.T) {
	t.Parallel()

	items := []any{"zebra", "alpha", "middle"}
	SortLexicographically(items)

	assert.Equal(t, []any{"alpha", "middle", "zebra"}, items)
}
