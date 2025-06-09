package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/interfaces"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	pkgModels "github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestFetchService_FetchSchemas(t *testing.T) {
	tests := []struct {
		name          string
		opts          interfaces.FetchOptions
		mockResponse  pkgModels.SchemasResponse
		expectedCount int
		expectError   bool
		errorContains string
	}{
		{
			name: "successful_fetch_single_page",
			opts: interfaces.FetchOptions{
				WriteKey: "test-key",
				DryRun:   false,
				Verbose:  true,
				PageSize: 10,
				Timeout:  5 * time.Second,
			},
			mockResponse: pkgModels.SchemasResponse{
				Results: []pkgModels.Schema{
					{
						UID:             "schema-1",
						WriteKey:        "test-key",
						EventType:       "track",
						EventIdentifier: "test_event",
						Schema:          map[string]interface{}{"prop1": "string"},
						CreatedAt:       mustParseTime("2023-01-01T00:00:00Z"),
						LastSeen:        mustParseTime("2023-01-01T00:00:00Z"),
						Count:           100,
					},
					{
						UID:             "schema-2",
						WriteKey:        "test-key",
						EventType:       "track",
						EventIdentifier: "another_event",
						Schema:          map[string]interface{}{"prop2": "number"},
						CreatedAt:       mustParseTime("2023-01-01T00:00:00Z"),
						LastSeen:        mustParseTime("2023-01-01T00:00:00Z"),
						Count:           50,
					},
				},
				HasNext: false,
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "dry_run_mode",
			opts: interfaces.FetchOptions{
				WriteKey: "test-key",
				DryRun:   true,
				Verbose:  false,
				PageSize: 10,
				Timeout:  5 * time.Second,
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "empty_response",
			opts: interfaces.FetchOptions{
				WriteKey: "",
				DryRun:   false,
				Verbose:  false,
				PageSize: 10,
				Timeout:  5 * time.Second,
			},
			mockResponse: pkgModels.SchemasResponse{
				Results: []pkgModels.Schema{},
				HasNext: false,
			},
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request path and method
				assert.Equal(t, "GET", r.Method)
				assert.Contains(t, r.URL.Path, "v2/schemas")

				// Check query parameters
				if tt.opts.WriteKey != "" {
					assert.Equal(t, tt.opts.WriteKey, r.URL.Query().Get("writeKey"))
				}

				// Return mock response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer server.Close()

			// Create client pointing to mock server
			apiClient, err := client.New("test-token", client.WithBaseURL(server.URL))
			require.NoError(t, err)

			// Create logger
			log := logger.New("test")

			// Create service
			service := NewFetchService(apiClient, log)

			// Execute test
			ctx := context.Background()
			result, err := service.FetchSchemas(ctx, tt.opts)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedCount, len(result.Schemas))

			// Verify schema content for non-empty responses
			if tt.expectedCount > 0 && !tt.opts.DryRun {
				schema := result.Schemas[0]
				assert.Equal(t, tt.mockResponse.Results[0].UID, schema.UID)
				assert.Equal(t, tt.mockResponse.Results[0].WriteKey, schema.WriteKey)
				assert.Equal(t, tt.mockResponse.Results[0].EventType, schema.EventType)
				assert.Equal(t, tt.mockResponse.Results[0].EventIdentifier, schema.EventIdentifier)
			}

			// Verify stats
			stats := service.GetFetchStats()
			assert.GreaterOrEqual(t, stats.Duration, time.Duration(0))
			if !tt.opts.DryRun {
				assert.Equal(t, tt.expectedCount, stats.TotalSchemas)
			}
		})
	}
}

func TestFetchService_ValidateConnection(t *testing.T) {
	tests := []struct {
		name        string
		serverFunc  func(w http.ResponseWriter, r *http.Request)
		expectError bool
	}{
		{
			name: "successful_connection",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(pkgModels.SchemasResponse{
					Results: []pkgModels.Schema{},
					HasNext: false,
				})
			},
			expectError: false,
		},
		{
			name: "connection_failure",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(tt.serverFunc))
			defer server.Close()

			// Create client pointing to mock server
			apiClient, err := client.New("test-token", client.WithBaseURL(server.URL))
			require.NoError(t, err)

			// Create logger
			log := logger.New("test")

			// Create service
			service := NewFetchService(apiClient, log)

			// Execute test
			ctx := context.Background()
			err = service.ValidateConnection(ctx)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "API connection validation failed")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFetchService_GetFetchStats(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pkgModels.SchemasResponse{
			Results: []pkgModels.Schema{
				{UID: "test-schema", WriteKey: "test-key"},
			},
			HasNext: false,
		})
	}))
	defer server.Close()

	// Create client pointing to mock server
	apiClient, err := client.New("test-token", client.WithBaseURL(server.URL))
	require.NoError(t, err)

	// Create logger
	log := logger.New("test")

	// Create service
	service := NewFetchService(apiClient, log)

	// Initial stats should be empty
	stats := service.GetFetchStats()
	assert.Equal(t, 0, stats.TotalSchemas)
	assert.Equal(t, 0, stats.PagesProcessed)
	assert.Equal(t, 0, stats.ErrorCount)

	// Perform a fetch operation
	ctx := context.Background()
	opts := interfaces.FetchOptions{
		WriteKey: "test-key",
		DryRun:   false,
		Verbose:  false,
		PageSize: 10,
		Timeout:  5 * time.Second,
	}

	result, err := service.FetchSchemas(ctx, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Stats should be updated
	stats = service.GetFetchStats()
	assert.Equal(t, 1, stats.TotalSchemas)
	assert.Equal(t, 1, stats.PagesProcessed)
	assert.Equal(t, 0, stats.ErrorCount)
	assert.Greater(t, stats.Duration, time.Duration(0))
}
