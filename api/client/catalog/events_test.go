package catalog_test

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"sync"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventCRUDAndExternalID(t *testing.T) {
	t.Run("create event success", func(t *testing.T) {
		calls := []testutils.Call{{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "POST", "v2/catalog/events", `{"name":"Order Completed","description":"desc","eventType":"track","categoryId":"cat-1","externalId":"ext-1"}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"evt-1","name":"Order Completed","eventType":"track","workspaceId":"ws-1"}`,
		}}

		categoryID := "cat-1"
		dc, httpClient := newDataCatalog(t, nil, calls...)
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
	})

	t.Run("create event request error", func(t *testing.T) {
		calls := []testutils.Call{{
			ResponseError: errors.New("network down"),
		}}

		dc, httpClient := newDataCatalog(t, nil, calls...)
		_, err := dc.CreateEvent(context.Background(), catalog.EventCreate{Name: "A"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "sending request")
		httpClient.AssertNumberOfCalls()
	})

	t.Run("update get delete and set external id", func(t *testing.T) {
		calls := []testutils.Call{
			{
				Validate: func(req *http.Request) bool {
					return validateRequest(t, req, "PUT", "v2/catalog/events/evt-1", `{"name":"Updated","description":"new","eventType":"track","categoryId":"cat-2"}`)
				},
				ResponseStatus: 200,
				ResponseBody:   `{"id":"evt-1","name":"Updated","eventType":"track","workspaceId":"ws-1"}`,
			},
			{
				Validate: func(req *http.Request) bool {
					return validateRequest(t, req, "GET", "v2/catalog/events/evt-1", "")
				},
				ResponseStatus: 200,
				ResponseBody:   `{"id":"evt-1","name":"Updated","eventType":"track","workspaceId":"ws-1"}`,
			},
			{
				Validate: func(req *http.Request) bool {
					return validateRequest(t, req, "PUT", "v2/catalog/events/evt-1/external-id", `{"externalId":"ext-updated"}`)
				},
				ResponseStatus: 200,
				ResponseBody:   `{}`,
			},
			{
				Validate: func(req *http.Request) bool {
					return validateRequest(t, req, "DELETE", "v2/catalog/events/evt-1", "")
				},
				ResponseStatus: 200,
				ResponseBody:   `{}`,
			},
		}

		categoryID := "cat-2"
		dc, httpClient := newDataCatalog(t, nil, calls...)

		event, err := dc.UpdateEvent(context.Background(), "evt-1", &catalog.EventUpdate{
			Name:        "Updated",
			Description: "new",
			EventType:   "track",
			CategoryId:  &categoryID,
		})
		require.NoError(t, err)
		assert.Equal(t, &catalog.Event{ID: "evt-1", Name: "Updated", EventType: "track", WorkspaceId: "ws-1"}, event)

		event, err = dc.GetEvent(context.Background(), "evt-1")
		require.NoError(t, err)
		assert.Equal(t, "evt-1", event.ID)

		err = dc.SetEventExternalId(context.Background(), "evt-1", "ext-updated")
		require.NoError(t, err)

		err = dc.DeleteEvent(context.Background(), "evt-1")
		require.NoError(t, err)

		httpClient.AssertNumberOfCalls()
	})

	t.Run("decode errors are returned", func(t *testing.T) {
		calls := []testutils.Call{{
			ResponseStatus: 200,
			ResponseBody:   "{",
		}}
		dc, httpClient := newDataCatalog(t, nil, calls...)
		_, err := dc.GetEvent(context.Background(), "evt-1")
		require.Error(t, err)
		assert.ErrorContains(t, err, "decoding response")
		httpClient.AssertNumberOfCalls()
	})
}

func TestGetEventsPagination(t *testing.T) {
	t.Run("aggregates multiple pages", func(t *testing.T) {
		hasExternalID := true
		var (
			mu       sync.Mutex
			seenPage = map[string]bool{}
		)
		calls := []testutils.Call{
			{
				Validate: func(req *http.Request) bool {
					return validateRequest(t, req, "GET", "v2/catalog/events?hasExternalId=true", "")
				},
				ResponseStatus: 200,
				ResponseBody:   `{"data":[{"id":"first"}],"total":3,"currentPage":1,"pageSize":2}`,
			},
			{
				Validate: func(req *http.Request) bool {
					if !validateRequest(t, req, "GET", req.URL.Path+"?"+req.URL.RawQuery, "") {
						return false
					}
					page := req.URL.Query().Get("page")
					if !assert.Contains(t, []string{"1", "2"}, page) {
						return false
					}
					mu.Lock()
					defer mu.Unlock()
					if seenPage[page] {
						return false
					}
					seenPage[page] = true
					return true
				},
				ResponseStatus: 200,
				ResponseBody:   `{"data":[{"id":"e1"},{"id":"e2"}],"total":3,"currentPage":1,"pageSize":2}`,
			},
			{
				Validate: func(req *http.Request) bool {
					if !validateRequest(t, req, "GET", req.URL.Path+"?"+req.URL.RawQuery, "") {
						return false
					}
					page := req.URL.Query().Get("page")
					if !assert.Contains(t, []string{"1", "2"}, page) {
						return false
					}
					mu.Lock()
					defer mu.Unlock()
					if seenPage[page] {
						return false
					}
					seenPage[page] = true
					return true
				},
				ResponseStatus: 200,
				ResponseBody:   `{"data":[{"id":"e3"}],"total":3,"currentPage":2,"pageSize":2}`,
			},
		}

		dc, httpClient := newDataCatalog(t, nil, calls...)
		events, err := dc.GetEvents(context.Background(), catalog.ListOptions{HasExternalID: &hasExternalID})
		require.NoError(t, err)
		require.Len(t, events, 3)

		ids := []string{events[0].ID, events[1].ID, events[2].ID}
		sort.Strings(ids)
		assert.Equal(t, []string{"e1", "e2", "e3"}, ids)
		assert.True(t, seenPage["1"])
		assert.True(t, seenPage["2"])
		httpClient.AssertNumberOfCalls()
	})

	t.Run("returns feature-flag first page as empty list", func(t *testing.T) {
		calls := []testutils.Call{{
			ResponseStatus: 403,
			ResponseBody:   `{"error":"Flag is not enabled for your account"}`,
		}}
		dc, httpClient := newDataCatalog(t, nil, calls...)
		events, err := dc.GetEvents(context.Background(), catalog.ListOptions{})
		require.NoError(t, err)
		assert.Empty(t, events)
		httpClient.AssertNumberOfCalls()
	})

	t.Run("returns first page request error", func(t *testing.T) {
		calls := []testutils.Call{{
			ResponseStatus: 500,
			ResponseBody:   `{"error":"internal"}`,
		}}
		dc, httpClient := newDataCatalog(t, nil, calls...)
		_, err := dc.GetEvents(context.Background(), catalog.ListOptions{})
		require.Error(t, err)
		assert.ErrorContains(t, err, "getting first page")
		httpClient.AssertNumberOfCalls()
	})
}
