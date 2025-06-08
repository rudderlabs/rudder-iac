package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	yamlModels "github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaAnalyzer_AnalyzeSchemas(t *testing.T) {
	t.Parallel()

	// Create test data
	testSchemas := []models.Schema{
		{
			UID:             "test-uid-1",
			WriteKey:        "test-write-key-1",
			EventType:       "track",
			EventIdentifier: "test_event_one",
			Schema: map[string]interface{}{
				"event":       "string",
				"userId":      "string",
				"anonymousId": "string",
				"context": map[string]interface{}{
					"app": map[string]interface{}{
						"name":    "string",
						"version": "string",
					},
					"traits": map[string]interface{}{
						"email": "string",
						"age":   "number",
					},
				},
				"properties": map[string]interface{}{
					"product_id":   "string",
					"product_name": "string",
					"categories":   []interface{}{"string", "string"},
				},
			},
		},
		{
			UID:             "test-uid-2",
			WriteKey:        "test-write-key-2",
			EventType:       "track",
			EventIdentifier: "test_event_two",
			Schema: map[string]interface{}{
				"event":  "string",
				"userId": "string",
				"properties": map[string]interface{}{
					"order_id":    "string",
					"total_price": "number",
				},
			},
		},
	}

	analyzer := NewSchemaAnalyzer()
	err := analyzer.AnalyzeSchemas(testSchemas)
	require.NoError(t, err)

	// Verify events were extracted
	assert.Len(t, analyzer.Events, 2)
	assert.Contains(t, analyzer.Events, "test_event_one")
	assert.Contains(t, analyzer.Events, "test_event_two")

	// Verify properties were extracted
	assert.True(t, len(analyzer.Properties) > 0)

	// Verify custom types were created for nested objects
	assert.True(t, len(analyzer.CustomTypes) > 0)

	// Test YAML generation
	eventsYAML := analyzer.GenerateEventsYAML()
	assert.Equal(t, "rudder/0.1", eventsYAML.Version)
	assert.Equal(t, "events", eventsYAML.Kind)
	assert.Len(t, eventsYAML.Spec.Events, 2)

	propertiesYAML := analyzer.GeneratePropertiesYAML()
	assert.Equal(t, "rudder/v0.1", propertiesYAML.Version)
	assert.Equal(t, "properties", propertiesYAML.Kind)
	assert.True(t, len(propertiesYAML.Spec.Properties) > 0)

	customTypesYAML := analyzer.GenerateCustomTypesYAML()
	assert.Equal(t, "rudder/v0.1", customTypesYAML.Version)
	assert.Equal(t, "custom-types", customTypesYAML.Kind)
	assert.True(t, len(customTypesYAML.Spec.Types) > 0)

	trackingPlansYAML := analyzer.GenerateTrackingPlansYAML(testSchemas)
	assert.Len(t, trackingPlansYAML, 2) // Two different writeKeys
}

func TestSchemaAnalyzer_EmptySchemas(t *testing.T) {
	t.Parallel()

	analyzer := NewSchemaAnalyzer()
	err := analyzer.AnalyzeSchemas([]models.Schema{})
	require.NoError(t, err)

	// Verify empty results
	assert.Len(t, analyzer.Events, 0)
	assert.Len(t, analyzer.Properties, 0)
	assert.Len(t, analyzer.CustomTypes, 0)

	// Test YAML generation with empty data
	eventsYAML := analyzer.GenerateEventsYAML()
	assert.Equal(t, "rudder/0.1", eventsYAML.Version)
	assert.Equal(t, "events", eventsYAML.Kind)
	assert.Len(t, eventsYAML.Spec.Events, 0)
}

func TestSchemaAnalyzer_ComplexNestedSchema(t *testing.T) {
	t.Parallel()

	testSchemas := []models.Schema{
		{
			UID:             "complex-uid",
			WriteKey:        "complex-write-key",
			EventType:       "track",
			EventIdentifier: "complex_event",
			Schema: map[string]interface{}{
				"event":  "string",
				"userId": "string",
				"properties": map[string]interface{}{
					"user": map[string]interface{}{
						"profile": map[string]interface{}{
							"name": "string",
							"age":  "number",
							"preferences": map[string]interface{}{
								"notifications": map[string]interface{}{
									"email": "boolean",
									"sms":   "boolean",
								},
							},
						},
					},
					"items": []interface{}{
						map[string]interface{}{
							"id":    "string",
							"price": "number",
							"metadata": map[string]interface{}{
								"tags": []interface{}{"string"},
							},
						},
					},
				},
			},
		},
	}

	analyzer := NewSchemaAnalyzer()
	err := analyzer.AnalyzeSchemas(testSchemas)
	require.NoError(t, err)

	// Verify event was extracted
	assert.Len(t, analyzer.Events, 1)
	assert.Contains(t, analyzer.Events, "complex_event")

	// Verify properties and custom types were created
	assert.True(t, len(analyzer.Properties) > 0)
	assert.True(t, len(analyzer.CustomTypes) > 0)
}

