package retl_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

func intPtr(i int) *int { return &i }

// assertCall validates an incoming mock HTTP request against the expected
// method, URL, and JSON body. testutils.ValidateRequest currently ignores its
// url argument, so we assert the URL here explicitly.
func assertCall(t *testing.T, req *http.Request, method, url, body string) bool {
	t.Helper()
	return assert.Equal(t, url, req.URL.String()) &&
		testutils.ValidateRequest(t, req, method, url, body)
}

func TestCreateConnection(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{
				"sourceId": "retl-src-123",
				"destinationId": "dest-456",
				"enabled": true,
				"schedule": {"type": "basic", "everyMinutes": 60},
				"syncBehaviour": "upsert",
				"identifiers": [{"from": "email", "to": "user_id"}],
				"mappings": [{"from": "name", "to": "first_name"}],
				"event": {"type": "identify"}
			}`
			return assertCall(t, req, "POST", "https://api.rudderstack.com/v2/retl-connections", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "conn-1",
			"sourceId": "retl-src-123",
			"destinationId": "dest-456",
			"enabled": true,
			"schedule": {"type": "basic", "everyMinutes": 60},
			"syncBehaviour": "upsert",
			"identifiers": [{"from": "email", "to": "user_id"}],
			"mappings": [{"from": "name", "to": "first_name"}],
			"event": {"type": "identify"},
			"createdAt": "2026-04-20T12:00:00Z",
			"updatedAt": "2026-04-20T12:00:00Z"
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	req := &retl.CreateRETLConnectionRequest{
		SourceID:      "retl-src-123",
		DestinationID: "dest-456",
		Enabled:       boolPtr(true),
		Schedule:      retl.Schedule{Type: retl.ScheduleTypeBasic, EveryMinutes: intPtr(60)},
		SyncBehaviour: retl.SyncBehaviourUpsert,
		Identifiers:   []retl.Mapping{{From: "email", To: "user_id"}},
		Mappings:      []retl.Mapping{{From: "name", To: "first_name"}},
		Event:         &retl.Event{Type: retl.EventTypeIdentify},
	}

	created, err := retlClient.CreateConnection(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "conn-1", created.ID)
	assert.Equal(t, "retl-src-123", created.SourceID)
	assert.Equal(t, "dest-456", created.DestinationID)
	assert.True(t, created.Enabled)
	assert.Equal(t, retl.SyncBehaviourUpsert, created.SyncBehaviour)
	assert.Equal(t, retl.ScheduleTypeBasic, created.Schedule.Type)
	require.NotNil(t, created.Schedule.EveryMinutes)
	assert.Equal(t, 60, *created.Schedule.EveryMinutes)
	assert.Equal(t, []retl.Mapping{{From: "email", To: "user_id"}}, created.Identifiers)

	httpClient.AssertNumberOfCalls()
}

func TestUpdateConnection(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{
				"schedule": {"type": "basic", "everyMinutes": 120},
				"mappings": [{"from": "name", "to": "first_name"}]
			}`
			return assertCall(t, req, "PUT", "https://api.rudderstack.com/v2/retl-connections/conn-1", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "conn-1",
			"sourceId": "retl-src-123",
			"destinationId": "dest-456",
			"enabled": true,
			"schedule": {"type": "basic", "everyMinutes": 120},
			"syncBehaviour": "upsert",
			"identifiers": [{"from": "email", "to": "user_id"}],
			"mappings": [{"from": "name", "to": "first_name"}]
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	req := &retl.UpdateRETLConnectionRequest{
		Schedule: retl.Schedule{Type: retl.ScheduleTypeBasic, EveryMinutes: intPtr(120)},
		Mappings: &[]retl.Mapping{{From: "name", To: "first_name"}},
	}

	updated, err := retlClient.UpdateConnection(context.Background(), "conn-1", req)
	require.NoError(t, err)

	assert.Equal(t, "conn-1", updated.ID)
	require.NotNil(t, updated.Schedule.EveryMinutes)
	assert.Equal(t, 120, *updated.Schedule.EveryMinutes)

	httpClient.AssertNumberOfCalls()
}

func TestUpdateConnection_ExplicitEmptyMappingsAndConstants(t *testing.T) {
	// Pointer-to-slice fields let callers send an empty array on the wire to
	// clear existing values, distinct from omitting the field entirely.
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{
				"schedule": {"type": "manual"},
				"mappings": [],
				"constants": []
			}`
			return assertCall(t, req, "PUT", "https://api.rudderstack.com/v2/retl-connections/conn-1", expected)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"conn-1","sourceId":"s","destinationId":"d","enabled":true,"schedule":{"type":"manual"},"syncBehaviour":"upsert","identifiers":[]}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	_, err = retlClient.UpdateConnection(context.Background(), "conn-1", &retl.UpdateRETLConnectionRequest{
		Schedule:  retl.Schedule{Type: retl.ScheduleTypeManual},
		Mappings:  &[]retl.Mapping{},
		Constants: &[]retl.Constant{},
	})
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestGetConnection(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "GET", "https://api.rudderstack.com/v2/retl-connections/conn-1", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "conn-1",
			"sourceId": "retl-src-123",
			"destinationId": "dest-456",
			"enabled": false,
			"schedule": {"type": "manual"},
			"syncBehaviour": "upsert",
			"identifiers": [{"from": "email", "to": "user_id"}]
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	got, err := retlClient.GetConnection(context.Background(), "conn-1")
	require.NoError(t, err)

	assert.Equal(t, "conn-1", got.ID)
	assert.False(t, got.Enabled)
	assert.Equal(t, retl.ScheduleTypeManual, got.Schedule.Type)
	assert.Nil(t, got.Schedule.EveryMinutes)

	httpClient.AssertNumberOfCalls()
}

func TestDeleteConnection(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "DELETE", "https://api.rudderstack.com/v2/retl-connections/conn-1", "")
		},
		ResponseStatus: 204,
		ResponseBody:   "",
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	require.NoError(t, retlClient.DeleteConnection(context.Background(), "conn-1"))
	httpClient.AssertNumberOfCalls()
}

func TestListConnections_NoFilters(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "GET", "https://api.rudderstack.com/v2/retl-connections", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [],
			"paging": {"total": 0}
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	page, err := retlClient.ListConnections(context.Background(), &retl.ListRETLConnectionsRequest{})
	require.NoError(t, err)
	assert.Empty(t, page.Data)
	assert.Equal(t, 0, page.Paging.Total)

	httpClient.AssertNumberOfCalls()
}

func TestListConnections_WithFiltersAndPaging(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			query := req.URL.Query()
			if query.Get("sourceId") != "retl-src-123" {
				return false
			}
			if query.Get("destinationId") != "dest-456" {
				return false
			}
			if query.Get("hasExternalId") != "true" {
				return false
			}
			if query.Get("page") != "2" {
				return false
			}
			if query.Get("pageSize") != "50" {
				return false
			}
			if req.URL.Path != "/v2/retl-connections" {
				return false
			}
			return req.Method == http.MethodGet
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "conn-1",
					"sourceId": "retl-src-123",
					"destinationId": "dest-456",
					"enabled": true,
					"externalId": "ext-1",
					"schedule": {"type": "basic", "everyMinutes": 60},
					"syncBehaviour": "upsert",
					"identifiers": [{"from": "email", "to": "user_id"}]
				}
			],
			"paging": {
				"total": 42,
				"next": "/v2/retl-connections?page=3&pageSize=50"
			}
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	page, err := retlClient.ListConnections(context.Background(), &retl.ListRETLConnectionsRequest{
		SourceID:      "retl-src-123",
		DestinationID: "dest-456",
		HasExternalID: boolPtr(true),
		Page:          2,
		PageSize:      50,
	})
	require.NoError(t, err)

	assert.Len(t, page.Data, 1)
	assert.Equal(t, "conn-1", page.Data[0].ID)
	assert.Equal(t, "ext-1", page.Data[0].ExternalID)
	assert.Equal(t, 42, page.Paging.Total)
	assert.Equal(t, "/v2/retl-connections?page=3&pageSize=50", page.Paging.Next)

	httpClient.AssertNumberOfCalls()
}

func TestSetConnectionExternalId(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"externalId": "ext-123"}`
			return assertCall(t, req, "PUT", "https://api.rudderstack.com/v2/retl-connections/conn-1/external-id", expected)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id": "conn-1", "externalId": "ext-123"}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	err = retlClient.SetConnectionExternalId(context.Background(), &retl.SetRETLConnectionExternalIDRequest{
		ID:         "conn-1",
		ExternalID: "ext-123",
	})
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestConnection_EmptyIDErrors(t *testing.T) {
	c, err := client.New("test-token")
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	_, err = retlClient.UpdateConnection(context.Background(), "", &retl.UpdateRETLConnectionRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection ID cannot be empty")

	err = retlClient.DeleteConnection(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection ID cannot be empty")

	_, err = retlClient.GetConnection(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection ID cannot be empty")

	err = retlClient.SetConnectionExternalId(context.Background(), &retl.SetRETLConnectionExternalIDRequest{ID: "", ExternalID: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection ID cannot be empty")
}

func TestUpdateConnection_RequiresSchedule(t *testing.T) {
	c, err := client.New("test-token")
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	// Zero-value Schedule would otherwise serialize as {"schedule":{"type":""}}
	// and fail server-side validation with a confusing enum error.
	_, err = retlClient.UpdateConnection(context.Background(), "conn-1", &retl.UpdateRETLConnectionRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schedule.type is required")
}

func TestCreateConnection_RequiresSchedule(t *testing.T) {
	c, err := client.New("test-token")
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	_, err = retlClient.CreateConnection(context.Background(), &retl.CreateRETLConnectionRequest{
		SourceID:      "retl-src-123",
		DestinationID: "dest-456",
		SyncBehaviour: retl.SyncBehaviourUpsert,
		Identifiers:   []retl.Mapping{{From: "email", To: "user_id"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schedule.type is required")
}

func TestConnection_NilRequestErrors(t *testing.T) {
	c, err := client.New("test-token")
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	_, err = retlClient.CreateConnection(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request cannot be nil")

	_, err = retlClient.UpdateConnection(context.Background(), "conn-1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request cannot be nil")

	err = retlClient.SetConnectionExternalId(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request cannot be nil")
}

func TestCreateConnection_APIError(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "POST", "https://api.rudderstack.com/v2/retl-connections", "")
		},
		ResponseStatus: 400,
		ResponseBody:   `{"error": "Bad Request"}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	_, err = retlClient.CreateConnection(context.Background(), &retl.CreateRETLConnectionRequest{
		Schedule: retl.Schedule{Type: retl.ScheduleTypeManual},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating RETL connection")
	httpClient.AssertNumberOfCalls()
}

func TestGetConnection_MalformedResponse(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "GET", "https://api.rudderstack.com/v2/retl-connections/conn-1", "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{malformed`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	_, err = retlClient.GetConnection(context.Background(), "conn-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling response")
	httpClient.AssertNumberOfCalls()
}

func TestListConnections_APIError(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "GET", "https://api.rudderstack.com/v2/retl-connections", "")
		},
		ResponseStatus: 500,
		ResponseBody:   `{"error": "Internal Server Error"}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	_, err = retlClient.ListConnections(context.Background(), &retl.ListRETLConnectionsRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing RETL connections")
	httpClient.AssertNumberOfCalls()
}

func TestDeleteConnection_APIError(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "DELETE", "https://api.rudderstack.com/v2/retl-connections/conn-1", "")
		},
		ResponseStatus: 500,
		ResponseBody:   `{"error": "Internal Server Error"}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	err = retlClient.DeleteConnection(context.Background(), "conn-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deleting RETL connection")
	httpClient.AssertNumberOfCalls()
}

func TestSetConnectionExternalId_APIError(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "PUT", "https://api.rudderstack.com/v2/retl-connections/conn-1/external-id", "")
		},
		ResponseStatus: 500,
		ResponseBody:   `{"error": "Internal Server Error"}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	retlClient := retl.NewRudderRETLStore(c)

	err = retlClient.SetConnectionExternalId(context.Background(), &retl.SetRETLConnectionExternalIDRequest{
		ID:         "conn-1",
		ExternalID: "ext-123",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "setting external ID")
	httpClient.AssertNumberOfCalls()
}
