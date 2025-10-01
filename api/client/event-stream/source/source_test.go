package source_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	esSource "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSource(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"externalId":"ext-123","name":"Test Source","type":"webhook","enabled":true}`
			return testutils.ValidateRequest(t, req, "POST", "/v2/event-stream-sources", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "src-123",
			"externalId": "ext-123",
			"name": "Test Source",
			"type": "webhook",
			"enabled": true
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	eventStreamClient := esSource.NewRudderSourceStore(c)

	source := &esSource.CreateSourceRequest{
		ExternalID: "ext-123",
		Name:       "Test Source",
		Type:       "webhook",
		Enabled:    true,
	}

	created, err := eventStreamClient.Create(context.Background(), source)
	require.NoError(t, err)

	assert.Equal(t, &esSource.EventStreamSource{
		ID:         "src-123",
		ExternalID: "ext-123",
		Name:       "Test Source",
		Type:       "webhook",
		Enabled:    true,
	}, created)
	httpClient.AssertNumberOfCalls()
}

func TestUpdateSource(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"Updated Source"}`
			return testutils.ValidateRequest(t, req, "PUT", "/v2/event-stream-sources/src-123", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "src-123",
			"externalId": "ext-123",
			"name": "Updated Source",
			"type": "webhook",
			"enabled": true
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	eventStreamClient := esSource.NewRudderSourceStore(c)

	source := &esSource.UpdateSourceRequest{
		Name: "Updated Source",
	}

	updated, err := eventStreamClient.Update(context.Background(), "src-123", source)
	require.NoError(t, err)

	assert.Equal(t, &esSource.EventStreamSource{
		ID:         "src-123",
		ExternalID: "ext-123",
		Name:       "Updated Source",
		Type:       "webhook",
		Enabled:    true,
	}, updated)

	httpClient.AssertNumberOfCalls()
}

func TestDeleteSource(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "DELETE", "/v2/event-stream-sources/src-123", "")
		},
		ResponseStatus: 204,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	eventStreamClient := esSource.NewRudderSourceStore(c)

	err = eventStreamClient.Delete(context.Background(), "src-123")
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestGetSources(t *testing.T) {
	tests := []struct {
		name           string
		calls          []testutils.Call
		expectedSources []esSource.EventStreamSource
	}{
		{
			name: "single page",
			calls: []testutils.Call{
				{
					Validate: func(req *http.Request) bool {
						return testutils.ValidateRequest(t, req, "GET", "/v2/event-stream-sources", "")
					},
					ResponseStatus: 200,
					ResponseBody: `{
						"data": [
							{
								"id": "src-123",
								"externalId": "ext-123",
								"name": "Source 1",
								"type": "webhook",
								"enabled": true
							},
							{
								"id": "src-456",
								"externalId": "ext-456",
								"name": "Source 2",
								"type": "api",
								"enabled": false
							}
						],
						"paging": {
							"total": 2,
							"next": ""
						}
					}`,
				},
			},
			expectedSources: []esSource.EventStreamSource{
				{
					ID:         "src-123",
					ExternalID: "ext-123",
					Name:       "Source 1",
					Type:       "webhook",
					Enabled:    true,
				},
				{
					ID:         "src-456",
					ExternalID: "ext-456",
					Name:       "Source 2",
					Type:       "api",
					Enabled:    false,
				},
			},
		},
		{
			name: "multiple pages",
			calls: []testutils.Call{
				{
					Validate: func(req *http.Request) bool {
						return testutils.ValidateRequest(t, req, "GET", "/v2/event-stream-sources", "")
					},
					ResponseStatus: 200,
					ResponseBody: `{
						"data": [
							{
								"id": "src-123",
								"externalId": "ext-123",
								"name": "Source 1",
								"type": "webhook",
								"enabled": true
							}
						],
						"paging": {
							"total": 3,
							"next": "/v2/event-stream-sources?page=2"
						}
					}`,
				},
				{
					Validate: func(req *http.Request) bool {
						return testutils.ValidateRequest(t, req, "GET", "/v2/event-stream-sources?page=2", "")
					},
					ResponseStatus: 200,
					ResponseBody: `{
						"data": [
							{
								"id": "src-456",
								"externalId": "ext-456",
								"name": "Source 2",
								"type": "api",
								"enabled": false
							}
						],
						"paging": {
							"total": 3,
							"next": "/v2/event-stream-sources?page=3"
						}
					}`,
				},
				{
					Validate: func(req *http.Request) bool {
						return testutils.ValidateRequest(t, req, "GET", "/v2/event-stream-sources?page=3", "")
					},
					ResponseStatus: 200,
					ResponseBody: `{
						"data": [
							{
								"id": "src-789",
								"externalId": "ext-789",
								"name": "Source 3",
								"type": "mobile",
								"enabled": true
							}
						],
						"paging": {
							"total": 3,
							"next": ""
						}
					}`,
				},
			},
			expectedSources: []esSource.EventStreamSource{
				{
					ID:         "src-123",
					ExternalID: "ext-123",
					Name:       "Source 1",
					Type:       "webhook",
					Enabled:    true,
				},
				{
					ID:         "src-456",
					ExternalID: "ext-456",
					Name:       "Source 2",
					Type:       "api",
					Enabled:    false,
				},
				{
					ID:         "src-789",
					ExternalID: "ext-789",
					Name:       "Source 3",
					Type:       "mobile",
					Enabled:    true,
				},
			},
		},
		{
			name: "empty response",
			calls: []testutils.Call{
				{
					Validate: func(req *http.Request) bool {
						return testutils.ValidateRequest(t, req, "GET", "/v2/event-stream-sources", "")
					},
					ResponseStatus: 200,
					ResponseBody: `{
						"data": [],
						"paging": {
							"total": 0
						}
					}`,
				},
			},
			expectedSources: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := testutils.NewMockHTTPClient(t, tt.calls...)

			c, err := client.New("test-token", client.WithHTTPClient(httpClient))
			require.NoError(t, err)

			eventStreamClient := esSource.NewRudderSourceStore(c)

			sources, err := eventStreamClient.GetSources(context.Background())
			require.NoError(t, err)

			assert.Equal(t, tt.expectedSources, sources)
			httpClient.AssertNumberOfCalls()
		})
	}
}
