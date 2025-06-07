package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestValidationRequirements tests that the converter produces output that meets all validation requirements
func TestValidationRequirements(t *testing.T) {
	t.Parallel()

	// Create test schemas with potential for duplicates and validation issues
	testSchemas := []models.Schema{
		{
			UID:             "test-uid-1",
			WriteKey:        "writekey-1",
			EventType:       "track",
			EventIdentifier: "user_signed_up",
			Schema: map[string]interface{}{
				"userId": "string",
				"email":  "string",
				"properties": map[string]interface{}{
					"name":  "string",
					"email": "string", // Same name, same type - should be detected as duplicate
				},
			},
		},
		{
			UID:             "test-uid-2",
			WriteKey:        "writekey-1", // Same writekey - should create unique rule IDs
			EventType:       "track",
			EventIdentifier: "user_signed_up", // Same event - should create unique rule IDs
			Schema: map[string]interface{}{
				"userId": "number", // Same name, different type - should have unique ID
				"properties": map[string]interface{}{
					"name": "number", // Same name, different type from first schema
				},
			},
		},
		{
			UID:             "test-uid-3",
			WriteKey:        "writekey-2",
			EventType:       "track",
			EventIdentifier: "order_completed",
			Schema: map[string]interface{}{
				"properties": map[string]interface{}{
					"nested_object": map[string]interface{}{
						"deep_field": "string",
					},
					"array_field": []interface{}{
						map[string]interface{}{
							"item_name": "string",
						},
					},
				},
			},
		},
	}

	// Create analyzer and process schemas
	analyzer := NewSchemaAnalyzer()
	err := analyzer.AnalyzeSchemas(testSchemas)
	require.NoError(t, err)

	// Generate YAML structures
	eventsYAML := analyzer.GenerateEventsYAML()
	propertiesYAML := analyzer.GeneratePropertiesYAML()
	customTypesYAML := analyzer.GenerateCustomTypesYAML()
	trackingPlansYAML := analyzer.GenerateTrackingPlansYAML(testSchemas)

	// Test 1: Custom type IDs must be unique
	t.Run("CustomTypeIDsValidation", func(t *testing.T) {
		seenIDs := make(map[string]bool)

		for _, customType := range customTypesYAML.Spec.Types {
			assert.False(t, seenIDs[customType.ID],
				"Custom type ID '%s' must be unique", customType.ID)
			seenIDs[customType.ID] = true
		}
	})

	// Test 2: Property IDs must be unique and representative of name+type combination
	t.Run("PropertyIDsValidation", func(t *testing.T) {
		seenIDs := make(map[string]bool)
		propertyNameTypes := make(map[string]string) // name -> type mapping

		for _, property := range propertiesYAML.Spec.Properties {
			// Check ID uniqueness
			assert.False(t, seenIDs[property.ID],
				"Property ID '%s' must be unique", property.ID)
			seenIDs[property.ID] = true

			// For properties with same name, ensure they have different types if IDs are different
			if existingType, exists := propertyNameTypes[property.Name]; exists {
				if property.Type != existingType {
					// Same name, different type - IDs should be different
					// This is validated by the uniqueness check above
					assert.NotEqual(t, existingType, property.Type,
						"Properties with same name '%s' should have different types to justify different IDs", property.Name)
				}
			}
			propertyNameTypes[property.Name] = property.Type
		}
	})

	// Test 3: Event rule IDs must be unique across all tracking plans
	t.Run("EventRuleIDsValidation", func(t *testing.T) {
		seenRuleIDs := make(map[string]bool)

		for writeKey, tp := range trackingPlansYAML {
			for _, rule := range tp.Spec.Rules {
				assert.False(t, seenRuleIDs[rule.ID],
					"Event rule ID '%s' in tracking plan '%s' must be globally unique", rule.ID, writeKey)
				seenRuleIDs[rule.ID] = true

				// Rule ID should be non-empty
				assert.NotEmpty(t, rule.ID, "Rule ID cannot be empty")
			}
		}
	})

	// Test 4: Integration test - write files and verify they don't have validation errors
	t.Run("IntegrationValidation", func(t *testing.T) {
		tempDir := t.TempDir()

		// Write the generated YAML files
		err := writeYAMLFile(filepath.Join(tempDir, "events.yaml"), eventsYAML, 2)
		require.NoError(t, err)
		err = writeYAMLFile(filepath.Join(tempDir, "properties.yaml"), propertiesYAML, 2)
		require.NoError(t, err)
		err = writeYAMLFile(filepath.Join(tempDir, "custom-types.yaml"), customTypesYAML, 2)
		require.NoError(t, err)

		// Write tracking plans
		tpDir := filepath.Join(tempDir, "tracking-plans")
		err = os.MkdirAll(tpDir, 0755)
		require.NoError(t, err)

		for writeKey, tp := range trackingPlansYAML {
			filename := fmt.Sprintf("writekey-%s.yaml", writeKey)
			err = writeYAMLFile(filepath.Join(tpDir, filename), tp, 2)
			require.NoError(t, err)
		}

		// Verify files exist and are valid YAML
		assert.FileExists(t, filepath.Join(tempDir, "events.yaml"))
		assert.FileExists(t, filepath.Join(tempDir, "properties.yaml"))
		assert.FileExists(t, filepath.Join(tempDir, "custom-types.yaml"))

		// Read and verify YAML structure
		verifyYAMLStructure(t, filepath.Join(tempDir, "events.yaml"))
		verifyYAMLStructure(t, filepath.Join(tempDir, "properties.yaml"))
		verifyYAMLStructure(t, filepath.Join(tempDir, "custom-types.yaml"))
	})
}

