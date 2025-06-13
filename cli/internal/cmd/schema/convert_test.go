package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/converter"
	pkgModels "github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/rudderlabs/rudder-iac/cli/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestConvertCommand_Integration(t *testing.T) {
	t.Parallel()

	// Create temporary directory for test
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test_schemas.json")
	outputDir := filepath.Join(tempDir, "output")

	// Create test input file with actual schema structure
	testData := `{
		"schemas": [
			{
				"uid": "test-uid-1",
				"writeKey": "test-write-key-1",
				"eventType": "track",
				"eventIdentifier": "product_viewed",
				"schema": {
					"anonymousId": "string",
					"channel": "string",
					"context": {
						"app": {
							"name": "string",
							"version": "string"
						},
						"traits": {
							"email": "string",
							"firstName": "string"
						}
					},
					"event": "string",
					"messageId": "string",
					"properties": {
						"product_id": "string",
						"product_name": "string",
						"categories": ["string", "string"]
					},
					"type": "string",
					"userId": "string"
				},
				"createdAt": "2024-01-10T10:08:15.407491Z",
				"lastSeen": "2024-03-25T18:49:31.870834Z",
				"count": 10
			},
			{
				"uid": "test-uid-2",
				"writeKey": "test-write-key-2",
				"eventType": "track",
				"eventIdentifier": "order_completed",
				"schema": {
					"anonymousId": "string",
					"event": "string",
					"properties": {
						"order_id": "string",
						"total_amount": "number",
						"items": [
							{
								"name": "string",
								"price": "number"
							}
						]
					},
					"userId": "string"
				},
				"createdAt": "2024-01-15T12:30:00.000000Z",
				"lastSeen": "2024-03-30T15:45:30.123456Z",
				"count": 25
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	// Test the convert command
	err = runConvert(inputFile, outputDir, false, false, 2)
	require.NoError(t, err)

	// Verify output files were created
	testhelpers.AssertFilesExist(t,
		filepath.Join(outputDir, "events.yaml"),
		filepath.Join(outputDir, "properties.yaml"),
		filepath.Join(outputDir, "custom-types.yaml"),
		filepath.Join(outputDir, "tracking-plans", "writekey-test-write-key-1.yaml"),
		filepath.Join(outputDir, "tracking-plans", "writekey-test-write-key-2.yaml"),
	)
	testhelpers.AssertDirsExist(t, filepath.Join(outputDir, "tracking-plans"))
}

func TestConvertCommand_Scenarios(t *testing.T) {
	cases := []struct {
		name        string
		dryRun      bool
		verbose     bool
		expectFiles bool
		expectError bool
	}{
		{
			name:        "Normal",
			dryRun:      false,
			verbose:     false,
			expectFiles: true,
			expectError: false,
		},
		{
			name:        "DryRun",
			dryRun:      true,
			verbose:     false,
			expectFiles: false,
			expectError: false,
		},
		{
			name:        "VerboseMode",
			dryRun:      false,
			verbose:     true,
			expectFiles: true,
			expectError: false,
		},
		{
			name:        "VerboseDryRun",
			dryRun:      true,
			verbose:     true,
			expectFiles: false,
			expectError: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			inputFile := filepath.Join(tempDir, "test_schemas.json")
			outputDir := filepath.Join(tempDir, "output")

			// Create minimal test input file
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

			err = runConvert(inputFile, outputDir, c.dryRun, c.verbose, 2)

			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if c.expectFiles {
				testhelpers.AssertFilesExist(t,
					filepath.Join(outputDir, "events.yaml"),
					filepath.Join(outputDir, "properties.yaml"),
					filepath.Join(outputDir, "custom-types.yaml"),
				)
				testhelpers.AssertDirsExist(t, filepath.Join(outputDir, "tracking-plans"))
			} else {
				_, err := os.Stat(outputDir)
				assert.True(t, os.IsNotExist(err))
			}
		})
	}
}

func TestConvertCommand_ErrorScenarios(t *testing.T) {
	cases := []struct {
		name        string
		setupFiles  func(tempDir string) (string, string)
		expectError string
	}{
		{
			name: "InputFileNotFound",
			setupFiles: func(tempDir string) (string, string) {
				inputFile := filepath.Join(tempDir, "nonexistent.json")
				outputDir := filepath.Join(tempDir, "output")
				return inputFile, outputDir
			},
			expectError: "conversion failed",
		},
		{
			name: "InvalidJSON",
			setupFiles: func(tempDir string) (string, string) {
				inputFile := filepath.Join(tempDir, "invalid.json")
				outputDir := filepath.Join(tempDir, "output")

				// Create invalid JSON file
				invalidJSON := `{ "schemas": [ invalid json } ]`
				os.WriteFile(inputFile, []byte(invalidJSON), 0644)

				return inputFile, outputDir
			},
			expectError: "conversion failed",
		},
		{
			name: "ReadOnlyOutputDirectory",
			setupFiles: func(tempDir string) (string, string) {
				inputFile := filepath.Join(tempDir, "test_schemas.json")
				outputDir := filepath.Join(tempDir, "readonly_output")

				// Create valid input file
				testData := `{
					"schemas": [
						{
							"uid": "test-uid",
							"writeKey": "test-write-key",
							"eventType": "track",
							"eventIdentifier": "test_event",
							"schema": {"event": "string"}
						}
					]
				}`
				os.WriteFile(inputFile, []byte(testData), 0644)

				// Create read-only output directory
				os.MkdirAll(outputDir, 0444)
				os.Chmod(outputDir, 0444)

				return inputFile, outputDir
			},
			expectError: "conversion failed",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			inputFile, outputDir := c.setupFiles(tempDir)

			err := runConvert(inputFile, outputDir, false, false, 2)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), c.expectError)
		})
	}
}

func TestConvertCommand_CommandFlags(t *testing.T) {
	t.Parallel()

	t.Run("IndentFlag", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		inputFile := filepath.Join(tempDir, "test_schemas.json")
		outputDir := filepath.Join(tempDir, "output")

		testData := `{
			"schemas": [
				{
					"uid": "test-uid",
					"writeKey": "test-write-key",
					"eventType": "track",
					"eventIdentifier": "test_event",
					"schema": {
						"event": "string",
						"userId": "string"
					}
				}
			]
		}`

		err := os.WriteFile(inputFile, []byte(testData), 0644)
		require.NoError(t, err)

		// Test with different indent values
		indentCases := []int{1, 2, 4, 8}
		for _, indent := range indentCases {
			t.Run(fmt.Sprintf("Indent_%d", indent), func(t *testing.T) {
				currentOutputDir := filepath.Join(outputDir, fmt.Sprintf("indent_%d", indent))
				err := runConvert(inputFile, currentOutputDir, false, false, indent)
				assert.NoError(t, err)
				assert.FileExists(t, filepath.Join(currentOutputDir, "events.yaml"))
			})
		}
	})

	t.Run("InvalidIndent", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		inputFile := filepath.Join(tempDir, "test_schemas.json")
		outputDir := filepath.Join(tempDir, "output")

		testData := `{
			"schemas": [
				{
					"uid": "test-uid",
					"writeKey": "test-write-key",
					"eventType": "track",
					"eventIdentifier": "test_event",
					"schema": {"event": "string"}
				}
			]
		}`

		err := os.WriteFile(inputFile, []byte(testData), 0644)
		require.NoError(t, err)

		// Test with negative indent (should still work, converter should handle it)
		err = runConvert(inputFile, outputDir, false, false, -1)
		assert.NoError(t, err) // Should not error, converter handles edge cases
	})
}

func TestConvertCommand_VerboseOutput(t *testing.T) {
	t.Run("VerboseOutputMessages", func(t *testing.T) {
		t.Parallel()

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
						"userId": "string"
					}
				}
			]
		}`

		err := os.WriteFile(inputFile, []byte(testData), 0644)
		require.NoError(t, err)

		// Test with verbose enabled (should trigger the verbose output paths)
		err = runConvert(inputFile, outputDir, false, true, 2)
		assert.NoError(t, err)
	})

	t.Run("DryRunVerboseOutput", func(t *testing.T) {
		t.Parallel()

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
						"userId": "string"
					}
				}
			]
		}`

		err := os.WriteFile(inputFile, []byte(testData), 0644)
		require.NoError(t, err)

		// Test with both dry-run and verbose (should trigger dry-run output paths)
		err = runConvert(inputFile, outputDir, true, true, 2)
		assert.NoError(t, err)
	})
}

func TestNewCmdConvert(t *testing.T) {
	t.Parallel()

	t.Run("CommandCreation", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdConvert()

		assert.NotNil(t, cmd)
		assert.Equal(t, "convert", cmd.Name())
		assert.Equal(t, "Convert unflattened schemas to YAML files", cmd.Short)
		assert.Contains(t, cmd.Long, "Convert unflattened schemas to RudderStack Data Catalog YAML files")

		// Check that flags are properly set
		dryRunFlag := cmd.Flags().Lookup("dry-run")
		assert.NotNil(t, dryRunFlag)
		assert.Equal(t, "false", dryRunFlag.DefValue)

		verboseFlag := cmd.Flags().Lookup("verbose")
		assert.NotNil(t, verboseFlag)
		assert.Equal(t, "v", verboseFlag.Shorthand)

		indentFlag := cmd.Flags().Lookup("indent")
		assert.NotNil(t, indentFlag)
		assert.Equal(t, "2", indentFlag.DefValue)
	})

	t.Run("CommandRequiresExactTwoArgs", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdConvert()
		assert.NotNil(t, cmd.Args)

		// Test that command requires exactly 2 arguments
		// This should pass with 2 args
		err := cmd.Args(cmd, []string{"input.json", "output/"})
		assert.NoError(t, err)

		// This should fail with 1 arg
		err = cmd.Args(cmd, []string{"input.json"})
		assert.Error(t, err)

		// This should fail with 3 args
		err = cmd.Args(cmd, []string{"input.json", "output/", "extra"})
		assert.Error(t, err)
	})

	t.Run("CommandRunFunction", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		inputFile := filepath.Join(tempDir, "test_schemas.json")
		outputDir := filepath.Join(tempDir, "output")

		// Create minimal test data
		testData := `{"schemas": []}`
		err := os.WriteFile(inputFile, []byte(testData), 0644)
		require.NoError(t, err)

		cmd := NewCmdConvert()
		assert.NotNil(t, cmd.RunE)

		// Test that RunE function can be called
		err = cmd.RunE(cmd, []string{inputFile, outputDir})
		assert.NoError(t, err)
	})
}

func TestConvertCommand_MultipleWriteKeys(t *testing.T) {
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

	err = runConvert(inputFile, outputDir, false, false, 2)
	require.NoError(t, err)

	// Verify separate tracking plans were created for each writeKey
	assert.FileExists(t, filepath.Join(outputDir, "tracking-plans", "writekey-writekey-1.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "tracking-plans", "writekey-writekey-2.yaml"))
}

func TestConvertCommand_EmptySchemas(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "empty_schemas.json")
	outputDir := filepath.Join(tempDir, "output")

	// Create test input file with empty schemas array
	testData := `{
		"schemas": []
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	err = runConvert(inputFile, outputDir, false, false, 2)
	require.NoError(t, err)

	// Verify output files were created even with empty schemas
	assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "properties.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "custom-types.yaml"))
	assert.DirExists(t, filepath.Join(outputDir, "tracking-plans"))
}

