package hs_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/hs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(hs.NewDefinition()))

	registered, err := registry.Get("hs", 1)
	require.NoError(t, err)

	assert.Equal(t, "hs", registered.Type)
	assert.Equal(t, "HS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"api_key", "access_token"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "cloud", "react_native", "flutter", "cordova",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	expectedModes := map[string][]string{
		"android":        {"cloud"},
		"android_kotlin": {"cloud"},
		"ios":            {"cloud"},
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

	assert.Nil(t, registered.SupportedSourcesValidation("web"))
	assert.Empty(t, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("HS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestHSConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(hs.NewDefinition()))
	registered, err := registry.Get("hs", 1)
	require.NoError(t, err)

	minimalConfig := func() map[string]any {
		return map[string]any{
			"authorization_type": "newPrivateAppApi",
			"api_version":        "newApi",
			"access_token":       "private-app-token",
			"lookup_field":       "email",
		}
	}

	t.Run("missing authorization_type", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		delete(config, "authorization_type")

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/authorization_type", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing api_version", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		delete(config, "api_version")

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_version", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("api_key required for legacy API key auth", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"authorization_type": "legacyApiKey",
			"api_version":        "legacyApi",
		})

		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_key", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("access_token required for private app auth", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"authorization_type": "newPrivateAppApi",
			"api_version":        "legacyApi",
		})

		require.NotEmpty(t, errors)
		assert.Equal(t, "/access_token", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("lookup_field required for new api", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"authorization_type": "newPrivateAppApi",
			"api_version":        "newApi",
			"access_token":       "private-app-token",
		})

		require.NotEmpty(t, errors)
		assert.Equal(t, "/lookup_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("lookup_field not required for legacy api", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"authorization_type": "newPrivateAppApi",
			"api_version":        "legacyApi",
			"access_token":       "private-app-token",
		})

		assert.Empty(t, errors)
	})

	t.Run("invalid enum values rejected", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			field string
			value string
			path  string
		}{
			{field: "authorization_type", value: "oauth", path: "/authorization_type"},
			{field: "api_version", value: "v2", path: "/api_version"},
		}

		for _, tc := range cases {
			tc := tc
			t.Run(tc.field, func(t *testing.T) {
				t.Parallel()
				config := minimalConfig()
				config[tc.field] = tc.value

				errors := registered.ValidateConfig(config)
				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	t.Run("single line fields reject newlines", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			field string
			value any
			path  string
		}{
			{field: "api_key", value: "bad\nkey", path: "/api_key"},
			{field: "access_token", value: "bad\ntoken", path: "/access_token"},
			{field: "hub_id", value: "bad\nhub", path: "/hub_id"},
			{field: "lookup_field", value: "bad\nfield", path: "/lookup_field"},
			{field: "event_filtering", value: map[string]any{"whitelist": []any{"bad\nevent"}}, path: "/event_filtering/whitelist/0"},
			{field: "event_filtering", value: map[string]any{"blacklist": []any{"bad\nevent"}}, path: "/event_filtering/blacklist/0"},
			{field: "hubspot_events", value: []any{map[string]any{"rs_event_name": "bad\nevent"}}, path: "/hubspot_events/0/rs_event_name"},
			{field: "hubspot_events", value: []any{map[string]any{"hubspot_event_name": "bad\nevent"}}, path: "/hubspot_events/0/hubspot_event_name"},
			{field: "hubspot_events", value: []any{map[string]any{"event_properties": []any{map[string]any{"from": "bad\nproperty"}}}}, path: "/hubspot_events/0/event_properties/0/from"},
			{field: "hubspot_events", value: []any{map[string]any{"event_properties": []any{map[string]any{"to": "bad\nproperty"}}}}, path: "/hubspot_events/0/event_properties/0/to"},
		}

		for _, tc := range cases {
			tc := tc
			t.Run(tc.path, func(t *testing.T) {
				t.Parallel()
				config := minimalConfig()
				config[tc.field] = tc.value
				if tc.field == "api_key" {
					config["authorization_type"] = "legacyApiKey"
					config["api_version"] = "legacyApi"
					delete(config, "access_token")
					delete(config, "lookup_field")
				}

				errors := registered.ValidateConfig(config)
				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	t.Run("single line fields accept dynamic templates", func(t *testing.T) {
		t.Parallel()
		config := map[string]any{
			"authorization_type": "legacyApiKey",
			"api_version":        "newApi",
			"api_key":            "{{ .HS_API_KEY || fallback-key }}",
			"access_token":       "{{ .HS_ACCESS_TOKEN || fallback-token }}",
			"hub_id":             "{{ .HS_HUB_ID || fallback-hub }}",
			"lookup_field":       "{{ .HS_LOOKUP_FIELD || email }}",
			"event_filtering": map[string]any{
				"blacklist": []any{"{{ .HS_BLOCKED_EVENT || Internal Event }}"},
			},
			"hubspot_events": []any{
				map[string]any{
					"rs_event_name":      "{{ .HS_RS_EVENT || Product Viewed }}",
					"hubspot_event_name": "{{ .HS_EVENT || Product Viewed }}",
					"event_properties": []any{
						map[string]any{
							"from": "{{ .HS_PROP_FROM || properties.plan }}",
							"to":   "{{ .HS_PROP_TO || plan }}",
						},
					},
				},
			},
		}

		errors := registered.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("over 100 character literals rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["hub_id"] = strings.Repeat("a", 101)

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/hub_id", errors[0].Path)
	})

	t.Run("event filtering lists are mutually exclusive", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["event_filtering"] = map[string]any{
			"whitelist": []any{"Product Viewed"},
			"blacklist": []any{"Internal Event"},
		}

		errors := registered.ValidateConfig(config)
		assertValidationPaths(t, errors, "/event_filtering/whitelist", "/event_filtering/blacklist")
	})

	t.Run("valid minimal private app example", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"authorization_type": "newPrivateAppApi",
			"api_version":        "newApi",
			"access_token":       "{{ .HS_ACCESS_TOKEN }}",
			"lookup_field":       "email",
		})

		assert.Empty(t, errors)
	})

	t.Run("valid legacy api key config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"authorization_type": "legacyApiKey",
			"api_version":        "legacyApi",
			"api_key":            "legacy-api-key",
		})

		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"authorization_type": "newPrivateAppApi",
			"api_version":        "newApi",
			"access_token":       "private-app-token",
			"api_key":            "legacy-api-key",
			"hub_id":             "123456",
			"lookup_field":       "email",
			"do_association":     true,
			"hubspot_events": []any{
				map[string]any{
					"rs_event_name":      "Product Viewed",
					"hubspot_event_name": "Product Viewed",
					"event_properties": []any{
						map[string]any{"from": "properties.plan", "to": "plan"},
					},
				},
			},
			"event_filtering": map[string]any{
				"whitelist": []any{"Product Viewed", "Order Completed"},
			},
			"use_native_sdk": map[string]any{
				"web": true,
			},
			"consent_management": map[string]any{
				"web": []any{
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
		config := minimalConfig()
		config["not_a_field"] = true

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("connection_mode is not a supported key", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["connection_mode"] = map[string]any{"web": "device"}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/connection_mode", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported source key rejected in use_native_sdk", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["use_native_sdk"] = map[string]any{"android": true}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/use_native_sdk/android", errors[0].Path)
	})

	t.Run("legacy consent blocks are not supported keys", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			key := key
			t.Run(key, func(t *testing.T) {
				t.Parallel()
				config := minimalConfig()
				config[key] = map[string]any{"web": []any{}}

				errors := registered.ValidateConfig(config)
				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+key, errors[0].Path)
				assert.Contains(t, errors[0].Message, "unknown config field")
			})
		}
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

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
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

func TestHSConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := hs.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal private app",
			LocalJSON: `{
				"authorization_type": "newPrivateAppApi",
				"api_version": "newApi",
				"access_token": "private-app-token",
				"lookup_field": "email"
			}`,
			APIJSON: `{
				"authorizationType": "newPrivateAppApi",
				"apiVersion": "newApi",
				"accessToken": "private-app-token",
				"lookupField": "email"
			}`,
		},
		{
			Name: "legacy api key",
			LocalJSON: `{
				"authorization_type": "legacyApiKey",
				"api_version": "legacyApi",
				"api_key": "legacy-api-key"
			}`,
			APIJSON: `{
				"authorizationType": "legacyApiKey",
				"apiVersion": "legacyApi",
				"apiKey": "legacy-api-key"
			}`,
		},
		{
			Name: "full config with hubspot events and whitelist",
			LocalJSON: `{
				"authorization_type": "newPrivateAppApi",
				"api_version": "newApi",
				"access_token": "private-app-token",
				"api_key": "legacy-api-key",
				"hub_id": "123456",
				"lookup_field": "email",
				"do_association": true,
				"hubspot_events": [
					{
						"rs_event_name": "Product Viewed",
						"hubspot_event_name": "Product Viewed",
						"event_properties": [
							{"from": "properties.plan", "to": "plan"},
							{"from": "properties.category", "to": "category"}
						]
					}
				],
				"event_filtering": {
					"whitelist": ["Product Viewed", "Order Completed"]
				},
				"use_native_sdk": {
					"web": true
				}
			}`,
			APIJSON: `{
				"authorizationType": "newPrivateAppApi",
				"apiVersion": "newApi",
				"accessToken": "private-app-token",
				"apiKey": "legacy-api-key",
				"hubID": "123456",
				"lookupField": "email",
				"doAssociation": true,
				"hubspotEvents": [
					{
						"rsEventName": "Product Viewed",
						"hubspotEventName": "Product Viewed",
						"eventProperties": [
							{"from": "properties.plan", "to": "plan"},
							{"from": "properties.category", "to": "category"}
						]
					}
				],
				"whitelistedEvents": [
					{"eventName": "Product Viewed"},
					{"eventName": "Order Completed"}
				],
				"eventFilteringOption": "whitelistedEvents",
				"useNativeSDK": {
					"web": true
				}
			}`,
		},
		{
			Name: "event filtering blacklist",
			LocalJSON: `{
				"authorization_type": "newPrivateAppApi",
				"api_version": "newApi",
				"access_token": "private-app-token",
				"lookup_field": "email",
				"event_filtering": {
					"blacklist": ["Internal Event"]
				}
			}`,
			APIJSON: `{
				"authorizationType": "newPrivateAppApi",
				"apiVersion": "newApi",
				"accessToken": "private-app-token",
				"lookupField": "email",
				"blacklistedEvents": [
					{"eventName": "Internal Event"}
				],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"authorization_type": "newPrivateAppApi",
				"api_version": "legacyApi",
				"access_token": "private-app-token",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}],
					"cloud": [{"provider": "custom", "resolution_strategy": "or", "consents": ["analytics"]}],
					"cordova": [{"provider": "oneTrust"}]
				}
			}`,
			APIJSON: `{
				"authorizationType": "newPrivateAppApi",
				"apiVersion": "legacyApi",
				"accessToken": "private-app-token",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}],
					"cloud": [{
						"provider": "custom",
						"resolutionStrategy": "or",
						"consents": [{"consent": "analytics"}]
					}],
					"cordova": [{"provider": "oneTrust"}]
				}
			}`,
		},
	})
}

func assertValidationPaths(t *testing.T, errors []definitions.ConfigError, paths ...string) {
	t.Helper()
	require.Len(t, errors, len(paths))

	actual := make([]string, 0, len(errors))
	for _, err := range errors {
		actual = append(actual, err.Path)
	}
	assert.ElementsMatch(t, paths, actual)
}
