package iterableaudience_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	iterableaudience "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/iterable_audience"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func register(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(iterableaudience.NewDefinition()))

	registered, err := registry.Get("iterable_audience", 1)
	require.NoError(t, err)
	return registered
}

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(iterableaudience.NewDefinition()))

	registered, err := registry.Get("iterable_audience", 1)
	require.NoError(t, err)

	assert.Equal(t, "iterable_audience", registered.Type)
	assert.Equal(t, "ITERABLE_AUDIENCE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"api_key"}, registered.SecretKeys())

	assert.Equal(t, []string{"warehouse"}, registered.SupportedSourceTypes())

	modes, err := registered.ConnectionModes("warehouse")
	require.NoError(t, err)
	assert.Equal(t, []string{"cloud"}, modes)

	// Warehouse-only upstream: rETL is the only way in.
	assert.NotContains(t, registered.SupportedSourceTypes(), "cloud")
	assert.NotContains(t, registered.SupportedSourceTypes(), "web")

	assert.Equal(t, []string{"mirror"}, registered.SyncBehaviours())
	assert.False(t, registered.SupportsVisualMapper())
	assert.True(t, registered.SupportsVisualMapperV2())

	byAPI, err := registry.GetByAPIType("ITERABLE_AUDIENCE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestIterableAudienceConfigValidation(t *testing.T) {
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

	t.Run("connection_mode accepts a supported mode", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "iterable-account",
			"connection_mode":   map[string]any{"warehouse": "cloud"},
		}))
	})

	t.Run("connection_mode rejects an unsupported mode", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "iterable-account",
			"connection_mode":   map[string]any{"warehouse": "device"},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	// An unsupported source type under connection_mode is not this layer's to
	// report: sourceTypeKeyResults rejects it generically in the rule pipeline,
	// so ValidateConfig skips the entry rather than double-reporting it.
	t.Run("connection_mode leaves an unsupported source type to the rule pipeline", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "iterable-account",
			"connection_mode":   map[string]any{"web": "cloud"},
		}))
	})

	// db-config scopes only connectionMode to warehouse, so there is no consent
	// surface to model.
	t.Run("consent_management rejected as unknown", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id":  "iterable-account",
			"consent_management": map[string]any{"warehouse": []any{}},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/consent_management", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	// The spec shipped in the README and used for the rETL validate proof.
	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "iterable-audience-account",
			"connection_mode":   map[string]any{"warehouse": "cloud"},
		}))
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "iterable-account",
			"api_key":           "iterable-api-key",
			"data_center":       "USDC",
			"project_type":      "web",
			"connection_mode":   map[string]any{"warehouse": "cloud"},
		}))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "iterable-account",
			"not_a_field":       true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})
}

func TestIterableAudienceConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := iterableaudience.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"rudder_account_id": "iterable-account"
			}`,
			APIJSON: `{
				"rudderAccountId": "iterable-account"
			}`,
		},
		{
			Name: "full",
			LocalJSON: `{
				"rudder_account_id": "iterable-account",
				"api_key": "iterable-api-key",
				"data_center": "USDC",
				"project_type": "web",
				"connection_mode": {"warehouse": "cloud"}
			}`,
			APIJSON: `{
				"rudderAccountId": "iterable-account",
				"apiKey": "iterable-api-key",
				"dataCenter": "USDC",
				"projectType": "web",
				"connectionMode": {"warehouse": "cloud"}
			}`,
		},
	})
}
