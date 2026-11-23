package catalog_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTrackingPlan(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "POST", catalogURL("v2/catalog/tracking-plans"), `{"name":"TP","description":"desc","externalId":"ext-1"}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"tp-1","name":"TP"}`,
	})

	trackingPlan, err := dc.CreateTrackingPlan(context.Background(), catalog.TrackingPlanCreate{Name: "TP", Description: "desc", ExternalID: "ext-1"})
	require.NoError(t, err)
	assert.Equal(t, &catalog.TrackingPlan{ID: "tp-1", Name: "TP"}, trackingPlan)
	httpClient.AssertNumberOfCalls()
}

func TestUpdateTrackingPlan(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "PUT", catalogURL("v2/catalog/tracking-plans/tp-1"), `{"name":"TP2","description":"desc2"}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"tp-1","name":"TP2"}`,
	})

	trackingPlan, err := dc.UpdateTrackingPlan(context.Background(), "tp-1", "TP2", "desc2")
	require.NoError(t, err)
	assert.Equal(t, &catalog.TrackingPlan{ID: "tp-1", Name: "TP2"}, trackingPlan)
	httpClient.AssertNumberOfCalls()
}

func TestUpsertTrackingPlan(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "PATCH", catalogURL("v2/catalog/tracking-plans/tp-1/events"), `{"name":"evt","description":"event","eventType":"track","identitySection":"properties","rules":{"type":"object","properties":{}}}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"tp-1","name":"TP2"}`,
	})

	trackingPlan, err := dc.UpsertTrackingPlan(context.Background(), "tp-1", catalog.TrackingPlanUpsertEvent{
		Name:            "evt",
		Description:     "event",
		EventType:       "track",
		IdentitySection: catalog.IdentitySectionProperties,
		Rules:           catalog.TrackingPlanUpsertEventRules{Type: "object"},
	})
	require.NoError(t, err)
	assert.Equal(t, &catalog.TrackingPlan{ID: "tp-1", Name: "TP2"}, trackingPlan)
	httpClient.AssertNumberOfCalls()
}

func TestGetTrackingPlan(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "GET", catalogURL("v2/catalog/tracking-plans/tp-1?rebuildSchemas=false"), "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"tp-1","name":"TP2"}`,
	})

	trackingPlan, err := dc.GetTrackingPlan(context.Background(), "tp-1")
	require.NoError(t, err)
	assert.Equal(t, &catalog.TrackingPlan{ID: "tp-1", Name: "TP2"}, trackingPlan)
	httpClient.AssertNumberOfCalls()
}

func TestSetTrackingPlanExternalID(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "PUT", catalogURL("v2/catalog/tracking-plans/tp-1/external-id"), `{"externalId":"ext-2"}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{}`,
	})

	err := dc.SetTrackingPlanExternalId(context.Background(), "tp-1", "ext-2")
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestDeleteTrackingPlanEvent(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "DELETE", catalogURL("v2/catalog/tracking-plans/tp-1/events/evt-1"), "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{}`,
	})

	err := dc.DeleteTrackingPlanEvent(context.Background(), "tp-1", "evt-1")
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestDeleteTrackingPlan(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "DELETE", catalogURL("v2/catalog/tracking-plans/tp-1"), "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{}`,
	})

	err := dc.DeleteTrackingPlan(context.Background(), "tp-1")
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestGetTrackingPlans(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "GET", catalogURL("v2/catalog/tracking-plans?hasExternalId=true"), "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{"trackingPlans":[{"id":"tp-1","name":"TP1"},{"id":"tp-2","name":"TP2"}]}`,
	})

	trackingPlans, err := dc.GetTrackingPlans(context.Background(), catalog.ListOptions{HasExternalID: boolPtr(true)})
	require.NoError(t, err)
	require.Len(t, trackingPlans, 2)
	assert.Equal(t, &catalog.TrackingPlan{ID: "tp-1", Name: "TP1"}, trackingPlans[0])
	assert.Equal(t, &catalog.TrackingPlan{ID: "tp-2", Name: "TP2"}, trackingPlans[1])
	httpClient.AssertNumberOfCalls()
}

