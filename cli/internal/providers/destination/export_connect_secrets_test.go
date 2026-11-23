package destination_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/braze"
	facebookpixel "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/facebook_pixel"
	retlConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProvider_FormatForExport_ConnectTimeSecretsFollowImportedConnections pins
// that a secret only connecting a warehouse source needs (V-C5) gets a
// reference only when the import connects a Reverse ETL source to that
// destination: a Braze destination fed only by a web source in device mode
// does not need rest_api_key.
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
		"no Reverse ETL connection in the import, so a warehouse source's requirement does not apply")
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
}
