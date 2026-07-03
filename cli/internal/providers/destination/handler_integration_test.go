package destination_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerImpl_GA4RoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	var createBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/destinations", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		createBody = string(body)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"destination": {
				"id": "dst-ga4",
				"externalId": "ga4-production",
				"name": "Production GA4",
				"type": "GA4",
				"enabled": true,
				"config": {
					"apiSecret": "secret-value",
					"measurementId": "G-123"
				}
			}
		}`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	input := &destination.DestinationResource{
		ID:                "ga4-production",
		DisplayName:       "Production GA4",
		Type:              "GA4",
		Enabled:           true,
		DefinitionVersion: 1,
		Config: map[string]any{
			"api_secret":     "secret-value",
			"measurement_id": "G-123",
		},
	}

	createdState, err := h.Impl.Create(ctx, input)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"name": "Production GA4",
		"type": "GA4",
		"enabled": true,
		"externalId": "ga4-production",
		"config": {
			"apiSecret": "secret-value",
			"measurementId": "G-123"
		}
	}`, createBody)

	resource, mappedState, err := h.Impl.MapRemoteToState(&destination.RemoteDestination{
		Destination: &client.Destination{
			ID:         createdState.ID,
			ExternalID: input.ID,
			Name:       input.DisplayName,
			Type:       input.Type,
			IsEnabled:  input.Enabled,
			Config: json.RawMessage(`{
				"apiSecret": "secret-value",
				"measurementId": "G-123"
			}`),
		},
	}, resources.NewRemoteResources())
	require.NoError(t, err)

	assert.Equal(t, input, resource)
	assert.Equal(t, createdState, mappedState)
}
