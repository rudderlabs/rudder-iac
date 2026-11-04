package intercomv2_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	intercomv2 "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/intercom_v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func register(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(intercomv2.NewDefinition()))

	registered, err := registry.Get("intercom_v2", 1)
	require.NoError(t, err)
	return registered
}

// validConfig is the minimal config every subtest starts from.
func validConfig() map[string]any {
	return map[string]any{
		"rudder_account_id": "intercom-account",
		"api_server":        "US",
	}
}

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(intercomv2.NewDefinition()))

	registered, err := registry.Get("intercom_v2", 1)
	require.NoError(t, err)

	assert.Equal(t, "intercom_v2", registered.Type)
	assert.Equal(t, "INTERCOM_V2", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web", "unity",
		"cloud", "react_native", "flutter", "cordova", "warehouse",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes, sourceType)
	}

	// Upstream lists amp; the CLI never declares it.
	assert.NotContains(t, registered.SupportedSourceTypes(), "amp")

	assert.Equal(t, []string{"mirror"}, registered.SyncBehaviours())
	assert.False(t, registered.SupportsVisualMapper())
	assert.True(t, registered.SupportsVisualMapperV2())

	byAPI, err := registry.GetByAPIType("INTERCOM_V2", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestIntercomV2ConfigValidation(t *testing.T) {
	t.Parallel()

	registered := register(t)

	for _, key := range []string{"rudder_account_id", "api_server"} {
		t.Run("missing "+key, func(t *testing.T) {
			t.Parallel()
			config := validConfig()
			delete(config, key)

			errors := registered.ValidateConfig(config)
			require.Len(t, errors, 1)
			assert.Equal(t, "/"+key, errors[0].Path)
			assert.Contains(t, errors[0].Message, "required")
		})
	}

	t.Run("invalid api_server rejected", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["api_server"] = "Asia"

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/api_server", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	// schema.json declares a plain enum with no {{ … || … }} branch, so a
	// templated value would be stored verbatim and rejected by the backend.
	t.Run("api_server rejects dynamic values", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["api_server"] = "{{ .INTERCOM_API_SERVER || US }}"

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/api_server", errors[0].Path)
	})

	t.Run("connection_mode accepts a supported mode", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["connection_mode"] = map[string]any{"web": "cloud", "warehouse": "cloud"}
		assert.Empty(t, registered.ValidateConfig(config))
	})

	t.Run("connection_mode rejects an unsupported mode", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["connection_mode"] = map[string]any{"android": "device"}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/android", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	// The spec shipped in the README.
	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "intercom-v2-account",
			"api_server":        "US",
			"send_anonymous_id": false,
			"connection_mode":   map[string]any{"warehouse": "cloud"},
		}))
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "intercom-account",
			"api_server":        "Europe",
			"send_anonymous_id": true,
			"connection_mode":   map[string]any{"android_kotlin": "cloud", "warehouse": "cloud"},
			"consent_management": map[string]any{
				"android_kotlin": []any{
					map[string]any{
						"provider":            "custom",
						"resolution_strategy": "or",
						"consents":            []any{"one", "two"},
					},
				},
			},
		}))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["not_a_field"] = true

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["consent_management"] = map[string]any{"amp": []any{}}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/amp", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'amp' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["consent_management"] = map[string]any{
			"web": []any{map[string]any{"provider": "unknown"}},
		}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestIntercomV2Defaults(t *testing.T) {
	t.Parallel()

	registered := register(t)

	// api_server is defaulted upstream too, but it is also required, so the
	// spec always supplies it and it carries no default tag.
	assert.Equal(t, map[string]any{
		"rudder_account_id": "intercom-account",
		"api_server":        "US",
		"send_anonymous_id": false,
	}, registered.ApplyDefaults(validConfig()))

	applied := registered.ApplyDefaults(map[string]any{
		"rudder_account_id": "intercom-account",
		"api_server":        "US",
		"send_anonymous_id": true,
	})
	assert.Equal(t, true, applied["send_anonymous_id"])
}

func TestIntercomV2ConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := intercomv2.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"rudder_account_id": "intercom-account",
				"api_server": "US"
			}`,
			APIJSON: `{
				"rudderAccountId": "intercom-account",
				"apiServer": "US"
			}`,
		},
		{
			Name: "full with consent on a boundary-mapped source type",
			LocalJSON: `{
				"rudder_account_id": "intercom-account",
				"api_server": "Europe",
				"send_anonymous_id": true,
				"connection_mode": {"android_kotlin": "cloud", "react_native": "cloud"},
				"consent_management": {
					"android_kotlin": [
						{
							"provider": "custom",
							"resolution_strategy": "or",
							"consents": ["one_android", "two_android"]
						}
					]
				}
			}`,
			APIJSON: `{
				"rudderAccountId": "intercom-account",
				"apiServer": "Europe",
				"sendAnonymousId": true,
				"connectionMode": {"androidKotlin": "cloud", "reactnative": "cloud"},
				"consentManagement": {
					"androidKotlin": [
						{
							"provider": "custom",
							"resolutionStrategy": "or",
							"consents": [
								{"consent": "one_android"},
								{"consent": "two_android"}
							]
						}
					]
				}
			}`,
		},
	})
}
