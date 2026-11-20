package destination_test

import (
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/redis"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/webhook"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManagedDestinationEmptyValuesConverge pins that a spec writing an empty
// value plans nothing once the remote holds the same empty value: a webhook's
// `headers: []` and a redis `prefix: ""` (the S6/S8 regressions).
func TestManagedDestinationEmptyValuesConverge(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(webhook.NewDefinition()))
	require.NoError(t, registry.Register(redis.NewDefinition()))
	h := destination.NewHandler(nil, registry)
	version := int64(1)

	cases := []struct {
		name, typ, apiType string
		spec               map[string]any
		remote             string
	}{
		{
			name: "webhook headers: []", typ: "webhook", apiType: "WEBHOOK",
			spec:   map[string]any{"webhook_url": "https://example.com/hook", "webhook_method": "POST", "headers": []any{}},
			remote: `{"webhookUrl":"https://example.com/hook","webhookMethod":"POST","headers":[]}`,
		},
		{
			name: `redis prefix: ""`, typ: "redis", apiType: "REDIS",
			spec:   map[string]any{"address": "redis:6379", "prefix": "", "secure": false, "cluster_mode": false, "database": "1"},
			remote: `{"address":"redis:6379","prefix":"","secure":false,"clusterMode":false,"database":"1","skipVerify":false,"useJSONModule":false}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			local, err := h.Impl.ExtractResourcesFromSpec("", &destination.DestinationSpec{
				ID: "dest", DisplayName: "Dest", Type: tc.typ, Enabled: true, DefinitionVersion: &version, Config: tc.spec,
			})
			require.NoError(t, err)
			remote, _, err := h.Impl.MapRemoteToState(&destination.RemoteDestination{Destination: &client.Destination{
				ID: "dst-1", ExternalID: "dest", Name: "Dest", Type: tc.apiType, Version: 1, IsEnabled: true, Config: []byte(tc.remote),
			}}, urnResolver{})
			require.NoError(t, err)

			var source, target map[string]any
			require.NoError(t, mapstructure.Decode(remote, &source))
			require.NoError(t, mapstructure.Decode(local["dest"], &target))
			diffs, _ := differ.CompareData(source, target)
			assert.Empty(t, diffs)
		})
	}
}
