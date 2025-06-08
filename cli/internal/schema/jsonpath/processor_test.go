package jsonpath

import (
	"fmt"
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

// TestProcessor_JSONPathNormalization tests JSONPath normalization for gjson compatibility
// This covers lines 44-45 in processor.go (normalizeJSONPath function)
func TestProcessor_JSONPathNormalization(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		inputPath    string
		expectedPath string
	}{
		{
			name:         "DollarDotPrefix",
			inputPath:    "$.properties.user",
			expectedPath: "properties.user",
		},
		{
			name:         "DollarOnly",
			inputPath:    "$",
			expectedPath: "",
		},
		{
			name:         "DotOnly",
			inputPath:    "$.",
			expectedPath: "",
		},
		{
			name:         "NoPrefix",
			inputPath:    "properties.user",
			expectedPath: "properties.user",
		},
		{
			name:         "EmptyPath",
			inputPath:    "",
			expectedPath: "",
		},
		{
			name:         "DeepPath",
			inputPath:    "$.context.app.version",
			expectedPath: "context.app.version",
		},
		{
			name:         "ArrayAccess",
			inputPath:    "$.items.0.tags.1",
			expectedPath: "items.0.tags.1",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			processor := NewProcessor(c.inputPath, true)
			normalizedPath := processor.normalizeJSONPath()
			assert.Equal(t, c.expectedPath, normalizedPath)
		})
	}
}

// TestProcessor_ResultParsing tests successful result parsing and type conversion
// This covers lines 56-59 in processor.go (parseGjsonResult function)
func TestProcessor_ResultParsing(t *testing.T) {
	t.Parallel()

	testSchema := map[string]interface{}{
		"string_field":  "hello world",
		"number_field":  42.5,
		"integer_field": 123,
		"boolean_true":  true,
		"boolean_false": false,
		"null_field":    nil,
		"object_field": map[string]interface{}{
			"nested_string": "nested value",
			"nested_number": 99,
		},
		"array_field": []interface{}{
			"item1", "item2", "item3",
		},
		"mixed_array": []interface{}{
			"string",
			123,
			true,
			map[string]interface{}{
				"nested": "object",
			},
		},
	}

	cases := []struct {
		name          string
		jsonPath      string
		expectedValue interface{}
		expectedType  string
	}{
		{
			name:          "StringField",
			jsonPath:      "$.string_field",
			expectedValue: "hello world",
			expectedType:  "string",
		},
		{
			name:          "NumberField",
			jsonPath:      "$.number_field",
			expectedValue: 42.5,
			expectedType:  "float64",
		},
		{
			name:          "IntegerField",
			jsonPath:      "$.integer_field",
			expectedValue: float64(123), // gjson returns all numbers as float64
			expectedType:  "float64",
		},
		{
			name:          "BooleanTrue",
			jsonPath:      "$.boolean_true",
			expectedValue: true,
			expectedType:  "bool",
		},
		{
			name:          "BooleanFalse",
			jsonPath:      "$.boolean_false",
			expectedValue: false,
			expectedType:  "bool",
		},
		{
			name:          "NullField",
			jsonPath:      "$.null_field",
			expectedValue: nil,
			expectedType:  "<nil>",
		},
		{
			name:     "ObjectField",
			jsonPath: "$.object_field",
			expectedValue: map[string]interface{}{
				"nested_string": "nested value",
				"nested_number": float64(99), // gjson parses numbers as float64
			},
			expectedType: "map[string]interface {}",
		},
		{
			name:          "ArrayField",
			jsonPath:      "$.array_field",
			expectedValue: []interface{}{"item1", "item2", "item3"},
			expectedType:  "[]interface {}",
		},
		{
			name:     "MixedArray",
			jsonPath: "$.mixed_array",
			expectedValue: []interface{}{
				"string",
				float64(123), // gjson parses numbers as float64
				true,
				map[string]interface{}{
					"nested": "object",
				},
			},
			expectedType: "[]interface {}",
		},
		{
			name:          "NestedStringField",
			jsonPath:      "$.object_field.nested_string",
			expectedValue: "nested value",
			expectedType:  "string",
		},
		{
			name:          "ArrayElement",
			jsonPath:      "$.array_field.1",
			expectedValue: "item2",
			expectedType:  "string",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			processor := NewProcessor(c.jsonPath, true)
			result := processor.ProcessSchema(testSchema)

			require.NoError(t, result.Error, "Expected no error for path: %s", c.jsonPath)
			assert.Equal(t, c.expectedValue, result.Value, "Value mismatch for path: %s", c.jsonPath)

			// Check type as well
			if result.Value != nil {
				assert.Contains(t, c.expectedType, fmt.Sprintf("%T", result.Value),
					"Type mismatch for path: %s", c.jsonPath)
			}
		})
	}
}

// TestProcessor_SuccessfulHappyPaths tests various successful scenarios to ensure coverage
func TestProcessor_SuccessfulHappyPaths(t *testing.T) {
	t.Parallel()

	t.Run("CompleteWorkflow", func(t *testing.T) {
		t.Parallel()

		// Test a complete workflow that exercises normalization and parsing
		schema := map[string]interface{}{
			"user": map[string]interface{}{
				"profile": map[string]interface{}{
					"preferences": map[string]interface{}{
						"theme":         "dark",
						"notifications": true,
						"settings": map[string]interface{}{
							"volume":  0.8,
							"quality": "high",
						},
					},
				},
			},
		}

		// Test deeply nested path that exercises normalization
		processor := NewProcessor("$.user.profile.preferences", true)
		result := processor.ProcessSchema(schema)

		require.NoError(t, result.Error)
		require.NotNil(t, result.Value)

		preferences, ok := result.Value.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, "dark", preferences["theme"])
		assert.Equal(t, true, preferences["notifications"])

		settings, ok := preferences["settings"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, 0.8, settings["volume"])
		assert.Equal(t, "high", settings["quality"])
	})

	t.Run("ArrayProcessing", func(t *testing.T) {
		t.Parallel()

		schema := map[string]interface{}{
			"events": []interface{}{
				map[string]interface{}{
					"name":      "event1",
					"timestamp": 1234567890,
					"data":      map[string]interface{}{"key": "value1"},
				},
				map[string]interface{}{
					"name":      "event2",
					"timestamp": 1234567891,
					"data":      map[string]interface{}{"key": "value2"},
				},
			},
		}

		// Test array access
		processor := NewProcessor("$.events.0.data", true)
		result := processor.ProcessSchema(schema)

		require.NoError(t, result.Error)
		require.NotNil(t, result.Value)

		data, ok := result.Value.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value1", data["key"])
	})

	t.Run("RootPathHandling", func(t *testing.T) {
		t.Parallel()

		schema := map[string]interface{}{
			"top_level": "value",
			"nested": map[string]interface{}{
				"field": "nested_value",
			},
		}

		// Test root path variations
		rootPaths := []string{"", "$", "$."}

		for _, path := range rootPaths {
			processor := NewProcessor(path, true)
			result := processor.ProcessSchema(schema)

			require.NoError(t, result.Error, "Root path should not error: %s", path)
			assert.Equal(t, schema, result.Value, "Root path should return entire schema: %s", path)
		}
	})
}
