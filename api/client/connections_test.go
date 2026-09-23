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
					"externalId": "external-id-1",
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
	assert.Equal(t, "external-id-1", connection.ExternalID)
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

func TestClientConnectionsCreateWithExternalID(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/connections", `{
					"externalId": "external-id-1",
					"sourceId": "source-id",
					"destinationId": "destination-id",
					"enabled": false
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"connection": {
					"id": "some-id",
					"externalId": "external-id-1",
					"sourceId": "source-id",
					"destinationId": "destination-id"
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	connection, err := c.Connections.Create(ctx, &client.Connection{
		ExternalID:    "external-id-1",
		SourceID:      "source-id",
		DestinationID: "destination-id",
	})
	require.NoError(t, err)
	assert.NotNil(t, connection)
	assert.Equal(t, "some-id", connection.ID)
	assert.Equal(t, "external-id-1", connection.ExternalID)

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

func TestClientConnectionsUpdateStripsExternalID(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			// externalId can only be set through the external-id endpoint; the
			// generic update rejects bodies carrying it, so it must be absent here.
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
					"externalId": "external-id-1",
					"sourceId": "source-id",
					"destinationId": "destination-id",
					"enabled": true
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	connection, err := c.Connections.Update(ctx, &client.Connection{
		ID:            "some-id",
		ExternalID:    "external-id-1",
		SourceID:      "source-id",
		DestinationID: "destination-id",
		IsEnabled:     true,
	})
	require.NoError(t, err)
	assert.NotNil(t, connection)
	assert.Equal(t, "external-id-1", connection.ExternalID)

	httpClient.AssertNumberOfCalls()
}

func TestClientConnectionsListWithHasExternalID(t *testing.T) {
	ctx := context.Background()

	httpClient := testutils.NewMockHTTPClient(t,
		testutils.Call{
			Validate: func(req *http.Request) bool {
				// testutils.ValidateRequest does not compare URLs, and the query
				// string is the behaviour under test here
				return assert.Equal(t, "https://api.rudderstack.com/v2/connections?hasExternalId=true", req.URL.String()) &&
					testutils.ValidateRequest(t, req, "GET", "", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"connections": [{
					"id": "id-1",
					"externalId": "external-id-1",
					"sourceId": "source-1",
					"destinationId": "destination-1",
					"enabled": true
				}],
				"paging": {
					"total": 1
				}
			}`,
		},
		testutils.Call{
			Validate: func(req *http.Request) bool {
				return assert.Equal(t, "https://api.rudderstack.com/v2/connections?hasExternalId=false", req.URL.String()) &&
					testutils.ValidateRequest(t, req, "GET", "", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"connections": [{
					"id": "id-2",
					"sourceId": "source-2",
					"destinationId": "destination-2",
					"enabled": true
				}],
				"paging": {
					"total": 1
				}
			}`,
		},
	)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	page, err := c.Connections.List(ctx, client.WithConnectionsHasExternalID(true))
	require.NoError(t, err)
	require.NotNil(t, page)
	require.Len(t, page.Connections, 1)
	assert.Equal(t, "external-id-1", page.Connections[0].ExternalID)

	page, err = c.Connections.List(ctx, client.WithConnectionsHasExternalID(false))
	require.NoError(t, err)
	require.NotNil(t, page)
	require.Len(t, page.Connections, 1)
	assert.Equal(t, "", page.Connections[0].ExternalID)

	httpClient.AssertNumberOfCalls()
}

func TestClientConnectionsSetExternalID(t *testing.T) {
	ctx := context.Background()

	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			// testutils.ValidateRequest does not compare URLs, and the path is
			// part of the behaviour under test here
			return assert.Equal(t, "https://api.rudderstack.com/v2/connections/some-id/external-id", req.URL.String()) &&
				testutils.ValidateRequest(t, req, "PUT", "", `{
					"externalId": "external-id-1"
				}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id": "some-id", "externalId": "external-id-1"}`,
	})

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	err = c.Connections.SetExternalID(ctx, "some-id", "external-id-1")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

// The set external ID endpoint answers 409 for two conditions that differ only
// in message, so the sentinel must not swallow what the backend said.
func TestClientConnectionsSetExternalIDConflict(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		responseBody string
		wantMessage  string
	}{
		{
			name:         "claimed by another connection",
			responseBody: `{"error": "A connection with this externalId already exists in the workspace"}`,
			wantMessage:  "A connection with this externalId already exists in the workspace",
		},
		{
			name:         "already set on this connection",
			responseBody: `{"error": "externalId is already set for this connection and cannot be changed"}`,
			wantMessage:  "externalId is already set for this connection and cannot be changed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
				ResponseStatus: 409,
				ResponseBody:   tc.responseBody,
			})

			c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
			require.NoError(t, err)

			err = c.Connections.SetExternalID(ctx, "some-id", "external-id-1")
			require.Error(t, err)
			assert.ErrorIs(t, err, client.ErrExternalIDAlreadyInUse)
			assert.Contains(t, err.Error(), tc.wantMessage)

			httpClient.AssertNumberOfCalls()
		})
	}
}

// Create answers 409 when the body carries an externalId and a live connection
// already exists for the source/destination pair.
func TestClientConnectionsCreateConflict(t *testing.T) {
	ctx := context.Background()

	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/connections", `{
				"externalId": "external-id-1",
				"sourceId": "source-id",
				"destinationId": "destination-id",
				"enabled": false
			}`)
		},
		ResponseStatus: 409,
		ResponseBody:   `{"error": "Connection already exists for this source and destination pair. Use PUT /v2/connections/:id/external-id to set its externalId"}`,
	})

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	connection, err := c.Connections.Create(ctx, &client.Connection{
		ExternalID:    "external-id-1",
		SourceID:      "source-id",
		DestinationID: "destination-id",
	})
	require.Error(t, err)
	assert.Nil(t, connection)
	assert.ErrorIs(t, err, client.ErrConnectionPairExists)
	assert.Contains(t, err.Error(), "Connection already exists for this source and destination pair")

	httpClient.AssertNumberOfCalls()
}

// A non-conflict create failure must stay unmapped.
func TestClientConnectionsCreateNonConflictErrorIsNotMapped(t *testing.T) {
	ctx := context.Background()

	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		ResponseStatus: 400,
		ResponseBody:   `{"error": "sourceId is required"}`,
	})

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	connection, err := c.Connections.Create(ctx, &client.Connection{
		SourceID:      "source-id",
		DestinationID: "destination-id",
	})
	require.Error(t, err)
	assert.Nil(t, connection)
	assert.NotErrorIs(t, err, client.ErrConnectionPairExists)
	assert.Contains(t, err.Error(), "sourceId is required")

	httpClient.AssertNumberOfCalls()
}
