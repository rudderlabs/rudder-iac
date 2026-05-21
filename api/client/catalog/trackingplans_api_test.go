package catalog_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrackingPlanCRUDAndExternalID(t *testing.T) {
	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "POST", "v2/catalog/tracking-plans", `{"name":"TP","description":"desc","externalId":"ext-1"}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"tp-1","name":"TP"}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "PUT", "v2/catalog/tracking-plans/tp-1", `{"name":"TP2","description":"desc2"}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"tp-1","name":"TP2"}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "PATCH", "v2/catalog/tracking-plans/tp-1/events", `{"name":"evt","description":"event","eventType":"track","identitySection":"properties","rules":{"type":"object","properties":{}}}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"tp-1","name":"TP2"}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "GET", "v2/catalog/tracking-plans/tp-1?rebuildSchemas=false", "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"tp-1","name":"TP2"}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "PUT", "v2/catalog/tracking-plans/tp-1/external-id", `{"externalId":"ext-2"}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "DELETE", "v2/catalog/tracking-plans/tp-1/events/evt-1", "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "DELETE", "v2/catalog/tracking-plans/tp-1", "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{}`,
		},
	}

	dc, httpClient := newDataCatalog(t, nil, calls...)
	trackingPlan, err := dc.CreateTrackingPlan(context.Background(), catalog.TrackingPlanCreate{Name: "TP", Description: "desc", ExternalID: "ext-1"})
	require.NoError(t, err)
	assert.Equal(t, &catalog.TrackingPlan{ID: "tp-1", Name: "TP"}, trackingPlan)

	trackingPlan, err = dc.UpdateTrackingPlan(context.Background(), "tp-1", "TP2", "desc2")
	require.NoError(t, err)
	assert.Equal(t, "TP2", trackingPlan.Name)

	trackingPlan, err = dc.UpsertTrackingPlan(context.Background(), "tp-1", catalog.TrackingPlanUpsertEvent{
		Name:            "evt",
		Description:     "event",
		EventType:       "track",
		IdentitySection: catalog.IdentitySectionProperties,
		Rules:           catalog.TrackingPlanUpsertEventRules{Type: "object"},
	})
	require.NoError(t, err)
	assert.Equal(t, "tp-1", trackingPlan.ID)

	trackingPlan, err = dc.GetTrackingPlan(context.Background(), "tp-1")
	require.NoError(t, err)
	assert.Equal(t, "tp-1", trackingPlan.ID)

	err = dc.SetTrackingPlanExternalId(context.Background(), "tp-1", "ext-2")
	require.NoError(t, err)

	err = dc.DeleteTrackingPlanEvent(context.Background(), "tp-1", "evt-1")
	require.NoError(t, err)

	err = dc.DeleteTrackingPlan(context.Background(), "tp-1")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestTrackingPlanReadVariants(t *testing.T) {
	t.Run("get tracking plans list", func(t *testing.T) {
		hasExternalID := true
		calls := []testutils.Call{{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "GET", "v2/catalog/tracking-plans?hasExternalId=true", "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{"trackingPlans":[{"id":"tp-1","name":"TP1"},{"id":"tp-2","name":"TP2"}]}`,
		}}

		dc, httpClient := newDataCatalog(t, nil, calls...)
		trackingPlans, err := dc.GetTrackingPlans(context.Background(), catalog.ListOptions{HasExternalID: &hasExternalID})
		require.NoError(t, err)
		require.Len(t, trackingPlans, 2)
		assert.Equal(t, &catalog.TrackingPlan{ID: "tp-1", Name: "TP1"}, trackingPlans[0])
		httpClient.AssertNumberOfCalls()
	})

	t.Run("get tracking plan with schemas", func(t *testing.T) {
		calls := []testutils.Call{
			{ResponseStatus: 200, ResponseBody: `{"id":"tp-1","name":"TP1"}`},
			{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"evt-1"}],"total":1,"currentPage":1,"pageSize":50}`},
			{
				Validate: func(req *http.Request) bool {
					return validateRequest(t, req, "GET", "v2/catalog/tracking-plans/tp-1/events/evt-1?format=schema", "")
				},
				ResponseStatus: 200,
				ResponseBody:   `{"id":"evt-1","name":"evt"}`,
			},
		}

		dc, httpClient := newDataCatalog(t, nil, calls...)
		trackingPlan, err := dc.GetTrackingPlanWithSchemas(context.Background(), "tp-1")
		require.NoError(t, err)
		assert.Equal(t, "tp-1", trackingPlan.ID)
		require.Len(t, trackingPlan.Events, 1)
		assert.Equal(t, "evt-1", trackingPlan.Events[0].ID)
		httpClient.AssertNumberOfCalls()
	})

	t.Run("get tracking plan with identifiers", func(t *testing.T) {
		calls := []testutils.Call{
			{
				Validate: func(req *http.Request) bool {
					return validateRequest(t, req, "GET", "v2/catalog/tracking-plans/tp-1?rebuildSchemas=true", "")
				},
				ResponseStatus: 200,
				ResponseBody:   `{"id":"tp-1","name":"TP1"}`,
			},
			{
				Validate: func(req *http.Request) bool {
					return validateRequest(t, req, "GET", "v2/catalog/tracking-plans/tp-1/events?rebuildSchemas=true", "")
				},
				ResponseStatus: 200,
				ResponseBody:   `{"data":[{"id":"evt-1"}],"total":1,"currentPage":1,"pageSize":50}`,
			},
			{
				Validate: func(req *http.Request) bool {
					return validateRequest(t, req, "GET", "v2/catalog/tracking-plans/tp-1/events/evt-1?format=properties&rebuildSchemas=true", "")
				},
				ResponseStatus: 200,
				ResponseBody:   `{"id":"evt-1","name":"evt"}`,
			},
		}

		dc, httpClient := newDataCatalog(t, nil, calls...)
		trackingPlan, err := dc.GetTrackingPlanWithIdentifiers(context.Background(), "tp-1", true)
		require.NoError(t, err)
		assert.Equal(t, "tp-1", trackingPlan.ID)
		require.Len(t, trackingPlan.Events, 1)
		assert.Equal(t, "evt-1", trackingPlan.Events[0].ID)
		httpClient.AssertNumberOfCalls()
	})

	t.Run("get tracking plans with identifiers", func(t *testing.T) {
		calls := []testutils.Call{
			{ResponseStatus: 200, ResponseBody: `{"trackingPlans":[{"id":"tp-1","name":"TP1"}]}`},
			{ResponseStatus: 200, ResponseBody: `{"id":"tp-1","name":"TP1"}`},
			{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"evt-1"}],"total":1,"currentPage":1,"pageSize":50}`},
			{ResponseStatus: 200, ResponseBody: `{"id":"evt-1","name":"evt"}`},
		}

		dc, httpClient := newDataCatalog(t, nil, calls...)
		trackingPlans, err := dc.GetTrackingPlansWithIdentifiers(context.Background(), catalog.ListOptions{RebuildSchemas: false})
		require.NoError(t, err)
		require.Len(t, trackingPlans, 1)
		assert.Equal(t, "tp-1", trackingPlans[0].ID)
		require.Len(t, trackingPlans[0].Events, 1)
		assert.Equal(t, "evt-1", trackingPlans[0].Events[0].ID)
		httpClient.AssertNumberOfCalls()
	})

	t.Run("get tracking plan event schema", func(t *testing.T) {
		calls := []testutils.Call{{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "GET", "v2/catalog/tracking-plans/tp-1/events/evt-1?format=schema", "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"evt-1","name":"evt"}`,
		}}

		dc, httpClient := newDataCatalog(t, nil, calls...)
		schema, err := dc.GetTrackingPlanEventSchema(context.Background(), "tp-1", "evt-1")
		require.NoError(t, err)
		assert.Equal(t, &catalog.TrackingPlanEventSchema{ID: "evt-1", Name: "evt"}, schema)
		httpClient.AssertNumberOfCalls()
	})
}

