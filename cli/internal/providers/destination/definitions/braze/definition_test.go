package braze_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/braze"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(braze.NewDefinition()))

	registered, err := registry.Get("braze", 1)
	require.NoError(t, err)

	assert.Equal(t, "braze", registered.Type)
	assert.Equal(t, "BRAZE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"rest_api_key"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "cloud", "react_native", "flutter", "cordova",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	assert.NotContains(t, registered.SupportedSourceTypes(), "amp")
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")

	expectedModes := map[string][]string{
		"android":        {"cloud", "device", "hybrid"},
		"android_kotlin": {"cloud", "device", "hybrid"},
		"ios":            {"cloud", "device", "hybrid"},
		"ios_swift":      {"cloud", "device", "hybrid"},
		"web":            {"cloud", "device", "hybrid"},
		"unity":          {"cloud"},
		"cloud":          {"cloud"},
		"react_native":   {"cloud", "device"},
		"flutter":        {"cloud", "device"},
		"cordova":        {"cloud"},
	}
	for sourceType, want := range expectedModes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, want, modes, "source type %s", sourceType)
	}

	assert.Equal(t, map[string][]string{
		"track_anonymous_user/web":           {"web"},
		"enable_braze_logging/web":           {"web"},
		"enable_push_notification/web":       {"web"},
		"allow_user_supplied_javascript/web": {"web"},
	}, registered.GatedKeyPaths())
	// rest_api_key is required wherever Braze talks to the API itself: every
	// cloud-mode source plus the hybrid modes. Device mode needs no entry.
	for _, sourceType := range expectedSourceTypes {
		assert.Equal(t, []string{"rest_api_key"}, registered.SupportedSourcesValidation(sourceType, "cloud"), sourceType)
		assert.Nil(t, registered.SupportedSourcesValidation(sourceType, "device"), sourceType)
	}
	for _, sourceType := range []string{"android", "android_kotlin", "ios", "ios_swift", "web"} {
		assert.Equal(t, []string{"rest_api_key"}, registered.SupportedSourcesValidation(sourceType, "hybrid"), sourceType)
	}

	byAPI, err := registry.GetByAPIType("BRAZE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestBrazeConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(braze.NewDefinition()))
	registered, err := registry.Get("braze", 1)
	require.NoError(t, err)

	minimalConfig := func() map[string]any {
		return map[string]any{
			"data_center": "US-01",
		}
	}

	t.Run("missing data_center", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rest_api_key": "rest-key",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/data_center", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("invalid data_center rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"data_center": "US-99",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/data_center", errors[0].Path)
	})

	t.Run("data_center accepts dynamic template", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"data_center": "{{ .BRAZE_DATA_CENTER || US-01 }}",
		})
		assert.Empty(t, errors)
	})

	t.Run("keys reject values over 100 characters", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"rest_api_key", "app_key", "android_api_key", "ios_api_key", "web_api_key"} {
			t.Run(field, func(t *testing.T) {
				t.Parallel()
				config := minimalConfig()
				config[field] = strings.Repeat("a", 101)

				errors := registered.ValidateConfig(config)
				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+field, errors[0].Path)
			})
		}
	})

	t.Run("keys accept dynamic templates", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["rest_api_key"] = "{{ .BRAZE_REST_API_KEY }}"
		config["app_key"] = "{{ .BRAZE_APP_KEY || fallback-app-key }}"
		config["android_api_key"] = "{{ .BRAZE_ANDROID_API_KEY }}"
		config["ios_api_key"] = "{{ .BRAZE_IOS_API_KEY }}"
		config["web_api_key"] = "{{ .BRAZE_WEB_API_KEY }}"

		errors := registered.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("event filtering lists are mutually exclusive", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["event_filtering"] = map[string]any{
			"whitelist": []any{"Order Completed"},
			"blacklist": []any{"Page Viewed"},
		}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
	})

	t.Run("event filtering event names reject newlines", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["event_filtering"] = map[string]any{
			"whitelist": []any{"Order\nCompleted"},
		}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/event_filtering/whitelist/0", errors[0].Path)
	})

	t.Run("event filtering event names accept dynamic templates", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["event_filtering"] = map[string]any{
			"blacklist": []any{"{{ .BRAZE_DENY_EVENT || Internal Event }}"},
		}

		errors := registered.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(minimalConfig())
		assert.Empty(t, errors)
	})

	t.Run("valid cloud config", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["rest_api_key"] = "rest-key"
		config["enable_subscription_group_in_group_call"] = true
		config["enable_nested_array_operations"] = true
		config["support_dedup"] = true
		config["send_purchase_event_with_extra_properties"] = true
		config["use_ecommerce_recommended_events"] = true

		errors := registered.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("valid device and hybrid config with platform keys", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["rest_api_key"] = "rest-key"
		config["app_key"] = "default-app-key"
		config["use_platform_specific_api_keys"] = true
		config["android_api_key"] = "android-key"
		config["ios_api_key"] = "ios-key"
		config["web_api_key"] = "web-key"
		config["use_native_sdk"] = map[string]any{
			"android_kotlin": true,
			"ios_swift":      true,
			"web":            true,
			"react_native":   true,
		}
		config["track_anonymous_user"] = map[string]any{"web": true}
		config["enable_braze_logging"] = map[string]any{"web": true}
		config["enable_push_notification"] = map[string]any{"web": false}
		config["allow_user_supplied_javascript"] = map[string]any{"web": true}

		errors := registered.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"data_center":  "US-01",
			"rest_api_key": "{{ .BRAZE_REST_API_KEY }}",
			"app_key":      "example-app-key",
			"enable_subscription_group_in_group_call":   false,
			"enable_nested_array_operations":            true,
			"send_purchase_event_with_extra_properties": false,
			"support_dedup":                    true,
			"use_ecommerce_recommended_events": true,
			"use_platform_specific_api_keys":   false,
			"use_native_sdk": map[string]any{
				"web": true,
			},
			"track_anonymous_user": map[string]any{
				"web": true,
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
			},
		})
		assert.Empty(t, errors)
	})

	// connection_mode legality is per source type, taken from this definition's
	// own ConnectionModes map rather than a shared enum.
	t.Run("connection_mode accepts a supported mode", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"connection_mode": map[string]any{"web": "cloud"},
		})

		for _, err := range errors {
			assert.NotEqual(t, "/connection_mode/web", err.Path)
		}
	})

	t.Run("connection_mode rejects an unsupported mode", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"connection_mode": map[string]any{"cloud": "device"},
		})

		var found bool
		for _, err := range errors {
			if err.Path == "/connection_mode/cloud" {
				found = true
				assert.Contains(t, err.Message, "must be one of")
			}
		}
		assert.True(t, found, "expected /connection_mode/cloud to be rejected")
	})

	t.Run("unsupported source key rejected in source-type blocks", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"use_native_sdk"} {
			t.Run(key, func(t *testing.T) {
				t.Parallel()
				config := minimalConfig()
				config[key] = map[string]any{"cloud_source": "cloud"}
				if key == "use_native_sdk" {
					config[key] = map[string]any{"cloud_source": true}
				}

				errors := registered.ValidateConfig(config)
				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+key+"/cloud_source", errors[0].Path)
			})
		}
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

	t.Run("legacy consent blocks are not supported keys", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
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

func TestBrazeConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := braze.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal cloud",
			LocalJSON: `{
				"data_center": "US-01",
				"rest_api_key": "rest-key"
			}`,
			APIJSON: `{
				"dataCenter": "US-01",
				"restApiKey": "rest-key"
			}`,
		},
		{
			Name: "device mode default app key",
			LocalJSON: `{
				"data_center": "US-02",
				"app_key": "default-app-key",
				"use_platform_specific_api_keys": false,
				"use_native_sdk": {
					"web": true,
					"android": true,
					"ios": true
				}
			}`,
			APIJSON: `{
				"dataCenter": "US-02",
				"appKey": "default-app-key",
				"usePlatformSpecificApiKeys": false,
				"useNativeSDK": {
					"web": true,
					"android": true,
					"ios": true
				}
			}`,
		},
		{
			Name: "platform specific keys and sdk booleans",
			LocalJSON: `{
				"data_center": "EU-01",
				"rest_api_key": "rest-key",
				"use_platform_specific_api_keys": true,
				"android_api_key": "android-key",
				"ios_api_key": "ios-key",
				"web_api_key": "web-key",
				"use_native_sdk": {
					"android_kotlin": true,
					"ios_swift": true,
					"react_native": true,
					"flutter": true,
					"web": true
				},
				"track_anonymous_user": {"web": true},
				"enable_braze_logging": {"web": true},
				"enable_push_notification": {"web": false},
				"allow_user_supplied_javascript": {"web": true}
			}`,
			APIJSON: `{
				"dataCenter": "EU-01",
				"restApiKey": "rest-key",
				"usePlatformSpecificApiKeys": true,
				"androidApiKey": "android-key",
				"iOSApiKey": "ios-key",
				"webApiKey": "web-key",
				"useNativeSDK": {
					"androidKotlin": true,
					"iosSwift": true,
					"reactnative": true,
					"flutter": true,
					"web": true
				},
				"trackAnonymousUser": {"web": true},
				"enableBrazeLogging": {"web": true},
				"enablePushNotification": {"web": false},
				"allowUserSuppliedJavascript": {"web": true}
			}`,
		},
		{
			Name: "cloud booleans and event filtering whitelist",
			LocalJSON: `{
				"data_center": "AU-01",
				"rest_api_key": "rest-key",
				"enable_subscription_group_in_group_call": true,
				"enable_nested_array_operations": true,
				"send_purchase_event_with_extra_properties": true,
				"support_dedup": true,
				"use_ecommerce_recommended_events": true,
				"event_filtering": {
					"whitelist": ["Product Viewed", "Order Completed"]
				}
			}`,
			APIJSON: `{
				"dataCenter": "AU-01",
				"restApiKey": "rest-key",
				"enableSubscriptionGroupInGroupCall": true,
				"enableNestedArrayOperations": true,
				"sendPurchaseEventWithExtraProperties": true,
				"supportDedup": true,
				"useEcommerceRecommendedEvents": true,
				"whitelistedEvents": [
					{"eventName": "Product Viewed"},
					{"eventName": "Order Completed"}
				],
				"eventFilteringOption": "whitelistedEvents"
			}`,
		},
		{
			Name: "event filtering blacklist",
			LocalJSON: `{
				"data_center": "US-03",
				"event_filtering": {
					"blacklist": ["Internal Event"]
				}
			}`,
			APIJSON: `{
				"dataCenter": "US-03",
				"blacklistedEvents": [
					{"eventName": "Internal Event"}
				],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"data_center": "US-04",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}],
					"cloud": [{"provider": "custom", "resolution_strategy": "or", "consents": ["analytics"]}]
				}
			}`,
			APIJSON: `{
				"dataCenter": "US-04",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}],
					"cloud": [{
						"provider": "custom",
						"resolutionStrategy": "or",
						"consents": [{"consent": "analytics"}]
					}]
				}
			}`,
		},
	})
}

