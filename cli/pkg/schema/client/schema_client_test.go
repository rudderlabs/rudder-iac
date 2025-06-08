package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
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

	client := NewSchemaClient("https://api.example.com", "test-token")
	assert.NotNil(t, client)
	assert.Equal(t, "https://api.example.com", client.baseURL)
	assert.Equal(t, "test-token", client.apiToken)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
}

// TestSchemaClient_SuccessfulRequestCreation tests successful API request creation
// This covers lines 60-61 in schema_client.go (fetchSchemasPage function - request creation)
func TestSchemaClient_SuccessfulRequestCreation(t *testing.T) {
	t.Parallel()

	t.Run("SuccessfulRequestWithParameters", func(t *testing.T) {
		t.Parallel()

		requestReceived := false
		var receivedRequest *http.Request

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestReceived = true
			receivedRequest = r

			// Return successful response
			response := models.SchemasResponse{
				Results: []models.Schema{
					{
						UID:             "success-uid",
						WriteKey:        "success-writekey",
						EventType:       "track",
						EventIdentifier: "success_event",
						Schema: map[string]interface{}{
							"event":  "string",
							"userId": "string",
							"properties": map[string]interface{}{
								"test": "string",
							},
						},
						Count: 10,
					},
				},
				CurrentPage: 3,
				HasNext:     true,
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewSchemaClient(server.URL, "success-token")
		schemas, hasNext, err := client.fetchSchemasPage(3, "success-writekey")

		// Verify successful request creation and processing
		require.NoError(t, err)
		assert.True(t, requestReceived, "Request should have been received")
		assert.Len(t, schemas, 1)
		assert.True(t, hasNext)

		// Verify request was created correctly (line 60-61)
		require.NotNil(t, receivedRequest)
		assert.Equal(t, "GET", receivedRequest.Method)
		assert.Equal(t, "/v2/schemas", receivedRequest.URL.Path)
		assert.Equal(t, "Bearer success-token", receivedRequest.Header.Get("Authorization"))
		assert.Equal(t, "application/json", receivedRequest.Header.Get("Content-Type"))

		// Verify query parameters
		assert.Equal(t, "3", receivedRequest.URL.Query().Get("page"))
		assert.Equal(t, "success-writekey", receivedRequest.URL.Query().Get("writeKey"))

		// Verify response data
		assert.Equal(t, "success-uid", schemas[0].UID)
		assert.Equal(t, "success-writekey", schemas[0].WriteKey)
		assert.Equal(t, "success_event", schemas[0].EventIdentifier)
		assert.Equal(t, 10, schemas[0].Count)
	})

	t.Run("SuccessfulRequestWithoutWriteKey", func(t *testing.T) {
		t.Parallel()

		var receivedRequest *http.Request

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedRequest = r

			response := models.SchemasResponse{
				Results: []models.Schema{
					{UID: "no-writekey-uid", EventIdentifier: "no_writekey_event"},
				},
				CurrentPage: 1,
				HasNext:     false,
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewSchemaClient(server.URL, "test-token")
		schemas, hasNext, err := client.fetchSchemasPage(1, "")

		require.NoError(t, err)
		assert.Len(t, schemas, 1)
		assert.False(t, hasNext)

		// Verify request creation without writeKey parameter
		require.NotNil(t, receivedRequest)
		assert.Equal(t, "1", receivedRequest.URL.Query().Get("page"))
		assert.Empty(t, receivedRequest.URL.Query().Get("writeKey"))
	})

	t.Run("SuccessfulComplexURL", func(t *testing.T) {
		t.Parallel()

		var receivedRequest *http.Request

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedRequest = r

			response := models.SchemasResponse{
				Results:     []models.Schema{},
				CurrentPage: 1,
				HasNext:     false,
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		// Test with complex base URL
		client := NewSchemaClient(server.URL+"/api", "complex-token")
		_, _, err := client.fetchSchemasPage(1, "complex-writekey")

		require.NoError(t, err)
		require.NotNil(t, receivedRequest)
		assert.Equal(t, "/api/v2/schemas", receivedRequest.URL.Path)
		assert.Equal(t, "Bearer complex-token", receivedRequest.Header.Get("Authorization"))
	})
}

// TestSchemaClient_SuccessfulResponseProcessing tests successful HTTP response processing
// This covers lines 73-74 in schema_client.go (fetchSchemasPage function - response processing)
func TestSchemaClient_SuccessfulResponseProcessing(t *testing.T) {
	t.Parallel()

	t.Run("SuccessfulCompleteResponseParsing", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return comprehensive response data to test parsing
			response := models.SchemasResponse{
				Results: []models.Schema{
					{
						UID:             "complete-uid-1",
						WriteKey:        "complete-writekey",
						EventType:       "track",
						EventIdentifier: "complete_event_1",
						Schema: map[string]interface{}{
							"event":       "string",
							"userId":      "string",
							"anonymousId": "string",
							"properties": map[string]interface{}{
								"product_id":   "string",
								"product_name": "string",
								"price":        "number",
								"categories":   []interface{}{"string"},
								"metadata": map[string]interface{}{
									"source":   "web",
									"campaign": "summer2024",
								},
							},
							"context": map[string]interface{}{
								"app": map[string]interface{}{
									"name":    "MyApp",
									"version": "1.2.3",
								},
								"library": map[string]interface{}{
									"name":    "analytics-js",
									"version": "4.0.0",
								},
							},
						},
						CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
						LastSeen:  time.Date(2024, 3, 20, 15, 45, 30, 0, time.UTC),
						Count:     42,
					},
					{
						UID:             "complete-uid-2",
						WriteKey:        "complete-writekey",
						EventType:       "identify",
						EventIdentifier: "complete_event_2",
						Schema: map[string]interface{}{
							"userId": "string",
							"traits": map[string]interface{}{
								"email":      "string",
								"name":       "string",
								"age":        "number",
								"premium":    "boolean",
								"signupDate": "string",
							},
						},
						CreatedAt: time.Date(2024, 2, 1, 8, 15, 0, 0, time.UTC),
						LastSeen:  time.Date(2024, 3, 19, 12, 30, 0, 0, time.UTC),
						Count:     18,
					},
				},
				CurrentPage: 2,
				HasNext:     true,
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewSchemaClient(server.URL, "complete-token")
		schemas, hasNext, err := client.fetchSchemasPage(2, "complete-writekey")

		// Verify successful response processing (lines 73-74)
		require.NoError(t, err)
		assert.True(t, hasNext)
		require.Len(t, schemas, 2)

		// Verify first schema was parsed correctly
		schema1 := schemas[0]
		assert.Equal(t, "complete-uid-1", schema1.UID)
		assert.Equal(t, "complete-writekey", schema1.WriteKey)
		assert.Equal(t, "track", schema1.EventType)
		assert.Equal(t, "complete_event_1", schema1.EventIdentifier)
		assert.Equal(t, 42, schema1.Count)
		assert.Equal(t, time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), schema1.CreatedAt)
		assert.Equal(t, time.Date(2024, 3, 20, 15, 45, 30, 0, time.UTC), schema1.LastSeen)

		// Verify schema structure parsing
		require.NotNil(t, schema1.Schema)
		assert.Equal(t, "string", schema1.Schema["event"])
		assert.Equal(t, "string", schema1.Schema["userId"])

		properties, ok := schema1.Schema["properties"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "string", properties["product_id"])
		assert.Equal(t, "string", properties["product_name"])
		assert.Equal(t, "number", properties["price"])

		metadata, ok := properties["metadata"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "web", metadata["source"])
		assert.Equal(t, "summer2024", metadata["campaign"])

		context, ok := schema1.Schema["context"].(map[string]interface{})
		require.True(t, ok)

		app, ok := context["app"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "MyApp", app["name"])
		assert.Equal(t, "1.2.3", app["version"])

		// Verify second schema
		schema2 := schemas[1]
		assert.Equal(t, "complete-uid-2", schema2.UID)
		assert.Equal(t, "identify", schema2.EventType)
		assert.Equal(t, "complete_event_2", schema2.EventIdentifier)
		assert.Equal(t, 18, schema2.Count)

		traits, ok := schema2.Schema["traits"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "string", traits["email"])
		assert.Equal(t, "string", traits["name"])
		assert.Equal(t, "number", traits["age"])
		assert.Equal(t, "boolean", traits["premium"])
	})

	t.Run("SuccessfulEmptyResponse", func(t *testing.T) {
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

		client := NewSchemaClient(server.URL, "empty-token")
		schemas, hasNext, err := client.fetchSchemasPage(1, "")

		require.NoError(t, err)
		assert.False(t, hasNext)
		assert.Len(t, schemas, 0)
	})

	t.Run("SuccessfulPaginationResponse", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			page := r.URL.Query().Get("page")

			var response models.SchemasResponse
			if page == "1" {
				response = models.SchemasResponse{
					Results: []models.Schema{
						{UID: "page1-schema1", EventIdentifier: "page1_event1"},
						{UID: "page1-schema2", EventIdentifier: "page1_event2"},
					},
					CurrentPage: 1,
					HasNext:     true,
				}
			} else if page == "2" {
				response = models.SchemasResponse{
					Results: []models.Schema{
						{UID: "page2-schema1", EventIdentifier: "page2_event1"},
					},
					CurrentPage: 2,
					HasNext:     false,
				}
			}

			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewSchemaClient(server.URL, "pagination-token")

		// Test first page
		schemas1, hasNext1, err1 := client.fetchSchemasPage(1, "")
		require.NoError(t, err1)
		assert.True(t, hasNext1)
		assert.Len(t, schemas1, 2)
		assert.Equal(t, "page1-schema1", schemas1[0].UID)
		assert.Equal(t, "page1-schema2", schemas1[1].UID)

		// Test second page
		schemas2, hasNext2, err2 := client.fetchSchemasPage(2, "")
		require.NoError(t, err2)
		assert.False(t, hasNext2)
		assert.Len(t, schemas2, 1)
		assert.Equal(t, "page2-schema1", schemas2[0].UID)
	})

	t.Run("SuccessfulLargeResponseProcessing", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a larger response to test processing capabilities
			schemas := make([]models.Schema, 50)
			for i := 0; i < 50; i++ {
				schemas[i] = models.Schema{
					UID:             fmt.Sprintf("large-uid-%d", i),
					WriteKey:        "large-writekey",
					EventType:       "track",
					EventIdentifier: fmt.Sprintf("large_event_%d", i),
					Schema: map[string]interface{}{
						"event":  "string",
						"userId": "string",
						"properties": map[string]interface{}{
							"item_id":    "string",
							"item_index": fmt.Sprintf("number_%d", i),
						},
					},
					Count: i + 1,
				}
			}

			response := models.SchemasResponse{
				Results:     schemas,
				CurrentPage: 1,
				HasNext:     false,
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewSchemaClient(server.URL, "large-token")
		schemas, hasNext, err := client.fetchSchemasPage(1, "large-writekey")

		require.NoError(t, err)
		assert.False(t, hasNext)
		assert.Len(t, schemas, 50)

		// Verify some schemas were processed correctly
		assert.Equal(t, "large-uid-0", schemas[0].UID)
		assert.Equal(t, "large_event_0", schemas[0].EventIdentifier)
		assert.Equal(t, 1, schemas[0].Count)

		assert.Equal(t, "large-uid-49", schemas[49].UID)
		assert.Equal(t, "large_event_49", schemas[49].EventIdentifier)
		assert.Equal(t, 50, schemas[49].Count)
	})
}
