package tiktokaudience_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
	tiktokaudience "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/tiktok_audience"
)

func register(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(tiktokaudience.NewDefinition()))

	registered, err := registry.Get("tiktok_audience", 1)
	require.NoError(t, err)
	return registered
}

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(tiktokaudience.NewDefinition()))

	registered, err := registry.Get("tiktok_audience", 1)
	require.NoError(t, err)

	assert.Equal(t, "tiktok_audience", registered.Type)
	assert.Equal(t, "TIKTOK_AUDIENCE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys())

	assert.Equal(t, []string{"warehouse"}, registered.SupportedSourceTypes())

	modes, err := registered.ConnectionModes("warehouse")
	require.NoError(t, err)
	assert.Equal(t, []string{"cloud"}, modes)

	// Not to be confused with tiktok_ads, the event stream destination.
	assert.NotContains(t, registered.SupportedSourceTypes(), "web")

	// ad_account_id sits under destConfig.warehouse rather than defaultConfig.
	assert.Equal(t, map[string][]string{
		"ad_account_id": {"warehouse"},
	}, registered.GatedKeyPaths())

	assert.Equal(t, []string{"mirror"}, registered.SyncBehaviours())
	assert.False(t, registered.SupportsVisualMapper())
	assert.True(t, registered.SupportsVisualMapperV2())

	byAPI, err := registry.GetByAPIType("TIKTOK_AUDIENCE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestTiktokAudienceConfigValidation(t *testing.T) {
	t.Parallel()

	registered := register(t)

	t.Run("missing rudder_account_id", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.Len(t, errors, 1)
		assert.Equal(t, "/rudder_account_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("connection_mode rejects an unsupported mode", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "tiktok-account",
			"connection_mode":   map[string]any{"warehouse": "device"},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("consent_management rejected as unknown", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id":  "tiktok-account",
			"consent_management": map[string]any{"warehouse": []any{}},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/consent_management", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	// The spec shipped in the README.
	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "tiktok-audience-account",
			"advertiser_id":     "7000000000000000000",
			"connection_mode":   map[string]any{"warehouse": "cloud"},
		}))
	})

	// Upstream declares no constraint on any key but rudder_account_id, so
	// these pass validation and TikTok reports a bad value at apply.
	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "tiktok-account",
			"advertiser_id":     "7000000000000000000",
			"ad_account_id":     "7000000000000000001",
			"audience_id":       "aud-1",
			"connection_mode":   map[string]any{"warehouse": "cloud"},
		}))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "tiktok-account",
			"not_a_field":       true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})
}

func TestTiktokAudienceConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := tiktokaudience.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name:      "minimal",
			LocalJSON: `{"rudder_account_id": "tiktok-account"}`,
			APIJSON:   `{"rudderAccountId": "tiktok-account"}`,
		},
		{
			Name: "full with the gated warehouse key",
			LocalJSON: `{
				"rudder_account_id": "tiktok-account",
				"advertiser_id": "7000000000000000000",
				"ad_account_id": "7000000000000000001",
				"audience_id": "aud-1",
				"connection_mode": {"warehouse": "cloud"}
			}`,
			APIJSON: `{
				"rudderAccountId": "tiktok-account",
				"advertiserId": "7000000000000000000",
				"adAccountId": "7000000000000000001",
				"audienceId": "aud-1",
				"connectionMode": {"warehouse": "cloud"}
			}`,
		},
	})
}
