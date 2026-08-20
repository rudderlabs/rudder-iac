package snowpipestreaming_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	snowpipestreaming "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/snowpipe_streaming"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func registeredDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(snowpipestreaming.NewDefinition()))
	registered, err := registry.Get("snowpipe_streaming", 1)
	require.NoError(t, err)
	return registered
}

func minimalConfig() map[string]any {
	return map[string]any{
		"account":     "rudder-cli-e2e.us-east-1",
		"database":    "RUDDER_E2E",
		"warehouse":   "RUDDER_WH",
		"user":        "RUDDER_CLI_E2E",
		"namespace":   "rudder_cli_e2e",
		"private_key": "rawSnowpipePrivateKeyXXXXXXXX",
	}
}

func exampleConfig() map[string]any {
	cfg := copyConfig(minimalConfig())
	cfg["role"] = "RUDDER_ROLE"
	cfg["skip_tracks_table"] = false
	cfg["json_paths"] = "context.traits,properties.metadata"
	cfg["enable_iceberg"] = true
	cfg["external_volume"] = "RUDDER_EXTERNAL_VOLUME"
	cfg["underscore_divide_numbers"] = false
	cfg["allow_users_context_traits"] = false
	cfg["consent_management"] = map[string]any{
		"android_kotlin": []any{map[string]any{"provider": "oneTrust"}},
	}
	return cfg
}

