package destination_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/am"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/redis"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/webhook"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
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
		"Config (API keys the import removes).consentManagement.amp": {
			Property: "Config (API keys the import removes).consentManagement.amp", SourceValue: "(list, 1 item)",
		},
		"Config (API keys the import removes).consentManagement.web[0].extraItem": {
			Property: "Config (API keys the import removes).consentManagement.web[0].extraItem", SourceValue: "(value)",
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

	assert.Equal(t, map[string]differ.ResourceDiff{"destination:f": {URN: "destination:f", Diffs: map[string]differ.PropertyDiff{
		// The blacklist change still shows as a local line...
		"Config": {
			Property:    "Config",
			SourceValue: map[string]any{"tracking_id": "T-1", "event_filtering": map[string]any{"blacklist": []any{"Old"}}},
			TargetValue: map[string]any{"tracking_id": "T-1", "event_filtering": map[string]any{"blacklist": []any{"New"}}},
		},
		// ...and does not hide the whitelist's removal.
		"Config (API keys the import removes).whitelistedEvents": {
			Property: "Config (API keys the import removes).whitelistedEvents", SourceValue: "(list, 1 item)",
		},
	}}}, diffs)
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

	// A link to a transformation the same apply creates never reaches the
	// preview: CheckPlan refuses it first (TestCheckPlanRefusesLinkToTransformationCreatedInSameApply).

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

// TestProvider_PreviewImports_RemovedSecretIsAChange pins that a secret only
// the remote holds, which the adopt's payload leaves out, is a removal rather
// than a secret re-application: the import changes the destination.
func TestProvider_PreviewImports_RemovedSecretIsAChange(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(redis.NewDefinition()))

	diffs := previewOne(t, registry, `{"destination":{"id":"dst-1","name":"R","type":"REDIS","version":1,"enabled":true,"config":{
		"address":"redis:6379","database":"1","secure":false,"clusterMode":false,"skipVerify":false,"useJSONModule":false,"password":""}}}`, "",
		&destination.DestinationResource{
			ID: "r", DisplayName: "R", Type: "redis", Enabled: true, DefinitionVersion: 1,
			Config: map[string]any{"address": "redis:6379", "database": "1", "secure": false, "cluster_mode": false, "skip_verify": false, "use_json_module": false},
		})

	remote := map[string]any{"address": "redis:6379", "database": "1", "secure": false, "cluster_mode": false, "skip_verify": false, "use_json_module": false, "password": ""}
	assert.Equal(t, map[string]differ.ResourceDiff{"destination:r": {URN: "destination:r", Diffs: map[string]differ.PropertyDiff{
		"Config": {
			Property:    "Config",
			SourceValue: secret.WrapUnknownSecrets(remote, []string{"password"}),
			TargetValue: map[string]any{"address": "redis:6379", "database": "1", "secure": false, "cluster_mode": false, "skip_verify": false, "use_json_module": false},
		},
	}}}, diffs, "a removed secret is a change, not a re-application")
}

// TestProvider_PreviewImports_RemovalsUnderOmittedParents pins that a parent
// the spec leaves out, which local config partly holds, does not hide its
// unmodelled children, and that a shortened list names the dropped element's
// unmapped fields. A removed null is listed too.
func TestProvider_PreviewImports_RemovalsUnderOmittedParents(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(webhook.NewDefinition()))

	diffs := previewOne(t, registry, `{"destination":{"id":"dst-1","name":"Hook","type":"WEBHOOK","version":1,"enabled":true,"config":{
		"webhookUrl":"https://example.com/hook","webhookMethod":"POST",
		"headers":[{"from":"A","to":"a"},{"from":"B","to":"b","description":"note"}],
		"consentManagement":{"web":[{"provider":"custom","resolutionStrategy":"and","consents":[{"consent":"c1"}]}],"amp":[{"provider":"ketch"}]},
		"legacyNull":null,"legacyOther":"x"}}}`, "", &destination.DestinationResource{
		ID: "hook", DisplayName: "Hook", Type: "webhook", Enabled: true, DefinitionVersion: 1,
		Config: map[string]any{
			"webhook_url": "https://example.com/hook", "webhook_method": "POST",
			"headers": []any{map[string]any{"from": "A", "to": "a"}},
		},
	})

	removed := func(path, summary string) differ.PropertyDiff {
		return differ.PropertyDiff{Property: "Config (API keys the import removes)." + path, SourceValue: summary}
	}
	expected := map[string]differ.PropertyDiff{
		"Config": {
			Property: "Config",
			SourceValue: secret.WrapUnknownSecrets(map[string]any{
				"webhook_url": "https://example.com/hook", "webhook_method": "POST",
				"headers":            []any{map[string]any{"from": "A", "to": "a"}, map[string]any{"from": "B", "to": "b"}},
				"consent_management": map[string]any{"web": []any{map[string]any{"provider": "custom", "resolution_strategy": "and", "consents": []any{"c1"}}}},
			}, []string{"headers.to"}),
			TargetValue: map[string]any{
				"webhook_url": "https://example.com/hook", "webhook_method": "POST",
				"headers": []any{map[string]any{"from": "A", "to": "a"}},
			},
		},
	}
	for path, summary := range map[string]string{
		"consentManagement.amp":  "(list, 1 item)",
		"headers[1].description": "(value)",
		"legacyNull":             "(null)",
		"legacyOther":            "(value)",
	} {
		expected[removed(path, summary).Property] = removed(path, summary)
	}
	assert.Equal(t, map[string]differ.ResourceDiff{"destination:hook": {URN: "destination:hook", Diffs: expected}}, diffs)
}

