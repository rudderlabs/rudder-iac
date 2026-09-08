package bingadsofflineconversions_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	bingadsofflineconversions "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/bingads_offline_conversions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(bingadsofflineconversions.NewDefinition()))

	registered, err := registry.Get("bingads_offline_conversions", 1)
	require.NoError(t, err)

	assert.Equal(t, "bingads_offline_conversions", registered.Type)
	assert.Equal(t, "BINGADS_OFFLINE_CONVERSIONS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys())
	assert.Empty(t, registered.GatedKeyPaths())

	expectedSourceTypes := []string{"warehouse"}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	assert.NotContains(t, registered.SupportedSourceTypes(), "web")
	assert.NotContains(t, registered.SupportedSourceTypes(), "cloud")

	byAPI, err := registry.GetByAPIType("BINGADS_OFFLINE_CONVERSIONS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestBingAdsOfflineConversionsApplyDefaults(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(bingadsofflineconversions.NewDefinition()))
	registered, err := registry.Get("bingads_offline_conversions", 1)
	require.NoError(t, err)

	t.Run("fills is_hash_required when omitted", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, map[string]any{
			"rudder_account_id":   "account-1",
			"customer_account_id": "53212345",
			"customer_id":         "34376598",
			"is_hash_required":    false,
		}, registered.ApplyDefaults(validMinimalConfig()))
	})

	t.Run("keeps explicit is_hash_required value", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["is_hash_required"] = true

		assert.Equal(t, map[string]any{
			"rudder_account_id":   "account-1",
			"customer_account_id": "53212345",
			"customer_id":         "34376598",
			"is_hash_required":    true,
		}, registered.ApplyDefaults(config))
	})
}

func TestBingAdsOfflineConversionsConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(bingadsofflineconversions.NewDefinition()))
	registered, err := registry.Get("bingads_offline_conversions", 1)
	require.NoError(t, err)

	t.Run("missing rudder_account_id", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		delete(config, "rudder_account_id")

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/rudder_account_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing customer_account_id", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		delete(config, "customer_account_id")

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/customer_account_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing customer_id", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		delete(config, "customer_id")

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/customer_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("numeric ids reject invalid literals", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name  string
			field string
			value string
		}{
			{name: "customer_account_id with letters", field: "customer_account_id", value: "acct-123"},
			{name: "customer_account_id empty", field: "customer_account_id", value: ""},
			{name: "customer_id with letters", field: "customer_id", value: "customer-456"},
			{name: "customer_id env value", field: "customer_id", value: "env.BINGADS_CUSTOMER_ID"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config[tc.field] = tc.value

				errors := registered.ValidateConfig(config)

				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+tc.field, errors[0].Path)
			})
		}
	})

	t.Run("numeric ids accept ui templates", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["customer_account_id"] = "{{ context.customerAccountId || 53212345 }}"
		config["customer_id"] = "{{ context.customerId || 34376598 }}"

		errors := registered.ValidateConfig(config)

		assert.Empty(t, errors)
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(validMinimalConfig())

		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(validFullConfig())

		assert.Empty(t, errors)
	})

	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(exampleConfig())

		assert.Empty(t, errors)
	})

	t.Run("is_hash_required false is valid", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["is_hash_required"] = false

		errors := registered.ValidateConfig(config)

		assert.Empty(t, errors)
	})

	t.Run("connection_mode accepts supported mode", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["connection_mode"] = map[string]any{"warehouse": "cloud"}

		errors := registered.ValidateConfig(config)

		assert.Empty(t, errors)
	})

	t.Run("connection_mode rejects unsupported mode", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["connection_mode"] = map[string]any{"warehouse": "device"}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["not_a_field"] = true

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("legacy one trust consent key rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["one_trust_cookie_categories"] = map[string]any{"warehouse": []any{}}

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/one_trust_cookie_categories", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("legacy ketch consent key rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["ketch_consent_purposes"] = map[string]any{"warehouse": []any{}}

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/ketch_consent_purposes", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["consent_management"] = map[string]any{
			"web": []any{},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'web' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["consent_management"] = map[string]any{
			"warehouse": []any{
				map[string]any{"provider": "unknown"},
			},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestBingAdsOfflineConversionsConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := bingadsofflineconversions.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"rudder_account_id": "account-1",
				"customer_account_id": "53212345",
				"customer_id": "34376598"
			}`,
			APIJSON: `{
				"rudderAccountId": "account-1",
				"customerAccountId": "53212345",
				"customerId": "34376598"
			}`,
		},
		{
			Name: "full scalar fields",
			LocalJSON: `{
				"rudder_account_id": "account-1",
				"customer_account_id": "53212345",
				"customer_id": "34376598",
				"is_hash_required": true,
				"connection_mode": {
					"warehouse": "cloud"
				}
			}`,
			APIJSON: `{
				"rudderAccountId": "account-1",
				"customerAccountId": "53212345",
				"customerId": "34376598",
				"isHashRequired": true,
				"connectionMode": {
					"warehouse": "cloud"
				}
			}`,
		},
		{
			Name: "warehouse consent",
			LocalJSON: `{
				"rudder_account_id": "account-1",
				"customer_account_id": "53212345",
				"customer_id": "34376598",
				"consent_management": {
					"warehouse": [
						{
							"provider": "custom",
							"resolution_strategy": "and",
							"consents": ["marketing", "analytics"]
						}
					]
				}
			}`,
			APIJSON: `{
				"rudderAccountId": "account-1",
				"customerAccountId": "53212345",
				"customerId": "34376598",
				"consentManagement": {
					"warehouse": [
						{
							"provider": "custom",
							"resolutionStrategy": "and",
							"consents": [
								{"consent": "marketing"},
								{"consent": "analytics"}
							]
						}
					]
				}
			}`,
		},
	})
}

func validMinimalConfig() map[string]any {
	return map[string]any{
		"rudder_account_id":   "account-1",
		"customer_account_id": "53212345",
		"customer_id":         "34376598",
	}
}

func validFullConfig() map[string]any {
	config := exampleConfig()
	config["is_hash_required"] = true
	return config
}

func exampleConfig() map[string]any {
	return map[string]any{
		"rudder_account_id":   "account-1",
		"customer_account_id": "53212345",
		"customer_id":         "34376598",
		"connection_mode":     map[string]any{"warehouse": "cloud"},
		"consent_management": map[string]any{
			"warehouse": []any{
				map[string]any{
					"provider":            "custom",
					"resolution_strategy": "and",
					"consents":            []any{"marketing", "analytics"},
				},
			},
		},
	}
}
