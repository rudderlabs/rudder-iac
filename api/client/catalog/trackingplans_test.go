package catalog_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

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

		dataCatalog, err := catalog.NewRudderDataCatalog(apiClient, catalog.Options{
			Concurrency:          1,
			EventUpdateBatchSize: 2,
		})
		require.NoError(t, err)
		events := createTestEvents(5)

		err = dataCatalog.UpdateTrackingPlanEvents(ctx, trackingPlanID, events)

		require.NoError(t, err)
		assert.Len(t, receivedBatches, 3)
		httpClient.AssertNumberOfCalls()
	})

	t.Run("no batches are created if there are no events to update", func(t *testing.T) {
		calls := []testutils.Call{} // No calls expected

		httpClient := testutils.NewMockHTTPClient(t, calls...)
		apiClient, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		dataCatalog, err := catalog.NewRudderDataCatalog(apiClient, catalog.Options{
			Concurrency:          1,
			EventUpdateBatchSize: 2,
		})
		require.NoError(t, err)
		events := createTestEvents(0)

		err = dataCatalog.UpdateTrackingPlanEvents(ctx, trackingPlanID, events)

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

		dataCatalog, err := catalog.NewRudderDataCatalog(apiClient, catalog.Options{
			Concurrency:          1,
			EventUpdateBatchSize: 2,
		})
		require.NoError(t, err)
		events := createTestEvents(2)

		err = dataCatalog.UpdateTrackingPlanEvents(ctx, trackingPlanID, events)

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

		dataCatalog, err := catalog.NewRudderDataCatalog(apiClient, catalog.Options{
			Concurrency:          1,
			EventUpdateBatchSize: 2,
		})
		require.NoError(t, err)
		events := createTestEvents(5)

		err = dataCatalog.UpdateTrackingPlanEvents(ctx, trackingPlanID, events)
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
