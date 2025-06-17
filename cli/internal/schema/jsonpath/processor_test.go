package jsonpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONPathProcessor(t *testing.T) {
	t.Parallel()

	// Shared test schemas
	testSchema := map[string]interface{}{
		"event":  "product_viewed",
		"userId": "12345",
		"properties": map[string]interface{}{
			"product_id":   "abc123",
			"product_name": "iPhone",
			"price":        999.99,
			"categories":   []interface{}{"electronics", "phones"},
		},
		"context": map[string]interface{}{
			"app": map[string]interface{}{
				"name":    "MyApp",
				"version": "1.0.0",
			},
			"traits": map[string]interface{}{
				"email": "user@example.com",
				"name":  "John Doe",
			},
		},
	}

	complexSchema := map[string]interface{}{
		"data": map[string]interface{}{
			"users": []interface{}{
				map[string]interface{}{"id": "1", "name": "Alice", "active": true},
				map[string]interface{}{"id": "2", "name": "Bob", "active": false},
				map[string]interface{}{"id": "3", "name": "Charlie", "active": true},
			},
			"metadata": map[string]interface{}{
				"version":  "2.0",
				"features": []interface{}{"auth", "analytics"},
			},
		},
	}

	integrationSchemas := []struct {
		name        string
		schema      map[string]interface{}
		jsonPath    string
		expectedVal interface{}
	}{
		{
			name:        "SimplePropertyExtraction",
			jsonPath:    "$.user_id",
			schema:      map[string]interface{}{"user_id": "12345", "event": "page_viewed"},
			expectedVal: "12345",
		},
		{
			name:     "NestedObjectExtraction",
			jsonPath: "$.properties",
			schema: map[string]interface{}{
				"event":      "purchase",
				"properties": map[string]interface{}{"total": 99.99, "currency": "USD"},
			},
			expectedVal: map[string]interface{}{"total": 99.99, "currency": "USD"},
		},
		{
			name:     "ArrayExtraction",
			jsonPath: "$.items",
			schema: map[string]interface{}{
				"items": []interface{}{"item1", "item2", "item3"},
			},
			expectedVal: []interface{}{"item1", "item2", "item3"},
		},
	}

	cases := []struct {
		category string
		name     string
		jsonPath string
		skipFail bool
		schema   map[string]interface{}
		validate func(t *testing.T, processor *Processor)
	}{
		// Processor Creation Tests
		{
			category: "Creation",
			name:     "CreateProcessor",
			jsonPath: "$.properties",
			skipFail: true,
			validate: func(t *testing.T, processor *Processor) {
				assert.NotNil(t, processor)
				assert.Equal(t, "$.properties", processor.jsonPath)
				assert.True(t, processor.skipFailed)
			},
		},
		{
			category: "Creation",
			name:     "CreateProcessorSkipFalse",
			jsonPath: "$.context",
			skipFail: false,
			validate: func(t *testing.T, processor *Processor) {
				assert.Equal(t, "$.context", processor.jsonPath)
				assert.False(t, processor.skipFailed)
			},
		},

		// Root Path Detection Tests
		{
			category: "RootPath",
			name:     "EmptyPath",
			jsonPath: "",
			skipFail: true,
			validate: func(t *testing.T, processor *Processor) {
				assert.True(t, processor.IsRootPath())
			},
		},
		{
			category: "RootPath",
			name:     "DollarSign",
			jsonPath: "$",
			skipFail: true,
			validate: func(t *testing.T, processor *Processor) {
				assert.True(t, processor.IsRootPath())
			},
		},
		{
			category: "RootPath",
			name:     "DollarDot",
			jsonPath: "$.",
			skipFail: true,
			validate: func(t *testing.T, processor *Processor) {
				assert.True(t, processor.IsRootPath())
			},
		},
		{
			category: "RootPath",
			name:     "PropertiesPath",
			jsonPath: "$.properties",
			skipFail: true,
			validate: func(t *testing.T, processor *Processor) {
				assert.False(t, processor.IsRootPath())
			},
		},
		{
			category: "RootPath",
			name:     "NestedPath",
			jsonPath: "$.context.app",
			skipFail: true,
			validate: func(t *testing.T, processor *Processor) {
				assert.False(t, processor.IsRootPath())
			},
		},

		// Schema Processing Tests
		{
			category: "Processing",
			name:     "RootPathExtraction",
			jsonPath: "$",
			skipFail: true,
			schema:   testSchema,
			validate: func(t *testing.T, processor *Processor) {
				processResult := processor.ProcessSchema(testSchema)
				assert.NoError(t, processResult.Error)
				assert.Equal(t, testSchema, processResult.Value)
			},
		},
		{
			category: "Processing",
			name:     "ExtractProperties",
			jsonPath: "$.properties",
			skipFail: true,
			schema:   testSchema,
			validate: func(t *testing.T, processor *Processor) {
				processResult := processor.ProcessSchema(testSchema)
				require.NoError(t, processResult.Error)
				require.NotNil(t, processResult.Value)

				extracted, ok := processResult.Value.(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "abc123", extracted["product_id"])
				assert.Equal(t, "iPhone", extracted["product_name"])
				assert.Equal(t, 999.99, extracted["price"])

				categories, ok := extracted["categories"].([]interface{})
				require.True(t, ok)
				assert.Len(t, categories, 2)
				assert.Equal(t, "electronics", categories[0])
				assert.Equal(t, "phones", categories[1])
			},
		},
		{
			category: "Processing",
			name:     "ExtractNestedField",
			jsonPath: "$.context.app",
			skipFail: true,
			schema:   testSchema,
			validate: func(t *testing.T, processor *Processor) {
				processResult := processor.ProcessSchema(testSchema)
				require.NoError(t, processResult.Error)
				require.NotNil(t, processResult.Value)

				extracted, ok := processResult.Value.(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "MyApp", extracted["name"])
				assert.Equal(t, "1.0.0", extracted["version"])
			},
		},
		{
			category: "Processing",
			name:     "ExtractPrimitiveValue",
			jsonPath: "$.userId",
			skipFail: true,
			schema:   testSchema,
			validate: func(t *testing.T, processor *Processor) {
				processResult := processor.ProcessSchema(testSchema)
				require.NoError(t, processResult.Error)
				assert.Equal(t, "12345", processResult.Value)
			},
		},
		{
			category: "Processing",
			name:     "ExtractArray",
			jsonPath: "$.properties.categories",
			skipFail: true,
			schema:   testSchema,
			validate: func(t *testing.T, processor *Processor) {
				processResult := processor.ProcessSchema(testSchema)
				require.NoError(t, processResult.Error)
				require.NotNil(t, processResult.Value)

				categories, ok := processResult.Value.([]interface{})
				require.True(t, ok)
				assert.Len(t, categories, 2)
				assert.Equal(t, "electronics", categories[0])
				assert.Equal(t, "phones", categories[1])
			},
		},
		{
			category: "Processing",
			name:     "ExtractArrayElement",
			jsonPath: "$.properties.categories.0",
			skipFail: true,
			schema:   testSchema,
			validate: func(t *testing.T, processor *Processor) {
				processResult := processor.ProcessSchema(testSchema)
				require.NoError(t, processResult.Error)
				assert.Equal(t, "electronics", processResult.Value)
			},
		},

		// Error Handling Tests
		{
			category: "ErrorHandling",
			name:     "NonExistentPath",
			jsonPath: "$.nonexistent",
			skipFail: true,
			schema:   testSchema,
			validate: func(t *testing.T, processor *Processor) {
				processResult := processor.ProcessSchema(testSchema)
				assert.Error(t, processResult.Error)
				assert.Nil(t, processResult.Value)
				assert.Contains(t, processResult.Error.Error(), "JSONPath '$.nonexistent' returned no results")
			},
		},
		{
			category: "ErrorHandling",
			name:     "InvalidJSONPath",
			jsonPath: "$.properties..invalid",
			skipFail: true,
			schema:   testSchema,
			validate: func(t *testing.T, processor *Processor) {
				processResult := processor.ProcessSchema(testSchema)
				assert.Error(t, processResult.Error)
				assert.Nil(t, processResult.Value)
				assert.Contains(t, processResult.Error.Error(), "returned no results")
			},
		},
		{
			category: "ErrorHandling",
			name:     "EmptySchema",
			jsonPath: "$.properties",
			skipFail: true,
			schema:   map[string]interface{}{},
			validate: func(t *testing.T, processor *Processor) {
				processResult := processor.ProcessSchema(map[string]interface{}{})
				assert.Error(t, processResult.Error)
				assert.Nil(t, processResult.Value)
				assert.Contains(t, processResult.Error.Error(), "returned no results")
			},
		},

		// Skip Behavior Tests
		{
			category: "SkipBehavior",
			name:     "SkipFailedTrue",
			jsonPath: "$.properties",
			skipFail: true,
			validate: func(t *testing.T, processor *Processor) {
				assert.True(t, processor.ShouldSkipOnError())
			},
		},
		{
			category: "SkipBehavior",
			name:     "SkipFailedFalse",
			jsonPath: "$.properties",
			skipFail: false,
			validate: func(t *testing.T, processor *Processor) {
				assert.False(t, processor.ShouldSkipOnError())
			},
		},

		// Complex JSONPath Tests
		{
			category: "ComplexPaths",
			name:     "ArrayFirstElement",
			jsonPath: "$.data.users.0",
			skipFail: true,
			schema:   complexSchema,
			validate: func(t *testing.T, processor *Processor) {
				processResult := processor.ProcessSchema(complexSchema)
				require.NoError(t, processResult.Error)
				require.NotNil(t, processResult.Value)

				user, ok := processResult.Value.(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "Alice", user["name"])
				assert.Equal(t, "1", user["id"])
				assert.True(t, user["active"].(bool))
			},
		},
		{
			category: "ComplexPaths",
			name:     "NestedArrayAccess",
			jsonPath: "$.data.users.0.name",
			skipFail: true,
			schema:   complexSchema,
			validate: func(t *testing.T, processor *Processor) {
				processResult := processor.ProcessSchema(complexSchema)
				require.NoError(t, processResult.Error)
				assert.Equal(t, "Alice", processResult.Value)
			},
		},
		{
			category: "ComplexPaths",
			name:     "DeepNestedPath",
			jsonPath: "$.data.metadata.version",
			skipFail: true,
			schema:   complexSchema,
			validate: func(t *testing.T, processor *Processor) {
				processResult := processor.ProcessSchema(complexSchema)
				require.NoError(t, processResult.Error)
				assert.Equal(t, "2.0", processResult.Value)
			},
		},

		// JSONPath Normalization Tests
		{
			category: "Normalization",
			name:     "NormalizeBasicPath",
			jsonPath: "properties",
			skipFail: true,
			validate: func(t *testing.T, processor *Processor) {
				normalizedPath := processor.normalizeJSONPath()
				assert.Equal(t, "properties", normalizedPath)
			},
		},
		{
			category: "Normalization",
			name:     "NormalizeComplexPath",
			jsonPath: "context.app.name",
			skipFail: true,
			validate: func(t *testing.T, processor *Processor) {
				normalizedPath := processor.normalizeJSONPath()
				assert.Equal(t, "context.app.name", normalizedPath)
			},
		},
		{
			category: "Normalization",
			name:     "AlreadyNormalizedPath",
			jsonPath: "$.data.users",
			skipFail: true,
			validate: func(t *testing.T, processor *Processor) {
				normalizedPath := processor.normalizeJSONPath()
				assert.Equal(t, "data.users", normalizedPath)
			},
		},
	}

	// Add integration test scenarios to main test matrix
	for _, integration := range integrationSchemas {
		cases = append(cases, struct {
			category string
			name     string
			jsonPath string
			skipFail bool
			schema   map[string]interface{}
			validate func(t *testing.T, processor *Processor)
		}{
			category: "Integration",
			name:     integration.name,
			jsonPath: integration.jsonPath,
			skipFail: true,
			schema:   integration.schema,
			validate: func(t *testing.T, processor *Processor) {
				result := processor.ProcessSchema(integration.schema)
				require.NoError(t, result.Error, "Processing should succeed for valid JSONPath")
				assert.Equal(t, integration.expectedVal, result.Value, "Extracted value should match expected")
			},
		})
	}

	for _, c := range cases {
		t.Run(c.category+"/"+c.name, func(t *testing.T) {
			t.Parallel()

			processor := NewProcessor(c.jsonPath, c.skipFail)
			schema := c.schema
			if schema == nil {
				schema = testSchema // Default schema
			}

			c.validate(t, processor)
		})
	}
}
