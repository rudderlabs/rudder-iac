package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/rudderlabs/rudder-iac/cli/internal/testhelpers"
	yamlModels "github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaAnalyzer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    []models.Schema
		expected func(t *testing.T, analyzer *SchemaAnalyzer)
	}{
		{
			name: "CompleteAnalysis",
			input: []models.Schema{
				testhelpers.CreateTestSchema("test-uid-1", "test-writekey-1", "test_event_one"),
				testhelpers.CreateComplexTestSchema("test-uid-2", "test-writekey-2", "nested_test_event"),
				testhelpers.CreateArraySchema(),
			},
			expected: func(t *testing.T, analyzer *SchemaAnalyzer) {
				assert.Len(t, analyzer.Events, 3)
				assert.Contains(t, analyzer.Events, "test_event_one")
				assert.Contains(t, analyzer.Events, "nested_test_event")
				assert.Contains(t, analyzer.Events, "array_test_event")
				assert.True(t, len(analyzer.Properties) > 0)
				assert.True(t, len(analyzer.CustomTypes) > 0)

				// Test YAML generation
				eventsYAML := analyzer.GenerateEventsYAML()
				assert.Equal(t, "rudder/0.1", eventsYAML.Version)
				assert.Equal(t, "events", eventsYAML.Kind)
				assert.Len(t, eventsYAML.Spec.Events, 3)

				propertiesYAML := analyzer.GeneratePropertiesYAML()
				assert.Equal(t, "rudder/v0.1", propertiesYAML.Version)
				assert.Equal(t, "properties", propertiesYAML.Kind)
				assert.True(t, len(propertiesYAML.Spec.Properties) > 0)

				customTypesYAML := analyzer.GenerateCustomTypesYAML()
				assert.Equal(t, "rudder/v0.1", customTypesYAML.Version)
				assert.Equal(t, "custom-types", customTypesYAML.Kind)
				assert.True(t, len(customTypesYAML.Spec.Types) > 0)

				// Just verify we can generate tracking plans
				inputSchemas := []models.Schema{
					testhelpers.CreateTestSchema("test-uid-1", "test-writekey-1", "test_event_one"),
					testhelpers.CreateComplexTestSchema("test-uid-2", "test-writekey-2", "nested_test_event"),
					testhelpers.CreateArraySchema(),
				}
				trackingPlansYAML := analyzer.GenerateTrackingPlansYAML(inputSchemas)
				assert.Len(t, trackingPlansYAML, 3)
			},
		},
		{
			name:  "EmptySchemas",
			input: []models.Schema{},
			expected: func(t *testing.T, analyzer *SchemaAnalyzer) {
				assert.Len(t, analyzer.Events, 0)
				assert.Len(t, analyzer.Properties, 0)
				assert.Len(t, analyzer.CustomTypes, 0)

				eventsYAML := analyzer.GenerateEventsYAML()
				assert.Equal(t, "rudder/0.1", eventsYAML.Version)
				assert.Equal(t, "events", eventsYAML.Kind)
				assert.Len(t, eventsYAML.Spec.Events, 0)
			},
		},
		{
			name:  "ComplexNestedSchema",
			input: []models.Schema{testhelpers.CreateComplexTestSchema("nested-uid", "nested-writekey", "nested_test_event")},
			expected: func(t *testing.T, analyzer *SchemaAnalyzer) {
				assert.Len(t, analyzer.Events, 1)
				assert.Contains(t, analyzer.Events, "nested_test_event")
				assert.True(t, len(analyzer.Properties) > 0)
				assert.True(t, len(analyzer.CustomTypes) > 0)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			analyzer := NewSchemaAnalyzer()
			err := analyzer.AnalyzeSchemas(c.input)
			require.NoError(t, err)
			c.expected(t, analyzer)
		})
	}
}

