package iterable_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/iterable"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(iterable.NewDefinition()))

	registered, err := registry.Get("iterable", 1)
	require.NoError(t, err)

	assert.Equal(t, "iterable", registered.Type)
	assert.Equal(t, "ITERABLE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{}, registered.SecretKeys())

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

	assert.Equal(t, map[string][]string{
		"initialisation_identifier/web":      {"web"},
		"get_in_app_event_mapping/web":       {"web"},
		"purchase_event_mapping/web":         {"web"},
		"send_track_for_inapp/web":           {"web"},
		"animation_duration/web":             {"web"},
		"display_interval/web":               {"web"},
		"on_open_screen_reader_message/web":  {"web"},
		"on_open_node_to_take_focus/web":     {"web"},
		"right_offset/web":                   {"web"},
		"top_offset/web":                     {"web"},
		"bottom_offset/web":                  {"web"},
		"handle_links/web":                   {"web"},
		"close_button_color/web":             {"web"},
		"close_button_size/web":              {"web"},
		"close_button_color_top_offset/web":  {"web"},
		"close_button_color_side_offset/web": {"web"},
		"icon_path/web":                      {"web"},
		"is_required_to_dismiss_message/web": {"web"},
		"close_button_position/web":          {"web"},
	}, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("ITERABLE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestIterableConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(iterable.NewDefinition()))
	registered, err := registry.Get("iterable", 1)
	require.NoError(t, err)

	t.Run("missing api_key", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_key", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("api_key too long", func(t *testing.T) {
		t.Parallel()
		long := make([]byte, 101)
		for i := range long {
			long[i] = 'a'
		}
		errors := registered.ValidateConfig(map[string]any{
			"api_key": string(long),
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_key", errors[0].Path)
		assert.Contains(t, errors[0].Message, "100")
	})

	t.Run("invalid initialisation_identifier", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key": "iterable-api-key",
			"initialisation_identifier": map[string]any{
				"web": "phone",
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/initialisation_identifier/web", errors[0].Path)
	})

	t.Run("invalid handle_links", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key": "iterable-api-key",
			"handle_links": map[string]any{
				"web": "invalid-option",
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/handle_links/web", errors[0].Path)
	})

	t.Run("invalid close_button_position", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key": "iterable-api-key",
			"close_button_position": map[string]any{
				"web": "bottom-right",
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/close_button_position/web", errors[0].Path)
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key": "iterable-api-key",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid example config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key":                 "iterable-api-key",
			"map_to_single_event":     true,
			"track_all_pages":         false,
			"track_categorized_pages": true,
			"track_named_pages":       true,
			"use_native_sdk": map[string]any{
				"web": true,
			},
			"initialisation_identifier": map[string]any{
				"web": "email",
			},
			"get_in_app_event_mapping": map[string]any{
				"web": []any{"Product Viewed", "Cart Updated"},
			},
			"purchase_event_mapping": map[string]any{
				"web": []any{"Order Completed"},
			},
			"send_track_for_inapp": map[string]any{
				"web": true,
			},
			"animation_duration": map[string]any{
				"web": "200",
			},
			"display_interval": map[string]any{
				"web": "2500",
			},
			"package_name": map[string]any{
				"web": "com.example.app",
			},
			"handle_links": map[string]any{
				"web": "open-all-new-tab",
			},
			"close_button_position": map[string]any{
				"web": "top-right",
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
			"api_key":     "iterable-api-key",
			"not_a_field": true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key": "iterable-api-key",
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
			"api_key": "iterable-api-key",
			"consent_management": map[string]any{
				"ios_swift": []any{
					map[string]any{"provider": "unknown"},
				},
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/ios_swift/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestIterableConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := iterable.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal api key",
			LocalJSON: `{
				"api_key": "iterable-api-key"
			}`,
			APIJSON: `{
				"apiKey": "iterable-api-key"
			}`,
		},
		{
			Name: "full TF fields",
			LocalJSON: `{
				"api_key": "iterable-api-key",
				"map_to_single_event": true,
				"track_all_pages": true,
				"track_categorized_pages": true,
				"track_named_pages": true,
				"use_native_sdk": {"web": true},
				"initialisation_identifier": {"web": "email"},
				"get_in_app_event_mapping": {"web": ["Product Viewed", "Cart Updated"]},
				"purchase_event_mapping": {"web": ["Order Completed"]},
				"send_track_for_inapp": {"web": true},
				"animation_duration": {"web": "200"},
				"display_interval": {"web": "2500"},
				"on_open_screen_reader_message": {"web": "New message"},
				"on_open_node_to_take_focus": {"web": "#main"},
				"package_name": {"web": "com.example.app"},
				"right_offset": {"web": "15"},
				"top_offset": {"web": "11"},
				"bottom_offset": {"web": "24%"},
				"handle_links": {"web": "open-all-new-tab"},
				"close_button_color": {"web": "blue"},
				"close_button_size": {"web": "16"},
				"close_button_color_top_offset": {"web": "3%"},
				"close_button_color_side_offset": {"web": "2%"},
				"icon_path": {"web": "/icons/close.svg"},
				"is_required_to_dismiss_message": {"web": true},
				"close_button_position": {"web": "top-right"}
			}`,
			APIJSON: `{
				"apiKey": "iterable-api-key",
				"mapToSingleEvent": true,
				"trackAllPages": true,
				"trackCategorisedPages": true,
				"trackNamedPages": true,
				"useNativeSDK": {"web": true},
				"initialisationIdentifier": {"web": "email"},
				"getInAppEventMapping": {"web": [{"eventName": "Product Viewed"}, {"eventName": "Cart Updated"}]},
				"purchaseEventMapping": {"web": [{"eventName": "Order Completed"}]},
				"sendTrackForInapp": {"web": true},
				"animationDuration": {"web": "200"},
				"displayInterval": {"web": "2500"},
				"onOpenScreenReaderMessage": {"web": "New message"},
				"onOpenNodeToTakeFocus": {"web": "#main"},
				"packageName": {"web": "com.example.app"},
				"rightOffset": {"web": "15"},
				"topOffset": {"web": "11"},
				"bottomOffset": {"web": "24%"},
				"handleLinks": {"web": "open-all-new-tab"},
				"closeButtonColor": {"web": "blue"},
				"closeButtonSize": {"web": "16"},
				"closeButtonColorTopOffset": {"web": "3%"},
				"closeButtonColorSideOffset": {"web": "2%"},
				"iconPath": {"web": "/icons/close.svg"},
				"isRequiredToDismissMessage": {"web": true},
				"closeButtonPosition": {"web": "top-right"}
			}`,
		},
		{
			Name: "array reshape get in app mapping",
			LocalJSON: `{
				"api_key": "iterable-api-key",
				"get_in_app_event_mapping": {"web": ["one", "two"]}
			}`,
			APIJSON: `{
				"apiKey": "iterable-api-key",
				"getInAppEventMapping": {"web": [{"eventName": "one"}, {"eventName": "two"}]}
			}`,
		},
		{
			Name: "consent for web",
			LocalJSON: `{
				"api_key": "iterable-api-key",
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
				"apiKey": "iterable-api-key",
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
				"api_key": "iterable-api-key",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"apiKey": "iterable-api-key",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
