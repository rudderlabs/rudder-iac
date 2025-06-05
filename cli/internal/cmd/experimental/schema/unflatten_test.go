package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnflattenCommand_Integration(t *testing.T) {
	t.Parallel()

	// Create temporary directory for test
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test_input.json")
	outputFile := filepath.Join(tempDir, "test_output.json")

	// Create test input file with flattened schema structure
	testData := `{
		"schemas": [
			{
				"uid": "test-uid-1",
				"writeKey": "test-write-key",
				"eventType": "track",
				"eventIdentifier": "product_viewed",
				"schema": {
					"context.app.name": "string",
					"context.traits.email": "string",
					"properties.product_id": "string",
					"properties.categories.0": "string",
					"properties.categories.1": "string"
				},
				"createdAt": "2024-01-10T10:08:15.407491Z",
				"lastSeen": "2024-03-25T18:49:31.870834Z",
				"count": 10
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	// Test the unflatten command
	err = runUnflatten(inputFile, outputFile, false, false, 2)
	require.NoError(t, err)

	// Verify output file was created
	assert.FileExists(t, outputFile)

	// Read and verify the content was unflattened
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	// Basic verification that the file contains expected structure
	contentStr := string(content)
	assert.Contains(t, contentStr, `"context"`)
	assert.Contains(t, contentStr, `"app"`)
	assert.Contains(t, contentStr, `"traits"`)
	assert.Contains(t, contentStr, `"properties"`)
	assert.Contains(t, contentStr, `"categories"`)
}

func TestUnflattenCommand_Scenarios(t *testing.T) {
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
			inputFile := filepath.Join(tempDir, "test_input.json")
			outputFile := filepath.Join(tempDir, "test_output.json")

			// Create test input file
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
							"properties.test": "string"
						}
					}
				]
			}`

			err := os.WriteFile(inputFile, []byte(testData), 0644)
			require.NoError(t, err)

			err = runUnflatten(inputFile, outputFile, c.dryRun, c.verbose, 2)

			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if c.expectFiles {
				assert.FileExists(t, outputFile)
			} else {
				_, err := os.Stat(outputFile)
				assert.True(t, os.IsNotExist(err))
			}
		})
	}
}

func TestUnflattenCommand_ErrorScenarios(t *testing.T) {
	cases := []struct {
		name        string
		setupFiles  func(tempDir string) (string, string)
		expectError string
	}{
		{
			name: "InputFileNotFound",
			setupFiles: func(tempDir string) (string, string) {
				inputFile := filepath.Join(tempDir, "nonexistent.json")
				outputFile := filepath.Join(tempDir, "output.json")
				return inputFile, outputFile
			},
			expectError: "does not exist",
		},
		{
			name: "InvalidJSON",
			setupFiles: func(tempDir string) (string, string) {
				inputFile := filepath.Join(tempDir, "invalid.json")
				outputFile := filepath.Join(tempDir, "output.json")

				// Create invalid JSON
				invalidJSON := `{ "schemas": [ invalid json } ]`
				os.WriteFile(inputFile, []byte(invalidJSON), 0644)

				return inputFile, outputFile
			},
			expectError: "failed to read input file",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			inputFile, outputFile := c.setupFiles(tempDir)

			err := runUnflatten(inputFile, outputFile, false, false, 2)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), c.expectError)
		})
	}
}

func TestCountKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{
			name:     "Primitive",
			input:    "string",
			expected: 0,
		},
		{
			name: "SimpleMap",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expected: 2,
		},
		{
			name: "NestedMap",
			input: map[string]interface{}{
				"key1": "value1",
				"nested": map[string]interface{}{
					"subkey": "subvalue",
				},
			},
			expected: 3, // key1 + nested + subkey
		},
		{
			name: "Array",
			input: []interface{}{
				"item1",
				"item2",
			},
			expected: 0, // Arrays themselves don't count as keys
		},
		{
			name: "ArrayOfMaps",
			input: []interface{}{
				map[string]interface{}{
					"key": "value",
				},
			},
			expected: 1, // The key inside the map
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			result := countKeys(c.input)
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestUnflattenCommand_EmptySchemas(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "empty_schemas.json")
	outputFile := filepath.Join(tempDir, "output.json")

	// Create test input file with empty schemas
	testData := `{
		"schemas": [
			{
				"uid": "test-uid",
				"writeKey": "test-write-key",
				"eventType": "track",
				"eventIdentifier": "test_event",
				"schema": {}
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	err = runUnflatten(inputFile, outputFile, false, true, 2)
	require.NoError(t, err)

	// Verify output file was created even with empty schemas
	assert.FileExists(t, outputFile)
}
