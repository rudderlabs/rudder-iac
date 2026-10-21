package catalog_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTrackingPlanWithSchemas_PreservesEventOrderWhileFetchingConcurrently(t *testing.T) {
	var (
		activeRequests    atomic.Int32
		concurrentFetches atomic.Bool
	)
	eventIDs := []string{"event-slowest", "event-medium", "event-fastest"}
	delays := map[string]time.Duration{
		"event-slowest": 30 * time.Millisecond,
		"event-medium":  20 * time.Millisecond,
		"event-fastest": 10 * time.Millisecond,
	}

	server := newTrackingPlanSchemasServer(t, eventIDs, func(w http.ResponseWriter, eventID string) {
		if activeRequests.Add(1) > 1 {
			concurrentFetches.Store(true)
		}
		defer activeRequests.Add(-1)
		time.Sleep(delays[eventID])
		writeJSON(t, w, catalog.TrackingPlanEventSchema{ID: eventID, Name: eventID})
	})
	defer server.Close()
	dataCatalog := newTrackingPlanSchemasCatalog(t, server, len(eventIDs))

	trackingPlan, err := dataCatalog.GetTrackingPlanWithSchemas(context.Background(), "tp-123")

	require.NoError(t, err)
	assert.True(t, concurrentFetches.Load())
	assert.Equal(t, []catalog.TrackingPlanEventSchema{
		{ID: eventIDs[0], Name: eventIDs[0]},
		{ID: eventIDs[1], Name: eventIDs[1]},
		{ID: eventIDs[2], Name: eventIDs[2]},
	}, trackingPlan.Events)
}

func TestGetTrackingPlanWithSchemas_PreservesEventOrderAcrossEventPages(t *testing.T) {
	eventPages := [][]string{
		{"page-1-event"},
		{"page-2-event"},
		{"page-3-event"},
	}
	server := newTrackingPlanSchemasServerWithPages(t, eventPages, map[int]time.Duration{
		2: 30 * time.Millisecond,
		3: 10 * time.Millisecond,
	}, nil)
	defer server.Close()
	dataCatalog := newTrackingPlanSchemasCatalog(t, server, 2)

	trackingPlan, err := dataCatalog.GetTrackingPlanWithSchemas(context.Background(), "tp-123")

	require.NoError(t, err)
	assert.Equal(t, []catalog.TrackingPlanEventSchema{
		{ID: "page-1-event", Name: "page-1-event"},
		{ID: "page-2-event", Name: "page-2-event"},
		{ID: "page-3-event", Name: "page-3-event"},
	}, trackingPlan.Events)
}

func TestGetTrackingPlanWithSchemas_IsInvariantAcrossConcurrencyLimits(t *testing.T) {
	eventIDs := []string{"event-1", "event-2", "event-3"}
	server := newTrackingPlanSchemasServer(t, eventIDs, nil)
	defer server.Close()

	loadEvents := func(concurrency int) []catalog.TrackingPlanEventSchema {
		dataCatalog := newTrackingPlanSchemasCatalog(t, server, concurrency)
		trackingPlan, err := dataCatalog.GetTrackingPlanWithSchemas(context.Background(), "tp-123")
		require.NoError(t, err)
		return trackingPlan.Events
	}

	assert.Equal(t, loadEvents(1), loadEvents(3))
}

func TestGetTrackingPlanWithSchemas_PropagatesSchemaFetchError(t *testing.T) {
	server := newTrackingPlanSchemasServer(t, []string{"event-1"}, func(w http.ResponseWriter, _ string) {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(t, w, map[string]string{"message": "schema unavailable"})
	})
	defer server.Close()
	dataCatalog := newTrackingPlanSchemasCatalog(t, server, 2)

	trackingPlan, err := dataCatalog.GetTrackingPlanWithSchemas(context.Background(), "tp-123")

	require.Error(t, err)
	assert.Nil(t, trackingPlan)
	assert.ErrorContains(t, err, "fetching event schemas on tracking plan: tp-123")
	assert.ErrorContains(t, err, "schema unavailable")
	var apiErr *client.APIError
	assert.ErrorAs(t, err, &apiErr)
}

func TestGetTrackingPlanWithSchemas_ReturnsEmptyEventsForEmptyPlan(t *testing.T) {
	server := newTrackingPlanSchemasServer(t, nil, nil)
	defer server.Close()
	dataCatalog := newTrackingPlanSchemasCatalog(t, server, 2)

	trackingPlan, err := dataCatalog.GetTrackingPlanWithSchemas(context.Background(), "tp-123")

	require.NoError(t, err)
	assert.Empty(t, trackingPlan.Events)
	assert.NotNil(t, trackingPlan.Events)
}