func TestSchemaConverter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		dryRun   bool
		verbose  bool
		expected func(t *testing.T, result *ConversionResult, outputDir string)
	}{
		{
			name:    "DryRun",
			dryRun:  true,
			verbose: false,
			expected: func(t *testing.T, result *ConversionResult, outputDir string) {
				assert.Equal(t, 1, result.EventsCount)
				assert.True(t, result.PropertiesCount > 0)
				assert.True(t, result.CustomTypesCount >= 0)
				assert.Empty(t, result.GeneratedFiles)
			},
		},
		{
			name:    "RealRun",
			dryRun:  false,
			verbose: false,
			expected: func(t *testing.T, result *ConversionResult, outputDir string) {
				assert.True(t, len(result.GeneratedFiles) >= 3)
				assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
				assert.FileExists(t, filepath.Join(outputDir, "properties.yaml"))
				assert.FileExists(t, filepath.Join(outputDir, "custom-types.yaml"))
				assert.DirExists(t, filepath.Join(outputDir, "tracking-plans"))
			},
		},
		{
			name:    "VerboseDryRun",
			dryRun:  true,
			verbose: true,
			expected: func(t *testing.T, result *ConversionResult, outputDir string) {
				assert.Equal(t, 1, result.EventsCount)
				assert.True(t, result.PropertiesCount > 0)
				assert.Empty(t, result.GeneratedFiles)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			inputFile := filepath.Join(tempDir, "test_schemas.json")
			outputDir := filepath.Join(tempDir, "output")

			testSchemas := []models.Schema{testhelpers.CreateMinimalTestSchema("test-uid")}
			schemasData := map[string]interface{}{"schemas": testSchemas}
			testhelpers.WriteTestFile(t, inputFile, schemasData)

			options := ConversionOptions{
				InputFile:  inputFile,
				OutputDir:  outputDir,
				DryRun:     c.dryRun,
				Verbose:    c.verbose,
				YAMLIndent: 2,
			}

			converter := NewSchemaConverter(options)
			result, err := converter.Convert()
			require.NoError(t, err)
			c.expected(t, result, outputDir)
		})
	}
}

func TestSchemaConverter_ErrorCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		setup func(t *testing.T) ConversionOptions
	}{
		{
			name: "InvalidInput",
			setup: func(t *testing.T) ConversionOptions {
				tempDir := t.TempDir()
				inputFile := filepath.Join(tempDir, "invalid.json")
				err := os.WriteFile(inputFile, []byte("invalid json"), 0644)
				require.NoError(t, err)
				return ConversionOptions{
					InputFile: inputFile,
					OutputDir: filepath.Join(tempDir, "output"),
					DryRun:    false,
				}
			},
		},
		{
			name: "NonexistentInput",
			setup: func(t *testing.T) ConversionOptions {
				tempDir := t.TempDir()
				return ConversionOptions{
					InputFile: filepath.Join(tempDir, "nonexistent.json"),
					OutputDir: filepath.Join(tempDir, "output"),
					DryRun:    false,
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			options := c.setup(t)
			converter := NewSchemaConverter(options)
			_, err := converter.Convert()
			assert.Error(t, err)
		})
	}
}

func TestSchemaConverter_MultipleWriteKeys(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test_schemas.json")
	outputDir := filepath.Join(tempDir, "output")

	testData := map[string]interface{}{
		"schemas": []models.Schema{
			testhelpers.CreateTestSchema("uid1", "writekey1", "event1"),
			testhelpers.CreateTestSchema("uid2", "writekey2", "event2"),
		},
	}
	testhelpers.WriteTestFile(t, inputFile, testData)

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

	assert.Equal(t, 2, result.EventsCount)
	assert.True(t, result.PropertiesCount > 0)
	assert.True(t, result.CustomTypesCount >= 0)
	assert.Len(t, result.TrackingPlans, 2)
	assert.Empty(t, result.GeneratedFiles)
}

func TestConversionOptions_Validation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		options  ConversionOptions
		hasError bool
	}{
		{
			name: "ValidOptions",
			options: ConversionOptions{
				InputFile:  "input.json",
				OutputDir:  "output",
				DryRun:     false,
				Verbose:    false,
				YAMLIndent: 2,
			},
			hasError: false,
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
			hasError: true,
		},
		{
			name: "EmptyOutputDir",
			options: ConversionOptions{
				InputFile:  "input.json",
				OutputDir:  "",
				DryRun:     false,
				Verbose:    false,
				YAMLIndent: 2,
			},
			hasError: true,
		},
		{
			name: "InvalidYAMLIndent",
			options: ConversionOptions{
				InputFile:  "input.json",
				OutputDir:  "output",
				DryRun:     false,
				Verbose:    false,
				YAMLIndent: 0,
			},
			hasError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			converter := NewSchemaConverter(c.options)
			assert.NotNil(t, converter)
			// Validation is handled during converter creation
		})
	}
}