func TestTrackingPlanFailures(t *testing.T) {
	t.Run("list request error", func(t *testing.T) {
		calls := []testutils.Call{{ResponseError: errors.New("network")}}
		dc, httpClient := newDataCatalog(t, nil, calls...)
		_, err := dc.GetTrackingPlans(context.Background(), catalog.ListOptions{})
		require.Error(t, err)
		assert.ErrorContains(t, err, "executing http request to fetch tracking plans")
		httpClient.AssertNumberOfCalls()
	})

	t.Run("tracking plan decode error", func(t *testing.T) {
		calls := []testutils.Call{{ResponseStatus: 200, ResponseBody: "{"}}
		dc, httpClient := newDataCatalog(t, nil, calls...)
		_, err := dc.GetTrackingPlan(context.Background(), "tp-1")
		require.Error(t, err)
		assert.ErrorContains(t, err, "decoding tracking plan response")
		httpClient.AssertNumberOfCalls()
	})

	t.Run("upsert request error", func(t *testing.T) {
		calls := []testutils.Call{{ResponseError: errors.New("network")}}
		dc, httpClient := newDataCatalog(t, nil, calls...)
		_, err := dc.UpsertTrackingPlan(context.Background(), "tp-1", catalog.TrackingPlanUpsertEvent{Name: "evt"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "executing http request")
		httpClient.AssertNumberOfCalls()
	})

	t.Run("get with schemas event schema fetch error", func(t *testing.T) {
		calls := []testutils.Call{
			{ResponseStatus: 200, ResponseBody: `{"id":"tp-1"}`},
			{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"evt-1"}],"total":1,"currentPage":1,"pageSize":50}`},
			{ResponseStatus: 500, ResponseBody: `{"error":"bad"}`},
		}
		dc, httpClient := newDataCatalog(t, nil, calls...)
		_, err := dc.GetTrackingPlanWithSchemas(context.Background(), "tp-1")
		require.Error(t, err)
		assert.ErrorContains(t, err, "fetching event schema")
		httpClient.AssertNumberOfCalls()
	})
}
