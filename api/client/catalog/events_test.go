package catalog_test

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEvent(t *testing.T) {
	categoryID := "cat-1"
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "POST", catalogURL("v2/catalog/events"), `{"name":"Order Completed","description":"desc","eventType":"track","categoryId":"cat-1","externalId":"ext-1"}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"evt-1","name":"Order Completed","eventType":"track","workspaceId":"ws-1"}`,
	})

	event, err := dc.CreateEvent(context.Background(), catalog.EventCreate{
		Name:        "Order Completed",
		Description: "desc",
		EventType:   "track",
		CategoryId:  &categoryID,
		ExternalId:  "ext-1",
	})
	require.NoError(t, err)
	assert.Equal(t, &catalog.Event{ID: "evt-1", Name: "Order Completed", EventType: "track", WorkspaceId: "ws-1"}, event)
	httpClient.AssertNumberOfCalls()
}

func TestUpdateEvent(t *testing.T) {
	categoryID := "cat-2"
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "PUT", catalogURL("v2/catalog/events/evt-1"), `{"name":"Updated","description":"new","eventType":"track","categoryId":"cat-2"}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"evt-1","name":"Updated","eventType":"track","workspaceId":"ws-1"}`,
	})

	event, err := dc.UpdateEvent(context.Background(), "evt-1", &catalog.EventUpdate{
		Name:        "Updated",
		Description: "new",
		EventType:   "track",
		CategoryId:  &categoryID,
	})
	require.NoError(t, err)
	assert.Equal(t, &catalog.Event{ID: "evt-1", Name: "Updated", EventType: "track", WorkspaceId: "ws-1"}, event)
	httpClient.AssertNumberOfCalls()
}

func TestGetEvent(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "GET", catalogURL("v2/catalog/events/evt-1"), "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"evt-1","name":"Updated","eventType":"track","workspaceId":"ws-1"}`,
	})

	event, err := dc.GetEvent(context.Background(), "evt-1")
	require.NoError(t, err)
	assert.Equal(t, &catalog.Event{ID: "evt-1", Name: "Updated", EventType: "track", WorkspaceId: "ws-1"}, event)
	httpClient.AssertNumberOfCalls()
}

func TestSetEventExternalID(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "PUT", catalogURL("v2/catalog/events/evt-1/external-id"), `{"externalId":"ext-updated"}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{}`,
	})

	err := dc.SetEventExternalId(context.Background(), "evt-1", "ext-updated")
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestDeleteEvent(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "DELETE", catalogURL("v2/catalog/events/evt-1"), "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{}`,
	})

	err := dc.DeleteEvent(context.Background(), "evt-1")
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestGetEvents(t *testing.T) {
	dc, httpClient := newDataCatalog(t,
		testutils.Call{
			Validate: func(req *http.Request) bool {
				return assertCall(t, req, "GET", catalogURL("v2/catalog/events?hasExternalId=true"), "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{"data":[{"id":"e1"},{"id":"e2"}],"total":3,"currentPage":1,"pageSize":2}`, // This request will be thrown away as it's used only to calculate concurrency
		},
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"e1"},{"id":"e2"}],"total":3,"currentPage":1,"pageSize":2}`},
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"e3"}],"total":3,"currentPage":2,"pageSize":2}`},
	)

	events, err := dc.GetEvents(context.Background(), catalog.ListOptions{HasExternalID: boolPtr(true)})
	require.NoError(t, err)
	require.Len(t, events, 3)

	ids := []string{events[0].ID, events[1].ID, events[2].ID}
	sort.Strings(ids)
	assert.Equal(t, []string{"e1", "e2", "e3"}, ids)
	httpClient.AssertNumberOfCalls()
}

func TestEventErrors(t *testing.T) {
	tests := []struct {
		name          string
		calls         []testutils.Call
		operation     func(dc catalog.DataCatalog) error
		expectedError string
	}{
		{
			name:  "CreateEvent request error",
			calls: []testutils.Call{{ResponseError: errors.New("network down")}},
			operation: func(dc catalog.DataCatalog) error {
				_, err := dc.CreateEvent(context.Background(), catalog.EventCreate{Name: "A"})
				return err
			},
			expectedError: "sending request",
		},
		{
			name:  "GetEvent decode error",
			calls: []testutils.Call{{ResponseStatus: 200, ResponseBody: "{"}},
			operation: func(dc catalog.DataCatalog) error {
				_, err := dc.GetEvent(context.Background(), "evt-1")
				return err
			},
			expectedError: "decoding response",
		},
		{
			name:  "GetEvents first page error",
			calls: []testutils.Call{{ResponseStatus: 500, ResponseBody: `{"error":"internal"}`}},
			operation: func(dc catalog.DataCatalog) error {
				_, err := dc.GetEvents(context.Background(), catalog.ListOptions{})
				return err
			},
			expectedError: "getting first page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc, httpClient := newDataCatalog(t, tt.calls...)
			err := tt.operation(dc)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.expectedError)
			httpClient.AssertNumberOfCalls()
		})
	}
}