// TestProvider_PreviewImports_SkipsBackendDerivedKeys pins that a key the
// backend sets from connectionMode on every write is not reported as removed:
// the CLI never sends useNativeSDK, and the destination keeps it after the
// adopt, so an otherwise unchanged destination is adopted unchanged.
func TestProvider_PreviewImports_SkipsBackendDerivedKeys(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(am.NewDefinition()))
	spec := func(config map[string]any) *destination.DestinationResource {
		return &destination.DestinationResource{ID: "amp", DisplayName: "Amp", Type: "am", Enabled: true, DefinitionVersion: 1, Config: config}
	}

	assert.Empty(t, previewOne(t, registry, `{"destination":{"id":"dst-1","name":"Amp","type":"AM","version":1,"enabled":true,"config":{
		"residencyServer":"standard","connectionMode":{"web":"cloud"},"useNativeSDK":{"web":false},"useNativeSDKToSend":{"web":false}}}}`, "",
		spec(map[string]any{"residency_server": "standard", "connection_mode": map[string]any{"web": "cloud"}})),
		"adopted unchanged")

	// The reported repro: the exported spec carries the returned api_key, so
	// the import only re-applies that secret, with no removal line.
	diffs := previewOne(t, registry, `{"destination":{"id":"dst-1","name":"Amp","type":"AM","version":1,"enabled":true,"config":{
		"apiKey":"k","residencyServer":"standard","connectionMode":{"web":"cloud"},"useNativeSDK":{"web":false}}}}`, "",
		spec(secret.WrapKnownSecrets(map[string]any{"api_key": "k", "residency_server": "standard", "connection_mode": map[string]any{"web": "cloud"}}, []string{"api_key"})))
	assert.Equal(t, map[string]differ.ResourceDiff{"destination:amp": {URN: "destination:amp", SecretOnly: true, Diffs: map[string]differ.PropertyDiff{
		"Config": {
			Property:    "Config",
			SourceValue: secret.WrapUnknownSecrets(map[string]any{"api_key": "k", "residency_server": "standard", "connection_mode": map[string]any{"web": "cloud"}}, []string{"api_key"}),
			TargetValue: secret.WrapKnownSecrets(map[string]any{"api_key": "k", "residency_server": "standard", "connection_mode": map[string]any{"web": "cloud"}}, []string{"api_key"}),
			SecretOnly:  true,
		},
	}}}, diffs, "the import only re-applies the secret, with no removal line")
}

// TestProvider_PreviewImports_ReadErrors pins that the preview fails, naming
// the destination, when it cannot read what the import would change: the
// destination itself, or its transformation link (a 404 there only means no
// link).
func TestProvider_PreviewImports_ReadErrors(t *testing.T) {
	t.Parallel()

	run := func(t *testing.T, failPath string) error {
		t.Helper()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case failPath:
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"boom"}`))
			case "/v2/destinations/dst-1":
				_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","name":"Hook","type":"WEBHOOK","version":1,"enabled":true,"config":{"webhookUrl":"https://example.com/hook"}}}`))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		}))
		t.Cleanup(srv.Close)

		data := &destination.DestinationResource{ID: "hook", DisplayName: "Hook", Type: "WEBHOOK", Enabled: true, DefinitionVersion: 1,
			Config: map[string]any{"webhook_url": "https://example.com/hook"}}
		plan := &planner.Plan{Operations: []*planner.Operation{{Type: planner.Import, Resource: resources.NewResource(
			"hook", destination.DestinationResourceType, resources.ResourceData{}, nil,
			resources.WithRawData(data), resources.WithResourceImportMetadata("dst-1", "ws-1"),
		)}}}
		_, err := destination.NewProvider(newTestClient(t, srv.URL), testRegistry(t)).PreviewImports(context.Background(), plan, state.EmptyState())
		return err
	}

	err := run(t, "/v2/destinations/dst-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "previewing import of destination:hook: getting destination")

	err = run(t, "/v2/destinations/dst-1/transformation")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "previewing import of destination:hook: getting destination transformation")
}

// TestProvider_PreviewImports_ListsUnderivableNativeSDK pins that a
// backend-derived key is skipped only where the backend can rederive it: a
// legacy useNativeSDK with no connectionMode for its source type is removed
// by the adopt, and the preview says so.
func TestProvider_PreviewImports_ListsUnderivableNativeSDK(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(am.NewDefinition()))

	diffs := previewOne(t, registry, `{"destination":{"id":"dst-1","name":"Amp","type":"AM","version":1,"enabled":true,"config":{
		"residencyServer":"standard","useNativeSDK":{"web":true}}}}`, "",
		&destination.DestinationResource{ID: "amp", DisplayName: "Amp", Type: "am", Enabled: true, DefinitionVersion: 1,
			Config: map[string]any{"residency_server": "standard"}})

	assert.Equal(t, map[string]differ.ResourceDiff{"destination:amp": {URN: "destination:amp", Diffs: map[string]differ.PropertyDiff{
		"Config (API keys the import removes).useNativeSDK.web": {
			Property: "Config (API keys the import removes).useNativeSDK.web", SourceValue: "(value)",
		},
	}}}, diffs)
}
