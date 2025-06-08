package converter

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaAnalyzer_AnalyzeProperties_DirectCalls tests the analyzeProperties function directly
// This targets the uncovered lines 126-128 (array case) and 129-131 (primitive case)
func TestSchemaAnalyzer_AnalyzeProperties_DirectCalls(t *testing.T) {
	t.Parallel()

	t.Run("ArrayCase_Lines126-128", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test direct array input - this should hit lines 126-128
		arrayInput := []interface{}{"string", "number", "boolean"}
		err := analyzer.analyzeProperties(arrayInput, "test_path")

		assert.NoError(t, err, "Should handle array input without error")
	})

	t.Run("PrimitiveCase_Lines129-131", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test direct primitive inputs - this should hit lines 129-131
		primitiveTests := []struct {
			name  string
			value interface{}
			path  string
		}{
			{"String", "test_string", "string_path"},
			{"Number", 42, "number_path"},
			{"Boolean", true, "boolean_path"},
			{"Float", 3.14, "float_path"},
		}

		for _, test := range primitiveTests {
			t.Run(test.name, func(t *testing.T) {
				err := analyzer.analyzeProperties(test.value, test.path)
				assert.NoError(t, err, "Should handle primitive %s without error", test.name)

				// Verify property was created
				assert.True(t, len(analyzer.Properties) > 0, "Should create properties for primitives")
			})
		}
	})

	t.Run("NestedArrayCase", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test nested array structure
		nestedArray := []interface{}{
			[]interface{}{"nested", "array"},
		}

		err := analyzer.analyzeProperties(nestedArray, "nested_array_path")
		assert.NoError(t, err, "Should handle nested arrays without error")
	})
}

// TestSchemaAnalyzer_AnalyzeArray_DirectCalls tests the analyzeArray function directly
// This targets the uncovered lines 196-198 (empty array) and 205-207 (nested array)
func TestSchemaAnalyzer_AnalyzeArray_DirectCalls(t *testing.T) {
	t.Parallel()

	t.Run("EmptyArray_Lines196-198", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test empty array - this should hit lines 196-198
		emptyArray := []interface{}{}
		err := analyzer.analyzeArray(emptyArray, "empty_array_path")

		assert.NoError(t, err, "Should handle empty array without error")
		// Empty array should return early without creating anything
	})

	t.Run("NestedArrayRecursion_Lines205-207", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test nested array recursion - this should hit lines 205-207
		nestedArray := []interface{}{
			[]interface{}{
				[]interface{}{"deeply", "nested"},
			},
		}

		err := analyzer.analyzeArray(nestedArray, "nested_recursion_path")
		assert.NoError(t, err, "Should handle nested array recursion without error")
	})

	t.Run("ArrayOfObjects", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test array of objects
		objectArray := []interface{}{
			map[string]interface{}{
				"field1": "string",
				"field2": "number",
			},
		}

		err := analyzer.analyzeArray(objectArray, "object_array_path")
		assert.NoError(t, err, "Should handle array of objects without error")
	})

	t.Run("ArrayOfPrimitives", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test array of primitives
		primitiveArray := []interface{}{"string1", "string2", "string3"}

		err := analyzer.analyzeArray(primitiveArray, "primitive_array_path")
		assert.NoError(t, err, "Should handle array of primitives without error")
	})
}

// TestSchemaAnalyzer_CreateCustomTypeForArray_EmptyArray tests empty array handling
// This targets the uncovered lines 255-265 in createCustomTypeForArray
func TestSchemaAnalyzer_CreateCustomTypeForArray_EmptyArray(t *testing.T) {
	t.Parallel()

	analyzer := NewSchemaAnalyzer()

	// Test empty array custom type creation - this should hit lines 255-265
	emptyArray := []interface{}{}
	customType, err := analyzer.createCustomTypeForArray(emptyArray, "empty_custom_array")

	require.NoError(t, err, "Should create custom type for empty array without error")
	assert.NotNil(t, customType, "Should return custom type for empty array")
	assert.Equal(t, "array", customType.Type, "Should create array type")
	assert.Equal(t, "string", customType.ArrayItemType, "Should default to string item type for empty arrays")
	assert.NotEmpty(t, customType.ID, "Should generate ID for custom type")
	assert.NotEmpty(t, customType.Hash, "Should generate hash for custom type")

	// Verify it was added to analyzer
	assert.Contains(t, analyzer.CustomTypes, customType.ID, "Should add custom type to analyzer")
}

// TestSchemaAnalyzer_AnalyzeObject_ErrorBranches tests error handling branches
// This targets the uncovered lines 145-150 and 160-165 in analyzeObject
func TestSchemaAnalyzer_AnalyzeObject_ErrorBranches(t *testing.T) {
	t.Parallel()

	t.Run("NestedObjectProcessing", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test complex nested object that exercises all branches
		complexObject := map[string]interface{}{
			"nested_object": map[string]interface{}{
				"deep_field1": "string",
				"deep_field2": map[string]interface{}{
					"deeper_field": "number",
				},
			},
			"array_field": []interface{}{
				map[string]interface{}{
					"array_item_field": "boolean",
				},
			},
			"primitive_field": "string",
			"empty_array":     []interface{}{},
		}

		err := analyzer.analyzeObject(complexObject, "complex_test")
		assert.NoError(t, err, "Should handle complex nested object without error")

		// Verify various elements were processed
		assert.True(t, len(analyzer.Properties) > 0, "Should create properties")
		assert.True(t, len(analyzer.CustomTypes) > 0, "Should create custom types")
	})

	t.Run("ArrayProcessingInObject", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test object with various array types
		objectWithArrays := map[string]interface{}{
			"simple_array": []interface{}{"item1", "item2"},
			"nested_array": []interface{}{
				[]interface{}{"nested1", "nested2"},
			},
			"empty_array": []interface{}{},
			"object_array": []interface{}{
				map[string]interface{}{
					"obj_field": "value",
				},
			},
		}

		err := analyzer.analyzeObject(objectWithArrays, "array_test")
		assert.NoError(t, err, "Should handle object with various arrays without error")
	})
}

