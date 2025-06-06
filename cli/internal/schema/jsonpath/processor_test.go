package jsonpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProcessor(t *testing.T) {
	t.Parallel()

	t.Run("CreateProcessor", func(t *testing.T) {
		t.Parallel()

		processor := NewProcessor("$.properties", true)
		assert.NotNil(t, processor)
		assert.Equal(t, "$.properties", processor.jsonPath)
		assert.True(t, processor.skipFailed)
	})

	t.Run("CreateProcessorSkipFalse", func(t *testing.T) {
		t.Parallel()

		processor := NewProcessor("$.context", false)
		assert.Equal(t, "$.context", processor.jsonPath)
		assert.False(t, processor.skipFailed)
	})
}

func TestProcessor_IsRootPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		jsonPath string
		expected bool
	}{
		{
			name:     "EmptyPath",
			jsonPath: "",
			expected: true,
		},
		{
			name:     "DollarSign",
			jsonPath: "$",
			expected: true,
		},
		{
			name:     "DollarDot",
			jsonPath: "$.",
			expected: true,
		},
		{
			name:     "PropertiesPath",
			jsonPath: "$.properties",
			expected: false,
		},
		{
			name:     "NestedPath",
			jsonPath: "$.context.app",
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			processor := NewProcessor(c.jsonPath, true)
			result := processor.IsRootPath()
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestProcessor_ProcessSchema(t *testing.T) {
	t.Parallel()

	// Test schema data
	testSchema := map[string]interface{}{
		"event":  "product_viewed",
		"userId": "12345",
		"properties": map[string]interface{}{
			"product_id":   "abc123",
			"product_name": "iPhone",
			"price":        999.99,
			"categories":   []interface{}{"electronics", "phones"},
		},
		"context": map[string]interface{}{
			"app": map[string]interface{}{
				"name":    "MyApp",
				"version": "1.0.0",
			},
			"traits": map[string]interface{}{
				"email": "user@example.com",
				"name":  "John Doe",
			},
		},
	}

	t.Run("RootPath", func(t *testing.T) {
		t.Parallel()

		cases := []string{"", "$", "$."}
		for _, jsonPath := range cases {
			t.Run("Path_"+jsonPath, func(t *testing.T) {
				t.Parallel()

				processor := NewProcessor(jsonPath, true)
				result := processor.ProcessSchema(testSchema)

				assert.NoError(t, result.Error)
				assert.Equal(t, testSchema, result.Value)
			})
		}
	})

	t.Run("ExtractProperties", func(t *testing.T) {
		t.Parallel()

		processor := NewProcessor("$.properties", true)
		result := processor.ProcessSchema(testSchema)

		require.NoError(t, result.Error)
		require.NotNil(t, result.Value)

		extracted, ok := result.Value.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, "abc123", extracted["product_id"])
		assert.Equal(t, "iPhone", extracted["product_name"])
		assert.Equal(t, 999.99, extracted["price"])

		categories, ok := extracted["categories"].([]interface{})
		require.True(t, ok)
		assert.Len(t, categories, 2)
		assert.Equal(t, "electronics", categories[0])
		assert.Equal(t, "phones", categories[1])
	})

	t.Run("ExtractNestedField", func(t *testing.T) {
		t.Parallel()

		processor := NewProcessor("$.context.app", true)
		result := processor.ProcessSchema(testSchema)

		require.NoError(t, result.Error)
		require.NotNil(t, result.Value)

		extracted, ok := result.Value.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, "MyApp", extracted["name"])
		assert.Equal(t, "1.0.0", extracted["version"])
	})

	t.Run("ExtractPrimitiveValue", func(t *testing.T) {
		t.Parallel()

		processor := NewProcessor("$.userId", true)
		result := processor.ProcessSchema(testSchema)

		require.NoError(t, result.Error)
		assert.Equal(t, "12345", result.Value)
	})

	t.Run("ExtractArray", func(t *testing.T) {
		t.Parallel()

		processor := NewProcessor("$.properties.categories", true)
		result := processor.ProcessSchema(testSchema)

		require.NoError(t, result.Error)
		require.NotNil(t, result.Value)

		categories, ok := result.Value.([]interface{})
		require.True(t, ok)
		assert.Len(t, categories, 2)
		assert.Equal(t, "electronics", categories[0])
		assert.Equal(t, "phones", categories[1])
	})

	t.Run("ExtractArrayElement", func(t *testing.T) {
		t.Parallel()

		processor := NewProcessor("$.properties.categories.0", true)
		result := processor.ProcessSchema(testSchema)

		require.NoError(t, result.Error)
		assert.Equal(t, "electronics", result.Value)
	})

	t.Run("NonExistentPath", func(t *testing.T) {
		t.Parallel()

		processor := NewProcessor("$.nonexistent", true)
		result := processor.ProcessSchema(testSchema)

		assert.Error(t, result.Error)
		assert.Nil(t, result.Value)
		assert.Contains(t, result.Error.Error(), "JSONPath '$.nonexistent' returned no results")
	})

	t.Run("InvalidJSONPath", func(t *testing.T) {
		t.Parallel()

		processor := NewProcessor("$.properties..invalid", true)
		result := processor.ProcessSchema(testSchema)

		assert.Error(t, result.Error)
		assert.Nil(t, result.Value)
		assert.Contains(t, result.Error.Error(), "returned no results")
	})

	t.Run("EmptySchema", func(t *testing.T) {
		t.Parallel()

		emptySchema := map[string]interface{}{}
		processor := NewProcessor("$.properties", true)
		result := processor.ProcessSchema(emptySchema)

		assert.Error(t, result.Error)
		assert.Nil(t, result.Value)
		assert.Contains(t, result.Error.Error(), "returned no results")
	})
}

