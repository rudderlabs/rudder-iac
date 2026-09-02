package tiktokads_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
	tiktokads "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/tiktok_ads"
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

	// connection_mode is not destination config: no other definition models it,
	// and per-source connection modes belong to the connections work. The
	// definition still advertises them via ConnectionModes() metadata.
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
			"connection_mode": map[string]any{"web": "hybrid"},
		})

		var found bool
		for _, err := range errors {
			if err.Path == "/connection_mode/web" {
				found = true
				assert.Contains(t, err.Message, "must be one of")
			}
		}
		assert.True(t, found, "expected /connection_mode/web to be rejected")
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
			"event_filtering": map[string]any{
				"whitelist": []any{"Order Completed", "Product Added"},
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
			"event_filtering": map[string]any{
				"blacklist": []any{"Page Viewed"},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("event filtering rejects whitelist and blacklist together", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"pixel_code": "C12345",
			"event_filtering": map[string]any{
				"whitelist": []any{"Order Completed"},
				"blacklist": []any{"Page Viewed"},
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/event_filtering/whitelist", errors[0].Path)
		assert.Contains(t, errors[0].Message, "cannot be specified together")
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

// Empty-string handling on enum fields is not decided by the enum: omitempty
// and dynamic_or_oneof both pass "" through regardless of what schema.json
// lists. These cases pin the resulting verdicts so a later tightening has to
// confront them deliberately — see docs/destination-validation-discrepancies.md.
func TestTiktokAdsEmptyEnumValues(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(tiktokads.NewDefinition()))
	registered, err := registry.Get("tiktok_ads", 1)
	require.NoError(t, err)

	// schema.json lists "" as the 23rd member of the eventsToStandard `to`
	// enum, and terraform encodes it as a trailing empty alternative. The UI
	// persists half-filled mapping rows, so rejecting it would break importing
	// a workspace that has one.
	t.Run("events_to_standard.to accepts an empty mapping", func(t *testing.T) {
		t.Parallel()

		config := map[string]any{
			"pixel_code":         "C12345",
			"events_to_standard": []any{map[string]any{"from": "Order Completed", "to": ""}},
		}
		assert.Empty(t, registered.ValidateConfig(config))

		api, err := registered.LocalToAPI(config)
		require.NoError(t, err)
		assert.Equal(t, []any{map[string]any{"from": "Order Completed", "to": ""}}, api["eventsToStandard"],
			"an empty mapping must survive conversion rather than being dropped")
	})

	t.Run("events_to_standard.to still rejects a non-standard event", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"pixel_code":         "C12345",
			"events_to_standard": []any{map[string]any{"from": "Order Completed", "to": "NotAStandardEvent"}},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/events_to_standard/0/to", errors[0].Path)
	})

	// Known divergence: schema.json's version enum is ["v2","v1"] with no empty
	// member, yet "" validates and ships as an empty value the backend rejects.
	// Pinned as-is; flipping this assertion is the marker for that fix landing.
	t.Run("version accepts empty despite the upstream enum", func(t *testing.T) {
		t.Parallel()

		config := map[string]any{"pixel_code": "C12345", "version": ""}
		assert.Empty(t, registered.ValidateConfig(config),
			"known gap: omitempty passes \"\" before the enum is consulted")

		api, err := registered.LocalToAPI(config)
		require.NoError(t, err)
		assert.Equal(t, "", api["version"], "the empty value reaches the payload")

		absent, err := registered.LocalToAPI(map[string]any{"pixel_code": "C12345"})
		require.NoError(t, err)
		assert.NotContains(t, absent, "version", "an absent key stays absent — distinguishable from \"\"")
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
				"event_filtering": {"whitelist": ["Order Completed", "Product Added"]},
				"use_native_sdk": {"web": true}
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
				"useNativeSDK": {"web": true}
			}`,
		},
		{
			Name: "blacklist reshape",
			LocalJSON: `{
				"pixel_code": "C12345",
				"event_filtering": {"blacklist": ["Page Viewed"]}
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
