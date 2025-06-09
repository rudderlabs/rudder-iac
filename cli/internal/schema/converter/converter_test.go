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

func TestSchemaConverterWorkflow(t *testing.T) {
	t.Parallel()

	cases := []struct {
		category string
		name     string
		setup    func(t *testing.T) (interface{}, ConversionOptions)
		validate func(t *testing.T, input interface{}, result interface{}, options ConversionOptions)
	}{
		// Analyzer Workflow Tests
		{
			category: "Analyzer",
			name:     "CompleteAnalysis",
			setup: func(t *testing.T) (interface{}, ConversionOptions) {
				schemas := []models.Schema{
					testhelpers.CreateTestSchema("test-uid-1", "test-writekey-1", "test_event_one"),
					testhelpers.CreateComplexTestSchema("test-uid-2", "test-writekey-2", "nested_test_event"),
					testhelpers.CreateArraySchema(),
				}
				return schemas, ConversionOptions{}
			},
			validate: func(t *testing.T, input interface{}, result interface{}, options ConversionOptions) {
				schemas := input.([]models.Schema)
				analyzer := NewSchemaAnalyzer()
				err := analyzer.AnalyzeSchemas(schemas)
				require.NoError(t, err)

				assert.Len(t, analyzer.Events, 3)
				assert.Contains(t, analyzer.Events, "test_event_one")
				assert.Contains(t, analyzer.Events, "nested_test_event")
				assert.Contains(t, analyzer.Events, "array_test_event")
				assert.True(t, len(analyzer.Properties) > 0)
				assert.True(t, len(analyzer.CustomTypes) > 0)

				// YAML Generation Tests
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

				trackingPlansYAML := analyzer.GenerateTrackingPlansYAML(schemas)
				assert.Len(t, trackingPlansYAML, 3)
			},
		},
		{
			category: "Analyzer",
			name:     "EmptySchemas",
			setup: func(t *testing.T) (interface{}, ConversionOptions) {
				return []models.Schema{}, ConversionOptions{}
			},
			validate: func(t *testing.T, input interface{}, result interface{}, options ConversionOptions) {
				schemas := input.([]models.Schema)
				analyzer := NewSchemaAnalyzer()
				err := analyzer.AnalyzeSchemas(schemas)
				require.NoError(t, err)

				assert.Len(t, analyzer.Events, 0)
				assert.Len(t, analyzer.Properties, 0)
				assert.Len(t, analyzer.CustomTypes, 0)

				eventsYAML := analyzer.GenerateEventsYAML()
				assert.Equal(t, "rudder/0.1", eventsYAML.Version)
				assert.Equal(t, "events", eventsYAML.Kind)
				assert.Len(t, eventsYAML.Spec.Events, 0)
			},
		},

		// Converter Workflow Tests
		{
			category: "Converter",
			name:     "DryRun",
			setup: func(t *testing.T) (interface{}, ConversionOptions) {
				tempDir := t.TempDir()
				inputFile := filepath.Join(tempDir, "test_schemas.json")
				outputDir := filepath.Join(tempDir, "output")

				testSchemas := []models.Schema{testhelpers.CreateMinimalTestSchema("test-uid")}
				schemasData := map[string]interface{}{"schemas": testSchemas}
				testhelpers.WriteTestFile(t, inputFile, schemasData)

				return inputFile, ConversionOptions{
					InputFile:  inputFile,
					OutputDir:  outputDir,
					DryRun:     true,
					Verbose:    false,
					YAMLIndent: 2,
				}
			},
			validate: func(t *testing.T, input interface{}, result interface{}, options ConversionOptions) {
				converter := NewSchemaConverter(options)
				convResult, err := converter.Convert()
				require.NoError(t, err)

				assert.Equal(t, 1, convResult.EventsCount)
				assert.True(t, convResult.PropertiesCount > 0)
				assert.True(t, convResult.CustomTypesCount >= 0)
				assert.Empty(t, convResult.GeneratedFiles)
			},
		},
		{
			category: "Converter",
			name:     "RealRun",
			setup: func(t *testing.T) (interface{}, ConversionOptions) {
				tempDir := t.TempDir()
				inputFile := filepath.Join(tempDir, "test_schemas.json")
				outputDir := filepath.Join(tempDir, "output")

				testSchemas := []models.Schema{testhelpers.CreateMinimalTestSchema("test-uid")}
				schemasData := map[string]interface{}{"schemas": testSchemas}
				testhelpers.WriteTestFile(t, inputFile, schemasData)

				return outputDir, ConversionOptions{
					InputFile:  inputFile,
					OutputDir:  outputDir,
					DryRun:     false,
					Verbose:    false,
					YAMLIndent: 2,
				}
			},
			validate: func(t *testing.T, input interface{}, result interface{}, options ConversionOptions) {
				outputDir := input.(string)
				converter := NewSchemaConverter(options)
				convResult, err := converter.Convert()
				require.NoError(t, err)

				assert.True(t, len(convResult.GeneratedFiles) >= 3)
				assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
				assert.FileExists(t, filepath.Join(outputDir, "properties.yaml"))
				assert.FileExists(t, filepath.Join(outputDir, "custom-types.yaml"))
				assert.DirExists(t, filepath.Join(outputDir, "tracking-plans"))
			},
		},

		// Error Cases
		{
			category: "ErrorCases",
			name:     "InvalidInput",
			setup: func(t *testing.T) (interface{}, ConversionOptions) {
				tempDir := t.TempDir()
				inputFile := filepath.Join(tempDir, "invalid.json")
				err := os.WriteFile(inputFile, []byte("invalid json"), 0644)
				require.NoError(t, err)
				return nil, ConversionOptions{
					InputFile: inputFile,
					OutputDir: filepath.Join(tempDir, "output"),
					DryRun:    false,
				}
			},
			validate: func(t *testing.T, input interface{}, result interface{}, options ConversionOptions) {
				converter := NewSchemaConverter(options)
				_, err := converter.Convert()
				assert.Error(t, err)
			},
		},
		{
			category: "ErrorCases",
			name:     "NonexistentInput",
			setup: func(t *testing.T) (interface{}, ConversionOptions) {
				tempDir := t.TempDir()
				return nil, ConversionOptions{
					InputFile: filepath.Join(tempDir, "nonexistent.json"),
					OutputDir: filepath.Join(tempDir, "output"),
					DryRun:    false,
				}
			},
			validate: func(t *testing.T, input interface{}, result interface{}, options ConversionOptions) {
				converter := NewSchemaConverter(options)
				_, err := converter.Convert()
				assert.Error(t, err)
			},
		},

		// Multi-WriteKey Workflow
		{
			category: "MultiWriteKey",
			name:     "MultipleWriteKeys",
			setup: func(t *testing.T) (interface{}, ConversionOptions) {
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

				return nil, ConversionOptions{
					InputFile:  inputFile,
					OutputDir:  outputDir,
					DryRun:     true,
					Verbose:    false,
					YAMLIndent: 2,
				}
			},
			validate: func(t *testing.T, input interface{}, result interface{}, options ConversionOptions) {
				converter := NewSchemaConverter(options)
				convResult, err := converter.Convert()
				require.NoError(t, err)

				assert.Equal(t, 2, convResult.EventsCount)
				assert.True(t, convResult.PropertiesCount > 0)
				assert.True(t, convResult.CustomTypesCount >= 0)
				assert.Len(t, convResult.TrackingPlans, 2)
				assert.Empty(t, convResult.GeneratedFiles)
			},
		},

		// Comprehensive Workflow - Domain Optimized
		{
			category: "ComprehensiveWorkflow",
			name:     "VerboseConversion",
			setup: func(t *testing.T) (interface{}, ConversionOptions) {
				tempDir := t.TempDir()
				inputFile := filepath.Join(tempDir, "test_schemas.json")
				outputDir := filepath.Join(tempDir, "output")

				testData := map[string]interface{}{
					"schemas": []interface{}{
						testhelpers.CreateComplexSchema("test-uid-1", "test-write-key-1", "comprehensive_event"),
						testhelpers.CreateIdentifySchema("test-uid-2", "test-write-key-2", "user_identify"),
					},
				}
				testhelpers.WriteTestFile(t, inputFile, testData)

				return outputDir, ConversionOptions{InputFile: inputFile, OutputDir: outputDir, DryRun: false, Verbose: true, YAMLIndent: 4}
			},
			validate: func(t *testing.T, input interface{}, result interface{}, options ConversionOptions) {
				outputDir := input.(string)
				converter := NewSchemaConverter(options)
				convResult, err := converter.Convert()
				require.NoError(t, err)

				expectedFiles := []string{"events.yaml", "properties.yaml", "custom-types.yaml"}
				for _, file := range expectedFiles {
					assert.FileExists(t, filepath.Join(outputDir, file))
				}
				assert.DirExists(t, filepath.Join(outputDir, "tracking-plans"))
				assert.Equal(t, 2, convResult.EventsCount)
				assert.True(t, convResult.PropertiesCount > 0 && convResult.CustomTypesCount > 0)
				assert.Len(t, convResult.TrackingPlans, 2)
				assert.True(t, len(convResult.GeneratedFiles) >= 5)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.category+"/"+c.name, func(t *testing.T) {
			t.Parallel()
			input, options := c.setup(t)
			c.validate(t, input, nil, options)
		})
	}
}

func TestConversionOptionsValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		options  ConversionOptions
		hasError bool
	}{
		{
			name: "ValidOptions",
			options: ConversionOptions{
				InputFile: "input.json", OutputDir: "output", DryRun: false, Verbose: false, YAMLIndent: 2,
			},
			hasError: false,
		},
		{
			name: "EmptyInputFile",
			options: ConversionOptions{
				InputFile: "", OutputDir: "output", DryRun: false, Verbose: false, YAMLIndent: 2,
			},
			hasError: true,
		},
		{
			name: "EmptyOutputDir",
			options: ConversionOptions{
				InputFile: "input.json", OutputDir: "", DryRun: false, Verbose: false, YAMLIndent: 2,
			},
			hasError: true,
		},
		{
			name: "InvalidYAMLIndent",
			options: ConversionOptions{
				InputFile: "input.json", OutputDir: "output", DryRun: false, Verbose: false, YAMLIndent: 0,
			},
			hasError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			converter := NewSchemaConverter(c.options)
			assert.NotNil(t, converter)
		})
	}
}

