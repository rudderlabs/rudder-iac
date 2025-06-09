package converter

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaAnalyzer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		category string
		name     string
		validate func(t *testing.T)
	}{
		// Direct Property Analysis Tests (lines 126-128, 129-131)
		{
			category: "DirectCalls",
			name:     "ArrayCase_Lines126-128",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				arrayInput := []interface{}{"string", "number", "boolean"}
				err := analyzer.analyzeProperties(arrayInput, "test_path")
				assert.NoError(t, err, "Should handle array input without error")
			},
		},
		{
			category: "DirectCalls",
			name:     "PrimitiveCase_Lines129-131",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
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
					err := analyzer.analyzeProperties(test.value, test.path)
					assert.NoError(t, err, "Should handle primitive %s without error", test.name)
					assert.True(t, len(analyzer.Properties) > 0, "Should create properties for primitives")
				}
			},
		},
		{
			category: "DirectCalls",
			name:     "NestedArrayCase",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				nestedArray := []interface{}{[]interface{}{"nested", "array"}}
				err := analyzer.analyzeProperties(nestedArray, "nested_array_path")
				assert.NoError(t, err, "Should handle nested arrays without error")
			},
		},

		// Array Analysis Tests (lines 196-198, 205-207)
		{
			category: "Arrays",
			name:     "EmptyArray_Lines196-198",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				emptyArray := []interface{}{}
				err := analyzer.analyzeArray(emptyArray, "empty_array_path")
				assert.NoError(t, err, "Should handle empty array without error")
			},
		},
		{
			category: "Arrays",
			name:     "NestedArrayRecursion_Lines205-207",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				nestedArray := []interface{}{[]interface{}{[]interface{}{"deeply", "nested"}}}
				err := analyzer.analyzeArray(nestedArray, "nested_recursion_path")
				assert.NoError(t, err, "Should handle nested array recursion without error")
			},
		},
		{
			category: "Arrays",
			name:     "ArrayOfObjects",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				objectArray := []interface{}{map[string]interface{}{"field1": "string", "field2": "number"}}
				err := analyzer.analyzeArray(objectArray, "object_array_path")
				assert.NoError(t, err, "Should handle array of objects without error")
			},
		},
		{
			category: "Arrays",
			name:     "ArrayOfPrimitives",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				primitiveArray := []interface{}{"string1", "string2", "string3"}
				err := analyzer.analyzeArray(primitiveArray, "primitive_array_path")
				assert.NoError(t, err, "Should handle array of primitives without error")
			},
		},

		// Custom Type Tests (lines 255-265)
		{
			category: "CustomTypes",
			name:     "EmptyArrayCustomType_Lines255-265",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				emptyArray := []interface{}{}
				customType, err := analyzer.createCustomTypeForArray(emptyArray, "empty_custom_array")

				require.NoError(t, err, "Should create custom type for empty array without error")
				assert.NotNil(t, customType, "Should return custom type for empty array")
				assert.Equal(t, "array", customType.Type, "Should create array type")
				assert.Equal(t, "string", customType.ArrayItemType, "Should default to string item type for empty arrays")
				assert.NotEmpty(t, customType.ID, "Should generate ID for custom type")
				assert.NotEmpty(t, customType.Hash, "Should generate hash for custom type")
				assert.Contains(t, analyzer.CustomTypes, customType.ID, "Should add custom type to analyzer")
			},
		},

		// Object Analysis Tests (lines 145-150, 160-165)
		{
			category: "Objects",
			name:     "NestedObjectProcessing",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				complexObject := map[string]interface{}{
					"nested_object": map[string]interface{}{
						"deep_field1": "string",
						"deep_field2": map[string]interface{}{"deeper_field": "number"},
					},
					"array_field":     []interface{}{map[string]interface{}{"array_item_field": "boolean"}},
					"primitive_field": "string",
					"empty_array":     []interface{}{},
				}

				err := analyzer.analyzeObject(complexObject, "complex_test")
				assert.NoError(t, err, "Should handle complex nested object without error")
				assert.True(t, len(analyzer.Properties) > 0, "Should create properties")
				assert.True(t, len(analyzer.CustomTypes) > 0, "Should create custom types")
			},
		},
		{
			category: "Objects",
			name:     "ArrayProcessingInObject",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				objectWithArrays := map[string]interface{}{
					"simple_array": []interface{}{"item1", "item2"},
					"nested_array": []interface{}{[]interface{}{"nested1", "nested2"}},
					"empty_array":  []interface{}{},
					"object_array": []interface{}{map[string]interface{}{"obj_field": "value"}},
				}

				err := analyzer.analyzeObject(objectWithArrays, "array_test")
				assert.NoError(t, err, "Should handle object with various arrays without error")
			},
		},

		// Schema Analysis Edge Cases
		{
			category: "Schemas",
			name:     "SchemaWithDirectArrays",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				testSchemas := []models.Schema{
					{
						UID:             "array-test-uid",
						WriteKey:        "array-test-key",
						EventType:       "track",
						EventIdentifier: "array_test_event",
						Schema: map[string]interface{}{
							"root_array": []interface{}{"direct_array_item1", "direct_array_item2"},
						},
					},
				}

				err := analyzer.AnalyzeSchemas(testSchemas)
				assert.NoError(t, err, "Should handle schemas with direct arrays")
			},
		},
		{
			category: "Schemas",
			name:     "SchemaWithDirectPrimitives",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				testSchemas := []models.Schema{
					{
						UID:             "primitive-test-uid",
						WriteKey:        "primitive-test-key",
						EventType:       "track",
						EventIdentifier: "primitive_test_event",
						Schema:          map[string]interface{}{"root_field": "direct_string_value"},
					},
				}

				err := analyzer.AnalyzeSchemas(testSchemas)
				assert.NoError(t, err, "Should handle schemas with direct primitives")
			},
		},
		{
			category: "Schemas",
			name:     "SchemaWithComplexNestedArrays",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
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
										map[string]interface{}{"deep_field": "value"},
									},
								},
							},
							"empty_nested": []interface{}{[]interface{}{}},
						},
					},
				}

				err := analyzer.AnalyzeSchemas(testSchemas)
				assert.NoError(t, err, "Should handle schemas with deeply nested arrays")
			},
		},

		// Property Creation Edge Cases
		{
			category: "Properties",
			name:     "EmptyPath",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				err := analyzer.createProperty("", "test_value")
				assert.NoError(t, err, "Should handle empty path gracefully")
				assert.Len(t, analyzer.Properties, 0, "Should not create property for empty path")
			},
		},
		{
			category: "Properties",
			name:     "InvalidPath",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				err := analyzer.createProperty("context.traits.", "test_value")
				assert.NoError(t, err, "Should handle invalid path gracefully")
			},
		},
		{
			category: "Properties",
			name:     "DuplicateProperties",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				err1 := analyzer.createProperty("test.field", "string")
				err2 := analyzer.createProperty("test.field", "string")

				assert.NoError(t, err1, "First property creation should succeed")
				assert.NoError(t, err2, "Duplicate property creation should not error")
				assert.Len(t, analyzer.Properties, 1, "Should only create one property for duplicates")
			},
		},

		// Custom Type Property Edge Cases
		{
			category: "CustomTypeProperties",
			name:     "EmptyPath",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				customType := &CustomTypeInfo{ID: "test_type", Name: "TestType", Type: "object"}
				err := analyzer.createPropertyWithCustomType("", customType)
				assert.NoError(t, err, "Should handle empty path gracefully")
				assert.Len(t, analyzer.Properties, 0, "Should not create property for empty path")
			},
		},
		{
			category: "CustomTypeProperties",
			name:     "InvalidPath",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				customType := &CustomTypeInfo{ID: "test_type", Name: "TestType", Type: "object"}
				err := analyzer.createPropertyWithCustomType("context.traits.", customType)
				assert.NoError(t, err, "Should handle invalid path gracefully")
			},
		},
		{
			category: "CustomTypeProperties",
			name:     "DuplicateCustomTypeProperties",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				customType := &CustomTypeInfo{ID: "test_type", Name: "TestType", Type: "object"}
				err1 := analyzer.createPropertyWithCustomType("test.field", customType)
				err2 := analyzer.createPropertyWithCustomType("test.field", customType)

				assert.NoError(t, err1, "First property creation should succeed")
				assert.NoError(t, err2, "Duplicate property creation should not error")
			},
		},
	}

	for _, c := range cases {
		t.Run(c.category+"/"+c.name, func(t *testing.T) {
			t.Parallel()
			c.validate(t)
		})
	}
}
