package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
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

	// Test the unflatten command without JSONPath
	err = runUnflatten(inputFile, outputFile, false, false, 2, "", true)
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

func TestUnflattenCommand_WithJSONPath(t *testing.T) {
	t.Parallel()

	// Create test data with complex nested structure after unflattening
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test_input.json")

	testData := `{
		"schemas": [
			{
				"uid": "test-uid-1",
				"writeKey": "test-write-key",
				"eventType": "track",
				"eventIdentifier": "product_viewed",
				"schema": {
					"userId": "string",
					"event": "string",
					"properties.product_id": "string",
					"properties.product_name": "string",
					"properties.price": "number",
					"context.app.name": "string",
					"context.traits.email": "string"
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
					"userId": "string",
					"properties.order_id": "string",
					"properties.total": "number",
					"context.app.name": "string"
				},
				"createdAt": "2024-01-15T12:30:00.000000Z",
				"lastSeen": "2024-03-30T15:45:30.123456Z",
				"count": 25
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	cases := []struct {
		name        string
		jsonPath    string
		skipFailed  bool
		expectError bool
		validator   func(t *testing.T, outputFile string)
	}{
		{
			name:        "ExtractProperties",
			jsonPath:    "$.properties",
			skipFailed:  true,
			expectError: false,
			validator: func(t *testing.T, outputFile string) {
				content, err := os.ReadFile(outputFile)
				require.NoError(t, err)

				var result models.SchemasFile
				err = json.Unmarshal(content, &result)
				require.NoError(t, err)

				// Should have 2 schemas with only properties content
				assert.Len(t, result.Schemas, 2)

				// First schema should have properties extracted
				firstSchema := result.Schemas[0].Schema
				assert.Contains(t, firstSchema, "product_id")
				assert.Contains(t, firstSchema, "product_name")
				assert.Contains(t, firstSchema, "price")
				// Should not contain original top-level fields
				assert.NotContains(t, firstSchema, "userId")
				assert.NotContains(t, firstSchema, "event")

				// Second schema should have properties extracted
				secondSchema := result.Schemas[1].Schema
				assert.Contains(t, secondSchema, "order_id")
				assert.Contains(t, secondSchema, "total")
			},
		},
		{
			name:        "ExtractContext",
			jsonPath:    "$.context",
			skipFailed:  true,
			expectError: false,
			validator: func(t *testing.T, outputFile string) {
				content, err := os.ReadFile(outputFile)
				require.NoError(t, err)

				var result models.SchemasFile
				err = json.Unmarshal(content, &result)
				require.NoError(t, err)

				// Should have 2 schemas with only context content
				assert.Len(t, result.Schemas, 2)

				firstSchema := result.Schemas[0].Schema
				assert.Contains(t, firstSchema, "app")
				assert.Contains(t, firstSchema, "traits")
			},
		},
		{
			name:        "ExtractPrimitiveValue",
			jsonPath:    "$.userId",
			skipFailed:  true,
			expectError: false,
			validator: func(t *testing.T, outputFile string) {
				content, err := os.ReadFile(outputFile)
				require.NoError(t, err)

				var result models.SchemasFile
				err = json.Unmarshal(content, &result)
				require.NoError(t, err)

				// Should have 2 schemas with primitive value wrapped
				assert.Len(t, result.Schemas, 2)

				firstSchema := result.Schemas[0].Schema
				assert.Contains(t, firstSchema, "value")
				assert.Equal(t, "string", firstSchema["value"])
			},
		},
		{
			name:        "RootPath",
			jsonPath:    "$",
			skipFailed:  true,
			expectError: false,
			validator: func(t *testing.T, outputFile string) {
				content, err := os.ReadFile(outputFile)
				require.NoError(t, err)

				var result models.SchemasFile
				err = json.Unmarshal(content, &result)
				require.NoError(t, err)

				// Should have 2 schemas with full unflattened content (like no JSONPath)
				assert.Len(t, result.Schemas, 2)

				firstSchema := result.Schemas[0].Schema
				// Should have all top-level fields after unflattening
				assert.Contains(t, firstSchema, "userId")
				assert.Contains(t, firstSchema, "event")
				assert.Contains(t, firstSchema, "properties")
				assert.Contains(t, firstSchema, "context")
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			// Use unique output file for each test
			testOutputFile := filepath.Join(tempDir, "test_output_"+c.name+".json")

			err := runUnflatten(inputFile, testOutputFile, false, false, 2, c.jsonPath, c.skipFailed)
			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.FileExists(t, testOutputFile)
				c.validator(t, testOutputFile)
			}
		})
	}
}

func TestUnflattenCommand_JSONPathErrors(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test_input.json")

	// Test data with one schema that will fail JSONPath
	testData := `{
		"schemas": [
			{
				"uid": "test-uid-1",
				"writeKey": "test-write-key",
				"eventType": "track",
				"eventIdentifier": "valid_event",
				"schema": {
					"properties.product_id": "string",
					"properties.name": "string"
				}
			},
			{
				"uid": "test-uid-2",
				"writeKey": "test-write-key-2",
				"eventType": "track",
				"eventIdentifier": "incomplete_event",
				"schema": {
					"userId": "string",
					"event": "string"
				}
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	t.Run("SkipFailedTrue", func(t *testing.T) {
		t.Parallel()

		outputFile := filepath.Join(tempDir, "output_skip_true.json")

		// Extract properties - second schema doesn't have properties, should be skipped
		err := runUnflatten(inputFile, outputFile, false, false, 2, "$.properties", true)
		require.NoError(t, err)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		var result models.SchemasFile
		err = json.Unmarshal(content, &result)
		require.NoError(t, err)

		// Should only have 1 schema (the one that succeeded)
		assert.Len(t, result.Schemas, 1)
		assert.Equal(t, "test-uid-1", result.Schemas[0].UID)
		assert.Contains(t, result.Schemas[0].Schema, "product_id")
	})

	t.Run("SkipFailedFalse", func(t *testing.T) {
		t.Parallel()

		outputFile := filepath.Join(tempDir, "output_skip_false.json")

		// Extract properties - second schema doesn't have properties, should be kept
		err := runUnflatten(inputFile, outputFile, false, false, 2, "$.properties", false)
		require.NoError(t, err)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		var result models.SchemasFile
		err = json.Unmarshal(content, &result)
		require.NoError(t, err)

		// Should have 2 schemas (one extracted, one kept as unflattened)
		assert.Len(t, result.Schemas, 2)

		// First schema should have properties extracted
		assert.Equal(t, "test-uid-1", result.Schemas[0].UID)
		assert.Contains(t, result.Schemas[0].Schema, "product_id")

		// Second schema should have original unflattened content
		assert.Equal(t, "test-uid-2", result.Schemas[1].UID)
		assert.Contains(t, result.Schemas[1].Schema, "userId")
		assert.Contains(t, result.Schemas[1].Schema, "event")
	})
}

func TestUnflattenCommand_DryRun(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test_input.json")
	outputFile := filepath.Join(tempDir, "test_output.json")

	testData := `{
		"schemas": [
			{
				"uid": "test-uid-1",
				"writeKey": "test-write-key",
				"eventType": "track",
				"eventIdentifier": "test_event",
				"schema": {
					"properties.name": "string"
				}
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	// Test dry run - should not create output file
	err = runUnflatten(inputFile, outputFile, true, true, 2, "$.properties", true)
	require.NoError(t, err)

	// Output file should not exist
	assert.NoFileExists(t, outputFile)
}

func TestUnflattenCommand_Verbose(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test_input.json")
	outputFile := filepath.Join(tempDir, "test_output.json")

	testData := `{
		"schemas": [
			{
				"uid": "test-uid-1",
				"writeKey": "test-write-key",
				"eventType": "track",
				"eventIdentifier": "test_event",
				"schema": {
					"properties.name": "string"
				}
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	// Test verbose output - this is mainly for manual verification
	// The function should complete without error
	err = runUnflatten(inputFile, outputFile, false, true, 2, "$.properties", true)
	require.NoError(t, err)

	assert.FileExists(t, outputFile)
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

	err = runUnflatten(inputFile, outputFile, false, true, 2, "", true)
	require.NoError(t, err)

	// Verify output file was created even with empty schemas
	assert.FileExists(t, outputFile)
}

func TestUnflattenCommand_FileNotFound(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "nonexistent.json")
	outputFile := filepath.Join(tempDir, "output.json")

	err := runUnflatten(inputFile, outputFile, false, false, 2, "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestUnflattenCommand_InvalidJSON(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "invalid.json")
	outputFile := filepath.Join(tempDir, "output.json")

	// Create invalid JSON file
	err := os.WriteFile(inputFile, []byte(`{"invalid": json}`), 0644)
	require.NoError(t, err)

	err = runUnflatten(inputFile, outputFile, false, false, 2, "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")
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

func TestUnflattenCommand_ComplexJSONPaths(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "complex_input.json")

	// Complex nested structure that will be unflattened first
	testData := `{
		"schemas": [
			{
				"uid": "complex-test",
				"writeKey": "test-key",
				"eventType": "track", 
				"eventIdentifier": "complex_event",
				"schema": {
					"properties.items.0.name": "string",
					"properties.items.0.price": "number",
					"properties.items.1.name": "string",
					"properties.metadata.version": "string",
					"context.nested.deep.value": "string"
				}
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	cases := []struct {
		name      string
		jsonPath  string
		validator func(t *testing.T, schema interface{})
	}{
		{
			name:     "ExtractArray",
			jsonPath: "$.properties.items",
			validator: func(t *testing.T, schema interface{}) {
				// Should be a map with an "items" key containing the array
				schemaMap := schema.(map[string]interface{})
				items, ok := schemaMap["items"].([]interface{})
				require.True(t, ok, "Expected 'items' key with array value")
				assert.Len(t, items, 2)

				// First item should have name and price
				firstItem := items[0].(map[string]interface{})
				assert.Equal(t, "string", firstItem["name"])
				assert.Equal(t, "number", firstItem["price"])
			},
		},
		{
			name:     "ExtractFirstArrayItem",
			jsonPath: "$.properties.items.0",
			validator: func(t *testing.T, schema interface{}) {
				// Should be the first item object
				schemaMap := schema.(map[string]interface{})
				assert.Equal(t, "string", schemaMap["name"])
				assert.Equal(t, "number", schemaMap["price"])
			},
		},
		{
			name:     "ExtractDeepNested",
			jsonPath: "$.context.nested.deep.value",
			validator: func(t *testing.T, schema interface{}) {
				// Should be wrapped primitive value
				schemaMap := schema.(map[string]interface{})
				assert.Equal(t, "string", schemaMap["value"])
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			testOutputFile := filepath.Join(tempDir, "complex_output_"+c.name+".json")

			err := runUnflatten(inputFile, testOutputFile, false, false, 2, c.jsonPath, true)
			require.NoError(t, err)

			content, err := os.ReadFile(testOutputFile)
			require.NoError(t, err)

			var result models.SchemasFile
			err = json.Unmarshal(content, &result)
			require.NoError(t, err)

			assert.Len(t, result.Schemas, 1)
			c.validator(t, result.Schemas[0].Schema)
		})
	}
}

func TestReadSchemasFile_HappyPaths(t *testing.T) {
	// Test successful file reading and JSON parsing to cover lines 64-65
	tempDir := t.TempDir()

	t.Run("SuccessfulFileRead", func(t *testing.T) {
		t.Parallel()

		inputFile := filepath.Join(tempDir, "valid_schemas.json")

		// Create valid test data
		testData := models.SchemasFile{
			Schemas: []models.Schema{
				{
					UID:             "test-uid-success",
					WriteKey:        "test-write-key",
					EventType:       "track",
					EventIdentifier: "test_event",
					Schema: map[string]interface{}{
						"event":                 "string",
						"userId":                "string",
						"properties.product_id": "string",
						"context.app.name":      "string",
					},
				},
			},
		}

		// Write valid JSON to file
		data, err := json.MarshalIndent(testData, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(inputFile, data, 0644)
		require.NoError(t, err)

		// Test successful file reading (covers lines 64-65 in readSchemasFile)
		result, err := readSchemasFile(inputFile)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify successful parsing
		assert.Len(t, result.Schemas, 1)
		assert.Equal(t, "test-uid-success", result.Schemas[0].UID)
		assert.Equal(t, "test_event", result.Schemas[0].EventIdentifier)
		assert.Contains(t, result.Schemas[0].Schema, "event")
		assert.Contains(t, result.Schemas[0].Schema, "properties.product_id")
	})

	t.Run("SuccessfulParsingComplexSchema", func(t *testing.T) {
		t.Parallel()

		inputFile := filepath.Join(tempDir, "complex_schemas.json")

		// Create more complex test data to thoroughly test parsing
		testData := models.SchemasFile{
			Schemas: []models.Schema{
				{
					UID:             "complex-uid-1",
					WriteKey:        "complex-write-key",
					EventType:       "track",
					EventIdentifier: "complex_event",
					Schema: map[string]interface{}{
						"nested.object.prop":    "string",
						"array.items.0.name":    "string",
						"array.items.1.value":   "number",
						"deep.nested.structure": "boolean",
					},
				},
				{
					UID:             "complex-uid-2",
					WriteKey:        "another-write-key",
					EventType:       "identify",
					EventIdentifier: "user_identified",
					Schema: map[string]interface{}{
						"traits.email": "string",
						"traits.name":  "string",
						"context.ip":   "string",
					},
				},
			},
		}

		// Write valid JSON to file
		data, err := json.MarshalIndent(testData, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(inputFile, data, 0644)
		require.NoError(t, err)

		// Test successful file reading and parsing
		result, err := readSchemasFile(inputFile)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify successful parsing of multiple schemas
		assert.Len(t, result.Schemas, 2)
		assert.Equal(t, "complex-uid-1", result.Schemas[0].UID)
		assert.Equal(t, "complex-uid-2", result.Schemas[1].UID)
		assert.Contains(t, result.Schemas[0].Schema, "nested.object.prop")
		assert.Contains(t, result.Schemas[1].Schema, "traits.email")
	})
}

func TestRunUnflatten_JSONPathSuccessPaths(t *testing.T) {
	// Test successful JSONPath processing to cover lines 86-88 and 91-92
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "jsonpath_test.json")

	// Create test data with properties that will be successfully extracted
	testData := `{
		"schemas": [
			{
				"uid": "jsonpath-success-1",
				"writeKey": "test-write-key",
				"eventType": "track",
				"eventIdentifier": "successful_extraction",
				"schema": {
					"userId": "string",
					"event": "string",
					"properties.product_id": "string",
					"properties.category": "string",
					"properties.price": "number",
					"context.app.name": "string",
					"context.library.name": "string"
				},
				"count": 15
			},
			{
				"uid": "jsonpath-success-2",
				"writeKey": "test-write-key-2",
				"eventType": "track",
				"eventIdentifier": "another_successful_extraction",
				"schema": {
					"userId": "string",
					"properties.order_id": "string",
					"properties.total": "number",
					"context.app.version": "string"
				},
				"count": 8
			}
		]
	}`

	err := os.WriteFile(inputFile, []byte(testData), 0644)
	require.NoError(t, err)

	t.Run("SuccessfulJSONPathProcessing", func(t *testing.T) {
		outputFile := filepath.Join(tempDir, "jsonpath_success_output.json")

		// Test successful JSONPath extraction (covers lines 86-88: processor result success path)
		err := runUnflatten(inputFile, outputFile, false, true, 2, "$.properties", true)
		require.NoError(t, err)

		// Verify successful processing and output (covers lines 91-92: processedCount tracking)
		assert.FileExists(t, outputFile)

		// Read and verify successful extraction
		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		var result models.SchemasFile
		err = json.Unmarshal(content, &result)
		require.NoError(t, err)

		// Should have successfully processed both schemas
		assert.Len(t, result.Schemas, 2)

		// Verify that properties were successfully extracted
		firstSchema := result.Schemas[0].Schema
		assert.Contains(t, firstSchema, "product_id")
		assert.Contains(t, firstSchema, "category")
		assert.Contains(t, firstSchema, "price")
		// Should not contain non-properties fields
		assert.NotContains(t, firstSchema, "userId")
		assert.NotContains(t, firstSchema, "event")

		secondSchema := result.Schemas[1].Schema
		assert.Contains(t, secondSchema, "order_id")
		assert.Contains(t, secondSchema, "total")
		// Should not contain non-properties fields
		assert.NotContains(t, secondSchema, "userId")
	})

	t.Run("SuccessfulProcessingCompletion", func(t *testing.T) {
		outputFile := filepath.Join(tempDir, "processing_completion_output.json")

		// Test successful processing completion tracking (covers lines 91-92)
		err := runUnflatten(inputFile, outputFile, false, false, 2, "$.context", true)
		require.NoError(t, err)

		// Verify processing completion
		assert.FileExists(t, outputFile)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		var result models.SchemasFile
		err = json.Unmarshal(content, &result)
		require.NoError(t, err)

		// Verify both schemas were successfully processed
		assert.Len(t, result.Schemas, 2)

		// Verify context extraction worked
		firstSchema := result.Schemas[0].Schema
		assert.Contains(t, firstSchema, "app")
		assert.Contains(t, firstSchema, "library")

		secondSchema := result.Schemas[1].Schema
		assert.Contains(t, secondSchema, "app")
	})

	t.Run("SuccessfulProcessingWithoutJSONPath", func(t *testing.T) {
		outputFile := filepath.Join(tempDir, "no_jsonpath_output.json")

		// Test successful processing without JSONPath (pure unflatten success)
		err := runUnflatten(inputFile, outputFile, false, true, 2, "", true)
		require.NoError(t, err)

		assert.FileExists(t, outputFile)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		var result models.SchemasFile
		err = json.Unmarshal(content, &result)
		require.NoError(t, err)

		// Both schemas should be processed successfully
		assert.Len(t, result.Schemas, 2)

		// Verify unflattening worked (nested structures should be created)
		firstSchema := result.Schemas[0].Schema
		assert.Contains(t, firstSchema, "properties")
		assert.Contains(t, firstSchema, "context")

		// Properties should be a nested object now
		properties, ok := firstSchema["properties"].(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, properties, "product_id")
		assert.Contains(t, properties, "category")
		assert.Contains(t, properties, "price")
	})
}

func TestCountKeys_HappyPaths(t *testing.T) {
	// Test countKeys function thoroughly to cover lines 230-231 and different data types
	t.Run("CountKeysMapStructure", func(t *testing.T) {
		t.Parallel()

		// Test with nested map structure (covers map case in countKeys)
		testObj := map[string]interface{}{
			"level1_key1": "string_value",
			"level1_key2": map[string]interface{}{
				"level2_key1": "another_string",
				"level2_key2": map[string]interface{}{
					"level3_key": "deep_value",
				},
			},
			"level1_key3": "third_value",
		}

		count := countKeys(testObj)
		// Should count: level1_key1(1) + level1_key2(1) + level2_key1(1) + level2_key2(1) + level3_key(1) + level1_key3(1) = 6
		assert.Equal(t, 6, count)
	})

	t.Run("CountKeysArrayStructure", func(t *testing.T) {
		t.Parallel()

		// Test with array structure (covers array case in countKeys - line 230-231)
		testArray := []interface{}{
			map[string]interface{}{
				"item1_key": "value1",
			},
			map[string]interface{}{
				"item2_key1": "value2",
				"item2_key2": "value3",
			},
			"primitive_string",
		}

		count := countKeys(testArray)
		// Should count: item1_key(1) + item2_key1(1) + item2_key2(1) + primitive(0) = 3
		assert.Equal(t, 3, count)
	})

	t.Run("CountKeysComplexStructure", func(t *testing.T) {
		t.Parallel()

		// Test with complex mixed structure
		testObj := map[string]interface{}{
			"properties": map[string]interface{}{
				"product_id": "string",
				"categories": []interface{}{
					"electronics",
					"gadgets",
				},
				"metadata": map[string]interface{}{
					"source": "api",
					"tags": []interface{}{
						map[string]interface{}{
							"name": "important",
						},
					},
				},
			},
			"context": map[string]interface{}{
				"app": map[string]interface{}{
					"name":    "test-app",
					"version": "1.0.0",
				},
			},
		}

		count := countKeys(testObj)
		// Count all nested keys: properties(1) + product_id(1) + categories(1) + metadata(1) + source(1) + tags(1) + name(1) + context(1) + app(1) + name(1) + version(1) = 11
		assert.Equal(t, 11, count)
	})

	t.Run("CountKeysPrimitiveValues", func(t *testing.T) {
		t.Parallel()

		// Test with primitive values (covers default case)
		assert.Equal(t, 0, countKeys("string"))
		assert.Equal(t, 0, countKeys(123))
		assert.Equal(t, 0, countKeys(true))
		assert.Equal(t, 0, countKeys(nil))
	})

	t.Run("CountKeysEmptyStructures", func(t *testing.T) {
		t.Parallel()

		// Test with empty structures
		assert.Equal(t, 0, countKeys(map[string]interface{}{}))
		assert.Equal(t, 0, countKeys([]interface{}{}))
	})
}
