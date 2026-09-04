package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposeProvidersIncludesDataGraph(t *testing.T) {
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	c, err := client.New("test-token")
	require.NoError(t, err)

	composite, providers, err := composeProviders(c)
	require.NoError(t, err)
	require.NotNil(t, providers.DataGraph)

	cp, ok := composite.(*provider.CompositeProvider)
	require.True(t, ok)
	assert.Same(t, providers.DataGraph, cp.Providers["datagraph"])
}

func TestNewDestinationRegistryFlagMatrix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                   string
		destinationSupport     bool
		unverifiedDestinations bool
		wantTypes              []string
	}{
		{
			name:                   "both flags disabled",
			destinationSupport:     false,
			unverifiedDestinations: false,
			wantTypes:              []string{},
		},
		{
			name:                   "destinationSupport off ignores unverifiedDestinations",
			destinationSupport:     false,
			unverifiedDestinations: true,
			wantTypes:              []string{},
		},
		{
			name:                   "destinationSupport on without unverifiedDestinations",
			destinationSupport:     true,
			unverifiedDestinations: false,
			wantTypes:              []string{"bqstream", "s3"},
		},
		{
			name:                   "both flags enabled registers verified and unverified destinations",
			destinationSupport:     true,
			unverifiedDestinations: true,
			wantTypes:              []string{"active_campaign", "adj", "adobe_analytics", "am", "attentive_tag", "bingads_offline_conversions", "bq", "bqstream", "braze", "confluent_cloud", "customerio", "customerio_audience", "facebook_conversions", "facebook_pixel", "firebase", "ga", "ga4", "gcs", "google_adwords_offline_conversions", "googleads", "googlepubsub", "googlesheets", "gtm", "hs", "http", "intercom", "iterable", "kafka", "kinesis", "linkedin_ads", "linkedin_insight_tag", "marketo", "mp", "postgres", "posthog", "qualtrics", "redis", "rs", "s3", "s3_datalake", "salesforce", "sentry", "slack", "snowflake", "snowpipe_streaming", "statsig", "tiktok_ads", "vwo", "webhook", "zendesk"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.Config{
				ExperimentalFlags: config.ExperimentalConfig{
					DestinationSupport:     tc.destinationSupport,
					UnverifiedDestinations: tc.unverifiedDestinations,
				},
			}

			registry, err := newDestinationRegistry(cfg)
			require.NoError(t, err)
			assert.Equal(t, tc.wantTypes, registry.SupportedTypes())
		})
	}
}

// Every definition that models connection_mode must also convert it. A config
// field without the matching converter properties passes validation but drops
// the key from the API payload, erasing whatever the user set upstream on the
// first apply — so assert the mapping fleet-wide rather than per definition.
func TestDestinationConnectionModeIsConverted(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		ExperimentalFlags: config.ExperimentalConfig{
			DestinationSupport:     true,
			UnverifiedDestinations: true,
		},
	}

	registry, err := newDestinationRegistry(cfg)
	require.NoError(t, err)

	apiSourceTypes := common.LocalToAPISourceTypes()

	for _, destType := range registry.SupportedTypes() {
		versions, err := registry.Versions(destType)
		require.NoError(t, err)

		for _, version := range versions {
			def, err := registry.Get(destType, version)
			require.NoError(t, err)

			if !modelsConnectionMode(def) {
				continue
			}

			for _, sourceType := range def.SupportedSourceTypes() {
				modes, err := def.ConnectionModes(sourceType)
				require.NoError(t, err)
				require.NotEmpty(t, modes, "%s: no connection modes for %s", destType, sourceType)

				api, err := def.LocalToAPI(map[string]any{
					"connection_mode": map[string]any{sourceType: modes[0]},
				})
				require.NoError(t, err)

				assert.Equal(t,
					map[string]any{apiSourceTypes[sourceType]: modes[0]},
					api["connectionMode"],
					"%s: connection_mode.%s is not converted", destType, sourceType,
				)
			}
		}
	}
}

// modelsConnectionMode reports whether the definition's config struct declares
// the key, distinguished by the closed allowlist rejecting it when it does not.
func modelsConnectionMode(def *definitions.RegisteredDefinition) bool {
	for _, err := range def.ValidateConfig(map[string]any{"connection_mode": map[string]any{}}) {
		if err.Path == "/connection_mode" && strings.Contains(err.Message, "unknown config field") {
			return false
		}
	}
	return true
}
