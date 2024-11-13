package client_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientConnectionsList(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/connections", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"connections": [{
					"id": "id-1",
					"sourceId": "source-1",
					"destinationId": "destination-1",
					"enabled": true
				},  {
					"id": "id-2",
					"sourceId": "source-2",
					"destinationId": "destination-2",
					"enabled": false
				}],
				"paging": {
					"total": 3,
					"next": "/connections?page=2"
				}
			}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/connections?page=2", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"connections": [{
					"id": "id-3",
					"sourceId": "source-3",
					"destinationId": "destination-3"
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

	page, err := c.Connections.List(ctx)
	require.NoError(t, err)
	assert.NotNil(t, page)
	assert.Len(t, page.Connections, 2)
	assert.Equal(t, client.Connection{ID: "id-1", SourceID: "source-1", DestinationID: "destination-1", IsEnabled: true}, page.Connections[0])
	assert.Equal(t, client.Connection{ID: "id-2", SourceID: "source-2", DestinationID: "destination-2", IsEnabled: false}, page.Connections[1])
	assert.Equal(t, 3, page.Paging.Total)
	assert.Equal(t, "/connections?page=2", page.Paging.Next)

	page, err = c.Connections.Next(ctx, page.Paging)
	require.NoError(t, err)
	assert.NotNil(t, page)
	assert.Len(t, page.Connections, 1)
	assert.Equal(t, client.Connection{ID: "id-3", SourceID: "source-3", DestinationID: "destination-3"}, page.Connections[0])
	assert.Equal(t, 3, page.Paging.Total)
	assert.Equal(t, "", page.Paging.Next)

	page, err = c.Connections.Next(ctx, page.Paging)
	require.NoError(t, err)
	assert.Nil(t, page)

	httpClient.AssertNumberOfCalls()
}

func TestClientConnectionsGet(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/connections/some-id", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"connection": {
					"id": "some-id",
					"sourceId": "source-id",
					"destinationId": "destination-id",
					"enabled": true,
					"createdAt": "2020-01-01T01:01:01Z",
					"updatedAt": "2020-01-02T01:01:01Z"	
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	connection, err := c.Connections.Get(ctx, "some-id")
	require.NoError(t, err)
	assert.NotNil(t, connection)
	assert.Equal(t, "some-id", connection.ID)
	assert.Equal(t, "source-id", connection.SourceID)
	assert.Equal(t, "destination-id", connection.DestinationID)
	assert.Equal(t, true, connection.IsEnabled)
	assert.Equal(t, time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC), *connection.CreatedAt)
	assert.Equal(t, time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC), *connection.UpdatedAt)

	httpClient.AssertNumberOfCalls()
}

func TestClientConnectionsCreate(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/connections", `{
					"sourceId": "source-id",
					"destinationId": "destination-id",
					"enabled": false
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"connection": {
					"id": "some-id",
					"sourceId": "source-id",
					"destinationId": "destination-id",
					"createdAt": "2020-01-01T01:01:01Z",
					"updatedAt": "2020-01-02T01:01:01Z"
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	connection, err := c.Connections.Create(ctx, &client.Connection{
		SourceID:      "source-id",
		DestinationID: "destination-id",
	})
	require.NoError(t, err)
	assert.NotNil(t, connection)
	assert.Equal(t, "some-id", connection.ID)
	assert.Equal(t, "source-id", connection.SourceID)
	assert.Equal(t, "destination-id", connection.DestinationID)
	assert.Equal(t, time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC), *connection.CreatedAt)
	assert.Equal(t, time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC), *connection.UpdatedAt)

	httpClient.AssertNumberOfCalls()
}

func TestClientConnectionsUpdate(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/connections/some-id", `{
					"sourceId": "source-id",
					"destinationId": "destination-id",
					"enabled": true
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"connection": {
					"id": "some-id",
					"sourceId": "source-id",
					"destinationId": "destination-id",
					"createdAt": "2020-01-01T01:01:01Z",
					"updatedAt": "2020-01-02T01:01:01Z"
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	connection, err := c.Connections.Update(ctx, &client.Connection{
		ID:            "some-id",
		SourceID:      "source-id",
		DestinationID: "destination-id",
		IsEnabled:     true,
	})
	require.NoError(t, err)
	assert.NotNil(t, connection)
	assert.Equal(t, "some-id", connection.ID)
	assert.Equal(t, "source-id", connection.SourceID)
	assert.Equal(t, "destination-id", connection.DestinationID)
	assert.Equal(t, time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC), *connection.CreatedAt)
	assert.Equal(t, time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC), *connection.UpdatedAt)

	httpClient.AssertNumberOfCalls()
}