func TestGetTrackingPlanWithSchemas(t *testing.T) {
	dc, httpClient := newDataCatalog(t,
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"id":"tp-1","name":"TP1"}`},
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"evt-1"}],"total":1,"currentPage":1,"pageSize":50}`},
		testutils.Call{
			Validate: func(req *http.Request) bool {
				return assertCall(t, req, "GET", catalogURL("v2/catalog/tracking-plans/tp-1/events/evt-1?format=schema"), "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"evt-1","name":"evt"}`,
		},
	)

	trackingPlan, err := dc.GetTrackingPlanWithSchemas(context.Background(), "tp-1")
	require.NoError(t, err)
	assert.Equal(t, "tp-1", trackingPlan.ID)
	require.Len(t, trackingPlan.Events, 1)
	assert.Equal(t, "evt-1", trackingPlan.Events[0].ID)
	httpClient.AssertNumberOfCalls()
}

func TestGetTrackingPlanWithIdentifiers(t *testing.T) {
	dc, httpClient := newDataCatalog(t,
		testutils.Call{
			Validate: func(req *http.Request) bool {
				return assertCall(t, req, "GET", catalogURL("v2/catalog/tracking-plans/tp-1?rebuildSchemas=true"), "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"tp-1","name":"TP1"}`,
		},
		testutils.Call{
			Validate: func(req *http.Request) bool {
				return assertCall(t, req, "GET", catalogURL("v2/catalog/tracking-plans/tp-1/events?rebuildSchemas=true"), "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{"data":[{"id":"evt-1"}],"total":1,"currentPage":1,"pageSize":50}`,
		},
		testutils.Call{
			Validate: func(req *http.Request) bool {
				return assertCall(t, req, "GET", catalogURL("v2/catalog/tracking-plans/tp-1/events/evt-1?format=properties&rebuildSchemas=true"), "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"evt-1","name":"evt"}`,
		},
	)

	trackingPlan, err := dc.GetTrackingPlanWithIdentifiers(context.Background(), "tp-1", true)
	require.NoError(t, err)
	assert.Equal(t, "tp-1", trackingPlan.ID)
	require.Len(t, trackingPlan.Events, 1)
	assert.Equal(t, "evt-1", trackingPlan.Events[0].ID)
	httpClient.AssertNumberOfCalls()
}

func TestGetTrackingPlansWithIdentifiers(t *testing.T) {
	dc, httpClient := newDataCatalog(t,
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"trackingPlans":[{"id":"tp-1","name":"TP1"}]}`},
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"id":"tp-1","name":"TP1"}`},
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"evt-1"}],"total":1,"currentPage":1,"pageSize":50}`},
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"id":"evt-1","name":"evt"}`},
	)

	trackingPlans, err := dc.GetTrackingPlansWithIdentifiers(context.Background(), catalog.ListOptions{RebuildSchemas: false})
	require.NoError(t, err)
	require.Len(t, trackingPlans, 1)
	assert.Equal(t, "tp-1", trackingPlans[0].ID)
	require.Len(t, trackingPlans[0].Events, 1)
	assert.Equal(t, "evt-1", trackingPlans[0].Events[0].ID)
	httpClient.AssertNumberOfCalls()
}

func TestGetTrackingPlanEventSchema(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "GET", catalogURL("v2/catalog/tracking-plans/tp-1/events/evt-1?format=schema"), "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"evt-1","name":"evt"}`,
	})

	schema, err := dc.GetTrackingPlanEventSchema(context.Background(), "tp-1", "evt-1")
	require.NoError(t, err)
	assert.Equal(t, &catalog.TrackingPlanEventSchema{ID: "evt-1", Name: "evt"}, schema)
	httpClient.AssertNumberOfCalls()
}

