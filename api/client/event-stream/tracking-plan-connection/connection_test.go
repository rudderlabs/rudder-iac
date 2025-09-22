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

// Helper function to convert boolean to pointer
func boolPtr(b bool) *bool {
	return &b
}

// Helper function to convert Action to pointer
func actionPtr(a trackingplanconnection.Action) *trackingplanconnection.Action {
	return &a
}

func TestLinkTP(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{
				"config": {
					"track":{"propagateValidationErrors":true,"unplannedProperties":"forward","anyOtherViolation":"forward","allowUnplannedEvents":false},
					"identify":{"propagateValidationErrors":false,"unplannedProperties":"drop","anyOtherViolation":"forward"},
					"group":{"propagateValidationErrors":true,"unplannedProperties":"forward","anyOtherViolation":"forward"},
					"page":{"propagateValidationErrors":false,"unplannedProperties":"forward","anyOtherViolation":"drop"},
					"screen":{"propagateValidationErrors":true,"unplannedProperties":"forward","anyOtherViolation":"forward"}
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
				PropagateValidationErrors: boolPtr(true),
				UnplannedProperties:       actionPtr(trackingplanconnection.Forward),
				AnyOtherViolations:        actionPtr(trackingplanconnection.Forward),
			},
			AllowUnplannedEvents: boolPtr(false),
		},
		Identify: &trackingplanconnection.EventTypeConfig{
			PropagateValidationErrors: boolPtr(false),
			UnplannedProperties:       actionPtr(trackingplanconnection.Drop),
			AnyOtherViolations:        actionPtr(trackingplanconnection.Forward),
		},
		Group: &trackingplanconnection.EventTypeConfig{
			PropagateValidationErrors: boolPtr(true),
			UnplannedProperties:       actionPtr(trackingplanconnection.Forward),
			AnyOtherViolations:        actionPtr(trackingplanconnection.Forward),
		},
		Page: &trackingplanconnection.EventTypeConfig{
			PropagateValidationErrors: boolPtr(false),
			UnplannedProperties:       actionPtr(trackingplanconnection.Forward),
			AnyOtherViolations:        actionPtr(trackingplanconnection.Drop),
		},
		Screen: &trackingplanconnection.EventTypeConfig{
			PropagateValidationErrors: boolPtr(true),
			UnplannedProperties:       actionPtr(trackingplanconnection.Forward),
			AnyOtherViolations:        actionPtr(trackingplanconnection.Forward),
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
					"track":{"propagateValidationErrors":false,"unplannedProperties":"drop","anyOtherViolation":"forward","allowUnplannedEvents":true},
					"identify":{"unplannedProperties":"forward","anyOtherViolation":"drop"},
					"group":{"propagateValidationErrors":false,"unplannedProperties":"forward","anyOtherViolation":"drop"}
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
				PropagateValidationErrors: boolPtr(false),
				UnplannedProperties:       actionPtr(trackingplanconnection.Drop),
				AnyOtherViolations:        actionPtr(trackingplanconnection.Forward),
			},
			AllowUnplannedEvents: boolPtr(true),
		},
		Identify: &trackingplanconnection.EventTypeConfig{
			PropagateValidationErrors: nil,
			UnplannedProperties:       actionPtr(trackingplanconnection.Forward),
			AnyOtherViolations:        actionPtr(trackingplanconnection.Drop),
		},
		Group: &trackingplanconnection.EventTypeConfig{
			PropagateValidationErrors: boolPtr(false),
			UnplannedProperties:       actionPtr(trackingplanconnection.Forward),
			AnyOtherViolations:        actionPtr(trackingplanconnection.Drop),
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
