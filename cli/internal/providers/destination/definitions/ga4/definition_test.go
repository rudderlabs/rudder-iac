package ga4_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/ga4"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(ga4.NewDefinition()))

	registered, err := registry.Get("ga4", 1)
	require.NoError(t, err)

	assert.Equal(t, "ga4", registered.Type)
	assert.Equal(t, "GA4", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"api_secret"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "react_native", "flutter", "cordova", "cloud",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	expectedModes := map[string][]string{
		"android":        {"cloud", "device"},
		"android_kotlin": {"cloud"},
		"ios":            {"cloud", "device"},
		"ios_swift":      {"cloud"},
		"web":            {"cloud", "device", "hybrid"},
		"unity":          {"cloud"},
		"react_native":   {"cloud"},
		"flutter":        {"cloud"},
		"cordova":        {"cloud"},
		"cloud":          {"cloud"},
	}
	for sourceType, want := range expectedModes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, want, modes, "source type %s", sourceType)
	}

	assert.NotContains(t, registered.SupportedSourceTypes(), "amp")
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")

	assert.Equal(t, map[string][]string{
		"extend_page_view_params":             {"web"},
		"capture_page_view/web":               {"web"},
		"debug_view/web":                      {"web"},
		"override_client_and_session_ids/web": {"web"},
	}, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("GA4", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestGA4ConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(ga4.NewDefinition()))
	registered, err := registry.Get("ga4", 1)
	require.NoError(t, err)

	t.Run("missing api_secret", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"client_type":    "gtag",
			"measurement_id": "G-XXXXXXXXXX",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_secret", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing client_type", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret": "secret",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/client_type", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("measurement_id required when client_type gtag", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":  "secret",
			"client_type": "gtag",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/measurement_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("firebase_app_id required when client_type firebase", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":  "secret",
			"client_type": "firebase",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/firebase_app_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("invalid client_type rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":     "secret",
			"client_type":    "other",
			"measurement_id": "G-XXXXXXXXXX",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/client_type", errors[0].Path)
	})

	t.Run("valid minimal gtag", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":     "secret",
			"client_type":    "gtag",
			"measurement_id": "G-XXXXXXXXXX",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid minimal firebase", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":      "secret",
			"client_type":     "firebase",
			"firebase_app_id": "1:123:android:abc",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid full config example", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":     "my-api-secret",
			"client_type":    "gtag",
			"measurement_id": "G-XXXXXXXXXX",
			"debug_mode":     true,
			"sdk_base_url":   "https://www.googletagmanager.com",
			"pii_properties_to_ignore": []any{
				map[string]any{"pii_property": "email"},
			},
			"event_filtering": map[string]any{
				"whitelist": []any{"Product Viewed", "Order Completed"},
			},
			"use_native_sdk": map[string]any{
				"web": true,
			},
			"capture_page_view": map[string]any{
				"web": "rs",
			},
			"debug_view": map[string]any{
				"web": true,
			},
			"override_client_and_session_ids": map[string]any{
				"web": true,
			},
			"extend_page_view_params": true,
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
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":     "secret",
			"client_type":    "gtag",
			"measurement_id": "G-XXXXXXXXXX",
			"not_a_field":    true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":     "secret",
			"client_type":    "gtag",
			"measurement_id": "G-XXXXXXXXXX",
			"consent_management": map[string]any{
				"warehouse": []any{},
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'warehouse' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":     "secret",
			"client_type":    "gtag",
			"measurement_id": "G-XXXXXXXXXX",
			"consent_management": map[string]any{
				"android_kotlin": []any{
					map[string]any{"provider": "unknown"},
				},
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/android_kotlin/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})

	t.Run("invalid capture_page_view value rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":     "secret",
			"client_type":    "gtag",
			"measurement_id": "G-XXXXXXXXXX",
			"capture_page_view": map[string]any{
				"web": "other",
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/capture_page_view/web", errors[0].Path)
	})

	t.Run("sdk_base_url ngrok rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":     "secret",
			"client_type":    "gtag",
			"measurement_id": "G-XXXXXXXXXX",
			"sdk_base_url":   "https://abc.ngrok.io",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/sdk_base_url", errors[0].Path)
		assert.Contains(t, errors[0].Message, "ngrok")
	})

	t.Run("server_container_url invalid rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":           "secret",
			"client_type":          "gtag",
			"measurement_id":       "G-XXXXXXXXXX",
			"server_container_url": "not a url",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/server_container_url", errors[0].Path)
	})

	t.Run("sdk_base_url dynamic value allowed", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":     "secret",
			"client_type":    "gtag",
			"measurement_id": "G-XXXXXXXXXX",
			"sdk_base_url":   "env.SDK_BASE_URL",
		})
		assert.Empty(t, errors)
	})
}

