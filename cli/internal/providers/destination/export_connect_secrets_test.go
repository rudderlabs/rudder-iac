package destination_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/braze"
	facebookpixel "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/facebook_pixel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/intercom"
	esConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/connection"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	retlConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProvider_FormatForExport_ConnectTimeSecretsFollowImportedConnections pins
// that a secret only connecting a warehouse source needs (V-C5) gets a
// reference when the import connects a Reverse ETL source to that destination,
// or the retained config independently requires it. A Braze destination with
// web device mode and no rETL link does not need rest_api_key.
func TestProvider_FormatForExport_ConnectTimeSecretsFollowImportedConnections(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(braze.NewDefinition()))
	require.NoError(t, registry.Register(facebookpixel.NewDefinition()))

	export := func(t *testing.T, apiType, config string, connected bool) map[string]any {
		t.Helper()
		collection := resources.NewRemoteResources()
		collection.Set(destination.DestinationResourceType, map[string]*resources.RemoteResource{
			"dst-1": {ID: "dst-1", ExternalID: "dest", Data: &destination.RemoteDestination{Destination: &client.Destination{
				ID: "dst-1", Name: "Dest", Type: apiType, Version: 1, IsEnabled: true, Config: []byte(config),
			}}},
		})
		if connected {
			collection.Set(retlConnection.ResourceType, map[string]*resources.RemoteResource{
				"conn-1": {ID: "conn-1", ExternalID: "src-to-dest", Data: &retlConnection.RemoteConnection{
					RETLConnection: retlClient.RETLConnection{ID: "conn-1", DestinationID: "dst-1"},
				}},
			})
		}
		entities, _, err := destination.NewProvider(nil, registry).FormatForExport(collection, nil, stubResolver{})
		require.NoError(t, err)
		require.Len(t, entities, 1)
		spec, ok := entities[0].Content.(*specs.Spec)
		require.True(t, ok)
		config2, ok := spec.Spec["config"].(map[string]any)
		require.True(t, ok)
		return config2
	}

	webDevice := `{"dataCenter":"US-01","connectionMode":{"web":"device"}}`
	assert.NotContains(t, export(t, "BRAZE", webDevice, false), "rest_api_key",
		"device mode does not require the cloud credential")
	assert.Equal(t, "{{ .DEST_REST_API_KEY }}", export(t, "BRAZE", webDevice, true)["rest_api_key"],
		"a Reverse ETL connection in the import needs rest_api_key in cloud mode")
	assert.Equal(t, "{{ .DEST_REST_API_KEY }}", export(t, "BRAZE", `{"dataCenter":"US-01"}`, true)["rest_api_key"],
		"no warehouse mode stored: an absent warehouse mode means cloud")
	assert.Equal(t, "{{ .DEST_ACCESS_TOKEN }}", export(t, "FACEBOOK_PIXEL", `{"connectionMode":{"warehouse":"cloud"}}`, true)["access_token"])
	assert.NotContains(t, export(t, "FACEBOOK_PIXEL", `{"connectionMode":{"warehouse":"cloud"}}`, false), "access_token")
}

// TestRETLConnectionResourceTypeMatches pins the resource type the destination
// provider reads Reverse ETL connections under to the one that package uses.
func TestRETLConnectionResourceTypeMatches(t *testing.T) {
	t.Parallel()
	assert.Equal(t, retlConnection.ResourceType, destination.RETLConnectionResourceType)
	assert.Equal(t, esConnection.EventStreamConnectionResourceType, destination.EventStreamConnectionResourceType)
}