func copyConfig(src map[string]any) map[string]any {
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(snowpipestreaming.NewDefinition()))

	registered, err := registry.Get("snowpipe_streaming", 1)
	require.NoError(t, err)

	assert.Equal(t, "snowpipe_streaming", registered.Type)
	assert.Equal(t, "SNOWPIPE_STREAMING", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.GatedKeyPaths())
	assert.Equal(t, []string{"private_key", "private_key_passphrase"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web", "unity",
		"amp", "cloud", "react_native", "cloud_source", "flutter", "cordova", "shopify",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	byAPI, err := registry.GetByAPIType("SNOWPIPE_STREAMING", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestSnowpipeStreamingConfigValidation(t *testing.T) {
	t.Parallel()

	registered := registeredDefinition(t)

	t.Run("required fields missing", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"account", "database", "warehouse", "user", "namespace", "private_key"} {
			cfg := copyConfig(minimalConfig())
			delete(cfg, field)

			errors := registered.ValidateConfig(cfg)
			require.NotEmpty(t, errors, field)
			assert.Equal(t, "/"+field, errors[0].Path)
			assert.Contains(t, errors[0].Message, "required")
		}
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(minimalConfig()))
	})

	t.Run("valid example config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(exampleConfig()))
	})

	t.Run("external volume required when iceberg enabled", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["enable_iceberg"] = true

		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/external_volume", errors[0].Path)
	})

	t.Run("namespace rejects reserved pg prefix", func(t *testing.T) {
		t.Parallel()
		for _, ns := range []string{"pg_catalog", "PG_x", "pG_x", "Pg_x"} {
			cfg := copyConfig(minimalConfig())
			cfg["namespace"] = ns
			errors := registered.ValidateConfig(cfg)
			require.NotEmpty(t, errors, ns)
			assert.Equal(t, "/namespace", errors[0].Path)
		}

		cfg := copyConfig(minimalConfig())
		cfg["namespace"] = "analytics_pg_data"
		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("single line fields reject line breaks", func(t *testing.T) {
		t.Parallel()
		for _, field := range []string{"account", "database", "warehouse", "user", "role", "namespace", "private_key_passphrase", "external_volume"} {
			cfg := copyConfig(minimalConfig())
			if field == "external_volume" {
				cfg["enable_iceberg"] = true
			}
			cfg[field] = "bad\nvalue"
			errors := registered.ValidateConfig(cfg)
			require.NotEmpty(t, errors, field)
			assert.Equal(t, "/"+field, errors[0].Path)
		}
	})

	t.Run("pattern fields accept templates", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["account"] = "{{ config.account || " + strings.Repeat("a", 150) + " }}"
		cfg["namespace"] = "{{ config.namespace || rudder_events }}"
		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("raw private key accepted", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["private_key"] = "rawSnowpipePrivateKeyBody"
		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["not_a_field"] = true

		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("legacy consent blocks rejected as local unknown keys", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["one_trust_cookie_categories"] = map[string]any{"web": []any{}}

		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/one_trust_cookie_categories", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["consent_management"] = map[string]any{"warehouse": []any{}}

		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'warehouse' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["consent_management"] = map[string]any{
			"ios_swift": []any{map[string]any{"provider": "unknown"}},
		}

		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/ios_swift/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestSnowpipeStreamingConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := snowpipestreaming.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal pem key",
			LocalJSON: `{
				"account": "rudder-cli-e2e.us-east-1",
				"database": "RUDDER_E2E",
				"warehouse": "RUDDER_WH",
				"user": "RUDDER_CLI_E2E",
				"namespace": "rudder_cli_e2e",
				"private_key": "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----"
			}`,
			APIJSON: `{
				"account": "rudder-cli-e2e.us-east-1",
				"database": "RUDDER_E2E",
				"warehouse": "RUDDER_WH",
				"user": "RUDDER_CLI_E2E",
				"namespace": "rudder_cli_e2e",
				"privateKey": "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----"
			}`,
		},
		{
			Name: "full config with consent",
			LocalJSON: `{
				"account": "rudder-cli-e2e.us-east-1",
				"database": "RUDDER_E2E",
				"warehouse": "RUDDER_WH",
				"user": "RUDDER_CLI_E2E",
				"role": "RUDDER_ROLE",
				"namespace": "rudder_cli_e2e",
				"private_key": "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----",
				"private_key_passphrase": "phrase",
				"skip_tracks_table": false,
				"json_paths": "context.traits,properties.metadata",
				"enable_iceberg": true,
				"external_volume": "RUDDER_EXTERNAL_VOLUME",
				"underscore_divide_numbers": false,
				"allow_users_context_traits": false,
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}],
					"cloud_source": [{"provider": "custom", "resolution_strategy": "and", "consents": ["marketing"]}]
				}
			}`,
			APIJSON: `{
				"account": "rudder-cli-e2e.us-east-1",
				"database": "RUDDER_E2E",
				"warehouse": "RUDDER_WH",
				"user": "RUDDER_CLI_E2E",
				"role": "RUDDER_ROLE",
				"namespace": "rudder_cli_e2e",
				"privateKey": "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----",
				"privateKeyPassphrase": "phrase",
				"skipTracksTable": false,
				"jsonPaths": "context.traits,properties.metadata",
				"enableIceberg": true,
				"externalVolume": "RUDDER_EXTERNAL_VOLUME",
				"underscoreDivideNumbers": false,
				"allowUsersContextTraits": false,
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}],
					"cloudSource": [{"provider": "custom", "resolutionStrategy": "and", "consents": [{"consent":"marketing"}]}]
				}
			}`,
		},
	})
}

func TestSnowpipeStreamingRawPrivateKeyWrapsForAPI(t *testing.T) {
	t.Parallel()

	def := snowpipestreaming.NewDefinition()
	actual, err := converter.LocalToAPI(def.Properties, map[string]any{
		"account":     "rudder-cli-e2e.us-east-1",
		"database":    "RUDDER_E2E",
		"warehouse":   "RUDDER_WH",
		"user":        "RUDDER_CLI_E2E",
		"namespace":   "rudder_cli_e2e",
		"private_key": "rawSnowpipePrivateKeyBody",
	})
	require.NoError(t, err)
	assert.Equal(t, "-----BEGIN PRIVATE KEY-----\nrawSnowpipePrivateKeyBody\n-----END PRIVATE KEY-----", actual["privateKey"])
}
