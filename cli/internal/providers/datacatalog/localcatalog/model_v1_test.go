package localcatalog

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/utils"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := utils.ToSnakeCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertConfigKeysToSnakeCase(t *testing.T) {
	t.Parallel()

	t.Run("converts all camelCase keys", func(t *testing.T) {
		t.Parallel()

		config := map[string]interface{}{
			"minLength":        5,
			"maxLength":        50,
			"multipleOf":       3,
			"itemTypes":        []interface{}{"string", "integer"},
			"enum":             []interface{}{"value1", "value2"},
			"minimum":          0,
			"maximum":          100,
			"pattern":          "^[a-z]+$",
			"exclusiveMinimum": 1,
			"exclusiveMaximum": 99,
		}

		result := convertConfigKeysToSnakeCase(config)

		assert.Equal(t, 5, result["min_length"])
		assert.Equal(t, 50, result["max_length"])
		assert.Equal(t, 3, result["multiple_of"])
		assert.Equal(t, []interface{}{"string", "integer"}, result["item_types"])
		assert.Equal(t, []interface{}{"value1", "value2"}, result["enum"])
		assert.Equal(t, 0, result["minimum"])
		assert.Equal(t, 100, result["maximum"])
		assert.Equal(t, "^[a-z]+$", result["pattern"])
		assert.Equal(t, 1, result["exclusive_minimum"])
		assert.Equal(t, 99, result["exclusive_maximum"])

		// Verify camelCase keys don't exist
		assert.NotContains(t, result, "minLength")
		assert.NotContains(t, result, "maxLength")
		assert.NotContains(t, result, "multipleOf")
		assert.NotContains(t, result, "itemTypes")
		assert.NotContains(t, result, "exclusiveMinimum")
		assert.NotContains(t, result, "exclusiveMaximum")
	})

	t.Run("returns nil for nil config", func(t *testing.T) {
		t.Parallel()

		result := convertConfigKeysToSnakeCase(nil)
		assert.Nil(t, result)
	})

	t.Run("handles empty config", func(t *testing.T) {
		t.Parallel()

		config := map[string]interface{}{}
		result := convertConfigKeysToSnakeCase(config)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("preserves already snake_case keys", func(t *testing.T) {
		t.Parallel()

		config := map[string]interface{}{
			"min_length": 10,
			"max_length": 20,
		}

		result := convertConfigKeysToSnakeCase(config)

		assert.Equal(t, 10, result["min_length"])
		assert.Equal(t, 20, result["max_length"])
	})
}

func TestPropertyV1_FromV0_TypeConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		v0Type            string
		v0Config          map[string]interface{}
		expectedType      string
		expectedTypes     []string
		expectedItemTypes []string
		expectedConfig    map[string]interface{}
	}{
		{
			name:           "single primitive type",
			v0Type:         "string",
			v0Config:       nil,
			expectedType:   "string",
			expectedTypes:  nil,
			expectedConfig: nil,
		},
		{
			name:           "single type with camelCase config",
			v0Type:         "string",
			v0Config:       map[string]interface{}{"minLength": 5, "maxLength": 50},
			expectedType:   "string",
			expectedTypes:  nil,
			expectedConfig: map[string]interface{}{"min_length": 5, "max_length": 50},
		},
		{
			name:           "comma-separated types",
			v0Type:         "string,number",
			v0Config:       nil,
			expectedType:   "",
			expectedTypes:  []string{"string", "number"},
			expectedConfig: nil,
		},
		{
			name:           "comma-separated types with config",
			v0Type:         "string,number",
			v0Config:       map[string]interface{}{"multipleOf": 3, "pattern": "^[a-z]+$"},
			expectedType:   "",
			expectedTypes:  []string{"string", "number"},
			expectedConfig: map[string]interface{}{"multiple_of": 3, "pattern": "^[a-z]+$"},
		},
		{
			name:           "comma-separated types with spaces",
			v0Type:         "string, number, boolean",
			v0Config:       nil,
			expectedType:   "",
			expectedTypes:  []string{"string", "number", "boolean"},
			expectedConfig: nil,
		},
		{
			name:           "custom type reference",
			v0Type:         "#/custom-types/myGroup/MyType",
			v0Config:       nil,
			expectedType:   "#/custom-types/myGroup/MyType",
			expectedTypes:  nil,
			expectedConfig: nil,
		},
		{
			name:              "array type with itemTypes config",
			v0Type:            "array",
			v0Config:          map[string]interface{}{"itemTypes": []interface{}{"string", "number"}},
			expectedType:      "array",
			expectedTypes:     nil,
			expectedItemTypes: []string{"string", "number"},
			expectedConfig:    map[string]interface{}{},
		},
		{
			name:           "object type with nested config",
			v0Type:         "object",
			v0Config:       map[string]interface{}{"minProperties": 1, "maxProperties": 10},
			expectedType:   "object",
			expectedTypes:  nil,
			expectedConfig: map[string]interface{}{"min_properties": 1, "max_properties": 10},
		},
		{
			name:           "number type with complex config",
			v0Type:         "number",
			v0Config:       map[string]interface{}{"minimum": 0, "maximum": 100, "exclusiveMinimum": true, "exclusiveMaximum": false},
			expectedType:   "number",
			expectedTypes:  nil,
			expectedConfig: map[string]interface{}{"minimum": 0, "maximum": 100, "exclusive_minimum": true, "exclusive_maximum": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v0 := Property{
				LocalID: "test-prop",
				Name:    "Test Property",
				Type:    tt.v0Type,
				Config:  tt.v0Config,
			}

			var v1 PropertyV1
			v1.FromV0(v0)

			assert.Equal(t, tt.expectedType, v1.Type, "Type field mismatch")
			assert.Equal(t, tt.expectedTypes, v1.Types, "Types field mismatch")
			assert.Equal(t, tt.expectedItemTypes, v1.ItemTypes, "ItemTypes field mismatch")
			assert.Equal(t, tt.expectedConfig, v1.Config, "Config field mismatch")
			assert.Equal(t, v0.LocalID, v1.LocalID)
			assert.Equal(t, v0.Name, v1.Name)
		})
	}
}
