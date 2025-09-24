package trackingplanconnection_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	trackingplanconnection "github.com/rudderlabs/rudder-iac/api/client/event-stream/tracking-plan-connection"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestLinkTP(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{
				"config": {
					"track":{"propagateValidationErrors":true,"unplannedProperties":"forward","anyOtherViolations":"forward","allowUnplannedEvents":false},
					"identify":{"propagateValidationErrors":false,"unplannedProperties":"drop","anyOtherViolations":"forward"},
					"group":{"propagateValidationErrors":true,"unplannedProperties":"forward","anyOtherViolations":"forward"},
					"page":{"propagateValidationErrors":false,"unplannedProperties":"forward","anyOtherViolations":"drop"},
					"screen":{"propagateValidationErrors":true,"unplannedProperties":"forward","anyOtherViolations":"forward"}
				}
			}`
			return testutils.ValidateRequest(t, req, "POST", "v2/catalog/tracking-plans/tp-123/sources/src-456", expected)
		},
		ResponseStatus: 200,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	tpConnectionClient := trackingplanconnection.NewRudderTrackingPlanConnectionStore(c)
	config := &trackingplanconnection.ConnectionConfig{
		Track: &trackingplanconnection.TrackConfig{
			EventTypeConfig: &trackingplanconnection.EventTypeConfig{
				PropagateValidationErrors: true,
				UnplannedProperties:       trackingplanconnection.Forward,
				AnyOtherViolations:        trackingplanconnection.Forward,
			},
			AllowUnplannedEvents: false,
		},
		Identify: &trackingplanconnection.EventTypeConfig{
			PropagateValidationErrors: false,
			UnplannedProperties:       trackingplanconnection.Drop,
			AnyOtherViolations:        trackingplanconnection.Forward,
		},
		Group: &trackingplanconnection.EventTypeConfig{
			PropagateValidationErrors: true,
			UnplannedProperties:       trackingplanconnection.Forward,
			AnyOtherViolations:        trackingplanconnection.Forward,
		},
		Page: &trackingplanconnection.EventTypeConfig{
			PropagateValidationErrors: false,
			UnplannedProperties:       trackingplanconnection.Forward,
			AnyOtherViolations:        trackingplanconnection.Drop,
		},
		Screen: &trackingplanconnection.EventTypeConfig{
			PropagateValidationErrors: true,
			UnplannedProperties:       trackingplanconnection.Forward,
			AnyOtherViolations:        trackingplanconnection.Forward,
		},
	}

	err = tpConnectionClient.LinkTP(context.Background(), "tp-123", "src-456", config)
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestUpdateTPConnection(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{
				"config": {
					"track":{"propagateValidationErrors":false,"unplannedProperties":"drop","anyOtherViolations":"forward","allowUnplannedEvents":true},
					"identify":{"propagateValidationErrors":true,"unplannedProperties":"forward","anyOtherViolations":"drop"},
					"group":{"propagateValidationErrors":false,"unplannedProperties":"forward","anyOtherViolations":"drop"}
				}
			}`
			return testutils.ValidateRequest(t, req, "PUT", "v2/catalog/tracking-plans/tp-123/sources/src-456", expected)
		},
		ResponseStatus: 200,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	tpConnectionClient := trackingplanconnection.NewRudderTrackingPlanConnectionStore(c)

	config := &trackingplanconnection.ConnectionConfig{
		Track: &trackingplanconnection.TrackConfig{
			EventTypeConfig: &trackingplanconnection.EventTypeConfig{
				PropagateValidationErrors: false,
				UnplannedProperties:       trackingplanconnection.Drop,
				AnyOtherViolations:        trackingplanconnection.Forward,
			},
			AllowUnplannedEvents: true,
		},
		Identify: &trackingplanconnection.EventTypeConfig{
			PropagateValidationErrors: true,
			UnplannedProperties:       trackingplanconnection.Forward,
			AnyOtherViolations:        trackingplanconnection.Drop,
		},
		Group: &trackingplanconnection.EventTypeConfig{
			PropagateValidationErrors: false,
			UnplannedProperties:       trackingplanconnection.Forward,
			AnyOtherViolations:        trackingplanconnection.Drop,
		},
	}

	err = tpConnectionClient.UpdateTPConnection(context.Background(), "tp-123", "src-456", config)
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestUnlinkTP(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "DELETE", "v2/catalog/tracking-plans/tp-123/sources/src-456", "")
		},
		ResponseStatus: 200,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	tpConnectionClient := trackingplanconnection.NewRudderTrackingPlanConnectionStore(c)

	err = tpConnectionClient.UnlinkTP(context.Background(), "tp-123", "src-456")
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}
