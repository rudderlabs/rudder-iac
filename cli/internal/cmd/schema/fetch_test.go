package schema

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupViperForTests initializes viper with environment variable bindings for tests
func setupViperForTests() {
	viper.Reset()
	viper.BindEnv("auth.accessToken", "RUDDERSTACK_ACCESS_TOKEN")
	viper.BindEnv("apiURL", "RUDDERSTACK_API_URL")
}

func TestFetchCommand_Integration(t *testing.T) {
	t.Parallel()

	// Initialize viper for test
	setupViperForTests()

	// Create test response
	testSchemas := []models.Schema{
		{
			UID:             "test-uid-1",
			WriteKey:        "test-write-key",
			EventType:       "track",
			EventIdentifier: "test-event-1",
			Schema: map[string]interface{}{
				"event":      "string",
				"userId":     "string",
				"context.ip": "string",
			},
			CreatedAt: time.Now(),
			LastSeen:  time.Now(),
			Count:     10,
		},
		{
			UID:             "test-uid-2",
			WriteKey:        "test-write-key-2",
			EventType:       "identify",
			EventIdentifier: "test-event-2",
			Schema: map[string]interface{}{
				"userId":       "string",
				"traits.email": "string",
			},
			CreatedAt: time.Now(),
			LastSeen:  time.Now(),
			Count:     5,
		},
	}

	testResponse := models.SchemasResponse{
		Results:     testSchemas,
		CurrentPage: 1,
		HasNext:     false,
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "/v2/schemas", r.URL.Path)

		// Return test response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testResponse)
	}))
	defer server.Close()

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

	// Create temporary output file
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "test_output.json")

	// Run the command
	err := runFetch(outputFile, "", false, false, 2)
	require.NoError(t, err)

	// Verify output file was created
	assert.FileExists(t, outputFile)

	// Read and verify output file content
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var output models.SchemasFile
	err = json.Unmarshal(content, &output)
	require.NoError(t, err)

	assert.Len(t, output.Schemas, 2)
	assert.Equal(t, "test-uid-1", output.Schemas[0].UID)
	assert.Equal(t, "test-event-1", output.Schemas[0].EventIdentifier)
	assert.Equal(t, "test-uid-2", output.Schemas[1].UID)
	assert.Equal(t, "test-event-2", output.Schemas[1].EventIdentifier)
}

func TestFetchCommand_Scenarios(t *testing.T) {
	// Initialize viper for test
	setupViperForTests()

	cases := []struct {
		name        string
		writeKey    string
		dryRun      bool
		verbose     bool
		expectFiles bool
		expectError string
	}{
		{
			name:        "WithWriteKey",
			writeKey:    "specific-write-key",
			dryRun:      false,
			verbose:     false,
			expectFiles: true,
			expectError: "",
		},
		{
			name:        "DryRun",
			writeKey:    "",
			dryRun:      true,
			verbose:     false,
			expectFiles: false,
			expectError: "",
		},
		{
			name:        "VerboseMode",
			writeKey:    "",
			dryRun:      false,
			verbose:     true,
			expectFiles: true,
			expectError: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Remove t.Parallel() to avoid test contamination
			// t.Parallel()

			// Initialize viper for subtest
			setupViperForTests()

			// Create test server that verifies writeKey parameter if specified
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check that the writeKey parameter is passed correctly
				writeKeyParam := r.URL.Query().Get("writeKey")
				if c.writeKey != "" {
					assert.Equal(t, c.writeKey, writeKeyParam, "Expected writeKey parameter to match for test: %s", c.name)
				} else {
					assert.Empty(t, writeKeyParam, "Expected no writeKey parameter when not specified for test: %s", c.name)
				}

				// Only set WriteKey if one was actually provided
				var responseWriteKey string
				if writeKeyParam != "" {
					responseWriteKey = writeKeyParam
				}

				response := models.SchemasResponse{
					Results: []models.Schema{
						{UID: "test-uid", EventIdentifier: "test-event", WriteKey: responseWriteKey},
					},
					CurrentPage: 1,
					HasNext:     false,
				}
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

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

			tempDir := t.TempDir()
			outputFile := filepath.Join(tempDir, "test_output.json")

			err := runFetch(outputFile, c.writeKey, c.dryRun, c.verbose, 2)

			if c.expectError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), c.expectError)
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

func TestFetchCommand_ErrorScenarios(t *testing.T) {
	// Initialize viper for test
	setupViperForTests()

	cases := []struct {
		name        string
		setupEnv    func()
		expectError string
		useServer   bool
	}{
		{
			name: "MissingAPIToken",
			setupEnv: func() {
				os.Unsetenv("RUDDERSTACK_ACCESS_TOKEN")
				os.Setenv("RUDDERSTACK_API_URL", "https://example.com")
			},
			expectError: "access token is required",
			useServer:   false,
		},
		{
			name: "MissingAPIURL_UsesDefault",
			setupEnv: func() {
				os.Setenv("RUDDERSTACK_ACCESS_TOKEN", "test-token")
				os.Unsetenv("RUDDERSTACK_API_URL")
			},
			expectError: "", // Should pass with default URL
			useServer:   true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Remove t.Parallel() to avoid test contamination
			// t.Parallel()

			// Initialize viper for subtest
			setupViperForTests()

			var server *httptest.Server
			if c.useServer {
				// Create a mock server for tests that should succeed
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					response := models.SchemasResponse{
						Results: []models.Schema{
							{UID: "test-uid", EventIdentifier: "test-event", WriteKey: ""},
						},
						CurrentPage: 1,
						HasNext:     false,
					}
					json.NewEncoder(w).Encode(response)
				}))
				defer server.Close()
			}

			// Store original values
			originalToken := os.Getenv("RUDDERSTACK_ACCESS_TOKEN")
			originalURL := os.Getenv("RUDDERSTACK_API_URL")

			// Setup test environment
			c.setupEnv()

			// If using server, override the URL after setupEnv
			if c.useServer {
				os.Setenv("RUDDERSTACK_API_URL", server.URL)
			}

			// Restore environment after test
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

			tempDir := t.TempDir()
			outputFile := filepath.Join(tempDir, "test_output.json")

			err := runFetch(outputFile, "", false, false, 2)

			if c.expectError != "" {
				assert.Error(t, err, "Expected an error for test case: %s", c.name)
				if err != nil {
					assert.Contains(t, err.Error(), c.expectError)
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", c.name)
			}
		})
	}
}