func TestProvider_FormatForExport_EventStreamConnectSecrets(t *testing.T) {
	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(braze.NewDefinition()))
	require.NoError(t, registry.Register(intercom.NewDefinition()))

	// Resolver succeeds only for endpoints in the importable collection.
	// A managed endpoint with an external ID can still be referenced, and a
	// matched connection (--merge) adopts its existing spec without resolving.
	resolver := stubResolver{fn: func(kind, id string) (string, error) {
		if (kind == esSource.ResourceType && id == "src-1") ||
			(kind == destination.DestinationResourceType && id == "dst-1") {
			return "#" + kind + ":imported", nil
		}
		return "", errors.New("unresolvable")
	}}
	export := func(t *testing.T, apiType, config string, links ...*resources.RemoteResource) map[string]any {
		t.Helper()
		collection := resources.NewRemoteResources()
		collection.Set(destination.DestinationResourceType, map[string]*resources.RemoteResource{
			"dst-1": {ID: "dst-1", ExternalID: "dest", Data: &destination.RemoteDestination{Destination: &client.Destination{
				ID: "dst-1", Name: "Dest", Type: apiType, Version: 1, IsEnabled: true, Config: []byte(config),
			}}},
		})
		conns := map[string]*resources.RemoteResource{}
		for _, link := range links {
			conns[link.ID] = link
		}
		collection.Set(esConnection.EventStreamConnectionResourceType, conns)
		entities, _, err := destination.NewProvider(nil, registry).FormatForExport(collection, nil, resolver)
		require.NoError(t, err)
		require.Len(t, entities, 1)
		spec := entities[0].Content.(*specs.Spec)
		return spec.Spec["config"].(map[string]any)
	}
	link := func(id, srcID, srcType, dstID, srcExternalID, dstExternalID string, matched bool) *resources.RemoteResource {
		res := &resources.RemoteResource{ID: id, Data: &esConnection.RemoteConnection{
			Connection: client.Connection{SourceID: srcID, DestinationID: dstID},
			SourceType: srcType, SourceExternalID: srcExternalID, DestinationExternalID: dstExternalID,
		}}
		if matched {
			res.MatchedWith = resources.NewResource(id, esConnection.EventStreamConnectionResourceType, resources.ResourceData{}, nil)
		}
		return res
	}

	cases := []struct {
		name, apiType, config, secret string
		links                         []*resources.RemoteResource
		wantSecret                    bool
	}{
		{"no connections but cloud mode requires local credential", "BRAZE", `{"dataCenter":"US-01","connectionMode":{"web":"cloud"}}`, "rest_api_key", nil, true},
		{"web omitted mode implies cloud", "BRAZE", `{"dataCenter":"US-01"}`, "rest_api_key", []*resources.RemoteResource{link("c1", "src-1", "javascript", "dst-1", "", "", false)}, true},
		{"cloud source omitted mode", "BRAZE", `{"dataCenter":"US-01"}`, "rest_api_key", []*resources.RemoteResource{link("c1", "src-1", "python", "dst-1", "", "", false)}, true},
		{"web device mode does not need cloud secret", "BRAZE", `{"dataCenter":"US-01","connectionMode":{"web":"device"}}`, "rest_api_key", []*resources.RemoteResource{link("c1", "src-1", "javascript", "dst-1", "", "", false)}, false},
		{"stored cloud mode requires credential regardless of active link", "BRAZE", `{"dataCenter":"US-01","connectionMode":{"web":"cloud","android":"device"}}`, "rest_api_key", []*resources.RemoteResource{link("c1", "src-1", "android", "dst-1", "", "", false)}, true},
		{"connection to another destination ignored", "BRAZE", `{"dataCenter":"US-01"}`, "rest_api_key", []*resources.RemoteResource{link("c1", "src-1", "javascript", "dst-other", "", "", false)}, false},
		{"unresolvable source skipped", "BRAZE", `{"dataCenter":"US-01"}`, "rest_api_key", []*resources.RemoteResource{link("c1", "src-missing", "javascript", "dst-1", "", "", false)}, false},
		{"managed endpoint fallback", "BRAZE", `{"dataCenter":"US-01"}`, "rest_api_key", []*resources.RemoteResource{link("c1", "src-managed", "javascript", "dst-1", "local-src", "", false)}, true},
		{"merge matched link", "BRAZE", `{"dataCenter":"US-01"}`, "rest_api_key", []*resources.RemoteResource{link("c1", "src-missing", "javascript", "dst-1", "", "", true)}, true},
		{"explicit empty mode not cloud", "BRAZE", `{"dataCenter":"US-01","connectionMode":{"web":""}}`, "rest_api_key", []*resources.RemoteResource{link("c1", "src-1", "javascript", "dst-1", "", "", false)}, false},
		{"explicit invalid mode not cloud", "BRAZE", `{"dataCenter":"US-01","connectionMode":{"web":"invalid"}}`, "rest_api_key", []*resources.RemoteResource{link("c1", "src-1", "javascript", "dst-1", "", "", false)}, false},
		{"empty returned secret stays unset", "BRAZE", `{"dataCenter":"US-01","restApiKey":""}`, "rest_api_key", []*resources.RemoteResource{link("c1", "src-1", "javascript", "dst-1", "", "", false)}, false},
		{"null returned secret stays unset", "BRAZE", `{"dataCenter":"US-01","restApiKey":null}`, "rest_api_key", []*resources.RemoteResource{link("c1", "src-1", "javascript", "dst-1", "", "", false)}, false},
		{"intercom cloud mode omitted", "INTERCOM", `{}`, "api_key", []*resources.RemoteResource{link("c1", "src-1", "javascript", "dst-1", "", "", false)}, true},
		{"intercom device key follows explicit mode", "INTERCOM", `{"connectionMode":{"web":"device"}}`, "app_id", []*resources.RemoteResource{link("c1", "src-1", "javascript", "dst-1", "", "", false)}, true},
		{"intercom stored cloud mode requires credential", "INTERCOM", `{"connectionMode":{"web":"cloud"}}`, "api_key", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := export(t, tc.apiType, tc.config, tc.links...)
			if tc.wantSecret {
				assert.Equal(t, "{{ .DEST_"+strings.ToUpper(tc.secret)+" }}", config[tc.secret])
				if tc.name == "no connections but cloud mode requires local credential" {
					brazeDef, err := registry.Get("braze", 1)
					require.NoError(t, err)
					assert.Empty(t, brazeDef.ValidateConfig(config), "exported destination must validate once its secret variable is supplied")
				}
			} else {
				assert.NotContains(t, config, tc.secret)
			}
		})
	}

}
