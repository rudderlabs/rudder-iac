package brazeaudience_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	brazeaudience "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/braze_audience"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func register(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(brazeaudience.NewDefinition()))

	registered, err := registry.Get("braze_audience", 1)
	require.NoError(t, err)
	return registered
}

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(brazeaudience.NewDefinition()))

	registered, err := registry.Get("braze_audience", 1)
	require.NoError(t, err)

	assert.Equal(t, "braze_audience", registered.Type)
	assert.Equal(t, "BRAZE_AUDIENCE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"rest_api_key"}, registered.SecretKeys())

	assert.Equal(t, []string{"warehouse"}, registered.SupportedSourceTypes())

	modes, err := registered.ConnectionModes("warehouse")
	require.NoError(t, err)
	assert.Equal(t, []string{"cloud"}, modes)

	// Warehouse-only upstream: rETL is the only way in.
	assert.NotContains(t, registered.SupportedSourceTypes(), "cloud")

	assert.Equal(t, []string{"mirror"}, registered.SyncBehaviours())
	assert.False(t, registered.SupportsVisualMapper())
	assert.True(t, registered.SupportsVisualMapperV2())

	byAPI, err := registry.GetByAPIType("BRAZE_AUDIENCE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestBrazeAudienceConfigValidation(t *testing.T) {
	t.Parallel()

	registered := register(t)

	t.Run("missing rudder_account_id", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.Len(t, errors, 1)
		assert.Equal(t, "/rudder_account_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	// schema.json constrains rudderAccountId to a bare `^.{1,100}$` with no
	// {{ … || … }} branch, so the tag is plain `pattern`: length and
	// single-line-ness, nothing more.
	t.Run("rudder_account_id rejects overlong and multiline values", func(t *testing.T) {
		t.Parallel()
		for _, value := range []string{strings.Repeat("a", 101), "bad\nvalue"} {
			errors := registered.ValidateConfig(map[string]any{"rudder_account_id": value})
			require.Len(t, errors, 1, value)
			assert.Equal(t, "/rudder_account_id", errors[0].Path)
		}
	})

	t.Run("connection_mode rejects an unsupported mode", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "braze-account",
			"connection_mode":   map[string]any{"warehouse": "device"},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	// db-config scopes only connectionMode to warehouse, so there is no consent
	// surface to model.
	t.Run("consent_management rejected as unknown", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id":  "braze-account",
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
			"rudder_account_id": "braze-audience-account",
			"connection_mode":   map[string]any{"warehouse": "cloud"},
		}))
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "braze-account",
			"rest_api_key":      "braze-rest-api-key",
			"data_center":       "US-03",
			"connection_mode":   map[string]any{"warehouse": "cloud"},
		}))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "braze-account",
			"not_a_field":       true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})
}

func TestBrazeAudienceConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := brazeaudience.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name:      "minimal",
			LocalJSON: `{"rudder_account_id": "braze-account"}`,
			APIJSON:   `{"rudderAccountId": "braze-account"}`,
		},
		{
			Name: "full",
			LocalJSON: `{
				"rudder_account_id": "braze-account",
				"rest_api_key": "braze-rest-api-key",
				"data_center": "US-03",
				"connection_mode": {"warehouse": "cloud"}
			}`,
			APIJSON: `{
				"rudderAccountId": "braze-account",
				"restApiKey": "braze-rest-api-key",
				"dataCenter": "US-03",
				"connectionMode": {"warehouse": "cloud"}
			}`,
		},
	})
}
