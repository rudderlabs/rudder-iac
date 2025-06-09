package testhelpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// viperMutex protects concurrent access to viper configuration
var viperMutex sync.Mutex

// SetupViper initializes viper with environment variable bindings for tests
// Returns a cleanup function that should be called in defer
func SetupViper(t *testing.T) func() {
	// For parallel tests, we'll create isolated viper configuration
	// by saving and restoring the global state
	originalSettings := viper.AllSettings()

	// Reset and setup for test (no mutex needed since we restore at end)
	viper.Reset()
	viper.BindEnv("auth.accessToken", "RUDDERSTACK_ACCESS_TOKEN")
	viper.BindEnv("apiURL", "RUDDERSTACK_API_URL")

	return func() {
		viper.Reset()
		for key, value := range originalSettings {
			viper.Set(key, value)
		}
	}
}

// SetupEnvVars sets environment variables for the test and returns a cleanup function
func SetupEnvVars(t *testing.T, vars map[string]string) func() {
	originalVars := make(map[string]string)

	// Save original values and set new ones
	for key, value := range vars {
		originalVars[key] = os.Getenv(key)
		os.Setenv(key, value)
	}

	return func() {
		// Restore original values
		for key, originalValue := range originalVars {
			if originalValue == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, originalValue)
			}
		}
	}
}

// SetupMockServer creates an HTTP test server with the provided handler
func SetupMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// CreateTestSchemaResponse creates a standard test schema response
func CreateTestSchemaResponse(schemas []models.Schema, currentPage int, hasNext bool) models.SchemasResponse {
	return models.SchemasResponse{
		Results:     schemas,
		CurrentPage: currentPage,
		HasNext:     hasNext,
	}
}

// CreateTestSchema creates a test schema with sensible defaults
func CreateTestSchema(uid, writeKey, eventIdentifier string) models.Schema {
	return models.Schema{
		UID:             uid,
		WriteKey:        writeKey,
		EventType:       "track",
		EventIdentifier: eventIdentifier,
		Schema: map[string]interface{}{
			"event":  "string",
			"userId": "string",
			"properties": map[string]interface{}{
				"test_prop": "string",
			},
		},
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		Count:     10,
	}
}

// WriteTestFile writes test data to a file and ensures cleanup
func WriteTestFile(t *testing.T, filename string, data interface{}) {
	var content []byte
	var err error

	switch v := data.(type) {
	case string:
		content = []byte(v)
	case []byte:
		content = v
	default:
		content, err = json.Marshal(data)
		require.NoError(t, err)
	}

	err = os.WriteFile(filename, content, 0644)
	require.NoError(t, err)
}

// AssertFilesExist asserts that all specified files exist
func AssertFilesExist(t *testing.T, paths ...string) {
	for _, path := range paths {
		assert.FileExists(t, path, "Expected file to exist: %s", path)
	}
}

// AssertDirsExist asserts that all specified directories exist
func AssertDirsExist(t *testing.T, paths ...string) {
	for _, path := range paths {
		assert.DirExists(t, path, "Expected directory to exist: %s", path)
	}
}

// AssertValidJSON asserts that the file contains valid JSON
func AssertValidJSON(t *testing.T, filepath string) {
	content, err := os.ReadFile(filepath)
	require.NoError(t, err)

	var jsonData interface{}
	err = json.Unmarshal(content, &jsonData)
	assert.NoError(t, err, "File should contain valid JSON: %s", filepath)
}

// AssertHTTPRequest validates common HTTP request properties
func AssertHTTPRequest(t *testing.T, req *http.Request, expectedMethod, expectedPath string) {
	assert.Equal(t, expectedMethod, req.Method)
	assert.Equal(t, expectedPath, req.URL.Path)
}

// CreateStandardMockHandler creates a mock handler that returns schemas
func CreateStandardMockHandler(t *testing.T, schemas []models.Schema) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := CreateTestSchemaResponse(schemas, 1, false)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err := json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
	}
}

// CreatePaginatedMockHandler creates a mock handler that supports pagination
func CreatePaginatedMockHandler(t *testing.T, allSchemas []models.Schema, pageSize int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			// Basic page parsing
			if p == "2" {
				page = 2
			}
		}

		start := (page - 1) * pageSize
		end := start + pageSize
		if end > len(allSchemas) {
			end = len(allSchemas)
		}

		schemas := allSchemas[start:end]
		hasNext := end < len(allSchemas)

		response := CreateTestSchemaResponse(schemas, page, hasNext)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err := json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
	}
}

// CreateErrorMockHandler creates a mock handler that returns an error
func CreateErrorMockHandler(statusCode int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(message))
	}
}

// CreateTimeoutMockHandler creates a mock handler that simulates timeouts
func CreateTimeoutMockHandler(delay time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(http.StatusOK)
	}
}

// SetupTempDirWithFiles creates a temporary directory with specified files
func SetupTempDirWithFiles(t *testing.T, files map[string]interface{}) string {
	tempDir := t.TempDir()

	for filename, content := range files {
		fullPath := filepath.Join(tempDir, filename)

		// Create directory if needed
		if dir := filepath.Dir(fullPath); dir != tempDir {
			err := os.MkdirAll(dir, 0755)
			require.NoError(t, err)
		}

		WriteTestFile(t, fullPath, content)
	}

	return tempDir
}

// CreateMinimalTestSchema creates a minimal schema for basic testing
func CreateMinimalTestSchema(uid string) models.Schema {
	return models.Schema{
		UID:             uid,
		WriteKey:        "test-writekey",
		EventType:       "track",
		EventIdentifier: "minimal_test_event",
		Schema: map[string]interface{}{
			"event":  "string",
			"userId": "string",
		},
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		Count:     1,
	}
}

// CreateComplexTestSchema creates a schema with nested structures for testing
func CreateComplexTestSchema(uid, writeKey, eventIdentifier string) models.Schema {
	return models.Schema{
		UID:             uid,
		WriteKey:        writeKey,
		EventType:       "track",
		EventIdentifier: eventIdentifier,
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
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		Count:     10,
	}
}

// CreateArraySchema creates a schema with array structures for testing
func CreateArraySchema() models.Schema {
	return models.Schema{
		UID:             "array-test-uid",
		WriteKey:        "test-writekey-array",
		EventType:       "track",
		EventIdentifier: "array_test_event",
		Schema: map[string]interface{}{
			"event": "string",
			"items": []interface{}{
				map[string]interface{}{
					"name":  "string",
					"price": "number",
				},
			},
			"tags": []interface{}{"string"},
		},
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		Count:     5,
	}
}

// CreateFlattenedTestSchema creates a schema with flattened dot-notation keys
func CreateFlattenedTestSchema(uid, writeKey, eventIdentifier string) models.Schema {
	return models.Schema{
		UID:             uid,
		WriteKey:        writeKey,
		EventType:       "track",
		EventIdentifier: eventIdentifier,
		Schema: map[string]interface{}{
			"event":                   "string",
			"userId":                  "string",
			"context.app.name":        "string",
			"context.device.type":     "string",
			"properties.product_id":   "string",
			"properties.product_name": "string",
			"properties.price":        "number",
			"properties.categories.0": "string",
			"properties.categories.1": "string",
		},
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		Count:     100,
	}
}