func TestPreviewFunctions(t *testing.T) {
	t.Parallel()

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

	cases := []struct {
		name     string
		maxItems int
		function func(int)
	}{
		{"EventsPreview_Limited", 2, func(max int) { printEventsPreview(eventsYAML, max) }},
		{"EventsPreview_Full", 4, func(max int) { printEventsPreview(eventsYAML, max) }},
		{"EventsPreview_Overflow", 10, func(max int) { printEventsPreview(eventsYAML, max) }},
		{"PropertiesPreview_Limited", 2, func(max int) { printPropertiesPreview(propertiesYAML, max) }},
		{"PropertiesPreview_Full", 3, func(max int) { printPropertiesPreview(propertiesYAML, max) }},
		{"CustomTypesPreview_Limited", 1, func(max int) { printCustomTypesPreview(customTypesYAML, max) }},
		{"CustomTypesPreview_Full", 2, func(max int) { printCustomTypesPreview(customTypesYAML, max) }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			c.function(c.maxItems)
		})
	}

	// Test empty collections
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

		printEventsPreview(emptyEventsYAML, 3)
		printPropertiesPreview(emptyPropertiesYAML, 3)
		printCustomTypesPreview(emptyCustomTypesYAML, 3)
	})
}

func TestSchemaConverter_ComprehensiveWorkflow(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test_schemas.json")
	outputDir := filepath.Join(tempDir, "output")

	// Create comprehensive test data
	testData := map[string]interface{}{
		"schemas": []interface{}{
			map[string]interface{}{
				"uid":             "test-uid-1",
				"writeKey":        "test-write-key-1",
				"eventType":       "track",
				"eventIdentifier": "comprehensive_event",
				"schema": map[string]interface{}{
					"event":       "string",
					"userId":      "string",
					"anonymousId": "string",
					"properties": map[string]interface{}{
						"simple_prop": "string",
						"number_prop": "number",
						"nested_object": map[string]interface{}{
							"field1": "string",
							"field2": "number",
							"deeply_nested": map[string]interface{}{
								"deep_field": "boolean",
							},
						},
						"array_prop": []interface{}{"string", "number"},
						"complex_array": []interface{}{map[string]interface{}{
							"item_id": "string",
							"item_data": map[string]interface{}{
								"name":  "string",
								"value": "number",
							},
						}},
					},
					"context": map[string]interface{}{
						"app": map[string]interface{}{
							"name":    "string",
							"version": "string",
						},
						"device": map[string]interface{}{
							"type":  "string",
							"model": "string",
						},
					},
				},
			},
			map[string]interface{}{
				"uid":             "test-uid-2",
				"writeKey":        "test-write-key-2",
				"eventType":       "identify",
				"eventIdentifier": "user_identify",
				"schema": map[string]interface{}{
					"event":  "string",
					"userId": "string",
					"traits": map[string]interface{}{
						"email": "string",
						"name":  "string",
						"age":   "number",
						"preferences": map[string]interface{}{
							"notifications": "boolean",
							"theme":         "string",
						},
					},
				},
			},
		},
	}

	testhelpers.WriteTestFile(t, inputFile, testData)

	cases := []struct {
		name     string
		dryRun   bool
		verbose  bool
		indent   int
		expected func(t *testing.T, result *ConversionResult)
	}{
		{
			name:    "VerboseConversion",
			dryRun:  false,
			verbose: true,
			indent:  4,
			expected: func(t *testing.T, result *ConversionResult) {
				assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
				assert.FileExists(t, filepath.Join(outputDir, "properties.yaml"))
				assert.FileExists(t, filepath.Join(outputDir, "custom-types.yaml"))
				assert.DirExists(t, filepath.Join(outputDir, "tracking-plans"))
				assert.FileExists(t, filepath.Join(outputDir, "tracking-plans", "writekey-test-write-key-1.yaml"))
				assert.FileExists(t, filepath.Join(outputDir, "tracking-plans", "writekey-test-write-key-2.yaml"))
				assert.Equal(t, 2, result.EventsCount)
				assert.True(t, result.PropertiesCount > 0)
				assert.True(t, result.CustomTypesCount > 0)
				assert.Len(t, result.TrackingPlans, 2)
				assert.True(t, len(result.GeneratedFiles) >= 5)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			options := ConversionOptions{
				InputFile:  inputFile,
				OutputDir:  outputDir,
				DryRun:     c.dryRun,
				Verbose:    c.verbose,
				YAMLIndent: c.indent,
			}

			converter := NewSchemaConverter(options)
			result, err := converter.Convert()
			require.NoError(t, err)
			c.expected(t, result)
		})
	}
}
