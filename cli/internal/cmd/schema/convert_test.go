package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "properties.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "custom-types.yaml"))
	assert.DirExists(t, filepath.Join(outputDir, "tracking-plans"))

	// Verify tracking plan files were created for each writeKey
	assert.FileExists(t, filepath.Join(outputDir, "tracking-plans", "writekey-test-write-key-1.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "tracking-plans", "writekey-test-write-key-2.yaml"))
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
				assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
				assert.FileExists(t, filepath.Join(outputDir, "properties.yaml"))
				assert.FileExists(t, filepath.Join(outputDir, "custom-types.yaml"))
				assert.DirExists(t, filepath.Join(outputDir, "tracking-plans"))
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
