package catalog_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
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

func TestGetTrackingPlanWithSchemasFetchesSchemasConcurrently(t *testing.T) {
	const (
		eventCount  = 6
		concurrency = 3
	)

	var (
		inFlight    atomic.Int32
		maxInFlight atomic.Int32
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.URL.Query().Get("format") == "schema":
			current := inFlight.Add(1)
			defer inFlight.Add(-1)
			for {
				seen := maxInFlight.Load()
				if current <= seen || maxInFlight.CompareAndSwap(seen, current) {
					break
				}
			}
			// Hold the request open so overlapping fetches are observable
			time.Sleep(30 * time.Millisecond)
			fmt.Fprintf(w, `{"id": %q}`, path.Base(req.URL.Path))
		case strings.HasSuffix(req.URL.Path, "/events"):
			ids := make([]string, eventCount)
			for i := range eventCount {
				ids[i] = fmt.Sprintf(`{"id": %q}`, eventID(i))
			}
			fmt.Fprintf(w, `{"data": [%s], "total": %d, "currentPage": 1, "pageSize": 50}`, strings.Join(ids, ","), eventCount)
		default:
			fmt.Fprint(w, `{"id": "tp-123", "name": "Test Plan", "version": 4}`)
		}
	}))
	defer server.Close()

	apiClient, err := client.New("test-token", client.WithBaseURL(server.URL))
	require.NoError(t, err)

	dataCatalog, err := catalog.NewRudderDataCatalog(apiClient, catalog.WithConcurrency(concurrency))
	require.NoError(t, err)

	tp, err := dataCatalog.GetTrackingPlanWithSchemas(context.Background(), "tp-123")
	require.NoError(t, err)

	gotIDs := make([]string, len(tp.Events))
	for i, ev := range tp.Events {
		gotIDs[i] = ev.ID
	}
	assert.Equal(t, []string{"event-0", "event-1", "event-2", "event-3", "event-4", "event-5"}, gotIDs)
	assert.Greater(t, maxInFlight.Load(), int32(1), "schemas should be fetched concurrently")
	assert.LessOrEqual(t, maxInFlight.Load(), int32(concurrency), "concurrency limit should hold")
}

func TestGetTrackingPlanVersionWithSchemas(t *testing.T) {
	newCatalog := func(t *testing.T, handler http.HandlerFunc) catalog.DataCatalog {
		t.Helper()

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		apiClient, err := client.New("test-token", client.WithBaseURL(server.URL))
		require.NoError(t, err)

		dataCatalog, err := catalog.NewRudderDataCatalog(apiClient)
		require.NoError(t, err)
		return dataCatalog
	}

	t.Run("fetches the pinned version and its schemas in one request", func(t *testing.T) {
		var requests []string
		dataCatalog := newCatalog(t, func(w http.ResponseWriter, req *http.Request) {
			requests = append(requests, req.URL.String())
			fmt.Fprint(w, `{
				"id": "tp-123", "name": "Test Plan", "version": 3,
				"events": [{
					"id": "ev_1", "name": "Signed Up", "description": "A user signed up",
					"eventType": "track", "identitySection": "properties",
					"rules": {"type": "object", "properties": {}}
				}]
			}`)
		})

		tp, err := dataCatalog.GetTrackingPlanVersionWithSchemas(context.Background(), "tp-123", 3)
		require.NoError(t, err)

		assert.Equal(t, []string{"/v2/catalog/tracking-plans/tp-123?version=3&fetchSchemas=true"}, requests)
		assert.Equal(t, 3, tp.Version)
		require.Len(t, tp.Events, 1)
		assert.Equal(t, "Signed Up", tp.Events[0].Name)
		assert.Equal(t, "A user signed up", tp.Events[0].Description)
		assert.Equal(t, "properties", tp.Events[0].IdentitySection)
	})

	t.Run("accepts a version with no events", func(t *testing.T) {
		dataCatalog := newCatalog(t, func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `{"id": "tp-123", "name": "Empty", "version": 1, "events": []}`)
		})

		tp, err := dataCatalog.GetTrackingPlanVersionWithSchemas(context.Background(), "tp-123", 1)
		require.NoError(t, err)
		assert.Empty(t, tp.Events)
	})

	t.Run("fails when the server ignores fetchSchemas", func(t *testing.T) {
		dataCatalog := newCatalog(t, func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `{"id": "tp-123", "name": "Test Plan", "version": 3}`)
		})

		_, err := dataCatalog.GetTrackingPlanVersionWithSchemas(context.Background(), "tp-123", 3)
		assert.ErrorIs(t, err, catalog.ErrVersionedSchemasUnsupported)
	})
}