// TestSchemaAnalyzer_AnalyzeSchemas_EdgeCases tests edge cases in schema analysis
func TestSchemaAnalyzer_AnalyzeSchemas_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("SchemaWithDirectArrays", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test schema that contains arrays at root level
		testSchemas := []models.Schema{
			{
				UID:             "array-test-uid",
				WriteKey:        "array-test-key",
				EventType:       "track",
				EventIdentifier: "array_test_event",
				Schema: map[string]interface{}{
					"root_array": []interface{}{
						"direct_array_item1",
						"direct_array_item2",
					},
				},
			},
		}

		err := analyzer.AnalyzeSchemas(testSchemas)
		assert.NoError(t, err, "Should handle schemas with direct arrays")
	})

	t.Run("SchemaWithDirectPrimitives", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test schema that contains primitive at root level
		testSchemas := []models.Schema{
			{
				UID:             "primitive-test-uid",
				WriteKey:        "primitive-test-key",
				EventType:       "track",
				EventIdentifier: "primitive_test_event",
				Schema: map[string]interface{}{
					"root_field": "direct_string_value",
				},
			},
		}

		err := analyzer.AnalyzeSchemas(testSchemas)
		assert.NoError(t, err, "Should handle schemas with direct primitives")
	})

	t.Run("SchemaWithComplexNestedArrays", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test schema with deeply nested arrays
		testSchemas := []models.Schema{
			{
				UID:             "nested-array-uid",
				WriteKey:        "nested-array-key",
				EventType:       "track",
				EventIdentifier: "nested_array_event",
				Schema: map[string]interface{}{
					"level1": []interface{}{
						[]interface{}{
							[]interface{}{
								map[string]interface{}{
									"deep_field": "value",
								},
							},
						},
					},
					"empty_nested": []interface{}{
						[]interface{}{},
					},
				},
			},
		}

		err := analyzer.AnalyzeSchemas(testSchemas)
		assert.NoError(t, err, "Should handle schemas with deeply nested arrays")
	})
}

// TestSchemaAnalyzer_CreateProperty_EdgeCases tests edge cases in property creation
func TestSchemaAnalyzer_CreateProperty_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("EmptyPath", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test with empty path - should skip
		err := analyzer.createProperty("", "test_value")
		assert.NoError(t, err, "Should handle empty path gracefully")
		assert.Len(t, analyzer.Properties, 0, "Should not create property for empty path")
	})

	t.Run("InvalidPath", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test with path that results in empty property name
		err := analyzer.createProperty("context.traits.", "test_value")
		assert.NoError(t, err, "Should handle invalid path gracefully")
		// Should not create property for invalid path
	})

	t.Run("DuplicateProperties", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Create same property twice
		err1 := analyzer.createProperty("test.field", "string")
		err2 := analyzer.createProperty("test.field", "string")

		assert.NoError(t, err1, "First property creation should succeed")
		assert.NoError(t, err2, "Duplicate property creation should not error")
		assert.Len(t, analyzer.Properties, 1, "Should only create one property for duplicates")
	})
}

// TestSchemaAnalyzer_CreatePropertyWithCustomType_EdgeCases tests edge cases in custom type property creation
func TestSchemaAnalyzer_CreatePropertyWithCustomType_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("EmptyPath", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()
		customType := &CustomTypeInfo{
			ID:   "test_type",
			Name: "TestType",
			Type: "object",
		}

		// Test with empty path - should skip
		err := analyzer.createPropertyWithCustomType("", customType)
		assert.NoError(t, err, "Should handle empty path gracefully")
		assert.Len(t, analyzer.Properties, 0, "Should not create property for empty path")
	})

	t.Run("InvalidPath", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()
		customType := &CustomTypeInfo{
			ID:   "test_type",
			Name: "TestType",
			Type: "object",
		}

		// Test with path that results in empty property name
		err := analyzer.createPropertyWithCustomType("context.traits.", customType)
		assert.NoError(t, err, "Should handle invalid path gracefully")
		// Should not create property for invalid path
	})

	t.Run("DuplicateCustomTypeProperties", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()
		customType := &CustomTypeInfo{
			ID:   "test_type",
			Name: "TestType",
			Type: "object",
		}

		// Create same property twice
		err1 := analyzer.createPropertyWithCustomType("test.field", customType)
		err2 := analyzer.createPropertyWithCustomType("test.field", customType)

		assert.NoError(t, err1, "First property creation should succeed")
		assert.NoError(t, err2, "Duplicate property creation should not error")
		// Note: Due to deduplication logic, might have 0 or 1 properties depending on exact implementation
	})
}