func TestNewCmdFetch(t *testing.T) {
	t.Parallel()

	t.Run("CommandCreation", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdFetch()

		assert.NotNil(t, cmd)
		assert.Equal(t, "fetch", cmd.Name())
		assert.Equal(t, "Fetch event schemas from the API", cmd.Short)
		assert.Contains(t, cmd.Long, "Fetch event schemas from the Event Audit API")

		// Check that flags are properly set
		writeKeyFlag := cmd.Flags().Lookup("write-key")
		assert.NotNil(t, writeKeyFlag)
		assert.Equal(t, "", writeKeyFlag.DefValue)

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
}

func TestFetchCommand_EnhancedScenarios(t *testing.T) {
	// Initialize viper for test
	setupViperForTests()

	cases := []struct {
		name             string
		writeKey         string
		dryRun           bool
		verbose          bool
		expectFiles      bool
		expectError      string
		serverResponse   models.SchemasResponse
		serverStatusCode int
	}{
		{
			name:        "LargeDataset",
			writeKey:    "",
			dryRun:      false,
			verbose:     true,
			expectFiles: true,
			expectError: "",
			serverResponse: models.SchemasResponse{
				Results:     generateLargeSchemaSet(100),
				CurrentPage: 1,
				HasNext:     false,
			},
			serverStatusCode: 200,
		},
		{
			name:        "EmptyResponse",
			writeKey:    "",
			dryRun:      false,
			verbose:     false,
			expectFiles: true,
			expectError: "",
			serverResponse: models.SchemasResponse{
				Results:     []models.Schema{},
				CurrentPage: 1,
				HasNext:     false,
			},
			serverStatusCode: 200,
		},
		{
			name:             "ServerError500",
			writeKey:         "",
			dryRun:           false,
			verbose:          false,
			expectFiles:      false,
			expectError:      "failed to fetch schemas",
			serverResponse:   models.SchemasResponse{},
			serverStatusCode: 500,
		},
		{
			name:        "VerboseDryRunLargeDataset",
			writeKey:    "specific-key",
			dryRun:      true,
			verbose:     true,
			expectFiles: false,
			expectError: "",
			serverResponse: models.SchemasResponse{
				Results:     generateLargeSchemaSet(50),
				CurrentPage: 1,
				HasNext:     false,
			},
			serverStatusCode: 200,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Initialize viper for subtest
			setupViperForTests()

			// Create test server with custom response
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.serverStatusCode)
				if c.serverStatusCode == 200 {
					json.NewEncoder(w).Encode(c.serverResponse)
				} else {
					w.Write([]byte("Internal Server Error"))
				}
			}))
			defer server.Close()

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

			tempDir := t.TempDir()
			outputFile := filepath.Join(tempDir, "test_output.json")

			err := runFetch(outputFile, c.writeKey, c.dryRun, c.verbose, 2)

			if c.expectError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), c.expectError)
			} else {
				assert.NoError(t, err)
			}

			if c.expectFiles && c.expectError == "" {
				if !c.dryRun {
					assert.FileExists(t, outputFile)
				} else {
					// Dry run should not create files
					_, err := os.Stat(outputFile)
					assert.True(t, os.IsNotExist(err))
				}
			}
		})
	}
}

// Helper function to generate a large set of schemas for testing
func generateLargeSchemaSet(count int) []models.Schema {
	schemas := make([]models.Schema, count)
	for i := 0; i < count; i++ {
		schemas[i] = models.Schema{
			UID:             fmt.Sprintf("test-uid-%d", i),
			WriteKey:        fmt.Sprintf("write-key-%d", i%5), // Distribute across 5 write keys
			EventType:       "track",
			EventIdentifier: fmt.Sprintf("event_%d", i),
			Schema: map[string]interface{}{
				"userId": "string",
				"event":  "string",
				"properties": map[string]interface{}{
					fmt.Sprintf("prop_%d", i): "string",
				},
			},
			CreatedAt: time.Now(),
			LastSeen:  time.Now(),
			Count:     i + 1,
		}
	}
	return schemas
}

func TestWriteJSONFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "test.json")

	testData := models.SchemasFile{
		Schemas: []models.Schema{
			{
				UID:             "test-uid",
				EventIdentifier: "test-event",
				Schema:          map[string]interface{}{"test": "value"},
			},
		},
	}

	err := writeJSONFile(outputFile, testData, 2)
	require.NoError(t, err)

	// Verify file exists and contains correct data
	assert.FileExists(t, outputFile)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var result models.SchemasFile
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	assert.Len(t, result.Schemas, 1)
	assert.Equal(t, "test-uid", result.Schemas[0].UID)
}
