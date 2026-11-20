package destination_test

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
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

// TestProvider_PreviewImports_UpdateRemovals pins that an update of a managed
// destination names the remote keys it removes. Update replaces the whole config
// with the spec's, so a key local config cannot hold is deleted, and the plan's own
// comparison never sees it: MapRemoteToState has already converted it away. The
// state is built by MapRemoteToState, as the apply builds it.
func TestProvider_PreviewImports_UpdateRemovals(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(webhook.NewDefinition()))
	h := destination.NewHandler(nil, registry)
	version := int64(1)
	specs, err := h.Impl.ExtractResourcesFromSpec("", &destination.DestinationSpec{
		ID: "hook", DisplayName: "Hook", Type: "webhook", Enabled: true, DefinitionVersion: &version,
		Config: map[string]any{
			"webhook_url": "https://example.com/hook", "webhook_method": "POST",
			"headers": []any{map[string]any{"from": "A", "to": "a"}, map[string]any{"from": "B", "to": "b"}},
		},
	})
	require.NoError(t, err)
	spec := specs["hook"]

	stored := func(t *testing.T, config string) *state.State {
		t.Helper()
		resource, remote, err := h.Impl.MapRemoteToState(&destination.RemoteDestination{Destination: &client.Destination{
			ID: "dst-1", ExternalID: "hook", Name: "Hook", Type: "WEBHOOK", Version: 1, IsEnabled: true, Config: []byte(config),
		}}, urnResolver{})
		require.NoError(t, err)
		st := state.EmptyState()
		st.AddResource(&state.ResourceState{ID: "hook", Type: destination.DestinationResourceType, InputRaw: resource, OutputRaw: remote})
		return st
	}
	preview := func(t *testing.T, st *state.State) map[string]differ.ResourceDiff {
		t.Helper()
		plan := &planner.Plan{Operations: []*planner.Operation{{Type: planner.Update, Resource: resources.NewResource(
			"hook", destination.DestinationResourceType, resources.ResourceData{}, nil, resources.WithRawData(spec),
		)}}}
		// No client: an update preview reads the config the plan was built from.
		diffs, err := destination.NewProvider(nil, registry).PreviewImports(context.Background(), plan, st)
		require.NoError(t, err)
		return diffs
	}

	t.Run("names keys local config cannot hold", func(t *testing.T) {
		t.Parallel()
		diffs := preview(t, stored(t, `{"webhookUrl":"https://example.com/hook","webhookMethod":"POST",
			"headers":[{"from":"A","to":""},{"from":"B","to":"","description":"note"}],
			"consentManagement":{"amp":[{"provider":"custom","resolutionStrategy":"and","consents":[{"consent":"c1"}]}]}}`))

		// amp is the only consent entry, so the whole block goes and is named once.
		assert.Equal(t, map[string]differ.ResourceDiff{"destination:hook": {URN: "destination:hook", Diffs: map[string]differ.PropertyDiff{
			"Config (API keys the update removes).consentManagement": {
				Property: "Config (API keys the update removes).consentManagement", SourceValue: "(object, 1 key)",
			},
			"Config (API keys the update removes).headers[1].description": {
				Property: "Config (API keys the update removes).headers[1].description", SourceValue: "(value)",
			},
		}}}, diffs)
	})

	t.Run("nothing when local config holds every remote key", func(t *testing.T) {
		t.Parallel()
		diffs := preview(t, stored(t, `{"webhookUrl":"https://example.com/hook","webhookMethod":"POST",
			"headers":[{"from":"A","to":""},{"from":"B","to":""}]}`))

		assert.Empty(t, diffs)
	})

	t.Run("nothing without a stored remote", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, preview(t, state.EmptyState()))
	})

	t.Run("a preview that fails is skipped, not returned", func(t *testing.T) {
		t.Parallel()
		// ExtractResourcesFromSpec refuses an unknown version, so the spec is
		// copied by hand. Without the version error this remote would give lines.
		unknown := *spec
		unknown.DefinitionVersion = 99
		plan := &planner.Plan{Operations: []*planner.Operation{{Type: planner.Update, Resource: resources.NewResource(
			"hook", destination.DestinationResourceType, resources.ResourceData{}, nil, resources.WithRawData(&unknown),
		)}}}
		st := stored(t, `{"webhookUrl":"https://example.com/hook","consentManagement":{"amp":[]}}`)

		diffs, err := destination.NewProvider(nil, registry).PreviewImports(context.Background(), plan, st)
		require.NoError(t, err)
		assert.Empty(t, diffs)
	})

	t.Run("nothing when the spec changes the type", func(t *testing.T) {
		t.Parallel()
		// Update refuses a type change; converting the remote with the spec's
		// definition would only produce noise first.
		st := stored(t, `{"webhookUrl":"https://example.com/hook","consentManagement":{"amp":[]}}`)
		st.GetResource("destination:hook").InputRaw.(*destination.DestinationResource).Type = "ga4"

		assert.Empty(t, preview(t, st))
	})
}
