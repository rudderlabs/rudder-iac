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
		{
			name:     "simple snake_case",
			input:    "min_length",
			expected: "minLength",
		},
		{
			name:     "snake_case with multiple words",
			input:    "max_length",
			expected: "maxLength",
		},
		{
			name:     "snake_case with of",
			input:    "multiple_of",
			expected: "multipleOf",
		},
		{
			name:     "snake_case with types",
			input:    "item_types",
			expected: "itemTypes",
		},
		{
			name:     "lowercase only",
			input:    "enum",
			expected: "enum",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single lowercase letter",
			input:    "a",
			expected: "a",
		},
		{
			name:     "snake_case three words",
			input:    "exclusive_minimum",
			expected: "exclusiveMinimum",
		},
		{
			name:     "snake_case four words",
			input:    "exclusive_maximum_value",
			expected: "exclusiveMaximumValue",
		},
		{
			name:     "already camelCase",
			input:    "minLength",
			expected: "minLength",
		},
		{
			name:     "trailing underscore",
			input:    "min_length_",
			expected: "minLength",
		},
		{
			name:     "leading underscore",
			input:    "_min_length",
			expected: "MinLength",
		},
		{
			name:     "multiple consecutive underscores",
			input:    "min__length",
			expected: "minLength",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ToCamelCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
