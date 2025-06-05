package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/experimental/schema/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaClient_FetchAllSchemas(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		writeKey      string
		serverHandler func(w http.ResponseWriter, r *http.Request)
		expectError   bool
		expectedCount int
	}{
		{
			name:     "SinglePageResponse",
			writeKey: "",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				assert.Equal(t, "/v2/schemas", r.URL.Path)

				response := models.SchemasResponse{
					Results: []models.Schema{
						{UID: "schema-1", EventIdentifier: "event-1"},
						{UID: "schema-2", EventIdentifier: "event-2"},
					},
					CurrentPage: 1,
					HasNext:     false,
				}
				json.NewEncoder(w).Encode(response)
			},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:     "WithWriteKeyFilter",
			writeKey: "test-writekey",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "test-writekey", r.URL.Query().Get("writeKey"))
				assert.Equal(t, "1", r.URL.Query().Get("page"))

				response := models.SchemasResponse{
					Results: []models.Schema{
						{UID: "schema-1", WriteKey: "test-writekey", EventIdentifier: "event-1"},
					},
					CurrentPage: 1,
					HasNext:     false,
				}
				json.NewEncoder(w).Encode(response)
			},
			expectError:   false,
			expectedCount: 1,
		},
		{
			name:     "MultiplePages",
			writeKey: "",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				page := r.URL.Query().Get("page")

				if page == "1" || page == "" {
					response := models.SchemasResponse{
						Results: []models.Schema{
							{UID: "schema-1", EventIdentifier: "event-1"},
							{UID: "schema-2", EventIdentifier: "event-2"},
						},
						CurrentPage: 1,
						HasNext:     true,
					}
					json.NewEncoder(w).Encode(response)
				} else if page == "2" {
					response := models.SchemasResponse{
						Results: []models.Schema{
							{UID: "schema-3", EventIdentifier: "event-3"},
						},
						CurrentPage: 2,
						HasNext:     false,
					}
					json.NewEncoder(w).Encode(response)
				}
			},
			expectError:   false,
			expectedCount: 3,
		},
		{
			name:     "ServerError",
			writeKey: "",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			},
			expectError:   true,
			expectedCount: 0,
		},
		{
			name:     "InvalidJSON",
			writeKey: "",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte("invalid json"))
			},
			expectError:   true,
			expectedCount: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(c.serverHandler))
			defer server.Close()

			client := NewSchemaClient(server.URL, "test-token")
			schemas, err := client.FetchAllSchemas(c.writeKey)

			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, schemas, c.expectedCount)
			}
		})
	}
}

func TestSchemaClient_FetchSchemasPage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "/v2/schemas", r.URL.Path)

		// Verify query parameters
		assert.Equal(t, "2", r.URL.Query().Get("page"))
		assert.Equal(t, "filter-writekey", r.URL.Query().Get("writeKey"))

		response := models.SchemasResponse{
			Results: []models.Schema{
				{
					UID:             "test-uid",
					WriteKey:        "filter-writekey",
					EventType:       "track",
					EventIdentifier: "test-event",
					Schema: map[string]interface{}{
						"event":  "string",
						"userId": "string",
					},
					CreatedAt: time.Now(),
					LastSeen:  time.Now(),
					Count:     5,
				},
			},
			CurrentPage: 2,
			HasNext:     false,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewSchemaClient(server.URL, "test-token")
	schemas, hasNext, err := client.fetchSchemasPage(2, "filter-writekey")

	require.NoError(t, err)
	assert.Len(t, schemas, 1)
	assert.False(t, hasNext)
	assert.Equal(t, "test-uid", schemas[0].UID)
	assert.Equal(t, "filter-writekey", schemas[0].WriteKey)
	assert.Equal(t, "test-event", schemas[0].EventIdentifier)
}

func TestSchemaClient_NetworkTimeout(t *testing.T) {
	t.Parallel()

	// Create a server that will delay response beyond client timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Client timeout is 30 seconds, so this should work
		response := models.SchemasResponse{
			Results:     []models.Schema{},
			CurrentPage: 1,
			HasNext:     false,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Test with very short timeout by creating custom client
	client := &SchemaClient{
		baseURL:    server.URL,
		apiToken:   "test-token",
		httpClient: &http.Client{Timeout: 1 * time.Millisecond}, // Very short timeout
	}

	_, err := client.FetchAllSchemas("")
	assert.Error(t, err)
	// Check for the actual timeout error message
	assert.Contains(t, err.Error(), "deadline exceeded")
}

func TestSchemaClient_InvalidURL(t *testing.T) {
	t.Parallel()

	client := NewSchemaClient("invalid-url", "test-token")
	_, err := client.FetchAllSchemas("")
	assert.Error(t, err)
}

func TestSchemaClient_EmptyResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := models.SchemasResponse{
			Results:     []models.Schema{},
			CurrentPage: 1,
			HasNext:     false,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewSchemaClient(server.URL, "test-token")
	schemas, err := client.FetchAllSchemas("")

	require.NoError(t, err)
	assert.Len(t, schemas, 0)
}

func TestNewSchemaClient(t *testing.T) {
	t.Parallel()

	baseURL := "https://api.example.com"
	apiToken := "test-token"

	client := NewSchemaClient(baseURL, apiToken)

	assert.Equal(t, baseURL, client.baseURL)
	assert.Equal(t, apiToken, client.apiToken)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
}
