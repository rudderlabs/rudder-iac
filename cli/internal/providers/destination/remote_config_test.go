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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// webappShapedWebhook is a webhook config as the dashboard stores it: a key the
// definition does not model (with a value), the empty value a cleared field
// leaves behind, and a modelled value.
const webappShapedWebhook = `{
	"webhookUrl": "https://example.com/hook",
	"legacySetting": "keep-me",
	"oneTrustCookieCategories": {},
	"headers": [],
	"connectionMode": {"web": "cloud", "shopify": "cloud"}
}`

// TestHandlerImpl_Import_KeepsRemoteKeysTheSpecCannotCarry pins that adopting a
// destination does not erase what an imported spec cannot carry. The API
// replaces the whole config on update, and import leaves out unmodelled keys
// and empty values, so sending the spec alone used to remove them.
func TestHandlerImpl_Import_KeepsRemoteKeysTheSpecCannotCarry(t *testing.T) {
	t.Parallel()

	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/destinations/dst-1":
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","name":"Hook","type":"WEBHOOK","version":1,"enabled":true,"config":` + webappShapedWebhook + `}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/destinations/dst-1/transformation":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1":
			body, _ := io.ReadAll(r.Body)
			var req struct {
				Config map[string]any `json:"config"`
			}
			require.NoError(t, json.Unmarshal(body, &req))
			sent = req.Config
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","type":"WEBHOOK","version":1,"enabled":true,"config":{}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v2/destinations/dst-1/external-id":
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	h := destination.NewHandler(newTestClient(t, srv.URL), testRegistry(t))
	_, err := h.Impl.Import(context.Background(), &destination.DestinationResource{
		ID: "hook", DisplayName: "Hook", Type: "WEBHOOK", Enabled: true, DefinitionVersion: 1,
		Config: map[string]any{"webhook_url": "https://example.com/new-hook"},
	}, "dst-1")
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"webhookUrl":               "https://example.com/new-hook",
		"legacySetting":            "keep-me",
		"oneTrustCookieCategories": map[string]any{},
		"headers":                  []any{},
		"connectionMode":           map[string]any{"web": "cloud", "shopify": "cloud"},
	}, sent)
}

// TestHandlerImpl_Update_KeepsPreservedKeysButSpecWins pins that a managed
// update carries the preserved keys too, and that a key the spec does set is
// sent as the spec has it.
func TestHandlerImpl_Update_KeepsPreservedKeysButSpecWins(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	h := destination.NewHandler(nil, registry)
	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID: "dst-1", ExternalID: "hook", Name: "Hook", Type: "WEBHOOK", Version: 1, IsEnabled: true,
		Config: []byte(webappShapedWebhook),
	}}
	resource, state, err := h.Impl.MapRemoteToState(remote, urnResolver{})
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"webhook_url": "https://example.com/hook"}, resource.Config,
		"the remote side of the plan leaves out what the spec cannot carry, so a spec without it matches")
	assert.Equal(t, map[string]any{
		"legacySetting":            "keep-me",
		"oneTrustCookieCategories": map[string]any{},
		"headers":                  []any{},
		"connectionMode":           map[string]any{"web": "cloud", "shopify": "cloud"},
	}, state.PreservedConfig)

	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Config map[string]any `json:"config"`
		}
		require.NoError(t, json.Unmarshal(body, &req))
		sent = req.Config
		_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","type":"WEBHOOK","version":1,"enabled":true,"config":{}}}`))
	}))
	t.Cleanup(srv.Close)

	h = destination.NewHandler(newTestClient(t, srv.URL), registry)
	state.PreservedConfig["webhookUrl"] = "https://example.com/stale"
	_, err = h.Impl.Update(context.Background(), &destination.DestinationResource{
		ID: "hook", DisplayName: "Hook", Type: "WEBHOOK", Enabled: true, DefinitionVersion: 1,
		Config: map[string]any{"webhook_url": "https://example.com/new-hook"},
	}, resource, state)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/new-hook", sent["webhookUrl"])
	assert.Equal(t, "keep-me", sent["legacySetting"])
}

// TestHandlerImpl_MapRemoteToState_PreservesClearedModelledKeys pins the empty
// value of a modelled key: import prunes it, so the remote side of the plan
// leaves it out and an update sends it back rather than erasing it.
func TestHandlerImpl_MapRemoteToState_PreservesClearedModelledKeys(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, testRegistry(t))
	resource, state, err := h.Impl.MapRemoteToState(&destination.RemoteDestination{Destination: &client.Destination{
		ID: "dst-1", ExternalID: "ga4", Name: "GA4", Type: "GA4", Version: 1, IsEnabled: true,
		Config: []byte(`{"measurementId":"","debugMode":false,"blacklistedEvents":[{"eventName":""}]}`),
	}}, urnResolver{})
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"debug_mode": false}, resource.Config)
	assert.Equal(t, map[string]any{
		"measurementId":     "",
		"blacklistedEvents": []any{map[string]any{"eventName": ""}},
	}, state.PreservedConfig)
}
