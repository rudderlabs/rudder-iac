package localcatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestPropertySpecV1_FromV0(t *testing.T) {
	t.Parallel()

	t.Run("property type conversions", func(t *testing.T) {
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
				
			
	})

	t.Run("converts multiple properties with different configurations and preserves order", func(t *testing.T) {
		t.Parallel()

		v0Spec := PropertySpec{
			Properties: []Property{
				{
					LocalID:     "prop1",
					Name:        "Property 1",
					Description: "First property",
					Type:        "string",
					Config: map[string]interface{}{
						"minLength": 5,
					},
				},
				{
					LocalID:     "prop2",
					Name:        "Property 2",
					Description: "Second property",
					Type:        "integer",
					Config: map[string]interface{}{
						"minimum": 0,
						"maximum": 100,
					},
				},
				{
					LocalID: "prop3",
					Name:    "Property 3",
					Type:    "boolean",
					Config:  nil,
				},
				{
					LocalID: "prop4",
					Name:    "Property 4",
					Type:    "array",
				},
				{
					LocalID: "prop5",
					Name:    "Property 5",
					Type:    "#/custom-types/login_elements/email_type",
				},
			},
		}

		v1Spec := &PropertySpecV1{}
		err := v1Spec.FromV0(v0Spec)

		assert.NoError(t, err)
		assert.Len(t, v1Spec.Properties, 5)

		expected := []PropertyV1{
			{
				LocalID:     "prop1",
				Name:        "Property 1",
				Description: "First property",
				Type:        "string",
				Config: map[string]interface{}{
					"min_length": 5,
				},
			},
			{
				LocalID:     "prop2",
				Name:        "Property 2",
				Description: "Second property",
				Type:        "integer",
				Config: map[string]interface{}{
					"minimum": 0,
					"maximum": 100,
				},
			},
			{
				LocalID:     "prop3",
				Name:        "Property 3",
				Description: "",
				Type:        "boolean",
				Config:      nil,
			},
			{
				LocalID:     "prop4",
				Name:        "Property 4",
				Description: "",
				Type:        "array",
				Config:      nil,
			},
			{
				LocalID:     "prop5",
				Name:        "Property 5",
				Description: "",
				Type:        "#/custom-types/login_elements/email_type",
				Config:      nil,
			},
		}
		assert.Equal(t, expected, v1Spec.Properties)
	})
}
