package destination_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProvider_PreviewImports pins that the plan sees what adopting a
// destination writes: a value the spec sets that the remote lacks is a change,
// while keys the update carries over (unmodelled ones, cleared values) are not.
func TestProvider_PreviewImports(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/destinations/dst-changed":
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-changed","name":"GA4","type":"GA4","version":1,"enabled":true,"config":{"measurementId":"G-1","legacySetting":"keep-me","blacklistedEvents":[{"eventName":""}]}}}`))
		case "/v2/destinations/dst-same":
			_, _ = w.Write([]byte(`{"destination":{"id":"dst-same","name":"Hook","type":"WEBHOOK","version":1,"enabled":true,"config":{"webhookUrl":"https://example.com/hook","ketchConsentPurposes":{}}}}`))
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
		importing("hook", "dst-same", &destination.DestinationResource{
			ID: "hook", DisplayName: "Hook", Type: "WEBHOOK", Enabled: true, DefinitionVersion: 1,
			Config: map[string]any{"webhook_url": "https://example.com/hook"},
		}),
	}}

	p := destination.NewProvider(newTestClient(t, srv.URL), testRegistry(t))
	diffs, err := p.PreviewImports(context.Background(), plan)
	require.NoError(t, err)

	assert.Equal(t, map[string]differ.ResourceDiff{
		"destination:ga4": {URN: "destination:ga4", Diffs: map[string]differ.PropertyDiff{
			"Config": {
				Property:    "Config",
				SourceValue: map[string]any{"measurement_id": "G-1"},
				TargetValue: map[string]any{"measurement_id": "G-1", "debug_mode": true},
			},
		}},
	}, diffs)
}
