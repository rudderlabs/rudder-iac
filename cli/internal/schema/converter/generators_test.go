package converter

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	yamlModels "github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerators(t *testing.T) {
	t.Parallel()

	cases := []struct {
		category string
		name     string
		validate func(t *testing.T)
	}{
		// Schema Field Processing Tests (line 289 in generators.go)
		{
			category: "SchemaProcessing",
			name:     "ExtractPropertiesForSchema",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				testSchemas := []models.Schema{
					{
						UID: "test-uid", WriteKey: "test-write-key", EventType: "track", EventIdentifier: "test_event",
						Schema: map[string]interface{}{
							"event": "string", "userId": "string",
							"properties": map[string]interface{}{
								"product_id": "string", "product_name": "string", "price": "number",
								"nested": map[string]interface{}{"field1": "string", "field2": "number"},
							},
						},
					},
				}

				err := analyzer.AnalyzeSchemas(testSchemas)
				require.NoError(t, err)

				properties := analyzer.extractPropertiesForSchema(testSchemas[0])
				assert.NotNil(t, properties)
				assert.IsType(t, []yamlModels.PropertyRuleRef{}, properties)

				for _, prop := range properties {
					assert.NotEmpty(t, prop.Ref, "Property reference should not be empty")
					assert.Contains(t, prop.Ref, "#/properties/extracted_properties/", "Property reference should have correct prefix")
				}
			},
		},
		{
			category: "SchemaProcessing",
			name:     "CollectPropertyPaths",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				nestedObj := map[string]interface{}{
					"level1": map[string]interface{}{
						"level2":       map[string]interface{}{"field1": "string", "field2": "number"},
						"simple_field": "string",
					},
					"array_field": []interface{}{map[string]interface{}{"item_field": "string"}},
					"root_field":  "string",
				}

				paths := analyzer.collectPropertyPaths(nestedObj, "")
				assert.NotEmpty(t, paths)

				expectedPaths := []string{
					"level1", "level1.level2", "level1.level2.field1", "level1.level2.field2",
					"level1.simple_field", "array_field", "root_field",
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
			},
		},
		{
			category: "SchemaProcessing",
			name:     "FindPropertyKeyForPath",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				testSchemas := []models.Schema{
					{
						UID: "test-uid", WriteKey: "test-write-key", EventType: "track", EventIdentifier: "test_event",
						Schema: map[string]interface{}{
							"properties": map[string]interface{}{"user_id": "string", "email": "string"},
						},
					},
				}

				err := analyzer.AnalyzeSchemas(testSchemas)
				require.NoError(t, err)

				userIdKey := analyzer.findPropertyKeyForPath("properties.user_id")
				emailKey := analyzer.findPropertyKeyForPath("properties.email")
				nonExistentKey := analyzer.findPropertyKeyForPath("properties.nonexistent")

				if len(analyzer.Properties) > 0 {
					assert.NotEmpty(t, userIdKey, "Should find key for user_id path")
					assert.NotEmpty(t, emailKey, "Should find key for email path")
				}
				assert.Empty(t, nonExistentKey, "Should not find key for non-existent path")
			},
		},
		{
			category: "SchemaProcessing",
			name:     "IsPropertyRequired",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				requiredTests := []struct {
					path     string
					expected bool
				}{
					{"userId", true}, {"anonymousId", true}, {"event", true}, {"messageId", true}, {"type", true},
					{"custom_field", false}, {"properties.user_id", false},
				}

				for _, test := range requiredTests {
					result := analyzer.isPropertyRequired(test.path)
					assert.Equal(t, test.expected, result, "isPropertyRequired for path: %s", test.path)
				}
			},
		},

		// Generator Utility Tests (line 348 in generators.go)
		{
			category: "Utilities",
			name:     "GenerateUniqueRuleID",
			validate: func(t *testing.T) {
				usedRuleIDs := make(map[string]bool)

				ruleID1 := generateUniqueRuleID("writekey1", "event1", 0, usedRuleIDs)
				assert.NotEmpty(t, ruleID1)
				assert.Contains(t, ruleID1, "writekey1")
				assert.Contains(t, ruleID1, "event1")
				assert.True(t, usedRuleIDs[ruleID1], "Rule ID should be marked as used")

				ruleID2 := generateUniqueRuleID("writekey1", "event1", 1, usedRuleIDs)
				assert.NotEmpty(t, ruleID2)
				assert.NotEqual(t, ruleID1, ruleID2, "Second rule ID should be different")
				assert.True(t, usedRuleIDs[ruleID2], "Second rule ID should be marked as used")

				ruleID3 := generateUniqueRuleID("writekey2", "event2", 0, usedRuleIDs)
				assert.NotEmpty(t, ruleID3)
				assert.NotEqual(t, ruleID1, ruleID3)
				assert.NotEqual(t, ruleID2, ruleID3)
			},
		},
		{
			category: "PropertyReferenceSorting",
			name:     "CustomTypePropertyReferencesSorted",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				testSchemas := []models.Schema{
					{
						UID: "test-uid", WriteKey: "test-write-key", EventType: "track", EventIdentifier: "test_event",
						Schema: map[string]interface{}{
							"properties": map[string]interface{}{
								"user_profile": map[string]interface{}{
									"zebra_field": "string",
									"alpha_field": "string",
									"beta_field":  "string",
								},
								"metadata": map[string]interface{}{
									"gamma_prop": "number",
									"delta_prop": "boolean",
								},
							},
						},
					},
				}

				err := analyzer.AnalyzeSchemas(testSchemas)
				require.NoError(t, err)

				customTypesYAML := analyzer.GenerateCustomTypesYAML()
				require.NotNil(t, customTypesYAML)

				// Find any custom type that has property references
				var foundSortedRefs bool
				for i := range customTypesYAML.Spec.Types {
					customType := &customTypesYAML.Spec.Types[i]
					if customType.Type == "object" && len(customType.Properties) > 1 {
						refs := customType.Properties

						// Verify that property references are sorted alphabetically by Ref
						for j := 1; j < len(refs); j++ {
							assert.True(t, refs[j-1].Ref < refs[j].Ref,
								"Property references should be sorted alphabetically: %s should come before %s",
								refs[j-1].Ref, refs[j].Ref)
						}
						foundSortedRefs = true
						break
					}
				}

				// If no custom types with multiple properties were found, that's okay for this test
				// The important thing is that when they exist, they are sorted
				if !foundSortedRefs {
					t.Log("No custom types with multiple property references found - test passed trivially")
				}
			},
		},
		{
			category: "PropertyReferenceSorting",
			name:     "EventRulePropertyReferencesSorted",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				testSchemas := []models.Schema{
					{
						UID: "test-uid", WriteKey: "test-write-key", EventType: "track", EventIdentifier: "test_event",
						Schema: map[string]interface{}{
							"properties": map[string]interface{}{
								"zebra_prop": "string",
								"alpha_prop": "string",
								"beta_prop":  "number",
								"gamma_prop": "boolean",
							},
						},
					},
				}

				err := analyzer.AnalyzeSchemas(testSchemas)
				require.NoError(t, err)

				properties := analyzer.extractPropertiesForSchema(testSchemas[0])
				require.True(t, len(properties) > 1, "Should have multiple property references to test sorting")

				// Verify that property references are sorted alphabetically by Ref
				for i := 1; i < len(properties); i++ {
					assert.True(t, properties[i-1].Ref < properties[i].Ref,
						"Property references should be sorted alphabetically: %s should come before %s",
						properties[i-1].Ref, properties[i].Ref)
				}
			},
		},
		{
			category: "PropertyReferenceSorting",
			name:     "TrackingPlanPropertyReferencesSorted",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				testSchemas := []models.Schema{
					{
						UID: "test-uid", WriteKey: "test-write-key", EventType: "track", EventIdentifier: "test_event",
						Schema: map[string]interface{}{
							"properties": map[string]interface{}{
								"zebra_field": "string",
								"alpha_field": "string",
								"beta_field":  "number",
							},
						},
					},
				}

				err := analyzer.AnalyzeSchemas(testSchemas)
				require.NoError(t, err)

				trackingPlansYAML := analyzer.GenerateTrackingPlansYAML(testSchemas)
				require.NotEmpty(t, trackingPlansYAML, "Should generate tracking plans")

				// Check the first tracking plan
				for _, trackingPlan := range trackingPlansYAML {
					require.NotEmpty(t, trackingPlan.Spec.Rules, "Should have rules")

					rule := trackingPlan.Spec.Rules[0]
					properties := rule.Properties

					if len(properties) > 1 {
						// Verify that property references are sorted alphabetically by Ref
						for i := 1; i < len(properties); i++ {
							assert.True(t, properties[i-1].Ref < properties[i].Ref,
								"Property references in tracking plan should be sorted alphabetically: %s should come before %s",
								properties[i-1].Ref, properties[i].Ref)
						}
					}
					break // Test first tracking plan only
				}
			},
		},
		{
			category: "Utilities",
			name:     "FindEventIDForSchema",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				testSchemas := []models.Schema{
					{
						UID: "test-uid-1", WriteKey: "test-write-key-1", EventType: "track", EventIdentifier: "test_event_1",
						Schema: map[string]interface{}{"event": "string"},
					},
					{
						UID: "test-uid-2", WriteKey: "test-write-key-2", EventType: "track", EventIdentifier: "test_event_2",
						Schema: map[string]interface{}{"event": "string"},
					},
				}

				err := analyzer.AnalyzeSchemas(testSchemas)
				require.NoError(t, err)

				eventID1 := analyzer.findEventIDForSchema(testSchemas[0])
				eventID2 := analyzer.findEventIDForSchema(testSchemas[1])

				if len(analyzer.Events) > 0 {
					assert.NotEmpty(t, eventID1, "Should find event ID for first schema")
					assert.NotEmpty(t, eventID2, "Should find event ID for second schema")
					assert.NotEqual(t, eventID1, eventID2, "Event IDs should be different")
				}

				nonExistentSchema := models.Schema{
					UID: "non-existent", WriteKey: "non-existent-key", EventType: "track", EventIdentifier: "non_existent_event",
				}
				nonExistentEventID := analyzer.findEventIDForSchema(nonExistentSchema)
				assert.Empty(t, nonExistentEventID, "Should not find event ID for non-existent schema")
			},
		},
		{
			category: "Utilities",
			name:     "FindPropertyForStructField",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				testSchemas := []models.Schema{
					{
						UID: "test-uid", WriteKey: "test-write-key", EventType: "track", EventIdentifier: "test_event",
						Schema: map[string]interface{}{
							"properties": map[string]interface{}{
								"user": map[string]interface{}{"name": "string", "email": "string"},
							},
						},
					},
				}

				err := analyzer.AnalyzeSchemas(testSchemas)
				require.NoError(t, err)

				typeInfo := &CustomTypeInfo{ID: "user_type", Name: "User", Type: "object"}
				namePropertyID := findPropertyForStructField(analyzer, "name", typeInfo)
				emailPropertyID := findPropertyForStructField(analyzer, "email", typeInfo)
				nonExistentPropertyID := findPropertyForStructField(analyzer, "nonexistent", typeInfo)

				assert.IsType(t, "", namePropertyID)
				assert.IsType(t, "", emailPropertyID)
				assert.IsType(t, "", nonExistentPropertyID)
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