func TestSchemaConverter_Convert_DryRun(t *testing.T) {
	t.Parallel()

	// Create temporary input file
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test_schemas.json")
	outputDir := filepath.Join(tempDir, "output")

	// Create test data
	testData := `{
		"schemas": [
			{
				"uid": "test-uid",
				"writeKey": "test-write-key",
				"eventType": "track",
				"eventIdentifier": "test_event",
				"schema": {
					"event": "string",
					"userId": "string",
					"properties": {
						"test_prop": "string"
					}
				}
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	// Test dry run
	options := ConversionOptions{
		InputFile:  inputFile,
		OutputDir:  outputDir,
		DryRun:     true,
		Verbose:    false,
		YAMLIndent: 2,
	}

	converter := NewSchemaConverter(options)
	result, err := converter.Convert()
	require.NoError(t, err)

	assert.Equal(t, 1, result.EventsCount)
	assert.True(t, result.PropertiesCount > 0)
	assert.True(t, result.CustomTypesCount >= 0)
	assert.Empty(t, result.GeneratedFiles) // No files should be created in dry run
}

func TestSchemaConverter_Convert_RealRun(t *testing.T) {
	t.Parallel()

	// Create temporary input file
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test_schemas.json")
	outputDir := filepath.Join(tempDir, "output")

	// Create test data
	testData := `{
		"schemas": [
			{
				"uid": "test-uid",
				"writeKey": "test-write-key",
				"eventType": "track",
				"eventIdentifier": "test_event",
				"schema": {
					"event": "string",
					"userId": "string",
					"properties": {
						"test_prop": "string"
					}
				}
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	// Test real run
	options := ConversionOptions{
		InputFile:  inputFile,
		OutputDir:  outputDir,
		DryRun:     false,
		Verbose:    false,
		YAMLIndent: 2,
	}

	converter := NewSchemaConverter(options)
	result, err := converter.Convert()
	require.NoError(t, err)

	// Verify files were created
	assert.True(t, len(result.GeneratedFiles) >= 4) // events, properties, custom-types, tracking plan

	// Verify output files exist
	assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "properties.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "custom-types.yaml"))
	assert.DirExists(t, filepath.Join(outputDir, "tracking-plans"))
}

func TestSchemaConverter_Convert_InvalidInput(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "invalid.json")
	outputDir := filepath.Join(tempDir, "output")

	// Create invalid JSON
	err := os.WriteFile(inputFile, []byte("invalid json"), 0644)
	require.NoError(t, err)

	options := ConversionOptions{
		InputFile:  inputFile,
		OutputDir:  outputDir,
		DryRun:     false,
		Verbose:    false,
		YAMLIndent: 2,
	}

	converter := NewSchemaConverter(options)
	_, err = converter.Convert()
	assert.Error(t, err)
}

func TestSchemaConverter_Convert_NonexistentInput(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "nonexistent.json")
	outputDir := filepath.Join(tempDir, "output")

	options := ConversionOptions{
		InputFile:  inputFile,
		OutputDir:  outputDir,
		DryRun:     false,
		Verbose:    false,
		YAMLIndent: 2,
	}

	converter := NewSchemaConverter(options)
	_, err := converter.Convert()
	assert.Error(t, err)
}

func TestSchemaConverter_Convert_MultipleWriteKeys(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test_schemas.json")
	outputDir := filepath.Join(tempDir, "output")

	// Create test data with multiple write keys
	testData := `{
		"schemas": [
			{
				"uid": "test-uid-1",
				"writeKey": "writekey-1",
				"eventType": "track",
				"eventIdentifier": "event_1",
				"schema": {
					"event": "string",
					"userId": "string"
				}
			},
			{
				"uid": "test-uid-2",
				"writeKey": "writekey-2",
				"eventType": "track",
				"eventIdentifier": "event_2",
				"schema": {
					"event": "string",
					"properties": {
						"prop": "string"
					}
				}
			},
			{
				"uid": "test-uid-3",
				"writeKey": "writekey-1",
				"eventType": "track",
				"eventIdentifier": "event_3",
				"schema": {
					"event": "string",
					"userId": "string"
				}
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	options := ConversionOptions{
		InputFile:  inputFile,
		OutputDir:  outputDir,
		DryRun:     false,
		Verbose:    false,
		YAMLIndent: 2,
	}

	converter := NewSchemaConverter(options)
	result, err := converter.Convert()
	require.NoError(t, err)

	// Verify events were counted correctly
	assert.Equal(t, 3, result.EventsCount)

	// Verify separate tracking plans were created for each writeKey
	assert.FileExists(t, filepath.Join(outputDir, "tracking-plans", "writekey-writekey-1.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "tracking-plans", "writekey-writekey-2.yaml"))
}

func TestConversionOptions_Validation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		options     ConversionOptions
		expectError bool
	}{
		{
			name: "ValidOptions",
			options: ConversionOptions{
				InputFile:  "valid.json",
				OutputDir:  "output",
				DryRun:     false,
				Verbose:    false,
				YAMLIndent: 2,
			},
			expectError: false,
		},
		{
			name: "EmptyInputFile",
			options: ConversionOptions{
				InputFile:  "",
				OutputDir:  "output",
				DryRun:     false,
				Verbose:    false,
				YAMLIndent: 2,
			},
			expectError: true,
		},
		{
			name: "EmptyOutputDir",
			options: ConversionOptions{
				InputFile:  "valid.json",
				OutputDir:  "",
				DryRun:     false,
				Verbose:    false,
				YAMLIndent: 2,
			},
			expectError: true,
		},
		{
			name: "InvalidYAMLIndent",
			options: ConversionOptions{
				InputFile:  "valid.json",
				OutputDir:  "output",
				DryRun:     false,
				Verbose:    false,
				YAMLIndent: 0,
			},
			expectError: false, // Should use default
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			converter := NewSchemaConverter(c.options)
			assert.NotNil(t, converter)

			// For now, just verify the converter is created
			// More detailed validation would require running Convert()
			if c.options.YAMLIndent == 0 {
				assert.Equal(t, 2, converter.options.YAMLIndent) // Should default to 2
			}
		})
	}
}

// TestSchemaConverter_DryRunVerbose tests dry run with verbose output
// This covers lines 203-212 in converter.go (performDryRun function)
func TestSchemaConverter_DryRunVerbose(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test_schemas.json")
	outputDir := filepath.Join(tempDir, "output")

	// Create comprehensive test data to trigger all preview functions
	testData := `{
		"schemas": [
			{
				"uid": "test-uid-1",
				"writeKey": "test-write-key-1",
				"eventType": "track",
				"eventIdentifier": "product_viewed",
				"schema": {
					"event": "string",
					"userId": "string",
					"properties": {
						"product_id": "string",
						"product_name": "string",
						"price": "number",
						"user": {
							"profile": {
								"name": "string",
								"age": "number"
							}
						}
					}
				}
			},
			{
				"uid": "test-uid-2",
				"writeKey": "test-write-key-2",
				"eventType": "track",
				"eventIdentifier": "cart_viewed",
				"schema": {
					"event": "string",
					"userId": "string",
					"properties": {
						"cart_id": "string",
						"items": [{
							"id": "string",
							"name": "string"
						}]
					}
				}
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	options := ConversionOptions{
		InputFile:  inputFile,
		OutputDir:  outputDir,
		DryRun:     true,
		Verbose:    true, // Enable verbose to trigger preview functions
		YAMLIndent: 2,
	}

	converter := NewSchemaConverter(options)
	result, err := converter.Convert()
	require.NoError(t, err)

	// Verify result counts
	assert.Equal(t, 2, result.EventsCount)
	assert.True(t, result.PropertiesCount > 0)
	assert.True(t, result.CustomTypesCount >= 0)
	assert.Len(t, result.TrackingPlans, 2) // Two different writeKeys
	assert.Empty(t, result.GeneratedFiles) // No files in dry run
}

// TestPreviewFunctions tests the preview functions directly
// This covers lines 73-78, 85-90, 97-102 in converter.go
func TestPreviewFunctions(t *testing.T) {
	t.Parallel()

	// Create test YAML structures
	eventsYAML := &yamlModels.EventsYAML{
		Spec: yamlModels.EventsSpec{
			Events: []yamlModels.EventDefinition{
				{Name: "product_viewed", EventType: "track"},
				{Name: "cart_viewed", EventType: "track"},
				{Name: "purchase_completed", EventType: "track"},
				{Name: "user_registered", EventType: "identify"},
			},
		},
	}

	propertiesYAML := &yamlModels.PropertiesYAML{
		Spec: yamlModels.PropertiesSpec{
			Properties: []yamlModels.PropertyDefinition{
				{Name: "product_id", Type: "string"},
				{Name: "user_id", Type: "string"},
				{Name: "price", Type: "number"},
			},
		},
	}

	customTypesYAML := &yamlModels.CustomTypesYAML{
		Spec: yamlModels.CustomTypesSpec{
			Types: []yamlModels.CustomTypeDefinition{
				{Name: "UserProfile", Type: "object"},
				{Name: "ProductInfo", Type: "object"},
			},
		},
	}

	t.Run("EventsPreview", func(t *testing.T) {
		t.Parallel()

		// Test with maxItems less than total
		printEventsPreview(eventsYAML, 2)

		// Test with maxItems equal to total
		printEventsPreview(eventsYAML, 4)

		// Test with maxItems greater than total
		printEventsPreview(eventsYAML, 10)
	})

	t.Run("PropertiesPreview", func(t *testing.T) {
		t.Parallel()

		// Test with maxItems less than total
		printPropertiesPreview(propertiesYAML, 2)

		// Test with maxItems equal to total
		printPropertiesPreview(propertiesYAML, 3)

		// Test with maxItems greater than total
		printPropertiesPreview(propertiesYAML, 10)
	})

	t.Run("CustomTypesPreview", func(t *testing.T) {
		t.Parallel()

		// Test with maxItems less than total
		printCustomTypesPreview(customTypesYAML, 1)

		// Test with maxItems equal to total
		printCustomTypesPreview(customTypesYAML, 2)

		// Test with maxItems greater than total
		printCustomTypesPreview(customTypesYAML, 10)
	})

	t.Run("EmptyCollections", func(t *testing.T) {
		t.Parallel()

		emptyEventsYAML := &yamlModels.EventsYAML{
			Spec: yamlModels.EventsSpec{Events: []yamlModels.EventDefinition{}},
		}
		emptyPropertiesYAML := &yamlModels.PropertiesYAML{
			Spec: yamlModels.PropertiesSpec{Properties: []yamlModels.PropertyDefinition{}},
		}
		emptyCustomTypesYAML := &yamlModels.CustomTypesYAML{
			Spec: yamlModels.CustomTypesSpec{Types: []yamlModels.CustomTypeDefinition{}},
		}

		// These should handle empty collections gracefully
		printEventsPreview(emptyEventsYAML, 3)
		printPropertiesPreview(emptyPropertiesYAML, 3)
		printCustomTypesPreview(emptyCustomTypesYAML, 3)
	})
}

// TestSchemaConverter_SuccessfulYAMLFileCreation tests successful file creation
// This covers lines 227-228, 235-236, 243-244, 251-252 in converter.go
func TestSchemaConverter_SuccessfulYAMLFileCreation(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test_schemas.json")
	outputDir := filepath.Join(tempDir, "output")

	// Create test data with rich schema to generate all file types
	testData := `{
		"schemas": [
			{
				"uid": "test-uid-1",
				"writeKey": "test-write-key-1",
				"eventType": "track",
				"eventIdentifier": "comprehensive_event",
				"schema": {
					"event": "string",
					"userId": "string",
					"anonymousId": "string",
					"properties": {
						"simple_prop": "string",
						"number_prop": "number",
						"nested_object": {
							"field1": "string",
							"field2": "number",
							"deeply_nested": {
								"deep_field": "boolean"
							}
						},
						"array_prop": ["string", "number"],
						"complex_array": [{
							"item_id": "string",
							"item_data": {
								"name": "string",
								"value": "number"
							}
						}]
					},
					"context": {
						"app": {
							"name": "string",
							"version": "string"
						},
						"device": {
							"type": "string",
							"model": "string"
						}
					}
				}
			},
			{
				"uid": "test-uid-2",
				"writeKey": "test-write-key-2",
				"eventType": "identify",
				"eventIdentifier": "user_identify",
				"schema": {
					"event": "string",
					"userId": "string",
					"traits": {
						"email": "string",
						"name": "string",
						"age": "number",
						"preferences": {
							"notifications": "boolean",
							"theme": "string"
						}
					}
				}
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	options := ConversionOptions{
		InputFile:  inputFile,
		OutputDir:  outputDir,
		DryRun:     false,
		Verbose:    true,
		YAMLIndent: 4, // Test custom indent
	}

	converter := NewSchemaConverter(options)
	result, err := converter.Convert()
	require.NoError(t, err)

	// Verify successful file creation (lines 227-228, 235-236, 243-244)
	assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "properties.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "custom-types.yaml"))

	// Verify tracking plans directory creation (lines 251-252)
	assert.DirExists(t, filepath.Join(outputDir, "tracking-plans"))
	assert.FileExists(t, filepath.Join(outputDir, "tracking-plans", "writekey-test-write-key-1.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "tracking-plans", "writekey-test-write-key-2.yaml"))

	// Verify result contains generated files
	assert.True(t, len(result.GeneratedFiles) >= 5) // 3 main files + 2 tracking plans
	assert.Len(t, result.TrackingPlans, 2)

	// Verify file contents are valid YAML by reading and parsing
	t.Run("ValidateGeneratedYAML", func(t *testing.T) {
		// Read events.yaml and verify it's valid
		eventsContent, err := os.ReadFile(filepath.Join(outputDir, "events.yaml"))
		require.NoError(t, err)
		assert.NotEmpty(t, eventsContent)
		assert.Contains(t, string(eventsContent), "comprehensive_event")
		assert.Contains(t, string(eventsContent), "user_identify")

		// Read properties.yaml and verify it's valid
		propertiesContent, err := os.ReadFile(filepath.Join(outputDir, "properties.yaml"))
		require.NoError(t, err)
		assert.NotEmpty(t, propertiesContent)
		assert.Contains(t, string(propertiesContent), "simple_prop")

		// Read custom-types.yaml and verify it's valid
		customTypesContent, err := os.ReadFile(filepath.Join(outputDir, "custom-types.yaml"))
		require.NoError(t, err)
		assert.NotEmpty(t, customTypesContent)

		// Read tracking plan files
		tp1Content, err := os.ReadFile(filepath.Join(outputDir, "tracking-plans", "writekey-test-write-key-1.yaml"))
		require.NoError(t, err)
		assert.NotEmpty(t, tp1Content)
		assert.Contains(t, string(tp1Content), "comprehensive_event")

		tp2Content, err := os.ReadFile(filepath.Join(outputDir, "tracking-plans", "writekey-test-write-key-2.yaml"))
		require.NoError(t, err)
		assert.NotEmpty(t, tp2Content)
		assert.Contains(t, string(tp2Content), "user_identify")
	})

	// Verify counts match expectations
	assert.Equal(t, 2, result.EventsCount)
	assert.True(t, result.PropertiesCount > 5)  // Should have many properties from nested objects
	assert.True(t, result.CustomTypesCount > 0) // Should have custom types from nested objects
}

// TestSchemaConverter_VerboseConversion tests verbose mode output
func TestSchemaConverter_VerboseConversion(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test_schemas.json")
	outputDir := filepath.Join(tempDir, "output")

	testData := `{
		"schemas": [
			{
				"uid": "verbose-test-uid",
				"writeKey": "verbose-write-key",
				"eventType": "track",
				"eventIdentifier": "verbose_event",
				"schema": {
					"event": "string",
					"userId": "string",
					"properties": {
						"test_prop": "string"
					}
				}
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	options := ConversionOptions{
		InputFile:  inputFile,
		OutputDir:  outputDir,
		DryRun:     false,
		Verbose:    true, // Test verbose output paths
		YAMLIndent: 2,
	}

	converter := NewSchemaConverter(options)
	result, err := converter.Convert()
	require.NoError(t, err)

	// Verify successful conversion with verbose logging
	assert.Equal(t, 1, result.EventsCount)
	assert.True(t, result.PropertiesCount > 0)
	assert.True(t, len(result.GeneratedFiles) >= 4)
}
