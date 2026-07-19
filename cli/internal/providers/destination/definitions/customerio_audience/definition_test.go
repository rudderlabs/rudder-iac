package customerioaudience_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	customerioaudience "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/customerio_audience"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(customerioaudience.NewDefinition()))

	registered, err := registry.Get("customerio_audience", 1)
	require.NoError(t, err)

	assert.Equal(t, "customerio_audience", registered.Type)
	assert.Equal(t, "CUSTOMERIO_AUDIENCE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"api_key", "app_api_key"}, registered.SecretKeys())

	expectedSourceTypes := []string{"warehouse"}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	assert.NotContains(t, registered.SupportedSourceTypes(), "android")
	assert.NotContains(t, registered.SupportedSourceTypes(), "web")
	assert.NotContains(t, registered.SupportedSourceTypes(), "cloud")

	byAPI, err := registry.GetByAPIType("CUSTOMERIO_AUDIENCE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestCustomerioAudienceConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(customerioaudience.NewDefinition()))
	registered, err := registry.Get("customerio_audience", 1)
	require.NoError(t, err)

	t.Run("missing site_id", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key":     "api-key-1",
			"app_api_key": "app-api-key-1",
			"region":      "US",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/site_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing api_key", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"site_id":     "site-id-1",
			"app_api_key": "app-api-key-1",
			"region":      "US",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_key", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing app_api_key", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"site_id": "site-id-1",
			"api_key": "api-key-1",
			"region":  "US",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/app_api_key", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing region", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"site_id":     "site-id-1",
			"api_key":     "api-key-1",
			"app_api_key": "app-api-key-1",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/region", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("invalid region", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"site_id":     "site-id-1",
			"api_key":     "api-key-1",
			"app_api_key": "app-api-key-1",
			"region":      "APAC",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/region", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("invalid connection_mode warehouse", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"site_id":     "site-id-1",
			"api_key":     "api-key-1",
			"app_api_key": "app-api-key-1",
			"region":      "US",
			"connection_mode": map[string]any{
				"warehouse": "device",
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/connection_mode/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"site_id":     "site-id-1",
			"api_key":     "api-key-1",
			"app_api_key": "app-api-key-1",
			"region":      "US",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"site_id":     "site-id-1",
			"api_key":     "api-key-1",
			"app_api_key": "app-api-key-1",
			"region":      "EU",
			"connection_mode": map[string]any{
				"warehouse": "cloud",
			},
			"consent_management": map[string]any{
				"warehouse": []any{
					map[string]any{
						"provider":            "custom",
						"resolution_strategy": "and",
						"consents":            []any{"one_warehouse", "two_warehouse"},
					},
				},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"site_id":     "abc123site",
			"api_key":     "track-api-key-secret",
			"app_api_key": "app-api-key-secret",
			"region":      "US",
			"connection_mode": map[string]any{
				"warehouse": "cloud",
			},
			"consent_management": map[string]any{
				"warehouse": []any{
					map[string]any{
						"provider": "oneTrust",
						"consents": []any{"analytics"},
					},
				},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"site_id":     "site-id-1",
			"api_key":     "api-key-1",
			"app_api_key": "app-api-key-1",
			"region":      "US",
			"not_a_field": true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"site_id":     "site-id-1",
			"api_key":     "api-key-1",
			"app_api_key": "app-api-key-1",
			"region":      "US",
			"consent_management": map[string]any{
				"web": []any{},
			},
		})

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'web' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"site_id":     "site-id-1",
			"api_key":     "api-key-1",
			"app_api_key": "app-api-key-1",
			"region":      "US",
			"consent_management": map[string]any{
				"warehouse": []any{
					map[string]any{"provider": "unknown"},
				},
			},
		})

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestCustomerioAudienceConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := customerioaudience.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"site_id": "site-id-1",
				"api_key": "api-key-1",
				"app_api_key": "app-api-key-1",
				"region": "US"
			}`,
			APIJSON: `{
				"siteId": "site-id-1",
				"apiKey": "api-key-1",
				"appApiKey": "app-api-key-1",
				"region": "US"
			}`,
		},
		{
			Name: "full with connection mode and consent",
			LocalJSON: `{
				"site_id": "site-id-1",
				"api_key": "api-key-1",
				"app_api_key": "app-api-key-1",
				"region": "EU",
				"connection_mode": {
					"warehouse": "cloud"
				},
				"consent_management": {
					"warehouse": [
						{
							"provider": "custom",
							"resolution_strategy": "and",
							"consents": ["one_warehouse", "two_warehouse", "three_warehouse"]
						}
					]
				}
			}`,
			APIJSON: `{
				"siteId": "site-id-1",
				"apiKey": "api-key-1",
				"appApiKey": "app-api-key-1",
				"region": "EU",
				"connectionMode": {
					"warehouse": "cloud"
				},
				"consentManagement": {
					"warehouse": [
						{
							"provider": "custom",
							"resolutionStrategy": "and",
							"consents": [
								{"consent": "one_warehouse"},
								{"consent": "two_warehouse"},
								{"consent": "three_warehouse"}
							]
						}
					]
				}
			}`,
		},
	})
}
