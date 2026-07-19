package posthog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/posthog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(posthog.NewDefinition()))

	registered, err := registry.Get("posthog", 1)
	require.NoError(t, err)

	assert.Equal(t, "posthog", registered.Type)
	assert.Equal(t, "POSTHOG", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"api_key"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "react_native", "flutter", "cordova", "cloud",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	expectedModes := map[string][]string{
		"android":        {"cloud"},
		"android_kotlin": {"cloud"},
		"ios":            {"cloud"},
		"ios_swift":      {"cloud"},
		"web":            {"cloud", "device"},
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
		"disable_session_recording/web":        {"web"},
		"capture_page_view/web":                {"web"},
		"autocapture/web":                      {"web"},
		"enable_local_storage_persistence/web": {"web"},
		"xhr_headers":                          {"web"},
		"property_blacklist":                   {"web"},
		"person_profiles/web":                  {"web"},
	}, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("POSTHOG", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestPosthogConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(posthog.NewDefinition()))
	registered, err := registry.Get("posthog", 1)
	require.NoError(t, err)

	t.Run("missing api_key", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_key", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key": "phc_test_key",
		})
		assert.Empty(t, errors)
	})

	t.Run("endpoint too long rejected", func(t *testing.T) {
		t.Parallel()
		longEndpoint := make([]byte, 101)
		for i := range longEndpoint {
			longEndpoint[i] = 'a'
		}
		errors := registered.ValidateConfig(map[string]any{
			"api_key":  "phc_test_key",
			"endpoint": string(longEndpoint),
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/endpoint", errors[0].Path)
		assert.Contains(t, errors[0].Message, "100")
	})

	t.Run("invalid person_profiles rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key": "phc_test_key",
			"person_profiles": map[string]any{
				"web": "sometimes",
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/person_profiles/web", errors[0].Path)
	})

	t.Run("xhr_headers value too long rejected", func(t *testing.T) {
		t.Parallel()
		longValue := make([]byte, 101)
		for i := range longValue {
			longValue[i] = 'a'
		}
		errors := registered.ValidateConfig(map[string]any{
			"api_key": "phc_test_key",
			"xhr_headers": []any{
				map[string]any{"key": "X-Custom", "value": string(longValue)},
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/xhr_headers/0/value", errors[0].Path)
		assert.Contains(t, errors[0].Message, "100")
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key":      "phc_test_key",
			"endpoint":     "https://app.posthog.com",
			"use_v2_group": true,
			"event_filtering": map[string]any{
				"whitelist": []any{"Product Viewed", "Order Completed"},
			},
			"use_native_sdk": map[string]any{
				"web": true,
			},
			"autocapture": map[string]any{
				"web": true,
			},
			"capture_page_view": map[string]any{
				"web": false,
			},
			"disable_session_recording": map[string]any{
				"web": true,
			},
			"enable_local_storage_persistence": map[string]any{
				"web": true,
			},
			"person_profiles": map[string]any{
				"web": "identified_only",
			},
			"xhr_headers": []any{
				map[string]any{"key": "X-Custom", "value": "value"},
			},
			"property_blacklist": []any{
				map[string]any{"property": "ssn"},
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

	t.Run("example yaml config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key":      "phc_YOUR_PROJECT_API_KEY",
			"endpoint":     "https://app.posthog.com",
			"use_v2_group": true,
			"event_filtering": map[string]any{
				"whitelist": []any{"Product Viewed", "Order Completed"},
			},
			"use_native_sdk": map[string]any{
				"web": true,
			},
			"autocapture": map[string]any{
				"web": true,
			},
			"capture_page_view": map[string]any{
				"web": true,
			},
			"disable_session_recording": map[string]any{
				"web": false,
			},
			"enable_local_storage_persistence": map[string]any{
				"web": true,
			},
			"person_profiles": map[string]any{
				"web": "always",
			},
			"xhr_headers": []any{
				map[string]any{"key": "X-Custom-Header", "value": "custom-value"},
			},
			"property_blacklist": []any{
				map[string]any{"property": "ssn"},
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
		errors := registered.ValidateConfig(map[string]any{
			"api_key":     "phc_test_key",
			"not_a_field": true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key": "phc_test_key",
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
			"api_key": "phc_test_key",
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
}

func TestPosthogConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := posthog.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal api key",
			LocalJSON: `{
				"api_key": "phc_test_key"
			}`,
			APIJSON: `{
				"teamApiKey": "phc_test_key"
			}`,
		},
		{
			Name: "full fields",
			LocalJSON: `{
				"api_key": "phc_test_key",
				"endpoint": "https://app.posthog.com",
				"use_v2_group": true,
				"event_filtering": {
					"whitelist": ["Product Viewed", "Order Completed"]
				},
				"use_native_sdk": {"web": true},
				"autocapture": {"web": true},
				"capture_page_view": {"web": false},
				"disable_session_recording": {"web": true},
				"enable_local_storage_persistence": {"web": true},
				"person_profiles": {"web": "identified_only"},
				"xhr_headers": [
					{"key": "X-Custom", "value": "value"}
				],
				"property_blacklist": [
					{"property": "ssn"}
				]
			}`,
			APIJSON: `{
				"teamApiKey": "phc_test_key",
				"yourInstance": "https://app.posthog.com",
				"useV2Group": true,
				"eventFilteringOption": "whitelistedEvents",
				"whitelistedEvents": [
					{"eventName": "Product Viewed"},
					{"eventName": "Order Completed"}
				],
				"useNativeSDK": {"web": true},
				"autocapture": {"web": true},
				"capturePageView": {"web": false},
				"disableSessionRecording": {"web": true},
				"enableLocalStoragePersistence": {"web": true},
				"personProfiles": {"web": "identified_only"},
				"xhrHeaders": {
					"web": [
						{"key": "X-Custom", "value": "value"}
					]
				},
				"propertyBlacklist": {
					"web": [
						{"property": "ssn"}
					]
				}
			}`,
		},
		{
			Name: "event filtering blacklist",
			LocalJSON: `{
				"api_key": "phc_test_key",
				"event_filtering": {
					"blacklist": ["Application Opened"]
				}
			}`,
			APIJSON: `{
				"teamApiKey": "phc_test_key",
				"eventFilteringOption": "blacklistedEvents",
				"blacklistedEvents": [
					{"eventName": "Application Opened"}
				]
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"api_key": "phc_test_key",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"teamApiKey": "phc_test_key",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