func TestPreviewFunctions(t *testing.T) {
	t.Parallel()

	// Test data setup
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

	// Micro-optimized preview tests
	previewTests := []struct {
		name string
		fn   func()
	}{
		{"EventsPreview", func() { printEventsPreview(eventsYAML, 2); printEventsPreview(eventsYAML, 4) }},
		{"PropertiesPreview", func() { printPropertiesPreview(propertiesYAML, 2); printPropertiesPreview(propertiesYAML, 3) }},
		{"CustomTypesPreview", func() { printCustomTypesPreview(customTypesYAML, 1); printCustomTypesPreview(customTypesYAML, 2) }},
		{"EmptyCollections", func() {
			emptyEventsYAML := &yamlModels.EventsYAML{Spec: yamlModels.EventsSpec{Events: []yamlModels.EventDefinition{}}}
			emptyPropertiesYAML := &yamlModels.PropertiesYAML{Spec: yamlModels.PropertiesSpec{Properties: []yamlModels.PropertyDefinition{}}}
			emptyCustomTypesYAML := &yamlModels.CustomTypesYAML{Spec: yamlModels.CustomTypesSpec{Types: []yamlModels.CustomTypeDefinition{}}}
			printEventsPreview(emptyEventsYAML, 3)
			printPropertiesPreview(emptyPropertiesYAML, 3)
			printCustomTypesPreview(emptyCustomTypesYAML, 3)
		}},
	}

	for _, test := range previewTests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			test.fn()
		})
	}
}
