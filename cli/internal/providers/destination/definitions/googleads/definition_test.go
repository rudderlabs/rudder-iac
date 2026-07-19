package googleads_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/googleads"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googleads.NewDefinition()))

	registered, err := registry.Get("googleads", 1)
	require.NoError(t, err)

	assert.Equal(t, "googleads", registered.Type)
	assert.Equal(t, "GOOGLEADS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{}, registered.SecretKeys())

	expectedSourceTypes := []string{"web"}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	modes, err := registered.ConnectionModes("web")
	require.NoError(t, err)
	assert.Equal(t, []string{"device"}, modes)

	assert.Empty(t, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("GOOGLEADS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestGoogleAdsConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googleads.NewDefinition()))
	registered, err := registry.Get("googleads", 1)
	require.NoError(t, err)

	t.Run("missing conversion_id", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/conversion_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"conversion_id": "AW-123456789",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"conversion_id": "AW-123456789",
			"page_load_conversions": []any{
				map[string]any{"label": "page-label", "name": "home"},
			},
			"click_event_conversions": []any{
				map[string]any{"label": "click-label", "name": "Purchase"},
			},
			"default_page_conversion": "default-label",
			"dynamic_remarketing": map[string]any{
				"web": true,
			},
			"conversion_linker":          true,
			"send_page_view":             true,
			"disable_ad_personalization": true,
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

	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"conversion_id": "AW-123456789",
			"page_load_conversions": []any{
				map[string]any{"label": "abcDEF123", "name": "home"},
			},
			"click_event_conversions": []any{
				map[string]any{"label": "xyzABC456", "name": "Purchase"},
			},
			"default_page_conversion": "defaultLabel",
			"dynamic_remarketing": map[string]any{
				"web": true,
			},
			"conversion_linker":          true,
			"send_page_view":             true,
			"disable_ad_personalization": false,
			"event_filtering": map[string]any{
				"blacklist": []any{"Application Opened"},
			},
			"use_native_sdk": map[string]any{
				"web": true,
			},
			"consent_management": map[string]any{
				"web": []any{
					map[string]any{
						"provider":            "oneTrust",
						"resolution_strategy": "and",
						"consents":            []any{"analytics", "marketing"},
					},
				},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("page_load_conversions label required", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"conversion_id": "AW-123456789",
			"page_load_conversions": []any{
				map[string]any{"name": "home"},
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/page_load_conversions/0/label", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"conversion_id": "AW-123456789",
			"not_a_field":   true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"conversion_id": "AW-123456789",
			"consent_management": map[string]any{
				"android": []any{},
			},
		})

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/android", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'android' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"conversion_id": "AW-123456789",
			"consent_management": map[string]any{
				"web": []any{
					map[string]any{"provider": "unknown"},
				},
			},
		})

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestGoogleAdsConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := googleads.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal conversion id only",
			LocalJSON: `{
				"conversion_id": "AW-123456789"
			}`,
			APIJSON: `{
				"conversionID": "AW-123456789"
			}`,
		},
		{
			Name: "full TF fields with whitelist",
			LocalJSON: `{
				"conversion_id": "AW-123456789",
				"page_load_conversions": [
					{"label": "page-label", "name": "home"}
				],
				"click_event_conversions": [
					{"label": "click-label", "name": "Purchase"}
				],
				"default_page_conversion": "default-label",
				"dynamic_remarketing": {"web": true},
				"conversion_linker": true,
				"send_page_view": true,
				"disable_ad_personalization": true,
				"event_filtering": {
					"whitelist": ["Product Viewed", "Order Completed"]
				},
				"use_native_sdk": {"web": true}
			}`,
			APIJSON: `{
				"conversionID": "AW-123456789",
				"pageLoadConversions": [
					{"conversionLabel": "page-label", "name": "home"}
				],
				"clickEventConversions": [
					{"conversionLabel": "click-label", "name": "Purchase"}
				],
				"defaultPageConversion": "default-label",
				"dynamicRemarketing": {"web": true},
				"conversionLinker": true,
				"sendPageView": true,
				"disableAdPersonalization": true,
				"whitelistedEvents": [
					{"eventName": "Product Viewed"},
					{"eventName": "Order Completed"}
				],
				"eventFilteringOption": "whitelistedEvents",
				"useNativeSDK": {"web": true}
			}`,
		},
		{
			Name: "event filtering blacklist reshape",
			LocalJSON: `{
				"conversion_id": "AW-123456789",
				"event_filtering": {
					"blacklist": ["Application Opened"]
				}
			}`,
			APIJSON: `{
				"conversionID": "AW-123456789",
				"blacklistedEvents": [
					{"eventName": "Application Opened"}
				],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "conversion arrays reshape",
			LocalJSON: `{
				"conversion_id": "AW-123456789",
				"page_load_conversions": [
					{"label": "lbl1", "name": "Page Viewed"}
				],
				"click_event_conversions": [
					{"label": "lbl2", "name": "Signed Up"}
				]
			}`,
			APIJSON: `{
				"conversionID": "AW-123456789",
				"pageLoadConversions": [
					{"conversionLabel": "lbl1", "name": "Page Viewed"}
				],
				"clickEventConversions": [
					{"conversionLabel": "lbl2", "name": "Signed Up"}
				]
			}`,
		},
		{
			Name: "consent for web",
			LocalJSON: `{
				"conversion_id": "AW-123456789",
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
				"conversionID": "AW-123456789",
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
	})
}
