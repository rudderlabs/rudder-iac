package linkedinaudience_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	linkedinaudience "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/linkedin_audience"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func register(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(linkedinaudience.NewDefinition()))

	registered, err := registry.Get("linkedin_audience", 1)
	require.NoError(t, err)
	return registered
}

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(linkedinaudience.NewDefinition()))

	registered, err := registry.Get("linkedin_audience", 1)
	require.NoError(t, err)

	assert.Equal(t, "linkedin_audience", registered.Type)
	assert.Equal(t, "LINKEDIN_AUDIENCE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys())

	assert.Equal(t, []string{"warehouse"}, registered.SupportedSourceTypes())

	modes, err := registered.ConnectionModes("warehouse")
	require.NoError(t, err)
	assert.Equal(t, []string{"cloud"}, modes)

	// Not to be confused with linkedin_ads, which takes eleven source types.
	assert.NotContains(t, registered.SupportedSourceTypes(), "web")

	assert.Equal(t, []string{"mirror"}, registered.SyncBehaviours())
	assert.False(t, registered.SupportsVisualMapper())
	assert.True(t, registered.SupportsVisualMapperV2())

	byAPI, err := registry.GetByAPIType("LINKEDIN_AUDIENCE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestLinkedinAudienceConfigValidation(t *testing.T) {
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
			"rudder_account_id": "linkedin-account",
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
			"rudder_account_id":  "linkedin-account",
			"consent_management": map[string]any{"warehouse": []any{}},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/consent_management", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	// The spec shipped in the README. The account and the connection mode are
	// the whole config surface.
	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "linkedin-audience-account",
			"connection_mode":   map[string]any{"warehouse": "cloud"},
		}))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "linkedin-account",
			"ad_account_id":     "urn:li:sponsoredAccount:123",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/ad_account_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})
}

func TestLinkedinAudienceConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := linkedinaudience.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name:      "minimal",
			LocalJSON: `{"rudder_account_id": "linkedin-account"}`,
			APIJSON:   `{"rudderAccountId": "linkedin-account"}`,
		},
		{
			Name: "with connection mode",
			LocalJSON: `{
				"rudder_account_id": "linkedin-account",
				"connection_mode": {"warehouse": "cloud"}
			}`,
			APIJSON: `{
				"rudderAccountId": "linkedin-account",
				"connectionMode": {"warehouse": "cloud"}
			}`,
		},
	})
}
