package destination_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProvider_PreviewImports pins that the plan sees what adopting a
// destination writes. The adopt replaces the whole config with the spec's, so
// a value the spec adds is a change and so is one it leaves out, even an empty
// one import pruned. A key the definition does not model is named in the API's
// spelling, since no local line can show it. A destination the spec matches
// has no entry.
func TestProvider_PreviewImports(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/destinations/dst-changed":
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-changed","name":"GA4","type":"GA4","version":1,"enabled":true,"config":{"measurementId":"G-1","blacklistedEvents":[{"eventName":""}],"legacySetting":"keep-me"}}}`))
		case "/v2/destinations/dst-filtered":
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-filtered","name":"Filtered","type":"FILTERED","version":1,"enabled":true,"config":{"trackingID":"T-1","eventFilteringOption":"disable","blacklistedEvents":[{"eventName":"Old"}]}}}`))
		case "/v2/destinations/dst-same":
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-same","name":"Hook","type":"WEBHOOK","version":1,"enabled":true,"config":{"webhookUrl":"https://example.com/hook"}}}`))
		case "/v2/destinations/dst-changed/transformation", "/v2/destinations/dst-filtered/transformation", "/v2/destinations/dst-same/transformation":
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	importing := func(id, remoteID string, data *destination.DestinationResource) *planner.Operation {
		return &planner.Operation{Type: planner.Import, Resource: resources.NewResource(
			id, destination.DestinationResourceType, resources.ResourceData{}, nil,
			resources.WithRawData(data), resources.WithResourceImportMetadata(remoteID, "ws-1"),
		)}
	}
	plan := &planner.Plan{Operations: []*planner.Operation{
		importing("ga4", "dst-changed", &destination.DestinationResource{
			ID: "ga4", DisplayName: "GA4", Type: "GA4", Enabled: true, DefinitionVersion: 1,
			Config: map[string]any{"measurement_id": "G-1", "debug_mode": true},
		}),
		importing("filtered", "dst-filtered", &destination.DestinationResource{
			ID: "filtered", DisplayName: "Filtered", Type: "FILTERED", Enabled: true, DefinitionVersion: 1,
			Config: map[string]any{"tracking_id": "T-1"},
		}),
		importing("hook", "dst-same", &destination.DestinationResource{
			ID: "hook", DisplayName: "Hook", Type: "WEBHOOK", Enabled: true, DefinitionVersion: 1,
			Config: map[string]any{"webhook_url": "https://example.com/hook"},
		}),
	}}

	p := destination.NewProvider(newTestClient(t, srv.URL), testRegistry(t))
	diffs, err := p.PreviewImports(context.Background(), plan, state.EmptyState())
	require.NoError(t, err)

	assert.Equal(t, map[string]differ.ResourceDiff{
		"destination:ga4": {URN: "destination:ga4", Diffs: map[string]differ.PropertyDiff{
			"Config": {
				Property:    "Config",
				SourceValue: map[string]any{"measurement_id": "G-1", "blacklisted_events": []any{""}},
				TargetValue: map[string]any{"measurement_id": "G-1", "debug_mode": true},
			},
			"Config (API keys the import removes).legacySetting": {
				Property:    "Config (API keys the import removes).legacySetting",
				SourceValue: "(value)",
			},
		}},
		// The unselected list the converter drops is named; the discriminator
		// naming no list says what its absence says, so it is not.
		"destination:filtered": {URN: "destination:filtered", Diffs: map[string]differ.PropertyDiff{
			"Config (API keys the import removes).blacklistedEvents": {
				Property:    "Config (API keys the import removes).blacklistedEvents",
				SourceValue: "(list, 1 item)",
			},
		}},
	}, diffs)
}
