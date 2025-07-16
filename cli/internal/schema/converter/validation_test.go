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

func TestValidation(t *testing.T) {
	t.Parallel()

	// Shared test schemas for validation requirements
	testSchemas := []models.Schema{
		{
			UID: "test-uid-1", WriteKey: "writekey-1", EventType: "track", EventIdentifier: "user_signed_up",
			Schema: map[string]interface{}{
				"userId": "string", "email": "string",
				"properties": map[string]interface{}{"name": "string", "email": "string"},
			},
		},
		{
			UID: "test-uid-2", WriteKey: "writekey-1", EventType: "track", EventIdentifier: "user_signed_up",
			Schema: map[string]interface{}{
				"userId":     "number",
				"properties": map[string]interface{}{"name": "number"},
			},
		},
		{
			UID: "test-uid-3", WriteKey: "writekey-2", EventType: "track", EventIdentifier: "order_completed",
			Schema: map[string]interface{}{
				"properties": map[string]interface{}{
					"nested_object": map[string]interface{}{"deep_field": "string"},
					"array_field":   []interface{}{map[string]interface{}{"item_name": "string"}},
				},
			},
		},
	}

	cases := []struct {
		category string
		name     string
		validate func(t *testing.T)
	}{
		// Validation Requirements Tests
		{
			category: "Requirements",
			name:     "CustomTypeIDsValidation",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				err := analyzer.AnalyzeSchemas(testSchemas)
				require.NoError(t, err)
				customTypesYAML := analyzer.GenerateCustomTypesYAML()

				seenIDs := make(map[string]bool)
				for _, customType := range customTypesYAML.Spec.Types {
					assert.False(t, seenIDs[customType.ID], "Custom type ID '%s' must be unique", customType.ID)
					seenIDs[customType.ID] = true
				}
			},
		},
		{
			category: "Requirements",
			name:     "PropertyIDsValidation",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				err := analyzer.AnalyzeSchemas(testSchemas)
				require.NoError(t, err)
				propertiesYAML := analyzer.GeneratePropertiesYAML()

				seenIDs := make(map[string]bool)
				propertyNameTypes := make(map[string]string)

				for _, property := range propertiesYAML.Spec.Properties {
					assert.False(t, seenIDs[property.ID], "Property ID '%s' must be unique", property.ID)
					seenIDs[property.ID] = true

					if existingType, exists := propertyNameTypes[property.Name]; exists {
						if property.Type != existingType {
							assert.NotEqual(t, existingType, property.Type,
								"Properties with same name '%s' should have different types to justify different IDs", property.Name)
						}
					}
					propertyNameTypes[property.Name] = property.Type
				}
			},
		},
		{
			category: "Requirements",
			name:     "EventRuleIDsValidation",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				err := analyzer.AnalyzeSchemas(testSchemas)
				require.NoError(t, err)
				trackingPlansYAML := analyzer.GenerateTrackingPlansYAML(testSchemas)

				seenRuleIDs := make(map[string]bool)
				for writeKey, tp := range trackingPlansYAML {
					for _, rule := range tp.Spec.Rules {
						assert.False(t, seenRuleIDs[rule.ID],
							"Event rule ID '%s' in tracking plan '%s' must be globally unique", rule.ID, writeKey)
						seenRuleIDs[rule.ID] = true
						assert.NotEmpty(t, rule.ID, "Rule ID cannot be empty")
					}
				}
			},
		},
		{
			category: "Requirements",
			name:     "IntegrationValidation",
			validate: func(t *testing.T) {
				analyzer := NewSchemaAnalyzer()
				err := analyzer.AnalyzeSchemas(testSchemas)
				require.NoError(t, err)

				eventsYAML := analyzer.GenerateEventsYAML()
				propertiesYAML := analyzer.GeneratePropertiesYAML()
				customTypesYAML := analyzer.GenerateCustomTypesYAML()
				trackingPlansYAML := analyzer.GenerateTrackingPlansYAML(testSchemas)

				tempDir := t.TempDir()
				err = writeYAMLFile(filepath.Join(tempDir, "events.yaml"), eventsYAML, 2)
				require.NoError(t, err)
				err = writeYAMLFile(filepath.Join(tempDir, "properties.yaml"), propertiesYAML, 2)
				require.NoError(t, err)
				err = writeYAMLFile(filepath.Join(tempDir, "custom-types.yaml"), customTypesYAML, 2)
				require.NoError(t, err)

				tpDir := filepath.Join(tempDir, "tracking-plans")
				err = os.MkdirAll(tpDir, 0755)
				require.NoError(t, err)

				for writeKey, tp := range trackingPlansYAML {
					filename := fmt.Sprintf("writekey-%s.yaml", writeKey)
					err = writeYAMLFile(filepath.Join(tpDir, filename), tp, 2)
					require.NoError(t, err)
				}

				assert.FileExists(t, filepath.Join(tempDir, "events.yaml"))
				assert.FileExists(t, filepath.Join(tempDir, "properties.yaml"))
				assert.FileExists(t, filepath.Join(tempDir, "custom-types.yaml"))

				verifyYAMLStructure(t, filepath.Join(tempDir, "events.yaml"))
				verifyYAMLStructure(t, filepath.Join(tempDir, "properties.yaml"))
				verifyYAMLStructure(t, filepath.Join(tempDir, "custom-types.yaml"))
			},
		},

		// Custom Type Name Generation Tests
		{
			category: "CustomTypeNames",
			name:     "NameGenerationValidation",
			validate: func(t *testing.T) {
				tests := []struct {
					name        string
					path        string
					baseType    string
					expectedLen int
				}{
					{"Simple path", "user.profile", "object", 15},
					{"Path with symbols", "user_123.profile-data.info$test", "object", 21},
					{"Array type", "items", "array", 10},
					{"Empty path", "", "object", 13},
					{"Very long path", "very.long.path.with.many.nested.fields.that.should.be.truncated.to.fit.within.sixtyfive.character.limit", "object", 65},
				}

				customTypeNameRegex := regexp.MustCompile(`^[A-Z][a-zA-Z]{2,64}$`)
				customTypeFactory := NewCustomTypeFactory()

				for _, tt := range tests {
					result := customTypeFactory.generateCustomTypeName(tt.path, tt.baseType)
					assert.True(t, len(result) >= 3 && len(result) <= 65,
						"Generated name '%s' should be 3-65 characters, got %d", result, len(result))
					assert.True(t, customTypeNameRegex.MatchString(result),
						"Generated name '%s' should match validation regex", result)
					assert.True(t, result[0] >= 'A' && result[0] <= 'Z',
						"Generated name '%s' should start with uppercase letter", result)
				}
			},
		},

		// Property ID Generation Tests
		{
			category: "PropertyIDs",
			name:     "IDGenerationValidation",
			validate: func(t *testing.T) {
				tests := []struct {
					name     string
					propName string
					propType string
					expected string
				}{
					{"String property", "email", "string", "email_string"},
					{"Number property with same name", "email", "number", "email_number"},
					{"Custom type property", "user", "#/custom-types/test/UserType", "user_usertype"},
				}

				propertyFactory := NewPropertyFactory()
				for _, tt := range tests {
					mockAnalyzer := &SchemaAnalyzer{UsedPropertyIDs: make(map[string]bool)}
					result := propertyFactory.generateUniquePropertyID(mockAnalyzer, tt.propName, tt.propType)
					assert.Equal(t, tt.expected, result)
				}
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

func verifyYAMLStructure(t *testing.T, filename string) {
	data, err := os.ReadFile(filename)
	require.NoError(t, err)

	var parsed interface{}
	err = yaml.Unmarshal(data, &parsed)
	require.NoError(t, err, "File %s should contain valid YAML", filename)
}
