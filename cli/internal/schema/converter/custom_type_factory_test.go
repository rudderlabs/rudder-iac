package converter

import (
	"testing"
)

func TestCustomTypeFactory_generateUniquenessSignature(t *testing.T) {
	factory := NewCustomTypeFactory()
	analyzer := NewSchemaAnalyzer()

	// Add some properties to the analyzer
	analyzer.Properties["prop1_string"] = &PropertyInfo{
		ID:       "prop1_string",
		Name:     "prop1",
		Type:     "string",
		JsonType: "string",
		Path:     "test.prop1",
	}
	analyzer.Properties["prop2_number"] = &PropertyInfo{
		ID:       "prop2_number",
		Name:     "prop2",
		Type:     "number",
		JsonType: "number",
		Path:     "test.prop2",
	}

	tests := []struct {
		name       string
		customType *CustomTypeInfo
		expected   string
	}{
		{
			name: "array type with string items",
			customType: &CustomTypeInfo{
				Type:          "array",
				ArrayItemType: "string",
			},
			expected: "array:string",
		},
		{
			name: "array type with custom type items",
			customType: &CustomTypeInfo{
				Type:          "array",
				ArrayItemType: "#/custom-types/extracted_custom_types/some_custom_type",
			},
			expected: "array:#/custom-types/extracted_custom_types/some_custom_type",
		},
		{
			name: "object type with properties",
			customType: &CustomTypeInfo{
				Type: "object",
				Structure: map[string]string{
					"prop1": "string",
					"prop2": "number",
				},
			},
			expected: "object:#/properties/extracted_properties/prop1_string|#/properties/extracted_properties/prop2_number",
		},
		{
			name: "empty object type",
			customType: &CustomTypeInfo{
				Type:      "object",
				Structure: map[string]string{},
			},
			expected: "object:empty",
		},
		{
			name: "unknown type",
			customType: &CustomTypeInfo{
				Type: "unknown",
			},
			expected: "unknown:unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := factory.generateUniquenessSignature(analyzer, tt.customType)
			if result != tt.expected {
				t.Errorf("generateUniquenessSignature() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCustomTypeFactory_uniquenessLogic(t *testing.T) {
	factory := NewCustomTypeFactory()
	analyzer := NewSchemaAnalyzer()

	// Test that two custom types with the same properties are considered the same
	structure := map[string]string{
		"name": "string",
		"age":  "number",
	}

	// Create first custom type
	customType1, err := factory.createCustomTypeInternal(analyzer, "path1", "object", structure, "")
	if err != nil {
		t.Fatalf("Failed to create first custom type: %v", err)
	}

	// Create second custom type with same structure
	customType2, err := factory.createCustomTypeInternal(analyzer, "path2", "object", structure, "")
	if err != nil {
		t.Fatalf("Failed to create second custom type: %v", err)
	}

	// They should be the same instance (reused)
	if customType1.ID != customType2.ID {
		t.Errorf("Expected custom types with same structure to be reused, but got different IDs: %s vs %s", customType1.ID, customType2.ID)
	}

	// Test array types with same item type
	arrayType1, err := factory.createCustomTypeInternal(analyzer, "array1", "array", nil, "string")
	if err != nil {
		t.Fatalf("Failed to create first array type: %v", err)
	}

	arrayType2, err := factory.createCustomTypeInternal(analyzer, "array2", "array", nil, "string")
	if err != nil {
		t.Fatalf("Failed to create second array type: %v", err)
	}

	// They should be the same instance (reused)
	if arrayType1.ID != arrayType2.ID {
		t.Errorf("Expected array types with same item type to be reused, but got different IDs: %s vs %s", arrayType1.ID, arrayType2.ID)
	}
}
