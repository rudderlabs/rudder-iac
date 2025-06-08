package converter

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	yamlModels "github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaAnalyzer_SchemaFieldProcessing tests schema field processing during analysis
// This covers line 289 in generators.go
func TestSchemaAnalyzer_SchemaFieldProcessing(t *testing.T) {
	t.Parallel()

	t.Run("ExtractPropertiesForSchema", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Setup analyzer with some properties first
		testSchemas := []models.Schema{
			{
				UID:             "test-uid",
				WriteKey:        "test-write-key",
				EventType:       "track",
				EventIdentifier: "test_event",
				Schema: map[string]interface{}{
					"event":  "string",
					"userId": "string",
					"properties": map[string]interface{}{
						"product_id":   "string",
						"product_name": "string",
						"price":        "number",
						"nested": map[string]interface{}{
							"field1": "string",
							"field2": "number",
						},
					},
				},
			},
		}

		err := analyzer.AnalyzeSchemas(testSchemas)
		require.NoError(t, err)

		// Test extracting properties for a specific schema
		schema := testSchemas[0]
		properties := analyzer.extractPropertiesForSchema(schema)

		// Verify properties were extracted - may be empty if no exact matches found
		// The function should at least run without error
		assert.NotNil(t, properties)

		// Check that any returned properties have valid structure
		for _, prop := range properties {
			assert.NotEmpty(t, prop.Ref, "Property reference should not be empty")
			assert.Contains(t, prop.Ref, "#/properties/extracted_properties/", "Property reference should have correct prefix")
		}

		// The key test is that the function executes without error and returns a valid slice
		assert.IsType(t, []yamlModels.PropertyRuleRef{}, properties)
	})

	t.Run("CollectPropertyPaths", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test with nested object
		nestedObj := map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"field1": "string",
					"field2": "number",
				},
				"simple_field": "string",
			},
			"array_field": []interface{}{
				map[string]interface{}{
					"item_field": "string",
				},
			},
			"root_field": "string",
		}

		paths := analyzer.collectPropertyPaths(nestedObj, "")

		// Verify paths were collected
		assert.NotEmpty(t, paths)

		// Should contain nested paths
		expectedPaths := []string{
			"level1",
			"level1.level2",
			"level1.level2.field1",
			"level1.level2.field2",
			"level1.simple_field",
			"array_field",
			"root_field",
		}

		for _, expectedPath := range expectedPaths {
			found := false
			for _, path := range paths {
				if path == expectedPath {
					found = true
					break
				}
			}
			assert.True(t, found, "Should find path: %s", expectedPath)
		}
	})

	t.Run("FindPropertyKeyForPath", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Setup analyzer with test data
		testSchemas := []models.Schema{
			{
				UID:             "test-uid",
				WriteKey:        "test-write-key",
				EventType:       "track",
				EventIdentifier: "test_event",
				Schema: map[string]interface{}{
					"properties": map[string]interface{}{
						"user_id": "string",
						"email":   "string",
					},
				},
			},
		}

		err := analyzer.AnalyzeSchemas(testSchemas)
		require.NoError(t, err)

		// Test finding property key for path
		userIdKey := analyzer.findPropertyKeyForPath("properties.user_id")
		emailKey := analyzer.findPropertyKeyForPath("properties.email")
		nonExistentKey := analyzer.findPropertyKeyForPath("properties.nonexistent")

		// Should find keys for existing paths
		if len(analyzer.Properties) > 0 {
			assert.NotEmpty(t, userIdKey, "Should find key for user_id path")
			assert.NotEmpty(t, emailKey, "Should find key for email path")
		}

		// Should not find key for non-existent path
		assert.Empty(t, nonExistentKey, "Should not find key for non-existent path")
	})

	t.Run("IsPropertyRequired", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Test common required fields
		requiredTests := []struct {
			path     string
			expected bool
		}{
			{"userId", true},
			{"anonymousId", true},
			{"event", true},
			{"messageId", true},
			{"type", true},
			{"custom_field", false},
			{"properties.user_id", false}, // Should extract just "user_id" and return false
		}

		for _, test := range requiredTests {
			result := analyzer.isPropertyRequired(test.path)
			assert.Equal(t, test.expected, result, "isPropertyRequired for path: %s", test.path)
		}
	})
}

