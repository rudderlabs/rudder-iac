package linkedinads_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	linkedinads "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/linkedin_ads"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(linkedinads.NewDefinition()))

	registered, err := registry.Get("linkedin_ads", 1)
	require.NoError(t, err)

	assert.Equal(t, "linkedin_ads", registered.Type)
	assert.Equal(t, "LINKEDIN_ADS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys())
	assert.Empty(t, registered.GatedKeyPaths())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web", "unity", "amp",
		"cloud", "warehouse", "react_native", "flutter", "cordova", "shopify",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	byAPI, err := registry.GetByAPIType("LINKEDIN_ADS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestLinkedInAdsConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(linkedinads.NewDefinition()))
	registered, err := registry.Get("linkedin_ads", 1)
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

	t.Run("missing hash_data", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		delete(config, "hash_data")

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/hash_data", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("empty rudder_account_id rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["rudder_account_id"] = ""

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/rudder_account_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(validMinimalConfig())

		assert.Empty(t, errors)
	})

	t.Run("hash_data false satisfies required", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["hash_data"] = false

		errors := registered.ValidateConfig(config)

		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(validFullConfig())

		assert.Empty(t, errors)
	})

	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id":  "1234567890",
			"hash_data":          true,
			"ad_account_id":      "urn:li:sponsoredAccount:123456789",
			"deduplication_key":  "properties.order_id",
			"conversion_mapping": []any{map[string]any{"from": "Order Completed", "to": "123456"}},
		})

		assert.Empty(t, errors)
	})

	t.Run("optional string fields reject invalid literals", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name  string
			field string
			value string
		}{
			{name: "ad_account_id over 100 characters", field: "ad_account_id", value: strings.Repeat("a", 101)},
			{name: "ad_account_id with line break", field: "ad_account_id", value: "account\nbad"},
			{name: "deduplication_key over 100 characters", field: "deduplication_key", value: strings.Repeat("d", 101)},
			{name: "deduplication_key with line break", field: "deduplication_key", value: "message\nid"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config[tc.field] = tc.value

				errors := registered.ValidateConfig(config)

				require.Len(t, errors, 1)
				assert.Equal(t, "/"+tc.field, errors[0].Path)
			})
		}
	})

	t.Run("optional string fields accept ui templates", func(t *testing.T) {
		t.Parallel()
		long := "{{ config.value || " + strings.Repeat("x", 150) + " }}"
		for _, field := range []string{"ad_account_id", "deduplication_key"} {
			t.Run(field, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config[field] = long

				errors := registered.ValidateConfig(config)

				assert.Empty(t, errors)
			})
		}
	})

	t.Run("conversion mapping fields reject invalid literals", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name  string
			entry map[string]any
			path  string
		}{
			{name: "from missing", entry: map[string]any{"to": "conversion-1"}, path: "/conversion_mapping/0/from"},
			{name: "from empty", entry: map[string]any{"from": "", "to": "conversion-1"}, path: "/conversion_mapping/0/from"},
			{name: "from over 100 characters", entry: map[string]any{"from": strings.Repeat("a", 101), "to": "conversion-1"}, path: "/conversion_mapping/0/from"},
			{name: "from with line break", entry: map[string]any{"from": "Order\nCompleted", "to": "conversion-1"}, path: "/conversion_mapping/0/from"},
			{name: "to missing", entry: map[string]any{"from": "Order Completed"}, path: "/conversion_mapping/0/to"},
			{name: "to empty", entry: map[string]any{"from": "Order Completed", "to": ""}, path: "/conversion_mapping/0/to"},
			{name: "to over 100 characters", entry: map[string]any{"from": "Order Completed", "to": strings.Repeat("1", 101)}, path: "/conversion_mapping/0/to"},
			{name: "to with line break", entry: map[string]any{"from": "Order Completed", "to": "conversion\n1"}, path: "/conversion_mapping/0/to"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config["conversion_mapping"] = []any{tc.entry}

				errors := registered.ValidateConfig(config)

				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	t.Run("conversion mapping fields accept ui templates", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["conversion_mapping"] = []any{
			map[string]any{
				"from": "{{ event.name || Order Completed }}",
				"to":   "{{ destination.conversion || 123456 }}",
			},
		}

		errors := registered.ValidateConfig(config)

		assert.Empty(t, errors)
	})

	// connection_mode is definition-level metadata (ConnectionModes), never a
	// user-settable config key — the same treatment every other definition gives it.
	t.Run("connection_mode rejected as unknown key", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["connection_mode"] = map[string]any{"web": "cloud"}

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/connection_mode", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
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

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["consent_management"] = map[string]any{
			"cloud_source": []any{},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/cloud_source", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'cloud_source' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["consent_management"] = map[string]any{
			"android_kotlin": []any{
				map[string]any{"provider": "unknown"},
			},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/android_kotlin/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestLinkedInAdsConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := linkedinads.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"rudder_account_id": "account-1",
				"hash_data": false
			}`,
			APIJSON: `{
				"rudderAccountId": "account-1",
				"hashData": false
			}`,
		},
		{
			Name: "full scalar fields",
			LocalJSON: `{
				"rudder_account_id": "account-1",
				"hash_data": true,
				"ad_account_id": "urn:li:sponsoredAccount:123456789",
				"deduplication_key": "properties.order_id"
			}`,
			APIJSON: `{
				"rudderAccountId": "account-1",
				"hashData": true,
				"adAccountId": "urn:li:sponsoredAccount:123456789",
				"deduplicationKey": "properties.order_id"
			}`,
		},
		{
			Name: "conversion mapping",
			LocalJSON: `{
				"rudder_account_id": "account-1",
				"hash_data": true,
				"conversion_mapping": [
					{"from": "Product Viewed", "to": "111111"},
					{"from": "Order Completed", "to": "222222"}
				]
			}`,
			APIJSON: `{
				"rudderAccountId": "account-1",
				"hashData": true,
				"conversionMapping": [
					{"from": "Product Viewed", "to": "111111"},
					{"from": "Order Completed", "to": "222222"}
				]
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"rudder_account_id": "account-1",
				"hash_data": true,
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}],
					"warehouse": [{"provider": "custom", "resolution_strategy": "and", "consents": ["analytics"]}]
				}
			}`,
			APIJSON: `{
				"rudderAccountId": "account-1",
				"hashData": true,
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}],
					"warehouse": [{"provider": "custom", "resolutionStrategy": "and", "consents": [{"consent": "analytics"}]}]
				}
			}`,
		},
	})
}

func validMinimalConfig() map[string]any {
	return map[string]any{
		"rudder_account_id": "account-1",
		"hash_data":         true,
	}
}

func validFullConfig() map[string]any {
	return map[string]any{
		"rudder_account_id": "account-1",
		"hash_data":         true,
		"ad_account_id":     "urn:li:sponsoredAccount:123456789",
		"deduplication_key": "messageId",
		"conversion_mapping": []any{
			map[string]any{"from": "Product Viewed", "to": "111111"},
			map[string]any{"from": "Order Completed", "to": "222222"},
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider":            "oneTrust",
					"resolution_strategy": "and",
					"consents":            []any{"marketing"},
				},
			},
		},
	}
}
