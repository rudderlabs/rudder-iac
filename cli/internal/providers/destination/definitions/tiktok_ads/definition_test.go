package tiktokads_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	tiktokads "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/tiktok_ads"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(tiktokads.NewDefinition()))

	registered, err := registry.Get("tiktok_ads", 1)
	require.NoError(t, err)

	assert.Equal(t, "tiktok_ads", registered.Type)
	assert.Equal(t, "TIKTOK_ADS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"access_token"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "react_native", "flutter", "cordova", "cloud",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		if sourceType == "web" {
			assert.Equal(t, []string{"cloud", "device"}, modes)
			continue
		}
		assert.Equal(t, []string{"cloud"}, modes)
	}

	assert.NotContains(t, registered.SupportedSourceTypes(), "amp")
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")

	byAPI, err := registry.GetByAPIType("TIKTOK_ADS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestTiktokAdsConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(tiktokads.NewDefinition()))
	registered, err := registry.Get("tiktok_ads", 1)
	require.NoError(t, err)

	t.Run("missing pixel_code", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"access_token": "token",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/pixel_code", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("empty pixel_code rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"pixel_code": "",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/pixel_code", errors[0].Path)
	})

	t.Run("invalid version rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"pixel_code": "C12345",
			"version":    "v3",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/version", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("invalid events_to_standard.to rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"pixel_code": "C12345",
			"events_to_standard": []any{
				map[string]any{"from": "Product Viewed", "to": "NotAStandardEvent"},
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/events_to_standard/0/to", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("invalid connection_mode.web rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"pixel_code": "C12345",
			"connection_mode": map[string]any{
				"web": "hybrid",
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/connection_mode/web", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"pixel_code": "C12345",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid example config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"pixel_code":           "C12345ABCDEF",
			"access_token":         "tiktok-long-lived-token",
			"version":              "v2",
			"hash_user_properties": true,
			"send_custom_events":   true,
			"events_to_standard": []any{
				map[string]any{"from": "Order Completed", "to": "CompletePayment"},
				map[string]any{"from": "Product Added", "to": "AddToCart"},
			},
			"event_filtering_whitelist": []any{"Order Completed", "Product Added"},
			"use_native_sdk": map[string]any{
				"web": true,
			},
			"connection_mode": map[string]any{
				"web":   "device",
				"cloud": "cloud",
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

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"pixel_code":           "C12345",
			"access_token":         "secret-token",
			"version":              "v1",
			"hash_user_properties": false,
			"send_custom_events":   false,
			"events_to_standard": []any{
				map[string]any{"from": "Signed Up", "to": "CompleteRegistration"},
			},
			"event_filtering_blacklist": []any{"Page Viewed"},
			"connection_mode": map[string]any{
				"android":        "cloud",
				"android_kotlin": "cloud",
				"ios":            "cloud",
				"ios_swift":      "cloud",
				"react_native":   "cloud",
				"flutter":        "cloud",
				"cordova":        "cloud",
				"unity":          "cloud",
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"pixel_code":  "C12345",
			"not_a_field": true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"pixel_code": "C12345",
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
			"pixel_code": "C12345",
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

func TestTiktokAdsConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := tiktokads.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal pixel only",
			LocalJSON: `{
				"pixel_code": "C12345"
			}`,
			APIJSON: `{
				"pixelCode": "C12345"
			}`,
		},
		{
			Name: "full fields with whitelist reshape",
			LocalJSON: `{
				"pixel_code": "C12345",
				"access_token": "secret-token",
				"version": "v2",
				"hash_user_properties": true,
				"send_custom_events": true,
				"events_to_standard": [
					{"from": "Order Completed", "to": "CompletePayment"}
				],
				"event_filtering_whitelist": ["Order Completed", "Product Added"],
				"use_native_sdk": {"web": true},
				"connection_mode": {"web": "device", "cloud": "cloud"}
			}`,
			APIJSON: `{
				"pixelCode": "C12345",
				"accessToken": "secret-token",
				"version": "v2",
				"hashUserProperties": true,
				"sendCustomEvents": true,
				"eventsToStandard": [
					{"from": "Order Completed", "to": "CompletePayment"}
				],
				"whitelistedEvents": [
					{"eventName": "Order Completed"},
					{"eventName": "Product Added"}
				],
				"eventFilteringOption": "whitelistedEvents",
				"useNativeSDK": {"web": true},
				"connectionMode": {"web": "device", "cloud": "cloud"}
			}`,
		},
		{
			Name: "blacklist reshape",
			LocalJSON: `{
				"pixel_code": "C12345",
				"event_filtering_blacklist": ["Page Viewed"]
			}`,
			APIJSON: `{
				"pixelCode": "C12345",
				"blacklistedEvents": [
					{"eventName": "Page Viewed"}
				],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"pixel_code": "C12345",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"pixelCode": "C12345",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
