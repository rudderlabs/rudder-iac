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
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandlerImpl_GA4RoundTrip exercises the full happy path against the real
// registry converter: spec → extract → Create (camelCase payload) → remote
// response → MapRemoteToState (snake_case round-trip). It asserts the config
// survives a snake_case → camelCase → snake_case round-trip unchanged, which is
// the correctness invariant the differ relies on.
func TestHandlerImpl_GA4RoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	registry := testRegistry(t)

	localConfig := map[string]any{
		"api_secret":     "secret-value",
		"measurement_id": "G-123",
	}
	// The camelCase form we expect the API to receive and echo back.
	apiConfigJSON := `{"apiSecret":"secret-value","measurementId":"G-123"}`

	var createBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/destinations", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		createBody = string(body)

		// Echo back the config the API received, plus server-assigned IDs.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"destination": {
				"id": "dst-ga4",
				"externalId": "ga4-production",
				"name": "Production GA4",
				"type": "GA4",
				"enabled": true,
				"config": ` + apiConfigJSON + `
			}
		}`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	h := destination.NewHandler(c, registry)

	// 1. Spec → resource (ExtractResourcesFromSpec)
	spec := &destination.DestinationSpec{
		ID:                "ga4-production",
		DisplayName:       "Production GA4",
		Type:              "GA4",
		Enabled:           true,
		DefinitionVersion: 1,
		Config:            localConfig,
	}
	extracted, err := h.Impl.ExtractResourcesFromSpec("destinations/ga4.yaml", spec)
	require.NoError(t, err)
	resource := extracted["ga4-production"]
	require.NotNil(t, resource)
	assert.Equal(t, "G-123", resource.Config["measurement_id"])
	apiSecret := requireSecret(t, resource.Config, "api_secret")
	assert.Equal(t, "secret-value", apiSecret.Reveal())

	// 2. Create — verify camelCase payload
	state, err := h.Impl.Create(ctx, resource)
	require.NoError(t, err)
	assert.Equal(t, "dst-ga4", state.ID)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(createBody), &payload))
	configPayload, _ := payload["config"].(map[string]any)
	assert.Equal(t, "secret-value", configPayload["apiSecret"])
	assert.Equal(t, "G-123", configPayload["measurementId"])

	// 3. Remote response → state (MapRemoteToState) — verify snake_case round-trip
	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID:         state.ID,
		ExternalID: "ga4-production",
		Name:       "Production GA4",
		Type:       "GA4",
		Version:    1,
		IsEnabled:  true,
		Config:     []byte(apiConfigJSON),
	}}

	roundTripped, _, err := h.Impl.MapRemoteToState(remote, urnResolver{})
	require.NoError(t, err)
	require.NotNil(t, roundTripped)

	// Non-secret keys round-trip; secrets become unknown on the remote side.
	assert.Equal(t, "G-123", roundTripped.Config["measurement_id"])
	remoteSecret := requireSecret(t, roundTripped.Config, "api_secret")
	assert.True(t, remoteSecret.IsUnknown())
	assert.Equal(t, "ga4-production", roundTripped.ID)
	assert.Equal(t, "Production GA4", roundTripped.DisplayName)
	assert.Equal(t, "GA4", roundTripped.Type)
	assert.True(t, roundTripped.Enabled)
	assert.Equal(t, int64(1), roundTripped.DefinitionVersion)
}

// TestHandlerImpl_GA4RoundTripWithTransformationLink extends the round-trip to
// cover the transformation ref on both sides: spec-side parses the scalar ref
// into a PropertyRef, and MapRemoteToState resolves the remote link back to the
// same URN when the transformation is CLI-managed.
func TestHandlerImpl_GA4RoundTripWithTransformationLink(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	h := destination.NewHandler(nil, registry)

	// Spec side: "#transformation:my-transform" → PropertyRef with URN.
	extracted, err := h.Impl.ExtractResourcesFromSpec("x", &destination.DestinationSpec{
		ID:                "ga4",
		DisplayName:       "GA4",
		Type:              "GA4",
		DefinitionVersion: 1,
		Transformation:    "#transformation:my-transform",
		Config:            map[string]any{"api_secret": "s"},
	})
	require.NoError(t, err)
	specRef := extracted["ga4"].Transformation
	require.NotNil(t, specRef)
	expectedURN := resources.URN("my-transform", ttypes.TransformationResourceType)
	assert.Equal(t, expectedURN, specRef.URN)
	assert.Equal(t, "id", specRef.Property)

	// State side: remote link "trans-1" resolves to the same URN.
	resolver := urnResolver{transformationURNByID: map[string]string{
		"trans-1": expectedURN,
	}}
	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID:             "dst-1",
		ExternalID:     "ga4",
		Name:           "GA4",
		Type:           "GA4",
		Version:        1,
		Config:         []byte(`{"apiSecret":"s"}`),
		Transformation: &client.DestinationTransformationLink{ID: "trans-1"},
	}}

	resource, state, err := h.Impl.MapRemoteToState(remote, resolver)
	require.NoError(t, err)
	require.NotNil(t, resource.Transformation)
	assert.Equal(t, expectedURN, resource.Transformation.URN)
	assert.Equal(t, "id", resource.Transformation.Property)
	assert.Equal(t, "trans-1", state.TransformationID)
}
