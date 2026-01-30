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

	t.Run("converts single property and transforms config keys to snake_case", func(t *testing.T) {
		t.Parallel()

		v0Spec := PropertySpec{
			Properties: []Property{
				{
					LocalID:     "prop1",
					Name:        "Property 1",
					Description: "Test property",
					Type:        "string",
					Config: map[string]interface{}{
						"minLength":        5,
						"maxLength":        50,
						"multipleOf":       3,
						"itemTypes":        []interface{}{"string", "integer"},
						"exclusiveMinimum": 1,
						"exclusiveMaximum": 99,
					},
				},
			},
		}

		v1Spec := &PropertySpecV1{}
		err := v1Spec.FromV0(v0Spec)

		assert.NoError(t, err)
		assert.Len(t, v1Spec.Properties, 1)

		expected := PropertyV1{
			LocalID:     "prop1",
			Name:        "Property 1",
			Description: "Test property",
			Type:        "string",
			Config: map[string]interface{}{
				"min_length":        5,
				"max_length":        50,
				"multiple_of":       3,
				"item_types":        []interface{}{"string", "integer"},
				"exclusive_minimum": 1,
				"exclusive_maximum": 99,
			},
		}
		assert.Equal(t, expected, v1Spec.Properties[0])
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