func TestProcessor_ShouldSkipOnError(t *testing.T) {
	t.Parallel()

	t.Run("SkipFailedTrue", func(t *testing.T) {
		t.Parallel()

		processor := NewProcessor("$.properties", true)
		assert.True(t, processor.ShouldSkipOnError())
	})

	t.Run("SkipFailedFalse", func(t *testing.T) {
		t.Parallel()

		processor := NewProcessor("$.properties", false)
		assert.False(t, processor.ShouldSkipOnError())
	})
}

func TestProcessor_ComplexJSONPaths(t *testing.T) {
	t.Parallel()

	complexSchema := map[string]interface{}{
		"data": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{
					"id":   "item1",
					"tags": []interface{}{"tag1", "tag2"},
				},
				map[string]interface{}{
					"id":   "item2",
					"tags": []interface{}{"tag3", "tag4"},
				},
			},
			"metadata": map[string]interface{}{
				"version": "1.0",
				"nested": map[string]interface{}{
					"deep": map[string]interface{}{
						"value": "found_it",
					},
				},
			},
		},
	}

	cases := []struct {
		name     string
		jsonPath string
		expected interface{}
	}{
		{
			name:     "FirstItem",
			jsonPath: "$.data.items.0",
			expected: map[string]interface{}{
				"id":   "item1",
				"tags": []interface{}{"tag1", "tag2"},
			},
		},
		{
			name:     "FirstItemID",
			jsonPath: "$.data.items.0.id",
			expected: "item1",
		},
		{
			name:     "FirstItemFirstTag",
			jsonPath: "$.data.items.0.tags.0",
			expected: "tag1",
		},
		{
			name:     "DeepNested",
			jsonPath: "$.data.metadata.nested.deep.value",
			expected: "found_it",
		},
		{
			name:     "AllItems",
			jsonPath: "$.data.items",
			expected: []interface{}{
				map[string]interface{}{
					"id":   "item1",
					"tags": []interface{}{"tag1", "tag2"},
				},
				map[string]interface{}{
					"id":   "item2",
					"tags": []interface{}{"tag3", "tag4"},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			processor := NewProcessor(c.jsonPath, true)
			result := processor.ProcessSchema(complexSchema)

			require.NoError(t, result.Error)
			assert.Equal(t, c.expected, result.Value)
		})
	}
}
