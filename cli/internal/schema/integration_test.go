package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/unflatten"
	"github.com/rudderlabs/rudder-iac/cli/internal/testhelpers"
	pkgModels "github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullWorkflow tests the complete workflow: fetch -> unflatten -> convert
func TestFullWorkflow(t *testing.T) {
	// Remove t.Parallel() to avoid race conditions
	// t.Parallel()

	// Initialize viper for test
	cleanup := testhelpers.SetupViper(t)
	defer cleanup()

	// Create test schema with flattened structure
	testSchema := testhelpers.CreateFlattenedTestSchema("test-uid-1", "test-write-key", "product_viewed")

	// Setup mock server for fetch
	server := testhelpers.SetupMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testhelpers.AssertHTTPRequest(t, r, "GET", "/v2/schemas")
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		mockResponse := testhelpers.CreateTestSchemaResponse([]models.Schema{testSchema}, 1, false)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))

	// Create temporary directory for test
	tempDir := t.TempDir()

	// Set environment variables
	cleanupEnv := testhelpers.SetupEnvVars(t, map[string]string{
		"RUDDERSTACK_ACCESS_TOKEN": "test-token",
		"RUDDERSTACK_API_URL":      server.URL,
	})
	defer cleanupEnv()

	// Step 1: Fetch schemas
	fetchedFile := filepath.Join(tempDir, "fetched_schemas.json")
	err := performFetch([]string{"test-write-key"}, fetchedFile, false, false)
	require.NoError(t, err)

	// Verify fetched file exists and has correct content
	assert.FileExists(t, fetchedFile)
	fetchedData, err := os.ReadFile(fetchedFile)
	require.NoError(t, err)

	var fetchedSchemas models.SchemasFile
	err = json.Unmarshal(fetchedData, &fetchedSchemas)
	require.NoError(t, err)
	require.Len(t, fetchedSchemas.Schemas, 1)

	// Step 2: Unflatten schemas
	unflattenedFile := filepath.Join(tempDir, "unflattened_schemas.json")
	err = performUnflatten(fetchedFile, unflattenedFile, false, false)
	require.NoError(t, err)

	// Verify unflattened file exists and has correct structure
	assert.FileExists(t, unflattenedFile)
	unflattenedData, err := os.ReadFile(unflattenedFile)
	require.NoError(t, err)

	var unflattenedSchemas models.SchemasFile
	err = json.Unmarshal(unflattenedData, &unflattenedSchemas)
	require.NoError(t, err)
	require.Len(t, unflattenedSchemas.Schemas, 1)

	// Verify the schema was properly unflattened
	schema := unflattenedSchemas.Schemas[0].Schema
	assert.Contains(t, schema, "context")
	context, ok := schema["context"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, context, "app")

	// Step 3: Convert to YAML
	outputDir := filepath.Join(tempDir, "yaml_output")
	err = performConvert(unflattenedFile, outputDir, false, false, 2)
	require.NoError(t, err)

	// Verify output files were created
	assert.FileExists(t, filepath.Join(outputDir, "events.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "properties.yaml"))
	assert.FileExists(t, filepath.Join(outputDir, "custom-types.yaml"))
	assert.DirExists(t, filepath.Join(outputDir, "tracking-plans"))

	// Verify content of events.yaml
	eventsContent, err := os.ReadFile(filepath.Join(outputDir, "events.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(eventsContent), "product_viewed")
	assert.Contains(t, string(eventsContent), "kind: events")
}

// TestFetchWithPagination tests fetching with pagination
func TestFetchWithPagination(t *testing.T) {
	// Remove t.Parallel() to avoid race conditions
	// t.Parallel()

	// Initialize viper for test
	cleanup := testhelpers.SetupViper(t)
	defer cleanup()

	// Track request count with mutex for thread safety
	var requestCount int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		currentRequest := requestCount
		mu.Unlock()

		var response map[string]interface{}
		if currentRequest == 1 {
			// First page
			response = map[string]interface{}{
				"results": []map[string]interface{}{
					{
						"uid":             "test-uid-1",
						"writeKey":        "test-write-key",
						"eventType":       "track",
						"eventIdentifier": "event_1",
						"schema":          map[string]interface{}{"event": "string"},
					},
				},
				"currentPage": 1,
				"hasNext":     true,
			}
		} else {
			// Second page
			response = map[string]interface{}{
				"results": []map[string]interface{}{
					{
						"uid":             "test-uid-2",
						"writeKey":        "test-write-key",
						"eventType":       "track",
						"eventIdentifier": "event_2",
						"schema":          map[string]interface{}{"event": "string"},
					},
				},
				"currentPage": 2,
				"hasNext":     false,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	tempDir := t.TempDir()

	// Set environment variables
	originalToken := os.Getenv("RUDDERSTACK_ACCESS_TOKEN")
	originalURL := os.Getenv("RUDDERSTACK_API_URL")
	os.Setenv("RUDDERSTACK_ACCESS_TOKEN", "test-token")
	os.Setenv("RUDDERSTACK_API_URL", server.URL)
	defer func() {
		if originalToken == "" {
			os.Unsetenv("RUDDERSTACK_ACCESS_TOKEN")
		} else {
			os.Setenv("RUDDERSTACK_ACCESS_TOKEN", originalToken)
		}
		if originalURL == "" {
			os.Unsetenv("RUDDERSTACK_API_URL")
		} else {
			os.Setenv("RUDDERSTACK_API_URL", originalURL)
		}
	}()

	// Fetch with pagination
	outputFile := filepath.Join(tempDir, "paginated_schemas.json")
	err := performFetch([]string{"test-write-key"}, outputFile, false, false)
	require.NoError(t, err)

	// Verify both requests were made
	mu.Lock()
	finalRequestCount := requestCount
	mu.Unlock()
	assert.Equal(t, 2, finalRequestCount)

	// Verify both schemas were fetched
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var schemas models.SchemasFile
	err = json.Unmarshal(data, &schemas)
	require.NoError(t, err)
	require.Len(t, schemas.Schemas, 2)

	// Verify content
	assert.Equal(t, "event_1", schemas.Schemas[0].EventIdentifier)
	assert.Equal(t, "event_2", schemas.Schemas[1].EventIdentifier)
}

// TestDryRunMode tests that dry run mode doesn't create files
func TestDryRunMode(t *testing.T) {
	// Removed t.Parallel() to avoid race conditions with viper

	// Initialize viper for test
	cleanup := testhelpers.SetupViper(t)
	defer cleanup()

	tempDir := t.TempDir()

	// Create test input file
	inputFile := filepath.Join(tempDir, "test_schemas.json")
	testData := models.SchemasFile{
		Schemas: []models.Schema{
			{
				UID:             "test-uid",
				WriteKey:        "test-write-key",
				EventType:       "track",
				EventIdentifier: "test_event",
				Schema: map[string]interface{}{
					"event":                "string",
					"properties.test_prop": "string",
				},
			},
		},
	}

	data, err := json.Marshal(testData)
	require.NoError(t, err)
	err = os.WriteFile(inputFile, data, 0644)
	require.NoError(t, err)

	// Test unflatten dry run
	unflattenedFile := filepath.Join(tempDir, "unflattened.json")
	err = performUnflatten(inputFile, unflattenedFile, true, false) // dry run = true
	require.NoError(t, err)

	// File should NOT be created in dry run
	assert.NoFileExists(t, unflattenedFile)

	// Test convert dry run
	outputDir := filepath.Join(tempDir, "output")
	err = performConvert(inputFile, outputDir, true, false, 2) // dry run = true
	require.NoError(t, err)

	// Output directory should NOT be created in dry run
	assert.NoDirExists(t, outputDir)
}

// TestErrorHandling tests various error conditions
func TestErrorHandling(t *testing.T) {
	// Removed t.Parallel() to avoid race conditions with viper

	// Initialize viper for test
	cleanup := testhelpers.SetupViper(t)
	defer cleanup()

	tempDir := t.TempDir()

	cases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "FetchWithInvalidCredentials",
			testFunc: func(t *testing.T) {
				// Initialize viper for subtest
				cleanup := testhelpers.SetupViper(t)
				defer cleanup()

				// Store original values to restore later
				originalToken := os.Getenv("RUDDERSTACK_ACCESS_TOKEN")
				originalURL := os.Getenv("RUDDERSTACK_API_URL")

				// Test fetch with missing credentials
				os.Unsetenv("RUDDERSTACK_ACCESS_TOKEN")
				os.Unsetenv("RUDDERSTACK_API_URL")
				defer func() {
					if originalToken == "" {
						os.Unsetenv("RUDDERSTACK_ACCESS_TOKEN")
					} else {
						os.Setenv("RUDDERSTACK_ACCESS_TOKEN", originalToken)
					}
					if originalURL == "" {
						os.Unsetenv("RUDDERSTACK_API_URL")
					} else {
						os.Setenv("RUDDERSTACK_API_URL", originalURL)
					}
				}()

				err := performFetch([]string{}, "test.json", false, false)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "access token is required")
			},
		},
		{
			name: "UnflattenWithInvalidJSON",
			testFunc: func(t *testing.T) {
				// Create invalid JSON file
				inputFile := filepath.Join(tempDir, "invalid.json")
				err := os.WriteFile(inputFile, []byte("invalid json"), 0644)
				require.NoError(t, err)

				outputFile := filepath.Join(tempDir, "unflatten_error.json")
				err = performUnflatten(inputFile, outputFile, false, false)
				assert.Error(t, err)
			},
		},
		{
			name: "ConvertWithNonexistentInput",
			testFunc: func(t *testing.T) {
				nonexistentFile := filepath.Join(tempDir, "nonexistent.json")
				outputDir := filepath.Join(tempDir, "convert_error")
				err := performConvert(nonexistentFile, outputDir, false, false, 2)
				assert.Error(t, err)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			c.testFunc(t)
		})
	}
}

// Helper functions that would call the actual command implementations
// These would be implemented to call the real command functions

func performFetch(writeKeys []string, outputFile string, dryRun, verbose bool) error {
	// Get configuration from viper (which handles the same env vars)
	apiToken := viper.GetString("auth.accessToken")
	if apiToken == "" {
		return fmt.Errorf("access token is required. Please run 'rudder-cli auth login' or set RUDDERSTACK_ACCESS_TOKEN environment variable")
	}

	apiURL := viper.GetString("apiURL")
	var centralClient *client.Client
	var err error

	if apiURL == "" {
		// Use default central client (which defaults to https://api.rudderstack.com/v2)
		centralClient, err = client.New(apiToken)
	} else {
		// Use custom URL directly (v2 is added in the request path)
		centralClient, err = client.New(apiToken, client.WithBaseURL(apiURL))
	}
	if err != nil {
		return fmt.Errorf("failed to create central API client: %w", err)
	}

	// Determine writeKey parameter
	var writeKey string
	if len(writeKeys) > 0 && writeKeys[0] != "" {
		writeKey = writeKeys[0]
	}

	// Fetch schemas directly using central client
	pkgSchemas, err := fetchAllSchemas(centralClient, writeKey)
	if err != nil {
		return fmt.Errorf("failed to fetch schemas: %w", err)
	}

	if dryRun {
		return nil // Don't create file in dry run
	}

	// Create output structure using pkg models (internal models are re-exports)
	output := pkgModels.SchemasFile{
		Schemas: pkgSchemas,
	}

	// Write output file
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// fetchAllSchemas fetches all schemas with pagination from the API using the central client
func fetchAllSchemas(apiClient *client.Client, writeKey string) ([]pkgModels.Schema, error) {
	var allSchemas []pkgModels.Schema
	page := 1

	for {
		schemas, hasNext, err := fetchSchemasPage(apiClient, page, writeKey)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch schemas page %d: %w", page, err)
		}

		allSchemas = append(allSchemas, schemas...)

		if !hasNext {
			break
		}
		page++
	}

	return allSchemas, nil
}

// fetchSchemasPage fetches a single page of schemas for integration tests
func fetchSchemasPage(apiClient *client.Client, page int, writeKey string) ([]pkgModels.Schema, bool, error) {
	// Build path with query parameters
	path := "v2/schemas"
	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	if writeKey != "" {
		query.Set("writeKey", writeKey)
	}

	if queryStr := query.Encode(); queryStr != "" {
		path = path + "?" + queryStr
	}

	// Make request using the central client
	ctx := context.Background()
	data, err := apiClient.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to make request: %w", err)
	}

	// Parse response
	var response pkgModels.SchemasResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, false, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Results, response.HasNext, nil
}

func performUnflatten(inputFile, outputFile string, dryRun, verbose bool) error {
	if dryRun {
		return nil // Don't create file in dry run
	}

	// Read input file
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return err
	}

	// Parse JSON
	var schemasFile pkgModels.SchemasFile
	if err := json.Unmarshal(data, &schemasFile); err != nil {
		return err
	}

	// Actually perform the unflatten operation on each schema
	for i := range schemasFile.Schemas {
		unflattenedSchema := unflatten.UnflattenSchema(schemasFile.Schemas[i].Schema)
		schemasFile.Schemas[i].Schema = unflattenedSchema
	}

	// Write the unflattened result
	outputData, err := json.Marshal(schemasFile)
	if err != nil {
		return fmt.Errorf("failed to marshal unflattened data: %w", err)
	}

	return os.WriteFile(outputFile, outputData, 0644)
}

func performConvert(inputFile, outputDir string, dryRun, verbose bool, yamlIndent int) error {
	if dryRun {
		return nil // Don't create files in dry run
	}

	// Read and validate input file
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return err
	}

	var schemasFile pkgModels.SchemasFile
	if err := json.Unmarshal(data, &schemasFile); err != nil {
		return err
	}

	// Use the actual converter logic
	converter := converter.NewSchemaConverter(converter.ConversionOptions{
		InputFile:  inputFile,
		OutputDir:  outputDir,
		DryRun:     dryRun,
		Verbose:    verbose,
		YAMLIndent: yamlIndent,
	})

	_, err = converter.Convert()
	return err
}