func TestGetTrackingPlanWithSchemas_ReusesDuplicateEventSchema(t *testing.T) {
	var schemaRequests atomic.Int32
	server := newTrackingPlanSchemasServer(t, []string{"event-1", "event-1"}, func(w http.ResponseWriter, eventID string) {
		schemaRequests.Add(1)
		writeJSON(t, w, catalog.TrackingPlanEventSchema{ID: eventID, Name: eventID})
	})
	defer server.Close()
	dataCatalog := newTrackingPlanSchemasCatalog(t, server, 2)

	trackingPlan, err := dataCatalog.GetTrackingPlanWithSchemas(context.Background(), "tp-123")

	require.NoError(t, err)
	assert.Equal(t, int32(1), schemaRequests.Load())
	assert.Equal(t, []catalog.TrackingPlanEventSchema{
		{ID: "event-1", Name: "event-1"},
		{ID: "event-1", Name: "event-1"},
	}, trackingPlan.Events)
}

func TestUpdateTrackingPlanEvents(t *testing.T) {
	var (
		ctx            = context.Background()
		trackingPlanID = "tp-123"
	)

	createTestEvents := func(count int) []catalog.EventIdentifierDetail {
		events := make([]catalog.EventIdentifierDetail, count)
		for i := range count {
			events[i] = catalog.EventIdentifierDetail{
				ID:                   eventID(i),
				Properties:           []catalog.PropertyIdentifierDetail{},
				AdditionalProperties: false,
				IdentitySection:      "properties",
			}
		}
		return events
	}
	successResponse := `{"id": "tp-123", "name": "Test Plan", "version": 1}`

	t.Run("successfully creates batches of events to update the tracking plan", func(t *testing.T) {
		var receivedBatches [][]string

		calls := []testutils.Call{
			{
				Validate: func(req *http.Request) bool {
					if !assert.Equal(t, "PUT", req.Method) {
						return false
					}
					events := extractEventIDs(t, req)
					receivedBatches = append(receivedBatches, events)
					return assert.Equal(t, []string{"event-0", "event-1"}, events)
				},
				ResponseStatus: 200,
				ResponseBody:   successResponse,
			},
			{
				Validate: func(req *http.Request) bool {
					if !assert.Equal(t, "PUT", req.Method) {
						return false
					}
					events := extractEventIDs(t, req)
					receivedBatches = append(receivedBatches, events)
					return assert.Equal(t, []string{"event-2", "event-3"}, events)
				},
				ResponseStatus: 200,
				ResponseBody:   successResponse,
			},
			{
				Validate: func(req *http.Request) bool {
					if !assert.Equal(t, "PUT", req.Method) {
						return false
					}
					events := extractEventIDs(t, req)
					receivedBatches = append(receivedBatches, events)
					return assert.Equal(t, []string{"event-4"}, events)
				},
				ResponseStatus: 200,
				ResponseBody:   successResponse,
			},
		}

		httpClient := testutils.NewMockHTTPClient(t, calls...)
		apiClient, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		dataCatalog, err := catalog.NewRudderDataCatalog(
			apiClient,
			catalog.WithConcurrency(1),
			catalog.WithEventUpdateBatchSize(2),
		)
		require.NoError(t, err)
		events := createTestEvents(5)

		err = dataCatalog.UpdateTrackingPlanEvents(ctx, trackingPlanID, events, false)

		require.NoError(t, err)
		assert.Len(t, receivedBatches, 3)
		httpClient.AssertNumberOfCalls()
	})

	t.Run("no batches are created if there are no events to update", func(t *testing.T) {
		calls := []testutils.Call{} // No calls expected

		httpClient := testutils.NewMockHTTPClient(t, calls...)
		apiClient, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		dataCatalog, err := catalog.NewRudderDataCatalog(
			apiClient,
			catalog.WithConcurrency(1),
			catalog.WithEventUpdateBatchSize(2),
		)
		require.NoError(t, err)
		events := createTestEvents(0)

		err = dataCatalog.UpdateTrackingPlanEvents(ctx, trackingPlanID, events, false)

		require.NoError(t, err)
		httpClient.AssertNumberOfCalls()
	})

	t.Run("a single batch is created if there are exactly the number of events as the batch size", func(t *testing.T) {
		calls := []testutils.Call{
			{
				Validate: func(req *http.Request) bool {
					if !assert.Equal(t, "PUT", req.Method) {
						return false
					}
					events := extractEventIDs(t, req)
					return assert.Equal(t, []string{"event-0", "event-1"}, events)
				},
				ResponseStatus: 200,
				ResponseBody:   successResponse,
			},
		}

		httpClient := testutils.NewMockHTTPClient(t, calls...)
		apiClient, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		dataCatalog, err := catalog.NewRudderDataCatalog(
			apiClient,
			catalog.WithConcurrency(1),
			catalog.WithEventUpdateBatchSize(2),
		)
		require.NoError(t, err)
		events := createTestEvents(2)

		err = dataCatalog.UpdateTrackingPlanEvents(ctx, trackingPlanID, events, false)

		require.NoError(t, err)
		httpClient.AssertNumberOfCalls()
	})

	t.Run("an error is returned if it fails to update in the middle of the batches", func(t *testing.T) {
		calls := []testutils.Call{
			{
				Validate: func(req *http.Request) bool {
					return assert.Equal(t, "PUT", req.Method)
				},
				ResponseStatus: 200,
				ResponseBody:   successResponse,
			},
			{
				Validate: func(req *http.Request) bool {
					return assert.Equal(t, "PUT", req.Method)
				},
				ResponseStatus: 500,
				ResponseBody:   `{"message": "internal server error"}`,
			},
			// Third call should NOT happen due
			// to error in second batch
		}

		httpClient := testutils.NewMockHTTPClient(t, calls...)
		apiClient, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		dataCatalog, err := catalog.NewRudderDataCatalog(
			apiClient,
			catalog.WithConcurrency(1),
			catalog.WithEventUpdateBatchSize(2),
		)
		require.NoError(t, err)
		events := createTestEvents(5)

		err = dataCatalog.UpdateTrackingPlanEvents(ctx, trackingPlanID, events, false)
		require.Error(t, err)

		httpClient.AssertNumberOfCalls()
	})

}