// schema.json gates the five API keys on connection_mode combined with
// use_platform_specific_api_keys. Every branch was previously unexpressed, so a
// config missing a genuinely required key validated locally and failed upstream.
func TestBrazeAPIKeyConditionals(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(braze.NewDefinition()))
	registered, err := registry.Get("braze", 1)
	require.NoError(t, err)

	config := func(extra map[string]any) map[string]any {
		cfg := map[string]any{"data_center": "US-01"}
		for k, v := range extra {
			cfg[k] = v
		}
		return cfg
	}
	paths := func(errors []definitions.ConfigError) []string {
		out := make([]string, 0, len(errors))
		for _, err := range errors {
			out = append(out, err.Path)
		}
		return out
	}

	cases := []struct {
		name  string
		extra map[string]any
		want  []string
	}{
		{
			name:  "cloud requires rest_api_key",
			extra: map[string]any{"connection_mode": map[string]any{"web": "cloud"}},
			want:  []string{"/rest_api_key"},
		},
		{
			name:  "cloud satisfied by rest_api_key",
			extra: map[string]any{"connection_mode": map[string]any{"web": "cloud"}, "rest_api_key": "rest-key"},
			want:  []string{},
		},
		{
			name: "device without platform-specific keys requires app_key",
			extra: map[string]any{
				"connection_mode":                map[string]any{"web": "device"},
				"use_platform_specific_api_keys": false,
			},
			want: []string{"/app_key"},
		},
		{
			name: "platform-specific android requires android_api_key",
			extra: map[string]any{
				"connection_mode":                map[string]any{"android": "device"},
				"use_platform_specific_api_keys": true,
			},
			want: []string{"/android_api_key"},
		},
		{
			name: "platform-specific web requires web_api_key",
			extra: map[string]any{
				"connection_mode":                map[string]any{"web": "device"},
				"use_platform_specific_api_keys": true,
			},
			want: []string{"/web_api_key"},
		},
		{
			// react_native and flutter ship both native SDKs, so schema.json's
			// last branch requires the Android and iOS keys together.
			name: "platform-specific react_native requires both native keys",
			extra: map[string]any{
				"connection_mode":                map[string]any{"react_native": "device"},
				"use_platform_specific_api_keys": true,
			},
			want: []string{"/android_api_key", "/ios_api_key"},
		},
		{
			name:  "no connection mode requires no api key",
			extra: map[string]any{},
			want:  []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ElementsMatch(t, tc.want, paths(registered.ValidateConfig(config(tc.extra))))
		})
	}
}
