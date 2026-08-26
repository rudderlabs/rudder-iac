package confluentcloud_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	confluentcloud "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/confluent_cloud"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(confluentcloud.NewDefinition()))

	registered, err := registry.Get("confluent_cloud", 1)
	require.NoError(t, err)

	assert.Equal(t, "confluent_cloud", registered.Type)
	assert.Equal(t, "CONFLUENT_CLOUD", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"api_secret", "api_key"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "cloud", "react_native",
		"flutter", "cordova"}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	assert.Empty(t, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("CONFLUENT_CLOUD", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestConfluentCloudConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(confluentcloud.NewDefinition()))
	registered, err := registry.Get("confluent_cloud", 1)
	require.NoError(t, err)

	minimalConfig := func() map[string]any {
		return map[string]any{
			"bootstrap_server": "pkc-00000.us-central1.gcp.confluent.cloud:9092",
			"topic":            "rudder-cli-e2e",
			"api_key":          "confluent-cloud-api-key",
			"api_secret":       "confluent-cloud-api-secret",
		}
	}

	for _, field := range []string{"bootstrap_server", "topic", "api_key", "api_secret"} {
		field := field
		t.Run("missing "+field, func(t *testing.T) {
			t.Parallel()
			config := minimalConfig()
			delete(config, field)

			errors := registered.ValidateConfig(config)

			require.NotEmpty(t, errors)
			assert.Equal(t, "/"+field, errors[0].Path)
			assert.Contains(t, errors[0].Message, "required")
		})
	}

	t.Run("single line fields reject invalid literals", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name  string
			field string
			value string
		}{
			{name: "bootstrap_server over 100 characters", field: "bootstrap_server", value: strings.Repeat("a", 101)},
			{name: "topic with line break", field: "topic", value: "bad\ntopic"},
			{name: "api_key over 100 characters", field: "api_key", value: strings.Repeat("a", 101)},
			{name: "api_secret with line break", field: "api_secret", value: "bad\nsecret"},
		}

		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				config := minimalConfig()
				config[tc.field] = tc.value

				errors := registered.ValidateConfig(config)

				require.Len(t, errors, 1)
				assert.Equal(t, "/"+tc.field, errors[0].Path)
			})
		}
	})

	// Unlike most destinations, confluent_cloud's schema.json patterns are bare
	// `^(.{0,100})$` with no `(^\{\{.*\|\|(.*)\}\}$)` branch, so the backend
	// rejects UI templates here. Hence plain `pattern=`, not `dynamic_or_pattern=`.
	t.Run("ui templates rejected past the literal limit", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["topic"] = "{{ config.topic || " + strings.Repeat("a", 101) + " }}"

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/topic", errors[0].Path)
	})

	t.Run("deprecated env references get no template exemption", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["api_key"] = "env." + strings.Repeat("A", 101)

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/api_key", errors[0].Path)
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["not_a_field"] = true

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("connection_mode rejected as unknown key", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["connection_mode"] = map[string]any{"web": "cloud"}

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/connection_mode", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(minimalConfig())

		assert.Empty(t, errors)
	})

	t.Run("valid example config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"bootstrap_server": "pkc-00000.us-central1.gcp.confluent.cloud:9092",
			"topic":            "rudder-cli-events",
			"api_key":          "{{ .CONFLUENT_CLOUD_API_KEY }}",
			"api_secret":       "{{ .CONFLUENT_CLOUD_API_SECRET }}",
		})

		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["consent_management"] = map[string]any{
			"android_kotlin": []any{
				map[string]any{
					"provider":            "custom",
					"resolution_strategy": "and",
					"consents":            []any{"analytics", "marketing"},
				},
			},
		}

		errors := registered.ValidateConfig(config)

		assert.Empty(t, errors)
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["consent_management"] = map[string]any{
			"cloud_source": []any{},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/cloud_source", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'cloud_source' is not supported")
	})

	// schema.json declares oneTrustCookieCategories and ketchConsentPurposes, but
	// they are deliberately not modelled: when a payload carries them without
	// consentManagement the backend migrates them into consentManagement and drops
	// the legacy keys, so the CLI would re-send them on every apply forever.
	t.Run("legacy consent blocks are not supported keys", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			config := minimalConfig()
			config[key] = map[string]any{"web": []any{}}

			errors := registered.ValidateConfig(config)

			require.Len(t, errors, 1, key)
			assert.Equal(t, "/"+key, errors[0].Path)
			assert.Contains(t, errors[0].Message, "unknown config field")
		}
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["consent_management"] = map[string]any{
			"ios_swift": []any{
				map[string]any{"provider": "unknown"},
			},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/ios_swift/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})

	t.Run("duplicate consent provider rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["consent_management"] = map[string]any{
			"web": []any{
				map[string]any{"provider": "oneTrust"},
				map[string]any{"provider": "oneTrust"},
			},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web/1/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "only one consent entry")
	})

}

func TestConfluentCloudConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := confluentcloud.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"bootstrap_server": "pkc-00000.us-central1.gcp.confluent.cloud:9092",
				"topic": "rudder-cli-e2e",
				"api_key": "confluent-cloud-api-key",
				"api_secret": "confluent-cloud-api-secret"
			}`,
			APIJSON: `{
				"bootstrapServer": "pkc-00000.us-central1.gcp.confluent.cloud:9092",
				"topic": "rudder-cli-e2e",
				"apiKey": "confluent-cloud-api-key",
				"apiSecret": "confluent-cloud-api-secret"
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"bootstrap_server": "pkc-00000.us-central1.gcp.confluent.cloud:9092",
				"topic": "rudder-cli-e2e",
				"api_key": "confluent-cloud-api-key",
				"api_secret": "confluent-cloud-api-secret",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"bootstrapServer": "pkc-00000.us-central1.gcp.confluent.cloud:9092",
				"topic": "rudder-cli-e2e",
				"apiKey": "confluent-cloud-api-key",
				"apiSecret": "confluent-cloud-api-secret",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