func TestTrackingPlanErrors(t *testing.T) {
	tests := []struct {
		name          string
		calls         []testutils.Call
		operation     func(dc catalog.DataCatalog) error
		expectedError string
	}{
		{
			name:  "GetTrackingPlans request error",
			calls: []testutils.Call{{ResponseError: errors.New("network")}},
			operation: func(dc catalog.DataCatalog) error {
				_, err := dc.GetTrackingPlans(context.Background(), catalog.ListOptions{})
				return err
			},
			expectedError: "executing http request to fetch tracking plans",
		},
		{
			name:  "GetTrackingPlan decode error",
			calls: []testutils.Call{{ResponseStatus: 200, ResponseBody: "{"}},
			operation: func(dc catalog.DataCatalog) error {
				_, err := dc.GetTrackingPlan(context.Background(), "tp-1")
				return err
			},
			expectedError: "decoding tracking plan response",
		},
		{
			name:  "UpsertTrackingPlan request error",
			calls: []testutils.Call{{ResponseError: errors.New("network")}},
			operation: func(dc catalog.DataCatalog) error {
				_, err := dc.UpsertTrackingPlan(context.Background(), "tp-1", catalog.TrackingPlanUpsertEvent{Name: "evt"})
				return err
			},
			expectedError: "executing http request",
		},
		{
			name: "GetTrackingPlanWithSchemas event schema fetch error",
			calls: []testutils.Call{
				{ResponseStatus: 200, ResponseBody: `{"id":"tp-1"}`},
				{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"evt-1"}],"total":1,"currentPage":1,"pageSize":50}`},
				{ResponseStatus: 500, ResponseBody: `{"error":"bad"}`},
			},
			operation: func(dc catalog.DataCatalog) error {
				_, err := dc.GetTrackingPlanWithSchemas(context.Background(), "tp-1")
				return err
			},
			expectedError: "fetching event schema",
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

func TestUpdateTrackingPlanEvents(t *testing.T) {
	var (
		ctx            = context.Background()
		trackingPlanID = "tp-123"
		batchOpts      = []catalog.Opts{catalog.WithEventUpdateBatchSize(2)}
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

		dc, httpClient := newDataCatalogWithOptions(t, batchOpts,
			testutils.Call{
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
			testutils.Call{
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
			testutils.Call{
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
		)

		err := dc.UpdateTrackingPlanEvents(ctx, trackingPlanID, createTestEvents(5), false)
		require.NoError(t, err)
		assert.Len(t, receivedBatches, 3)
		httpClient.AssertNumberOfCalls()
	})

	t.Run("no batches are created if there are no events to update", func(t *testing.T) {
		dc, httpClient := newDataCatalogWithOptions(t, batchOpts)
		err := dc.UpdateTrackingPlanEvents(ctx, trackingPlanID, createTestEvents(0), false)
		require.NoError(t, err)
		httpClient.AssertNumberOfCalls()
	})

	t.Run("a single batch is created if there are exactly the number of events as the batch size", func(t *testing.T) {
		dc, httpClient := newDataCatalogWithOptions(t, batchOpts, testutils.Call{
			Validate: func(req *http.Request) bool {
				if !assert.Equal(t, "PUT", req.Method) {
					return false
				}
				return assert.Equal(t, []string{"event-0", "event-1"}, extractEventIDs(t, req))
			},
			ResponseStatus: 200,
			ResponseBody:   successResponse,
		})

		err := dc.UpdateTrackingPlanEvents(ctx, trackingPlanID, createTestEvents(2), false)
		require.NoError(t, err)
		httpClient.AssertNumberOfCalls()
	})

	t.Run("an error is returned if it fails to update in the middle of the batches", func(t *testing.T) {
		dc, httpClient := newDataCatalogWithOptions(t, batchOpts,
			testutils.Call{
				Validate:       func(req *http.Request) bool { return assert.Equal(t, "PUT", req.Method) },
				ResponseStatus: 200,
				ResponseBody:   successResponse,
			},
			testutils.Call{
				Validate:       func(req *http.Request) bool { return assert.Equal(t, "PUT", req.Method) },
				ResponseStatus: 500,
				ResponseBody:   `{"message": "internal server error"}`,
			},
		)

		err := dc.UpdateTrackingPlanEvents(ctx, trackingPlanID, createTestEvents(5), false)
		require.Error(t, err)
		httpClient.AssertNumberOfCalls()
	})
}

func eventID(index int) string {
	return fmt.Sprintf("event-%d", index)
}

func extractEventIDs(t *testing.T, req *http.Request) []string {
	t.Helper()

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

		event := catalog.TrackingPlanEventSchema{
			ID:              "ev_1",
			Name:            "Signed Up",
			Description:     "A user signed up",
			EventType:       "track",
			IdentitySection: "properties",
		}
		event.Rules.Type = "object"
		event.Rules.Properties = map[string]any{}

		assert.Equal(t, &catalog.TrackingPlanWithSchemas{
			ID:      "tp-123",
			Name:    "Test Plan",
			Version: 3,
			Events:  []catalog.TrackingPlanEventSchema{event},
		}, tp)
	})

	t.Run("fails when the server returns another version", func(t *testing.T) {
		dataCatalog := newCatalog(t, func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `{"id": "tp-123", "name": "Test Plan", "version": 5, "events": []}`)
		})

		_, err := dataCatalog.GetTrackingPlanVersionWithSchemas(context.Background(), "tp-123", 3)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "version 5")
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

func TestGetTrackingPlanWithSchemasStopsAtFirstFailure(t *testing.T) {
	const eventCount = 40

	for _, concurrency := range []int{1, 4} {
		t.Run(fmt.Sprintf("concurrency %d", concurrency), func(t *testing.T) {
			var schemaRequests atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				switch {
				case req.URL.Query().Get("format") == "schema":
					schemaRequests.Add(1)
					http.Error(w, `{"message": "boom"}`, http.StatusInternalServerError)
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

			require.Error(t, err)
			assert.Nil(t, tp)
			// Requests already in flight finish, but no new ones start after the failure
			assert.LessOrEqual(t, int(schemaRequests.Load()), concurrency)
			assert.Contains(t, err.Error(), "fetching event schemas on tracking plan")
			assert.NotContains(t, err.Error(), "overall job failed", "skipped events must not add a line each")
		})
	}
}

// newPlanCatalog serves a tracking plan whose event list holds the given IDs,
// and hands every schema request to schema.
func newPlanCatalog(t *testing.T, eventIDs []string, concurrency int, schema http.HandlerFunc) catalog.DataCatalog {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.URL.Query().Get("format") == "schema":
			schema(w, req)
		case strings.HasSuffix(req.URL.Path, "/events"):
			items := make([]string, len(eventIDs))
			for i, id := range eventIDs {
				items[i] = fmt.Sprintf(`{"id": %q}`, id)
			}
			fmt.Fprintf(w, `{"data": [%s], "total": %d, "currentPage": 1, "pageSize": 50}`, strings.Join(items, ","), len(eventIDs))
		default:
			fmt.Fprint(w, `{"id": "tp-123", "name": "Test Plan", "version": 4}`)
		}
	}))
	t.Cleanup(server.Close)

	apiClient, err := client.New("test-token", client.WithBaseURL(server.URL))
	require.NoError(t, err)

	dataCatalog, err := catalog.NewRudderDataCatalog(apiClient, catalog.WithConcurrency(concurrency))
	require.NoError(t, err)
	return dataCatalog
}

func TestGetTrackingPlanWithSchemasEdgeCases(t *testing.T) {
	t.Run("a null schema body is an error, not a panic", func(t *testing.T) {
		dataCatalog := newPlanCatalog(t, []string{"event-0"}, 2, func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `null`)
		})

		tp, err := dataCatalog.GetTrackingPlanWithSchemas(context.Background(), "tp-123")

		require.Error(t, err)
		assert.Nil(t, tp)
		assert.Contains(t, err.Error(), "event-0")
	})

	t.Run("a failure does not wait for a request that never answers", func(t *testing.T) {
		release := make(chan struct{})

		dataCatalog := newPlanCatalog(t, []string{"event-0", "event-1"}, 10, func(w http.ResponseWriter, req *http.Request) {
			if strings.Contains(req.URL.Path, "event-0") {
				http.Error(w, `{"message": "boom"}`, http.StatusInternalServerError)
				return
			}
			select {
			case <-req.Context().Done():
			case <-release:
			}
		})
		// Runs before the server closes, so a handler that is still waiting cannot hang the test
		t.Cleanup(func() { close(release) })

		type result struct {
			tp  *catalog.TrackingPlanWithSchemas
			err error
		}
		done := make(chan result, 1)
		go func() {
			tp, err := dataCatalog.GetTrackingPlanWithSchemas(context.Background(), "tp-123")
			done <- result{tp, err}
		}()

		select {
		case got := <-done:
			require.Error(t, got.err)
			assert.Nil(t, got.tp)
			assert.Contains(t, got.err.Error(), "500", "the error carries the real cause")
			assert.NotContains(t, got.err.Error(), "context canceled", "aborting the other request must not add noise")
		case <-time.After(5 * time.Second):
			t.Fatal("the call kept waiting for a request that never answers after another one failed")
		}
	})

	t.Run("an event listed twice is returned twice and fetched once", func(t *testing.T) {
		var schemaRequests atomic.Int32

		dataCatalog := newPlanCatalog(t, []string{"event-0", "event-0"}, 2, func(w http.ResponseWriter, _ *http.Request) {
			schemaRequests.Add(1)
			fmt.Fprint(w, `{"id": "event-0", "name": "Signed Up"}`)
		})

		tp, err := dataCatalog.GetTrackingPlanWithSchemas(context.Background(), "tp-123")

		require.NoError(t, err)
		require.Len(t, tp.Events, 2)
		assert.Equal(t, "Signed Up", tp.Events[0].Name)
		assert.Equal(t, "Signed Up", tp.Events[1].Name)
		assert.Equal(t, int32(1), schemaRequests.Load())
	})
}
