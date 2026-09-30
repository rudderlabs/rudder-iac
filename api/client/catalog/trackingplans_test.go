package catalog_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

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
