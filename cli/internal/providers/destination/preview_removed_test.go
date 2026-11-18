package destination_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/webhook"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// previewOne runs the destination import preview for one importable spec
// against a remote served from body, with the remote reporting link as its
// transformation (none when empty).
func previewOne(t *testing.T, registry *definitions.Registry, body, link string, data *destination.DestinationResource) map[string]differ.ResourceDiff {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/destinations/dst-1":
			_, _ = w.Write([]byte(body))
		case "/v2/destinations/dst-1/transformation":
			if link == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write([]byte(`{"destinationId":"dst-1","transformationId":"` + link + `"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	plan := &planner.Plan{Operations: []*planner.Operation{{Type: planner.Import, Resource: resources.NewResource(
		data.ID, destination.DestinationResourceType, resources.ResourceData{}, nil,
		resources.WithRawData(data), resources.WithResourceImportMetadata("dst-1", "ws-1"),
	)}}}
	diffs, err := destination.NewProvider(newTestClient(t, srv.URL), registry).PreviewImports(context.Background(), plan)
	require.NoError(t, err)
	return diffs
}

// TestProvider_PreviewImports_NestedRemovals pins that removals below the top
// level are named by path: a consent entry for a source type the webhook
// definition does not model, and a field of a consent item it does not map.
// Values are summarised, never printed.
func TestProvider_PreviewImports_NestedRemovals(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(webhook.NewDefinition()))

	diffs := previewOne(t, registry, `{"destination":{"id":"dst-1","name":"Hook","type":"WEBHOOK","version":1,"enabled":true,"config":{
		"webhookUrl":"https://example.com/hook","webhookMethod":"POST",
		"consentManagement":{
			"web":[{"provider":"custom","resolutionStrategy":"and","consents":[{"consent":"c1"}],"extraItem":"secret-looking-value"}],
			"amp":[{"provider":"ketch","consents":[{"consent":"k1"}]}]
		}}}}`, "", &destination.DestinationResource{
		ID: "hook", DisplayName: "Hook", Type: "webhook", Enabled: true, DefinitionVersion: 1,
		Config: map[string]any{
			"webhook_url": "https://example.com/hook", "webhook_method": "POST",
			"consent_management": map[string]any{"web": []any{map[string]any{"provider": "custom", "resolution_strategy": "and", "consents": []any{"c1"}}}},
		},
	})

	assert.Equal(t, map[string]differ.ResourceDiff{"destination:hook": {URN: "destination:hook", Diffs: map[string]differ.PropertyDiff{
		"Config (API keys the import removes)": {
			Property: "Config (API keys the import removes)",
			SourceValue: map[string]any{
				"consentManagement.amp":              "(list, 1 item)",
				"consentManagement.web[0].extraItem": "(value)",
			},
			TargetValue: map[string]any{},
		},
	}}}, diffs)
}

// TestProvider_PreviewImports_SiblingRemovalUnderChangedRoot pins that a change
// shown for one member of a local block does not hide another member's
// removal: editing the selected blacklist still names the stale, unselected
// whitelist the adopt drops.
func TestProvider_PreviewImports_SiblingRemovalUnderChangedRoot(t *testing.T) {
	t.Parallel()

	diffs := previewOne(t, testRegistry(t), `{"destination":{"id":"dst-1","name":"F","type":"FILTERED","version":1,"enabled":true,"config":{
		"trackingID":"T-1","eventFilteringOption":"blacklistedEvents",
		"blacklistedEvents":[{"eventName":"Old"}],"whitelistedEvents":[{"eventName":"Stale"}]}}}`, "", &destination.DestinationResource{
		ID: "f", DisplayName: "F", Type: "FILTERED", Enabled: true, DefinitionVersion: 1,
		Config: map[string]any{"tracking_id": "T-1", "event_filtering": map[string]any{"blacklist": []any{"New"}}},
	})

	require.Contains(t, diffs, "destination:f")
	removed := diffs["destination:f"].Diffs["Config (API keys the import removes)"]
	assert.Equal(t, map[string]any{"whitelistedEvents": "(list, 1 item)"}, removed.SourceValue)
	assert.Contains(t, diffs["destination:f"].Diffs, "Config", "the blacklist change still shows as a local line")
}

// TestProvider_PreviewImports_TransformationLink pins that the preview shows a
// transformation link the import changes: Import disconnects a remote link the
// spec leaves out, and links one the spec names.
func TestProvider_PreviewImports_TransformationLink(t *testing.T) {
	t.Parallel()

	body := `{"destination":{"id":"dst-1","name":"Hook","type":"WEBHOOK","version":1,"enabled":true,"config":{"webhookUrl":"https://example.com/hook"}}}`
	spec := func(ref *resources.PropertyRef) *destination.DestinationResource {
		return &destination.DestinationResource{
			ID: "hook", DisplayName: "Hook", Type: "WEBHOOK", Enabled: true, DefinitionVersion: 1,
			Transformation: ref, Config: map[string]any{"webhook_url": "https://example.com/hook"},
		}
	}
	ref := &resources.PropertyRef{URN: "transformation:clean"}

	disconnected := previewOne(t, testRegistry(t), body, "trn-remote", spec(nil))
	assert.Equal(t, differ.PropertyDiff{Property: "Transformation", SourceValue: "trn-remote (remote id)", TargetValue: nil},
		disconnected["destination:hook"].Diffs["Transformation"])

	linked := previewOne(t, testRegistry(t), body, "", spec(ref))
	assert.Equal(t, differ.PropertyDiff{Property: "Transformation", SourceValue: nil, TargetValue: ref},
		linked["destination:hook"].Diffs["Transformation"])

	assert.Empty(t, previewOne(t, testRegistry(t), body, "", spec(nil)), "no link on either side is no change")
}
