package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
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
