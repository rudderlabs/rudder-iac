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
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// previewOne runs the destination import preview for one importable spec
// against a remote served from body, with the remote reporting link as its
// transformation (none when empty).
func previewOne(t *testing.T, registry *definitions.Registry, body, link string, data *destination.DestinationResource) map[string]differ.ResourceDiff {
	return previewOneWithState(t, registry, body, link, data, state.EmptyState())
}

func previewOneWithState(t *testing.T, registry *definitions.Registry, body, link string, data *destination.DestinationResource, st *state.State, others ...*planner.Operation) map[string]differ.ResourceDiff {
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

	plan := &planner.Plan{Operations: append(others, &planner.Operation{Type: planner.Import, Resource: resources.NewResource(
		data.ID, destination.DestinationResourceType, resources.ResourceData{}, nil,
		resources.WithRawData(data), resources.WithResourceImportMetadata("dst-1", "ws-1"),
	)})}
	diffs, err := destination.NewProvider(newTestClient(t, srv.URL), registry).PreviewImports(context.Background(), plan, st)
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

// TestProvider_PreviewImports_TransformationLink pins that the preview shows
// every transformation link change Import makes, compared by remote ID as
// syncTransformationLink compares them: a disconnect, a new link, and a switch
// from one transformation to another; a link already in place shows none.
func TestProvider_PreviewImports_TransformationLink(t *testing.T) {
	t.Parallel()

	body := `{"destination":{"id":"dst-1","name":"Hook","type":"WEBHOOK","version":1,"enabled":true,"config":{"webhookUrl":"https://example.com/hook"}}}`
	spec := func(ref *resources.PropertyRef) *destination.DestinationResource {
		return &destination.DestinationResource{
			ID: "hook", DisplayName: "Hook", Type: "WEBHOOK", Enabled: true, DefinitionVersion: 1,
			Transformation: ref, Config: map[string]any{"webhook_url": "https://example.com/hook"},
		}
	}
	line := func(diffs map[string]differ.ResourceDiff) differ.PropertyDiff {
		return diffs["destination:hook"].Diffs["Transformation"]
	}

	// The managed transformation the spec names, as stored state holds it; the
	// spec's reference is still unresolved at plan time.
	st := state.EmptyState()
	st.AddResource(&state.ResourceState{ID: "new-one", Type: "transformation", OutputRaw: "trf-NEW"})
	unresolved := func() *resources.PropertyRef {
		return &resources.PropertyRef{URN: "transformation:new-one", Resolve: func(out any) (string, error) { return out.(string), nil }}
	}

	assert.Equal(t, differ.PropertyDiff{Property: "Transformation", SourceValue: "trf-OLD (remote id)"},
		line(previewOne(t, testRegistry(t), body, "trf-OLD", spec(nil))), "a remote link the spec leaves out is disconnected")

	assert.Equal(t, differ.PropertyDiff{Property: "Transformation", SourceValue: "trf-OLD (remote id)", TargetValue: "trf-NEW (transformation:new-one)"},
		line(previewOneWithState(t, testRegistry(t), body, "trf-OLD", spec(unresolved()), st)), "a link to another transformation is switched")

	assert.Equal(t, differ.PropertyDiff{Property: "Transformation", SourceValue: "trf-OLD (remote id)", TargetValue: "trf-NEW (transformation:new-one)"},
		line(previewOne(t, testRegistry(t), body, "trf-OLD", spec(&resources.PropertyRef{URN: "transformation:new-one", IsResolved: true, Value: "trf-NEW"}))),
		"an already resolved reference is compared by its value")

	assert.Equal(t, differ.PropertyDiff{Property: "Transformation", TargetValue: "trf-NEW (transformation:new-one)"},
		line(previewOneWithState(t, testRegistry(t), body, "", spec(unresolved()), st)), "a new link")

	assert.Equal(t, differ.PropertyDiff{Property: "Transformation", SourceValue: "trf-OLD (remote id)", TargetValue: "transformation:clean (created in this apply)"},
		line(previewOne(t, testRegistry(t), body, "trf-OLD", spec(&resources.PropertyRef{URN: "transformation:clean"}))),
		"a transformation the same apply creates has no ID yet and is a change")

	assert.Empty(t, previewOneWithState(t, testRegistry(t), body, "trf-NEW", spec(unresolved()), st), "the link already in place is no change")

	// A transformation the same apply imports keeps its remote ID, so the
	// reference resolves to that ID: the usual import workspace -> apply flow.
	adopting := &planner.Operation{Type: planner.Import, Resource: resources.NewResource(
		"t1", "transformation", resources.ResourceData{}, nil, resources.WithResourceImportMetadata("trf-1", "ws-1"),
	)}
	adoptedRef := func() *resources.PropertyRef {
		return &resources.PropertyRef{URN: "transformation:t1", Resolve: func(out any) (string, error) { return out.(string), nil }}
	}
	assert.Empty(t, previewOneWithState(t, testRegistry(t), body, "trf-1", spec(adoptedRef()), state.EmptyState(), adopting),
		"a link to the transformation being adopted, already in place, is no change")
	assert.Equal(t, differ.PropertyDiff{Property: "Transformation", SourceValue: "trf-OLD (remote id)", TargetValue: "trf-1 (transformation:t1, adopted in this apply)"},
		line(previewOneWithState(t, testRegistry(t), body, "trf-OLD", spec(adoptedRef()), state.EmptyState(), adopting)),
		"switching to a transformation the same apply adopts")
	assert.Empty(t, previewOne(t, testRegistry(t), body, "", spec(nil)), "no link on either side is no change")
}
