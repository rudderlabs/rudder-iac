package intercom_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/intercom"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(intercom.NewDefinition()))

	registered, err := registry.Get("intercom", 1)
	require.NoError(t, err)

	assert.Equal(t, "intercom", registered.Type)
	assert.Equal(t, "INTERCOM", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"api_key"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "cloud", "react_native", "flutter", "cordova",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	assert.NotContains(t, registered.SupportedSourceTypes(), "amp")
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")

	expectedModes := map[string][]string{
		"android":        {"cloud", "device"},
		"android_kotlin": {"cloud"},
		"ios":            {"cloud", "device"},
		"ios_swift":      {"cloud"},
		"web":            {"cloud", "device"},
		"unity":          {"cloud"},
		"cloud":          {"cloud"},
		"react_native":   {"cloud"},
		"flutter":        {"cloud"},
		"cordova":        {"cloud"},
	}
	for sourceType, want := range expectedModes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, want, modes, "source type %s", sourceType)
	}

	assert.Equal(t, map[string][]string{
		"mobile_api_key_android": {"android"},
		"mobile_api_key_ios":     {"ios"},
	}, registered.GatedKeyPaths())
	assert.Nil(t, registered.SupportedSourcesValidation("web"))

	byAPI, err := registry.GetByAPIType("INTERCOM", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestIntercomConfigValidation(t *testing.T) {
	t.Parallel()

	registered := registeredDefinition(t)

	t.Run("valid cloud config with api_key", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"api_key":           "intercom-access-token",
			"api_server":        "standard",
			"api_version":       "v2",
			"send_anonymous_id": true,
		}))
	})

	t.Run("valid device config with app_id", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"app_id": "fll5vd90",
			"use_native_sdk": map[string]any{
				"web": true,
			},
		}))
	})

	t.Run("credential requiredness is backend enforced", func(t *testing.T) {
		t.Parallel()

		// Intercom schema.json makes appId/apiKey conditional on source connection
		// modes. The definition supports both fields but does not globally require
		// either, so cloud-only and device-only specs are both valid locally.
		assert.Empty(t, registered.ValidateConfig(map[string]any{}))
	})

	t.Run("api enums rejected", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			key   string
			value string
			path  string
		}{
			{key: "api_server", value: "emea", path: "/api_server"},
			{key: "api_version", value: "v3", path: "/api_version"},
		} {
			t.Run(tc.key, func(t *testing.T) {
				t.Parallel()

				config := validFullConfig()
				config[tc.key] = tc.value

				errors := registered.ValidateConfig(config)
				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	// connection_mode is not destination config; modes stay in ConnectionModes() metadata.
	t.Run("connection_mode is not a supported key", func(t *testing.T) {
		t.Parallel()
		config := validFullConfig()
		config["connection_mode"] = map[string]any{"web": "device"}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/connection_mode", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("credential fields reject empty strings", func(t *testing.T) {
		t.Parallel()

		for _, tc := range credentialFieldCases("") {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(tc.config)
				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	t.Run("single line fields reject values over maximum length", func(t *testing.T) {
		t.Parallel()

		for _, tc := range patternFieldCases(strings.Repeat("x", 101)) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(tc.config)
				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	t.Run("single line fields reject line breaks", func(t *testing.T) {
		t.Parallel()

		for _, tc := range patternFieldCases("line\nbreak") {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(tc.config)
				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	t.Run("pattern fields accept ui templates", func(t *testing.T) {
		t.Parallel()
		longTemplate := "{{ config.value || " + strings.Repeat("x", 120) + " }}"

		for _, tc := range patternFieldCases(longTemplate) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				assert.Empty(t, registered.ValidateConfig(tc.config))
			})
		}
	})

	t.Run("deprecated env references get no template exemption", func(t *testing.T) {
		t.Parallel()
		config := map[string]any{"api_key": "env." + strings.Repeat("A", 101)}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/api_key", errors[0].Path)
	})

	t.Run("event filtering lists are mutually exclusive", func(t *testing.T) {
		t.Parallel()
		config := validFullConfig()
		config["event_filtering"] = map[string]any{
			"whitelist": []any{"Order Completed"},
			"blacklist": []any{"Page Viewed"},
		}

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
	})

	t.Run("event filtering events reject invalid values", func(t *testing.T) {
		t.Parallel()
		config := validFullConfig()
		config["event_filtering"] = map[string]any{
			"whitelist": []any{"Order\nCompleted"},
		}

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/event_filtering/whitelist/0", errors[0].Path)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, registered.ValidateConfig(validFullConfig()))
	})

	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"api_key":                "{{ .INTERCOM_API_KEY }}",
			"api_server":             "standard",
			"api_version":            "v2",
			"send_anonymous_id":      true,
			"update_last_request_at": true,
			"event_filtering": map[string]any{
				"whitelist": []any{"Order Completed", "Signed Up"},
			},
			"consent_management": map[string]any{
				"web": []any{
					map[string]any{
						"provider": "oneTrust",
						"consents": []any{"analytics"},
					},
				},
			},
		}))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		config := validFullConfig()
		config["not_a_field"] = true

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("terraform only collect_context rejected", func(t *testing.T) {
		t.Parallel()
		config := validFullConfig()
		config["collect_context"] = true

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/collect_context", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("legacy consent blocks are not supported keys", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			t.Run(key, func(t *testing.T) {
				t.Parallel()
				config := validFullConfig()
				config[key] = map[string]any{"web": []any{}}

				errors := registered.ValidateConfig(config)
				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+key, errors[0].Path)
				assert.Contains(t, errors[0].Message, "unknown config field")
			})
		}
	})

	t.Run("unsupported source key rejected in source-type blocks", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name  string
			key   string
			value any
		}{
			{name: "use_native_sdk", key: "use_native_sdk", value: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				config := validFullConfig()
				config[tc.key] = map[string]any{"cloud_source": tc.value}

				errors := registered.ValidateConfig(config)
				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+tc.key+"/cloud_source", errors[0].Path)
			})
		}
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		config := validFullConfig()
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
		config := validFullConfig()
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

func TestIntercomConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := intercom.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "cloud config",
			LocalJSON: `{
				"api_key": "intercom-access-token",
				"api_server": "standard",
				"api_version": "v2",
				"send_anonymous_id": true,
				"update_last_request_at": true
			}`,
			APIJSON: `{
				"apiKey": "intercom-access-token",
				"apiServer": "standard",
				"apiVersion": "v2",
				"sendAnonymousId": true,
				"updateLastRequestAt": true
			}`,
		},
		{
			Name: "device config",
			LocalJSON: `{
				"app_id": "fll5vd90",
				"use_native_sdk": {
					"web": true,
					"android": true,
					"ios": true
				},
				"mobile_api_key_android": "android-sdk-key",
				"mobile_api_key_ios": "ios-sdk-key",
				"event_filtering": {
					"blacklist": ["Internal Event"]
				}
			}`,
			APIJSON: `{
				"appId": "fll5vd90",
				"useNativeSDK": {
					"web": true,
					"android": true,
					"ios": true
				},
				"mobileApiKeyAndroid": {"android": "android-sdk-key"},
				"mobileApiKeyIOS": {"ios": "ios-sdk-key"},
				"blacklistedEvents": [
					{"eventName": "Internal Event"}
				],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "event filtering whitelist",
			LocalJSON: `{
				"event_filtering": {
					"whitelist": ["Product Viewed", "Order Completed"]
				}
			}`,
			APIJSON: `{
				"whitelistedEvents": [
					{"eventName": "Product Viewed"},
					{"eventName": "Order Completed"}
				],
				"eventFilteringOption": "whitelistedEvents"
			}`,
		},
		{
			Name: "boundary source mappings",
			LocalJSON: `{

				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{

				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}

func registeredDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(intercom.NewDefinition()))
	registered, err := registry.Get("intercom", 1)
	require.NoError(t, err)
	return registered
}

type fieldCase struct {
	name   string
	path   string
	config map[string]any
}

func credentialFieldCases(value string) []fieldCase {
	return []fieldCase{
		{
			name: "api_key",
			path: "/api_key",
			config: map[string]any{
				"api_key": value,
			},
		},
		{
			name: "app_id",
			path: "/app_id",
			config: map[string]any{
				"app_id": value,
			},
		},
	}
}

func patternFieldCases(value string) []fieldCase {
	return append(credentialFieldCases(value), []fieldCase{
		{
			name: "mobile_api_key_android",
			path: "/mobile_api_key_android",
			config: map[string]any{
				"mobile_api_key_android": value,
			},
		},
		{
			name: "mobile_api_key_ios",
			path: "/mobile_api_key_ios",
			config: map[string]any{
				"mobile_api_key_ios": value,
			},
		},
	}...)
}

func validFullConfig() map[string]any {
	return map[string]any{
		"api_key":                "intercom-access-token",
		"app_id":                 "fll5vd90",
		"api_server":             "eu",
		"api_version":            "v2",
		"send_anonymous_id":      true,
		"update_last_request_at": true,
		"mobile_api_key_android": "android-sdk-key",
		"mobile_api_key_ios":     "ios-sdk-key",
		"use_native_sdk": map[string]any{
			"web":     true,
			"android": true,
			"ios":     true,
		},
		"event_filtering": map[string]any{
			"whitelist": []any{"Product Viewed", "Order Completed"},
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{"analytics"},
				},
			},
			"android_kotlin": []any{
				map[string]any{"provider": "ketch"},
			},
		},
	}
}