func newTrackingPlanSchemasCatalog(
	t *testing.T,
	server *httptest.Server,
	concurrency int,
) catalog.DataCatalog {
	t.Helper()

	apiClient, err := client.New(
		"test-token",
		client.WithBaseURL(server.URL),
		client.WithHTTPClient(server.Client()),
	)
	require.NoError(t, err)
	dataCatalog, err := catalog.NewRudderDataCatalog(apiClient, catalog.WithConcurrency(concurrency))
	require.NoError(t, err)
	return dataCatalog
}

func newTrackingPlanSchemasServer(
	t *testing.T,
	eventIDs []string,
	schemaHandler func(http.ResponseWriter, string),
) *httptest.Server {
	t.Helper()

	return newTrackingPlanSchemasServerWithPages(t, [][]string{eventIDs}, nil, schemaHandler)
}

func newTrackingPlanSchemasServerWithPages(
	t *testing.T,
	eventPages [][]string,
	pageDelays map[int]time.Duration,
	schemaHandler func(http.ResponseWriter, string),
) *httptest.Server {
	t.Helper()

	totalEvents := countEventIDs(eventPages)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/catalog/tracking-plans/tp-123":
			writeJSON(t, w, catalog.TrackingPlanWithSchemas{ID: "tp-123", Name: "Test Plan"})
		case "/v2/catalog/tracking-plans/tp-123/events":
			writeEventsPage(t, w, r, eventPages, pageDelays, totalEvents)
		default:
			const eventPathPrefix = "/v2/catalog/tracking-plans/tp-123/events/"
			if !strings.HasPrefix(r.URL.Path, eventPathPrefix) || r.URL.Query().Get("format") != "schema" {
				http.NotFound(w, r)
				return
			}

			eventID := strings.TrimPrefix(r.URL.Path, eventPathPrefix)
			if schemaHandler != nil {
				schemaHandler(w, eventID)
				return
			}
			writeJSON(t, w, catalog.TrackingPlanEventSchema{ID: eventID, Name: eventID})
		}
	}))
}

func writeEventsPage(
	t *testing.T,
	w http.ResponseWriter,
	r *http.Request,
	eventPages [][]string,
	pageDelays map[int]time.Duration,
	totalEvents int,
) {
	t.Helper()

	page := 1
	if pageValue := r.URL.Query().Get("page"); pageValue != "" {
		_, err := fmt.Sscanf(pageValue, "%d", &page)
		if err != nil {
			http.Error(w, "invalid page", http.StatusBadRequest)
			return
		}
	}
	if delay := pageDelays[page]; delay > 0 {
		time.Sleep(delay)
	}
	if page < 1 || page > len(eventPages) {
		http.NotFound(w, r)
		return
	}

	pageEventIDs := eventPages[page-1]
	events := make([]*catalog.TrackingPlanEventResponse, len(pageEventIDs))
	for i, eventID := range pageEventIDs {
		events[i] = &catalog.TrackingPlanEventResponse{ID: eventID}
	}
	writeJSON(t, w, catalog.PaginatedResponse[*catalog.TrackingPlanEventResponse]{
		Data:        events,
		Total:       totalEvents,
		CurrentPage: page,
		PageSize:    max(1, len(pageEventIDs)),
	})
}

func countEventIDs(eventPages [][]string) int {
	count := 0
	for _, page := range eventPages {
		count += len(page)
	}
	return count
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encoding response JSON: %v", err)
	}
}

// Helper function to generate event IDs
func eventID(index int) string {
	return fmt.Sprintf("event-%d", index)
}

// Helper function to extract event IDs from request body
func extractEventIDs(t *testing.T, req *http.Request) []string {
	bodyBytes, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	var payload struct {
		Events []struct {
			ID string `json:"id"`
		} `json:"events"`
	}

	err = json.Unmarshal(bodyBytes, &payload)
	require.NoError(t, err)

	ids := make([]string, len(payload.Events))
	for i, e := range payload.Events {
		ids[i] = e.ID
	}
	return ids
}
