package catalog_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTrackingPlans(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.staging.rudderlabs.com/v2/catalog/tracking-plans", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"trackingPlans": [
					{
						"id": "tp-1",
						"name": "E-commerce Tracking Plan",
						"version": 1,
						"description": "Main tracking plan for e-commerce events",
						"workspaceId": "workspace-1",
						"createdAt": "2021-09-01T00:00:00Z",
						"updatedAt": "2021-09-02T00:00:00Z",
						"creationType": "backend",
						"events": []
					},
					{
						"id": "tp-2",
						"name": "Analytics Tracking Plan",
						"version": 2,
						"description": "Analytics events tracking plan",
						"workspaceId": "workspace-1",
						"createdAt": "2021-09-03T00:00:00Z",
						"updatedAt": "2021-09-04T00:00:00Z",
						"creationType": "frontend",
						"events": []
					}
				]
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	dc := catalog.NewRudderDataCatalog(c)

	trackingPlans, err := dc.ListTrackingPlans(ctx)
	require.NoError(t, err)
	require.Len(t, trackingPlans, 2)

	// Verify first tracking plan
	assert.Equal(t, "tp-1", trackingPlans[0].ID)
	assert.Equal(t, "E-commerce Tracking Plan", trackingPlans[0].Name)
	assert.Equal(t, 1, trackingPlans[0].Version)
	assert.Equal(t, "Main tracking plan for e-commerce events", *trackingPlans[0].Description)
	assert.Equal(t, "workspace-1", trackingPlans[0].WorkspaceID)
	assert.Equal(t, "backend", trackingPlans[0].CreationType)

	// Verify second tracking plan
	assert.Equal(t, "tp-2", trackingPlans[1].ID)
	assert.Equal(t, "Analytics Tracking Plan", trackingPlans[1].Name)
	assert.Equal(t, 2, trackingPlans[1].Version)
}

func TestListTrackingPlansWithFilter(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				expectedURL := "https://api.staging.rudderlabs.com/v2/catalog/tracking-plans?ids=tp-1%2Ctp-2"
				return testutils.ValidateRequest(t, req, "GET", expectedURL, "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"trackingPlans": [
					{
						"id": "tp-1",
						"name": "E-commerce Tracking Plan",
						"version": 1,
						"description": "Main tracking plan for e-commerce events",
						"workspaceId": "workspace-1",
						"createdAt": "2021-09-01T00:00:00Z",
						"updatedAt": "2021-09-02T00:00:00Z",
						"creationType": "backend",
						"events": []
					}
				]
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	dc := catalog.NewRudderDataCatalog(c)

	trackingPlans, err := dc.ListTrackingPlansWithFilter(ctx, []string{"tp-1", "tp-2"})
	require.NoError(t, err)
	require.Len(t, trackingPlans, 1)
	assert.Equal(t, "tp-1", trackingPlans[0].ID)
}

func TestListEvents(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				expectedURL := "https://api.staging.rudderlabs.com/v2/catalog/events?trackingPlanIds=tp-1%2Ctp-2&page=1"
				return testutils.ValidateRequest(t, req, "GET", expectedURL, "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"data": [
					{
						"id": "event-1",
						"name": "Product Viewed",
						"eventType": "track",
						"description": "User viewed a product",
						"categoryId": "cat-1",
						"workspaceId": "workspace-1"
					},
					{
						"id": "event-2",
						"name": "Purchase Completed",
						"eventType": "track",
						"description": "User completed a purchase",
						"categoryId": "cat-1",
						"workspaceId": "workspace-1"
					}
				],
				"total": 2,
				"currentPage": 1,
				"pageSize": 10
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	dc := catalog.NewRudderDataCatalog(c)

	response, err := dc.ListEvents(ctx, []string{"tp-1", "tp-2"}, 1)
	require.NoError(t, err)
	require.Len(t, response.Data, 2)
	assert.Equal(t, 2, response.Total)
	assert.Equal(t, 1, response.CurrentPage)

	// Verify first event
	assert.Equal(t, "event-1", response.Data[0].ID)
	assert.Equal(t, "Product Viewed", response.Data[0].Name)
	assert.Equal(t, "track", response.Data[0].EventType)
}

func TestListProperties(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				expectedURL := "https://api.staging.rudderlabs.com/v2/catalog/properties?trackingPlanIds=tp-1&page=1"
				return testutils.ValidateRequest(t, req, "GET", expectedURL, "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"data": [
					{
						"id": "prop-1",
						"name": "Product ID",
						"type": "string",
						"description": "Unique identifier for the product",
						"workspaceId": "workspace-1"
					}
				],
				"total": 1,
				"currentPage": 1,
				"pageSize": 10
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	dc := catalog.NewRudderDataCatalog(c)

	response, err := dc.ListProperties(ctx, []string{"tp-1"}, 1)
	require.NoError(t, err)
	require.Len(t, response.Data, 1)
	assert.Equal(t, "prop-1", response.Data[0].ID)
}

func TestListCustomTypes(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.staging.rudderlabs.com/v2/catalog/custom-types?page=1", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"data": [
					{
						"id": "ct-1",
						"name": "Email Type",
						"type": "string",
						"description": "Custom email validation type",
						"workspaceId": "workspace-1"
					}
				],
				"total": 1,
				"currentPage": 1,
				"pageSize": 10
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	dc := catalog.NewRudderDataCatalog(c)

	response, err := dc.ListCustomTypes(ctx, 1)
	require.NoError(t, err)
	require.Len(t, response.Data, 1)
	assert.Equal(t, "ct-1", response.Data[0].ID)
}

func TestListCategories(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.staging.rudderlabs.com/v2/catalog/categories?page=1", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"data": [
					{
						"id": "cat-1",
						"name": "E-commerce",
						"workspaceId": "workspace-1"
					}
				],
				"total": 1,
				"currentPage": 1,
				"pageSize": 10
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	dc := catalog.NewRudderDataCatalog(c)

	response, err := dc.ListCategories(ctx, 1)
	require.NoError(t, err)
	require.Len(t, response.Data, 1)
	assert.Equal(t, "cat-1", response.Data[0].ID)
}