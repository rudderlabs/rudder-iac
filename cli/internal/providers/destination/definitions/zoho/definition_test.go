package zoho_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/zoho"
)

func register(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(zoho.NewDefinition()))

	registered, err := registry.Get("zoho", 1)
	require.NoError(t, err)
	return registered
}

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(zoho.NewDefinition()))

	registered, err := registry.Get("zoho", 1)
	require.NoError(t, err)

	assert.Equal(t, "zoho", registered.Type)
	assert.Equal(t, "ZOHO", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys())

	assert.Equal(t, []string{"warehouse"}, registered.SupportedSourceTypes())

	modes, err := registered.ConnectionModes("warehouse")
	require.NoError(t, err)
	assert.Equal(t, []string{"cloud"}, modes)

	assert.Equal(t, []string{"mirror"}, registered.SyncBehaviours())
	assert.False(t, registered.SupportsVisualMapper())
	assert.True(t, registered.SupportsVisualMapperV2())

	byAPI, err := registry.GetByAPIType("ZOHO", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestZohoConfigValidation(t *testing.T) {
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
			"rudder_account_id": "zoho-account",
			"connection_mode":   map[string]any{"warehouse": "device"},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	// The spec shipped in the README.
	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "zoho-account",
			"connection_mode":   map[string]any{"warehouse": "cloud"},
		}))
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "zoho-account",
			"connection_mode":   map[string]any{"warehouse": "cloud"},
			"consent_management": map[string]any{
				"warehouse": []any{
					map[string]any{
						"provider":            "custom",
						"resolution_strategy": "and",
						"consents":            []any{"one", "two"},
					},
				},
			},
		}))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "zoho-account",
			"not_a_field":       true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id":  "zoho-account",
			"consent_management": map[string]any{"web": []any{}},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'web' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "zoho-account",
			"consent_management": map[string]any{
				"warehouse": []any{map[string]any{"provider": "unknown"}},
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestZohoConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := zoho.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name:      "minimal",
			LocalJSON: `{"rudder_account_id": "zoho-account"}`,
			APIJSON:   `{"rudderAccountId": "zoho-account"}`,
		},
		{
			Name: "full with consent",
			LocalJSON: `{
				"rudder_account_id": "zoho-account",
				"connection_mode": {"warehouse": "cloud"},
				"consent_management": {
					"warehouse": [
						{
							"provider": "custom",
							"resolution_strategy": "and",
							"consents": ["one_warehouse", "two_warehouse"]
						}
					]
				}
			}`,
			APIJSON: `{
				"rudderAccountId": "zoho-account",
				"connectionMode": {"warehouse": "cloud"},
				"consentManagement": {
					"warehouse": [
						{
							"provider": "custom",
							"resolutionStrategy": "and",
							"consents": [
								{"consent": "one_warehouse"},
								{"consent": "two_warehouse"}
							]
						}
					]
				}
			}`,
		},
	})
}
