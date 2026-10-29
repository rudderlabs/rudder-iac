package destination_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteOnlyConfigKeyCannotBeSilentlyDeleted(t *testing.T) {
	h := destination.NewHandler(nil, testRegistry(t))
	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID: "dst-1", ExternalID: "ga4-production", Name: "Production GA4",
		Type: "GA4", Version: 1, IsEnabled: true,
		Config: []byte(`{"measurementId":"G-123","addedInUI":"keep me"}`),
	}}

	resource, _, err := h.Impl.MapRemoteToState(remote, urnResolver{})
	require.Error(t, err)
	assert.Nil(t, resource)
	assert.ErrorContains(t, err, "addedInUI")
}

func TestImportScaffoldsRequiredSecretOmittedByAPI(t *testing.T) {
	h := destination.NewHandler(nil, testRegistry(t))
	collection := map[string]*destination.RemoteDestination{
		"ga4-production": {Destination: &client.Destination{
			ID: "dst-2", Name: "GA4", Type: "GA4", Version: 1, IsEnabled: true,
			Config: []byte(`{"measurementId":"G-123"}`),
		}},
	}
	entities, _, err := h.Impl.FormatForExport(collection, nil, stubResolver{})
	require.NoError(t, err)
	require.Len(t, entities, 1)
	spec, ok := entities[0].Content.(*specs.Spec)
	require.True(t, ok)
	config, ok := spec.Spec["config"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "{{ .GA4_PRODUCTION_API_SECRET }}", config["api_secret"])
}

func TestImmutableDestinationTypeRejectedAtPlanTime(t *testing.T) {
	p := destination.NewProvider(nil, testRegistry(t))
	checker, ok := any(p).(provider.PlanChecker)
	require.True(t, ok, "destination provider must check immutable changes before apply")
	old := &destination.DestinationResource{ID: "dst", Type: "GA4", DefinitionVersion: 1}
	newValue := &destination.DestinationResource{ID: "dst", Type: "WEBHOOK", DefinitionVersion: 1}
	r := resources.NewResource("dst", destination.DestinationResourceType,
		resources.ResourceData{"type": "WEBHOOK"}, nil, resources.WithRawData(newValue))
	st := state.EmptyState()
	st.AddResource(&state.ResourceState{ID: "dst", Type: destination.DestinationResourceType, InputRaw: old})
	plan := &planner.Plan{Operations: []*planner.Operation{{Type: planner.Update, Resource: r}}}

	err := checker.CheckPlan(context.Background(), plan, st)
	require.Error(t, err)
	assert.ErrorContains(t, err, "type")
}

func TestImportDoesNotSilentlySkipUnsupportedDestination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"destinations":[{"id":"dst-unsupported","name":"Unknown","type":"UNKNOWN","version":1,"config":{}}],"paging":{"total":1}}`))
	}))
	t.Cleanup(srv.Close)
	h := destination.NewHandler(newTestClient(t, srv.URL), testRegistry(t))

	_, err := h.Impl.LoadImportableResources(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, "dst-unsupported")
}