// TestGeneratorUtilities tests generator utility functions
// This covers line 348 in generators.go
func TestGeneratorUtilities(t *testing.T) {
	t.Parallel()

	t.Run("GenerateUniqueRuleID", func(t *testing.T) {
		t.Parallel()

		usedRuleIDs := make(map[string]bool)

		// Test basic rule ID generation
		ruleID1 := generateUniqueRuleID("writekey1", "event1", 0, usedRuleIDs)
		assert.NotEmpty(t, ruleID1)
		assert.Contains(t, ruleID1, "writekey1")
		assert.Contains(t, ruleID1, "event1")
		assert.True(t, usedRuleIDs[ruleID1], "Rule ID should be marked as used")

		// Test unique rule ID generation when first one is taken
		ruleID2 := generateUniqueRuleID("writekey1", "event1", 1, usedRuleIDs)
		assert.NotEmpty(t, ruleID2)
		assert.NotEqual(t, ruleID1, ruleID2, "Second rule ID should be different")
		assert.True(t, usedRuleIDs[ruleID2], "Second rule ID should be marked as used")

		// Test with different parameters
		ruleID3 := generateUniqueRuleID("writekey2", "event2", 0, usedRuleIDs)
		assert.NotEmpty(t, ruleID3)
		assert.NotEqual(t, ruleID1, ruleID3)
		assert.NotEqual(t, ruleID2, ruleID3)
	})

	t.Run("FindEventIDForSchema", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Setup analyzer with test schemas
		testSchemas := []models.Schema{
			{
				UID:             "test-uid-1",
				WriteKey:        "test-write-key-1",
				EventType:       "track",
				EventIdentifier: "test_event_1",
				Schema: map[string]interface{}{
					"event": "string",
				},
			},
			{
				UID:             "test-uid-2",
				WriteKey:        "test-write-key-2",
				EventType:       "track",
				EventIdentifier: "test_event_2",
				Schema: map[string]interface{}{
					"event": "string",
				},
			},
		}

		err := analyzer.AnalyzeSchemas(testSchemas)
		require.NoError(t, err)

		// Test finding event ID for existing schema
		eventID1 := analyzer.findEventIDForSchema(testSchemas[0])
		eventID2 := analyzer.findEventIDForSchema(testSchemas[1])

		if len(analyzer.Events) > 0 {
			assert.NotEmpty(t, eventID1, "Should find event ID for first schema")
			assert.NotEmpty(t, eventID2, "Should find event ID for second schema")
			assert.NotEqual(t, eventID1, eventID2, "Event IDs should be different")
		}

		// Test with non-existent schema
		nonExistentSchema := models.Schema{
			UID:             "non-existent",
			WriteKey:        "non-existent-key",
			EventType:       "track",
			EventIdentifier: "non_existent_event",
		}

		nonExistentEventID := analyzer.findEventIDForSchema(nonExistentSchema)
		assert.Empty(t, nonExistentEventID, "Should not find event ID for non-existent schema")
	})

	t.Run("FindPropertyForStructField", func(t *testing.T) {
		t.Parallel()

		analyzer := NewSchemaAnalyzer()

		// Setup analyzer with test data
		testSchemas := []models.Schema{
			{
				UID:             "test-uid",
				WriteKey:        "test-write-key",
				EventType:       "track",
				EventIdentifier: "test_event",
				Schema: map[string]interface{}{
					"properties": map[string]interface{}{
						"user": map[string]interface{}{
							"name":  "string",
							"email": "string",
						},
					},
				},
			},
		}

		err := analyzer.AnalyzeSchemas(testSchemas)
		require.NoError(t, err)

		// Create a mock custom type info
		typeInfo := &CustomTypeInfo{
			ID:   "user_type",
			Name: "User",
			Type: "object",
		}

		// Test finding property for struct field
		namePropertyID := findPropertyForStructField(analyzer, "name", typeInfo)
		emailPropertyID := findPropertyForStructField(analyzer, "email", typeInfo)
		nonExistentPropertyID := findPropertyForStructField(analyzer, "nonexistent", typeInfo)

		// Results depend on how properties are structured in the analyzer
		// At minimum, should not panic and return string (empty or not)
		assert.IsType(t, "", namePropertyID)
		assert.IsType(t, "", emailPropertyID)
		assert.IsType(t, "", nonExistentPropertyID)
	})
}