func TestGA4ConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := ga4.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal gtag",
			LocalJSON: `{
				"api_secret": "secret",
				"client_type": "gtag",
				"measurement_id": "G-XXXXXXXXXX"
			}`,
			APIJSON: `{
				"apiSecret": "secret",
				"typesOfClient": "gtag",
				"measurementId": "G-XXXXXXXXXX"
			}`,
		},
		{
			Name: "full TF fields",
			LocalJSON: `{
				"api_secret": "secret",
				"client_type": "gtag",
				"measurement_id": "G-XXXXXXXXXX",
				"firebase_app_id": "1:123:android:abc",
				"debug_mode": true,
				"block_page_view_event": true,
				"extend_page_view_params": true,
				"send_user_id": true,
				"sdk_base_url": "https://www.googletagmanager.com",
				"server_container_url": "https://gtm.example.com",
				"pii_properties_to_ignore": [
					{"pii_property": "email"},
					{"pii_property": "phone"}
				],
				"event_filtering": {
					"whitelist": ["Product Viewed", "Order Completed"]
				},
				"use_native_sdk": {
					"web": true,
					"android": true,
					"ios": false
				},
				"capture_page_view": {"web": "rs"},
				"debug_view": {"web": true},
				"override_client_and_session_ids": {"web": true}
			}`,
			APIJSON: `{
				"apiSecret": "secret",
				"typesOfClient": "gtag",
				"measurementId": "G-XXXXXXXXXX",
				"firebaseAppId": "1:123:android:abc",
				"debugMode": true,
				"blockPageViewEvent": true,
				"extendPageViewParams": true,
				"sendUserId": true,
				"sdkBaseUrl": "https://www.googletagmanager.com",
				"serverContainerUrl": "https://gtm.example.com",
				"piiPropertiesToIgnore": [
					{"piiProperty": "email"},
					{"piiProperty": "phone"}
				],
				"whitelistedEvents": [
					{"eventName": "Product Viewed"},
					{"eventName": "Order Completed"}
				],
				"eventFilteringOption": "whitelistedEvents",
				"useNativeSDK": {
					"web": true,
					"android": true,
					"ios": false
				},
				"capturePageView": {"web": "rs"},
				"debugView": {"web": true},
				"overrideClientAndSessionId": {"web": true}
			}`,
		},
		{
			Name: "event filtering blacklist",
			LocalJSON: `{
				"api_secret": "secret",
				"client_type": "gtag",
				"measurement_id": "G-XXXXXXXXXX",
				"event_filtering": {
					"blacklist": ["Application Opened"]
				}
			}`,
			APIJSON: `{
				"apiSecret": "secret",
				"typesOfClient": "gtag",
				"measurementId": "G-XXXXXXXXXX",
				"blacklistedEvents": [
					{"eventName": "Application Opened"}
				],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "firebase client",
			LocalJSON: `{
				"api_secret": "secret",
				"client_type": "firebase",
				"firebase_app_id": "1:123:android:abc"
			}`,
			APIJSON: `{
				"apiSecret": "secret",
				"typesOfClient": "firebase",
				"firebaseAppId": "1:123:android:abc"
			}`,
		},
		{
			Name: "consent for web",
			LocalJSON: `{
				"api_secret": "secret",
				"client_type": "gtag",
				"measurement_id": "G-XXXXXXXXXX",
				"consent_management": {
					"web": [
						{
							"provider": "oneTrust",
							"resolution_strategy": "and",
							"consents": ["analytics", "marketing"]
						}
					]
				}
			}`,
			APIJSON: `{
				"apiSecret": "secret",
				"typesOfClient": "gtag",
				"measurementId": "G-XXXXXXXXXX",
				"consentManagement": {
					"web": [
						{
							"provider": "oneTrust",
							"resolutionStrategy": "and",
							"consents": [
								{"consent": "analytics"},
								{"consent": "marketing"}
							]
						}
					]
				}
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"api_secret": "secret",
				"client_type": "gtag",
				"measurement_id": "G-XXXXXXXXXX",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"apiSecret": "secret",
				"typesOfClient": "gtag",
				"measurementId": "G-XXXXXXXXXX",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