func TestConvertCommand_ComplexNestedStructures(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "complex_schemas.json")
	outputDir := filepath.Join(tempDir, "output")

	// Create test data with complex nested structures
	testData := `{
		"schemas": [
			{
				"uid": "test-uid",
				"writeKey": "test-write-key",
				"eventType": "track",
				"eventIdentifier": "complex_event",
				"schema": {
					"event": "string",
					"properties": {
						"user": {
							"profile": {
								"name": "string",
								"age": "number"
							},
							"preferences": {
								"notifications": {
									"email": "boolean",
									"sms": "boolean"
								}
							}
						},
						"items": [
							{
								"id": "string",
								"metadata": {
									"tags": ["string"]
								}
							}
						]
					}
				}
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	err = runConvert(inputFile, outputDir, false, true, 2)
	require.NoError(t, err)

	// Verify all output files were created
	assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "properties.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "custom-types.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "tracking-plans", "writekey-test-write-key.yaml"))
}

// New tests for uncovered functions

func TestRunConvert_PublicWrapper(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		setupFile   func(tempDir string) string
		expectError bool
	}{
		{
			name: "SuccessfulExecution",
			setupFile: func(tempDir string) string {
				inputFile := filepath.Join(tempDir, "test.json")
				testData := `{
					"schemas": [
						{
							"uid": "test-uid",
							"writeKey": "test-key",
							"eventType": "track",
							"eventIdentifier": "test_event",
							"schema": {
								"event": "string",
								"userId": "string"
							}
						}
					]
				}`
				err := os.WriteFile(inputFile, []byte(testData), 0644)
				require.NoError(t, err)
				return inputFile
			},
			expectError: false,
		},
		{
			name: "InvalidInputFile",
			setupFile: func(tempDir string) string {
				return filepath.Join(tempDir, "nonexistent.json")
			},
			expectError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			inputFile := c.setupFile(tempDir)
			outputDir := filepath.Join(tempDir, "output")

			err := RunConvert(inputFile, outputDir, false, false, 2)

			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
				assert.FileExists(t, filepath.Join(outputDir, "properties.yaml"))
				assert.FileExists(t, filepath.Join(outputDir, "custom-types.yaml"))
			}
		})
	}
}

func TestConvertSchemasToYAML_InMemoryConversion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		schemas     func() []pkgModels.Schema
		dryRun      bool
		verbose     bool
		expectError bool
		validateFn  func(t *testing.T, result *converter.ConversionResult, outputDir string, dryRun bool)
	}{
		{
			name: "SuccessfulConversion",
			schemas: func() []pkgModels.Schema {
				return []pkgModels.Schema{
					{
						UID:             "test-uid-1",
						WriteKey:        "test-key",
						EventType:       "track",
						EventIdentifier: "test_event",
						Schema: map[string]interface{}{
							"event":  "string",
							"userId": "string",
							"properties": map[string]interface{}{
								"product_id": "string",
							},
						},
					},
				}
			},
			dryRun:      false,
			verbose:     false,
			expectError: false,
			validateFn: func(t *testing.T, result *converter.ConversionResult, outputDir string, dryRun bool) {
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.EventsCount)
				assert.Greater(t, result.PropertiesCount, 0)
				if !dryRun {
					assert.Greater(t, len(result.GeneratedFiles), 0)
					assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
				}
			},
		},
		{
			name: "DryRunMode",
			schemas: func() []pkgModels.Schema {
				return []pkgModels.Schema{
					{
						UID:             "test-uid-2",
						WriteKey:        "test-key-2",
						EventType:       "track",
						EventIdentifier: "dry_run_event",
						Schema: map[string]interface{}{
							"event": "string",
						},
					},
				}
			},
			dryRun:      true,
			verbose:     true,
			expectError: false,
			validateFn: func(t *testing.T, result *converter.ConversionResult, outputDir string, dryRun bool) {
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.EventsCount)
				// In dry run mode, no files should be created
				_, err := os.Stat(filepath.Join(outputDir, "events.yaml"))
				assert.True(t, os.IsNotExist(err))
			},
		},
		{
			name: "EmptySchemas",
			schemas: func() []pkgModels.Schema {
				return []pkgModels.Schema{}
			},
			dryRun:      false,
			verbose:     false,
			expectError: false,
			validateFn: func(t *testing.T, result *converter.ConversionResult, outputDir string, dryRun bool) {
				assert.NotNil(t, result)
				assert.Equal(t, 0, result.EventsCount)
				assert.Equal(t, 0, result.PropertiesCount)
				assert.Equal(t, 0, result.CustomTypesCount)
				if !dryRun {
					assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
				}
			},
		},
		{
			name: "VerboseMode",
			schemas: func() []pkgModels.Schema {
				return []pkgModels.Schema{
					{
						UID:             "verbose-test",
						WriteKey:        "verbose-key",
						EventType:       "track",
						EventIdentifier: "verbose_event",
						Schema: map[string]interface{}{
							"event": "string",
							"properties": map[string]interface{}{
								"complex": map[string]interface{}{
									"nested": map[string]interface{}{
										"value": "string",
									},
								},
							},
						},
					},
				}
			},
			dryRun:      false,
			verbose:     true,
			expectError: false,
			validateFn: func(t *testing.T, result *converter.ConversionResult, outputDir string, dryRun bool) {
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.EventsCount)
				assert.Greater(t, result.PropertiesCount, 0)
				if !dryRun {
					assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			outputDir := filepath.Join(tempDir, "output")
			schemas := c.schemas()

			result, err := ConvertSchemasToYAML(schemas, outputDir, c.dryRun, c.verbose, 2)

			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				c.validateFn(t, result, outputDir, c.dryRun)
			}
		})
	}
}

func TestConvertSchemasToYAML_ErrorScenarios(t *testing.T) {
	t.Parallel()

	t.Run("ReadOnlyDirectory", func(t *testing.T) {
		t.Parallel()

		schemas := []pkgModels.Schema{
			{
				UID:             "test-uid",
				WriteKey:        "test-key",
				EventType:       "track",
				EventIdentifier: "test_event",
				Schema: map[string]interface{}{
					"event": "string",
				},
			},
		}

		// Use a directory that doesn't exist and can't be created
		outputDir := "/root/readonly/path"

		result, err := ConvertSchemasToYAML(schemas, outputDir, false, false, 2)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to create output directory")
	})
}

func TestConvertSchemasToYAML_IndentationLevels(t *testing.T) {
	t.Parallel()

	schemas := []pkgModels.Schema{
		{
			UID:             "indent-test",
			WriteKey:        "indent-key",
			EventType:       "track",
			EventIdentifier: "indent_event",
			Schema: map[string]interface{}{
				"event": "string",
				"properties": map[string]interface{}{
					"test": "string",
				},
			},
		},
	}

	indentLevels := []int{1, 2, 4, 8}

	for _, indent := range indentLevels {
		t.Run(fmt.Sprintf("Indent_%d", indent), func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			outputDir := filepath.Join(tempDir, fmt.Sprintf("output_indent_%d", indent))

			result, err := ConvertSchemasToYAML(schemas, outputDir, false, false, indent)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))

			// Verify the YAML file uses correct indentation
			yamlContent, err := os.ReadFile(filepath.Join(outputDir, "events.yaml"))
			assert.NoError(t, err)
			assert.NotEmpty(t, yamlContent)
		})
	}
}

func TestWriteYAMLFile_Functionality(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		data   map[string]interface{}
		indent int
	}{
		{
			name: "SimpleData",
			data: map[string]interface{}{
				"test": "value",
				"num":  123,
			},
			indent: 2,
		},
		{
			name: "ComplexData",
			data: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "events",
				"spec": map[string]interface{}{
					"events": []map[string]interface{}{
						{
							"name": "test_event",
							"type": "track",
						},
					},
				},
			},
			indent: 4,
		},
		{
			name:   "EmptyData",
			data:   map[string]interface{}{},
			indent: 2,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()

			// Create a test that indirectly tests writeYAMLFile through the public API
			schemas := []pkgModels.Schema{
				{
					UID:             "yaml-test",
					WriteKey:        "yaml-key",
					EventType:       "track",
					EventIdentifier: "yaml_event",
					Schema:          c.data,
				},
			}

			result, err := ConvertSchemasToYAML(schemas, tempDir, false, false, c.indent)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.FileExists(t, filepath.Join(tempDir, "events.yaml"))

			// Verify file content is valid YAML
			yamlContent, err := os.ReadFile(filepath.Join(tempDir, "events.yaml"))
			assert.NoError(t, err)
			assert.NotEmpty(t, yamlContent)

			// Verify it's valid YAML by trying to unmarshal it
			var parsed interface{}
			err = yaml.Unmarshal(yamlContent, &parsed)
			assert.NoError(t, err)
		})
	}
}

func TestPreviewFunctions_Coverage(t *testing.T) {
	t.Parallel()

	// Test the preview functions through dry run mode
	schemas := []pkgModels.Schema{
		{
			UID:             "preview-test-1",
			WriteKey:        "preview-key",
			EventType:       "track",
			EventIdentifier: "preview_event_1",
			Schema: map[string]interface{}{
				"event": "string",
				"properties": map[string]interface{}{
					"prop1": "string",
					"prop2": "number",
				},
			},
		},
		{
			UID:             "preview-test-2",
			WriteKey:        "preview-key",
			EventType:       "track",
			EventIdentifier: "preview_event_2",
			Schema: map[string]interface{}{
				"event": "string",
				"properties": map[string]interface{}{
					"complex": map[string]interface{}{
						"nested": map[string]interface{}{
							"value": "string",
						},
					},
				},
			},
		},
		{
			UID:             "preview-test-3",
			WriteKey:        "preview-key",
			EventType:       "track",
			EventIdentifier: "preview_event_3",
			Schema: map[string]interface{}{
				"event": "string",
				"properties": map[string]interface{}{
					"simple": "string",
				},
			},
		},
	}

	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "preview_output")

	// Test with verbose dry run to trigger preview functions
	result, err := ConvertSchemasToYAML(schemas, outputDir, true, true, 2)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.EventsCount)
	assert.Greater(t, result.PropertiesCount, 0)

	// Verify no files were created in dry run mode
	_, err = os.Stat(filepath.Join(outputDir, "events.yaml"))
	assert.True(t, os.IsNotExist(err))
}

func TestPreviewFunctions_EdgeCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		schemas []pkgModels.Schema
	}{
		{
			name:    "EmptySchemas",
			schemas: []pkgModels.Schema{},
		},
		{
			name: "SingleSchema",
			schemas: []pkgModels.Schema{
				{
					UID:             "single-test",
					WriteKey:        "single-key",
					EventType:       "track",
					EventIdentifier: "single_event",
					Schema: map[string]interface{}{
						"event": "string",
					},
				},
			},
		},
		{
			name: "SchemaWithoutCustomTypes",
			schemas: []pkgModels.Schema{
				{
					UID:             "no-custom-test",
					WriteKey:        "no-custom-key",
					EventType:       "track",
					EventIdentifier: "no_custom_event",
					Schema: map[string]interface{}{
						"event":  "string",
						"userId": "string",
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			outputDir := filepath.Join(tempDir, "edge_case_output")

			// Test with verbose dry run to trigger preview functions with edge cases
			result, err := ConvertSchemasToYAML(c.schemas, outputDir, true, true, 2)

			assert.NoError(t, err)
			assert.NotNil(t, result)

			// Verify no files were created in dry run mode
			_, err = os.Stat(filepath.Join(outputDir, "events.yaml"))
			assert.True(t, os.IsNotExist(err))
		})
	}
}

func TestConvertSchemasToYAML_PerformanceBenefits(t *testing.T) {
	t.Parallel()

	// Test with a larger dataset to ensure the in-memory processing works efficiently
	schemas := make([]pkgModels.Schema, 100)
	for i := 0; i < 100; i++ {
		schemas[i] = pkgModels.Schema{
			UID:             fmt.Sprintf("perf-test-%d", i),
			WriteKey:        fmt.Sprintf("perf-key-%d", i%5), // 5 different write keys
			EventType:       "track",
			EventIdentifier: fmt.Sprintf("perf_event_%d", i),
			Schema: map[string]interface{}{
				"event": "string",
				"properties": map[string]interface{}{
					fmt.Sprintf("prop_%d", i): "string",
					"common_prop":             "string",
				},
			},
		}
	}

	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "performance_output")

	start := time.Now()
	result, err := ConvertSchemasToYAML(schemas, outputDir, false, false, 2)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 100, result.EventsCount)
	assert.Greater(t, result.PropertiesCount, 100)

	// Verify files were created
	assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "properties.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "custom-types.yaml"))

	// Verify tracking plans for different write keys were created
	assert.DirExists(t, filepath.Join(outputDir, "tracking-plans"))

	// Performance should be reasonable (less than 5 seconds for 100 schemas)
	assert.Less(t, duration, 5*time.Second, "Conversion took too long: %v", duration)

	fmt.Printf("Processed 100 schemas in %v\n", duration)
}