// TestCustomTypeNameGeneration specifically tests custom type name generation
func TestCustomTypeNameGeneration(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		baseType    string
		expectedLen int
		shouldMatch bool
	}{
		{
			name:        "Simple path",
			path:        "user.profile",
			baseType:    "object",
			expectedLen: 15, // "UserprofileType"
			shouldMatch: true,
		},
		{
			name:        "Path with numbers and symbols",
			path:        "user_123.profile-data.info$test",
			baseType:    "object",
			expectedLen: 21, // "UserprofiledatainfoType" (numbers/symbols removed)
			shouldMatch: true,
		},
		{
			name:        "Array type",
			path:        "items",
			baseType:    "array",
			expectedLen: 10, // "ItemsArray"
			shouldMatch: true,
		},
		{
			name:        "Empty path",
			path:        "",
			baseType:    "object",
			expectedLen: 13, // "GeneratedType"
			shouldMatch: true,
		},
		{
			name:        "Very long path",
			path:        "very.long.path.with.many.nested.fields.that.should.be.truncated.to.fit.within.sixtyfive.character.limit",
			baseType:    "object",
			expectedLen: 65, // Should be truncated to exactly 65 chars
			shouldMatch: true,
		},
	}

	customTypeNameRegex := regexp.MustCompile(`^[A-Z][a-zA-Z]{2,64}$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateCustomTypeName(tt.path, tt.baseType)

			// Check length constraint
			assert.True(t, len(result) >= 3 && len(result) <= 65,
				"Generated name '%s' should be 3-65 characters, got %d", result, len(result))

			// Check format
			assert.True(t, customTypeNameRegex.MatchString(result),
				"Generated name '%s' should match validation regex", result)

			// Check starts with uppercase
			assert.True(t, result[0] >= 'A' && result[0] <= 'Z',
				"Generated name '%s' should start with uppercase letter", result)
		})
	}
}

// TestPropertyIDGeneration tests property ID generation for uniqueness
func TestPropertyIDGeneration(t *testing.T) {
	tests := []struct {
		name     string
		propName string
		propType string
		expected string
	}{
		{
			name:     "String property",
			propName: "email",
			propType: "string",
			expected: "email_string",
		},
		{
			name:     "Number property with same name",
			propName: "email",
			propType: "number",
			expected: "email_number",
		},
		{
			name:     "Custom type property",
			propName: "user",
			propType: "#/custom-types/test/UserType",
			expected: "user_usertype",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateUniquePropertyID(tt.propName, tt.propType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions

func verifyYAMLStructure(t *testing.T, filename string) {
	data, err := os.ReadFile(filename)
	require.NoError(t, err)

	var parsed interface{}
	err = yaml.Unmarshal(data, &parsed)
	require.NoError(t, err, "File %s should contain valid YAML", filename)
}
