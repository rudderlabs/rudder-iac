package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientDestinationsList(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destinations": [{
					"id": "id-1",
					"type": "type-1",
					"name": "name-1",
					"config": {"key":"val-1"}
				},  {
					"id": "id-2",
					"type": "type-2",
					"name": "name-2",
					"config": {"key":"val-2"}
				}],
				"paging": {
					"total": 3,
					"next": "/destinations?page=2"
				}
			}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations?page=2", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destinations": [{
					"id": "id-3",
					"type": "type-3",
					"name": "name-3",
					"config": {"key":"val-3"}
				}],
				"paging": {
					"total": 3
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	page, err := c.Destinations.List(ctx)
	require.NoError(t, err)
	assert.NotNil(t, page)
	assert.Len(t, page.Destinations, 2)
	assert.Equal(t, client.Destination{ID: "id-1", Type: "type-1", Name: "name-1", Config: []byte(`{"key":"val-1"}`)}, page.Destinations[0])
	assert.Equal(t, client.Destination{ID: "id-2", Type: "type-2", Name: "name-2", Config: []byte(`{"key":"val-2"}`)}, page.Destinations[1])
	assert.Equal(t, 3, page.Paging.Total)
	assert.Equal(t, "/destinations?page=2", page.Paging.Next)

	page, err = c.Destinations.Next(ctx, page.Paging)
	require.NoError(t, err)
	assert.NotNil(t, page)
	assert.Len(t, page.Destinations, 1)
	assert.Equal(t, client.Destination{ID: "id-3", Type: "type-3", Name: "name-3", Config: []byte(`{"key":"val-3"}`)}, page.Destinations[0])
	assert.Equal(t, 3, page.Paging.Total)
	assert.Equal(t, "", page.Paging.Next)

	page, err = c.Destinations.Next(ctx, page.Paging)
	require.NoError(t, err)
	assert.Nil(t, page)

	httpClient.AssertNumberOfCalls()
}

func TestClientDestinationsGet(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations/some-id", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destination": {
					"id": "some-id",
					"name": "some-name",
					"type": "some-type",
					"config": {
						"key1": "val1"
					},
					"createdAt": "2020-01-01T01:01:01Z",
					"updatedAt": "2020-01-02T01:01:01Z"	
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	destination, err := c.Destinations.Get(ctx, "some-id")
	require.NoError(t, err)
	assert.NotNil(t, destination)
	assert.Equal(t, "some-id", destination.ID)
	assert.Equal(t, "some-name", destination.Name)
	assert.Equal(t, "some-type", destination.Type)
	assert.Equal(t, time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC), *destination.CreatedAt)
	assert.Equal(t, time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC), *destination.UpdatedAt)

	httpClient.AssertNumberOfCalls()
}

func TestClientDestinationsCreate(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/destinations", `{
					"name": "some-name",
					"type": "some-type",
					"enabled": true,
					"config": { "key1": "val1" }
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destination": {
					"id": "some-id",
					"name": "some-name",
					"type": "some-type",
					"config": {
						"key1": "val1"
					},
					"createdAt": "2020-01-01T01:01:01Z",
					"updatedAt": "2020-01-02T01:01:01Z"
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	destination, err := c.Destinations.Create(ctx, &client.Destination{
		Name:      "some-name",
		Type:      "some-type",
		IsEnabled: true,
		Config: json.RawMessage([]byte(`{
			"key1": "val1"
		}`)),
	})
	require.NoError(t, err)
	assert.NotNil(t, destination)
	assert.Equal(t, "some-id", destination.ID)
	assert.Equal(t, "some-name", destination.Name)
	assert.Equal(t, "some-type", destination.Type)
	assert.Equal(t, time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC), *destination.CreatedAt)
	assert.Equal(t, time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC), *destination.UpdatedAt)

	httpClient.AssertNumberOfCalls()
}

func TestClientDestinationsUpdate(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/destinations/some-id", `{
					"name": "some-name",
					"type": "some-type",
					"enabled": true,
					"config": { "key1": "val1" }
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destination": {
					"id": "some-id",
					"name": "some-name",
					"type": "some-type",
					"config": {
						"key1": "val1"
					},
					"createdAt": "2020-01-01T01:01:01Z",
					"updatedAt": "2020-01-02T01:01:01Z"
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	destination, err := c.Destinations.Update(ctx, &client.Destination{
		ID:        "some-id",
		Name:      "some-name",
		Type:      "some-type",
		IsEnabled: true,
		Config: json.RawMessage([]byte(`{
			"key1": "val1"
		}`)),
	})
	require.NoError(t, err)
	assert.NotNil(t, destination)
	assert.Equal(t, "some-id", destination.ID)
	assert.Equal(t, "some-name", destination.Name)
	assert.Equal(t, "some-type", destination.Type)
	assert.Equal(t, time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC), *destination.CreatedAt)
	assert.Equal(t, time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC), *destination.UpdatedAt)

	httpClient.AssertNumberOfCalls()
}
