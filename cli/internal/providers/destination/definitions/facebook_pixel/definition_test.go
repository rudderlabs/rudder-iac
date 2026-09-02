package facebookpixel_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	facebookpixel "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/facebook_pixel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(facebookpixel.NewDefinition()))

	registered, err := registry.Get("facebook_pixel", 1)
	require.NoError(t, err)

	assert.Equal(t, "facebook_pixel", registered.Type)
	assert.Equal(t, "FACEBOOK_PIXEL", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"access_token"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "cloud", "react_native", "flutter", "cordova",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	assert.NotContains(t, registered.SupportedSourceTypes(), "amp")
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")

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

	assert.Equal(t, map[string][]string{
		"auto_config/web":            {"web"},
		"legacy_conversion_pixel_id": {"web"},
	}, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("FACEBOOK_PIXEL", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestFacebookPixelConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(facebookpixel.NewDefinition()))
	registered, err := registry.Get("facebook_pixel", 1)
	require.NoError(t, err)

	t.Run("missing pixel_id", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"access_token": "access-token-1",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/pixel_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("invalid value_field_identifier", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"pixel_id":               "pixel-1",
			"access_token":           "fbAccessToken",
			"value_field_identifier": "properties.revenue",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/value_field_identifier", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	// schema.json declares a plain enum with no {{ … || … }} branch, so a
	// templated value would be stored verbatim and rejected by the backend.
	t.Run("value_field_identifier rejects dynamic values", func(t *testing.T) {
		t.Parallel()
		for _, value := range []string{
			"{{ .FACEBOOK_VALUE_FIELD_IDENTIFIER }}",
			`{{ .FACEBOOK_VALUE_FIELD_IDENTIFIER || properties.value }}`,
			"env.FACEBOOK_VALUE_FIELD_IDENTIFIER",
		} {
			errors := registered.ValidateConfig(map[string]any{
				"pixel_id":               "pixel-1",
				"access_token":           "fbAccessToken",
				"value_field_identifier": value,
			})
			require.NotEmpty(t, errors, value)
			assert.Equal(t, "/value_field_identifier", errors[0].Path)
		}
	})

	t.Run("invalid mapped event target", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"pixel_id":     "pixel-1",
			"access_token": "fbAccessToken",
			"events_to_events": []any{
				map[string]any{"from": "Signed Up", "to": "InvalidEvent"},
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/events_to_events/0/to", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("event filtering rejects whitelist and blacklist together", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"pixel_id":     "pixel-1",
			"access_token": "fbAccessToken",
			"event_filtering": map[string]any{
				"whitelist": []any{"Product Viewed"},
				"blacklist": []any{"Order Completed"},
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/event_filtering/whitelist", errors[0].Path)
		assert.Contains(t, errors[0].Message, "cannot be specified together")
	})

	t.Run("source type scoped config rejects unsupported source keys", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name   string
			key    string
			config map[string]any
			path   string
		}{
			{
				name: "use_native_sdk",
				key:  "use_native_sdk",
				config: map[string]any{
					"android": true,
				},
				path: "/use_native_sdk/android",
			},
			{
				name: "auto_config",
				key:  "auto_config",
				config: map[string]any{
					"ios": true,
				},
				path: "/auto_config/ios",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config[tc.key] = tc.config

				errors := registered.ValidateConfig(config)
				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
				assert.Contains(t, errors[0].Message, "unknown config field")
			})
		}
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

	t.Run("required string rejects empty value", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"pixel_id": "",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/pixel_id", errors[0].Path)
	})

	t.Run("string fields reject values over maximum length", func(t *testing.T) {
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

	// The upstream constraints are patterns, not length limits, so they reject line breaks.
	t.Run("string fields reject line breaks", func(t *testing.T) {
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

	// Upstream's template branch carries no length cap, so a template longer than
	// the literal limit is still valid.
	t.Run("string fields accept ui templates of any length", func(t *testing.T) {
		t.Parallel()

		long := "{{ config.value || " + strings.Repeat("x", 200) + " }}"
		for _, tc := range patternFieldCases(long) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				assert.Empty(t, registered.ValidateConfig(tc.config))
			})
		}
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(validMinimalConfig())
		assert.Empty(t, errors)
	})

	t.Run("valid full cloud config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(validFullConfig())
		assert.Empty(t, errors)
	})

	t.Run("valid web device config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(validWebDeviceConfig())
		assert.Empty(t, errors)
	})

	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(exampleYAMLConfig())
		assert.Empty(t, errors)
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["not_a_field"] = true

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("legacy consent include key blocks rejected", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			t.Run(key, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config[key] = map[string]any{
					"web": []any{map[string]any{"consent": "marketing"}},
				}

				errors := registered.ValidateConfig(config)
				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+key, errors[0].Path)
				assert.Contains(t, errors[0].Message, "unknown config field")
			})
		}
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
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

		config := validMinimalConfig()
		config["consent_management"] = map[string]any{
			"web": []any{
				map[string]any{"provider": "unknown"},
			},
		}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

// Both of these empty values are deliberately accepted, and a review flagged
// them as gaps, so pin the reasoning:
//   - eventsToEvents[].to lists "" among its schema enum members, and omitempty
//     lets it through before the enum tag runs;
//   - accessToken's ^(.{1,300})$ lower bound lives in an allOf branch gated on
//     connectionMode, which the CLI does not model. Enforcing a non-empty
//     minimum would reject configs whose branch does not apply.
func TestFacebookPixelEmptyOptionalValuesAccepted(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(facebookpixel.NewDefinition()))
	registered, err := registry.Get("facebook_pixel", 1)
	require.NoError(t, err)

	assert.Empty(t, registered.ValidateConfig(map[string]any{
		"pixel_id":         "1234567890",
		"access_token":     "fbAccessToken",
		"events_to_events": []any{map[string]any{"from": "Order Completed", "to": ""}},
	}), `eventsToEvents[].to accepts "" — it is a schema enum member`)

	// schema.json requires accessToken unless connection_mode.web is device, so an
	// empty token is accepted only in that one case.
	assert.Empty(t, registered.ValidateConfig(map[string]any{
		"pixel_id":        "1234567890",
		"access_token":    "",
		"connection_mode": map[string]any{"web": "device"},
	}), "device-mode web needs no access_token")

	assert.NotEmpty(t, registered.ValidateConfig(map[string]any{
		"pixel_id":     "1234567890",
		"access_token": "",
	}), "without a device-mode web source, access_token is required")
}

// schema.json bounds accessToken with ^(.{1,300})$ inside the allOf branch — a
// wider limit than the usual single_line_100, and previously unenforced.
func TestFacebookPixelAccessTokenPattern(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(facebookpixel.NewDefinition()))
	registered, err := registry.Get("facebook_pixel", 1)
	require.NoError(t, err)

	base := map[string]any{"pixel_id": "1234567890", "access_token": "fbAccessToken"}

	// 300 characters is allowed; 301 is not, and line breaks never are.
	ok := base
	ok["access_token"] = strings.Repeat("a", 300)
	assert.Empty(t, registered.ValidateConfig(ok))

	for _, bad := range []string{strings.Repeat("a", 301), "bad\ntoken"} {
		cfg := map[string]any{"pixel_id": "1234567890", "access_token": bad}
		errs := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errs)
		assert.Equal(t, "/access_token", errs[0].Path)
	}
}

func TestFacebookPixelConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := facebookpixel.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"pixel_id": "pixel-1"
			}`,
			APIJSON: `{
				"pixelId": "pixel-1"
			}`,
		},
		{
			Name: "full cloud",
			LocalJSON: `{
				"pixel_id": "pixel-1",
				"access_token": "access-token-1",
				"standard_page_call": true,
				"value_field_identifier": "properties.value",
				"advanced_mapping": true,
				"limited_data_usage": true,
				"test_destination": true,
				"test_event_code": "TEST12345",
				"remove_external_id": true,
				"use_updated_mapping": true,
				"events_to_events": [
					{"from": "Product Viewed", "to": "ViewContent"},
					{"from": "Order Completed", "to": "Purchase"}
				],
				"blacklist_pii_properties": [
					{"property": "email", "hash": true},
					{"property": "phone", "hash": false}
				],
				"whitelist_pii_properties": [
					{"property": "country"}
				]
			}`,
			APIJSON: `{
				"pixelId": "pixel-1",
				"accessToken": "access-token-1",
				"standardPageCall": true,
				"valueFieldIdentifier": "properties.value",
				"advancedMapping": true,
				"limitedDataUSage": true,
				"testDestination": true,
				"testEventCode": "TEST12345",
				"removeExternalId": true,
				"useUpdatedMapping": true,
				"eventsToEvents": [
					{"from": "Product Viewed", "to": "ViewContent"},
					{"from": "Order Completed", "to": "Purchase"}
				],
				"blacklistPiiProperties": [
					{"blacklistPiiProperties": "email", "blacklistPiiHash": true},
					{"blacklistPiiProperties": "phone", "blacklistPiiHash": false}
				],
				"whitelistPiiProperties": [
					{"whitelistPiiProperties": "country"}
				]
			}`,
		},
		{
			Name: "web device settings",
			LocalJSON: `{
				"pixel_id": "pixel-1",
				"use_native_sdk": {"web": true},
				"auto_config": {"web": false},
				"legacy_conversion_pixel_id": [
					{"from": "Signup", "to": "1234567890"},
					{"from": "Purchase", "to": "0987654321"}
				]
			}`,
			APIJSON: `{
				"pixelId": "pixel-1",
				"useNativeSDK": {"web": true},
				"autoConfig": {"web": false},
				"legacyConversionPixelId": {
					"web": [
						{"from": "Signup", "to": "1234567890"},
						{"from": "Purchase", "to": "0987654321"}
					]
				}
			}`,
		},
		{
			Name: "event filtering whitelist",
			LocalJSON: `{
				"pixel_id": "pixel-1",
				"event_filtering": {
					"whitelist": ["Product Viewed", "Order Completed"]
				}
			}`,
			APIJSON: `{
				"pixelId": "pixel-1",
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
				"pixel_id": "pixel-1",
				"event_filtering": {
					"blacklist": ["Page Viewed"]
				}
			}`,
			APIJSON: `{
				"pixelId": "pixel-1",
				"blacklistedEvents": [
					{"eventName": "Page Viewed"}
				],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"pixel_id": "pixel-1",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"react_native": [{"provider": "iubenda"}],
					"cloud": [{"provider": "ketch"}]
				}
			}`,
			APIJSON: `{
				"pixelId": "pixel-1",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"reactnative": [{"provider": "iubenda"}],
					"cloud": [{"provider": "ketch"}]
				}
			}`,
		},
	})
}

// patternFieldCase is one pattern-validated string field, with a config that is
// otherwise valid so the only error can come from the field under test.
type patternFieldCase struct {
	name   string
	path   string
	config map[string]any
}

// patternFieldCases returns one case per pattern-validated string field, each
// carrying value in that field.
func patternFieldCases(value string) []patternFieldCase {
	return []patternFieldCase{
		{
			name: "pixel_id",
			path: "/pixel_id",
			config: map[string]any{
				"pixel_id":     value,
				"access_token": "fbAccessToken",
			},
		},
		{
			name: "test_event_code",
			path: "/test_event_code",
			config: map[string]any{
				"pixel_id":        "pixel-1",
				"access_token":    "fbAccessToken",
				"test_event_code": value,
			},
		},
		{
			name: "event from",
			path: "/events_to_events/0/from",
			config: map[string]any{
				"pixel_id":     "pixel-1",
				"access_token": "fbAccessToken",
				"events_to_events": []any{
					map[string]any{"from": value, "to": "Purchase"},
				},
			},
		},
		{
			name: "denylist property",
			path: "/blacklist_pii_properties/0/property",
			config: map[string]any{
				"pixel_id":     "pixel-1",
				"access_token": "fbAccessToken",
				"blacklist_pii_properties": []any{
					map[string]any{"property": value, "hash": true},
				},
			},
		},
		{
			name: "allowlist property",
			path: "/whitelist_pii_properties/0/property",
			config: map[string]any{
				"pixel_id":     "pixel-1",
				"access_token": "fbAccessToken",
				"whitelist_pii_properties": []any{
					map[string]any{"property": value},
				},
			},
		},
		{
			name: "event filtering whitelist",
			path: "/event_filtering/whitelist/0",
			config: map[string]any{
				"pixel_id":     "pixel-1",
				"access_token": "fbAccessToken",
				"event_filtering": map[string]any{
					"whitelist": []any{value},
				},
			},
		},
		{
			name: "event filtering blacklist",
			path: "/event_filtering/blacklist/0",
			config: map[string]any{
				"pixel_id":     "pixel-1",
				"access_token": "fbAccessToken",
				"event_filtering": map[string]any{
					"blacklist": []any{value},
				},
			},
		},
		{
			name: "legacy conversion event",
			path: "/legacy_conversion_pixel_id/0/from",
			config: map[string]any{
				"pixel_id":     "pixel-1",
				"access_token": "fbAccessToken",
				"legacy_conversion_pixel_id": []any{
					map[string]any{"from": value, "to": "1234567890"},
				},
			},
		},
		{
			name: "legacy conversion pixel id",
			path: "/legacy_conversion_pixel_id/0/to",
			config: map[string]any{
				"pixel_id":     "pixel-1",
				"access_token": "fbAccessToken",
				"legacy_conversion_pixel_id": []any{
					map[string]any{"from": "Signup", "to": value},
				},
			},
		},
	}
}

func validMinimalConfig() map[string]any {
	return map[string]any{
		"pixel_id":     "pixel-1",
		"access_token": "fbAccessToken",
	}
}

func validFullConfig() map[string]any {
	return map[string]any{
		"pixel_id":               "pixel-1",
		"access_token":           "access-token-1",
		"standard_page_call":     true,
		"value_field_identifier": "properties.value",
		"advanced_mapping":       true,
		"limited_data_usage":     true,
		"test_destination":       true,
		"test_event_code":        "TEST12345",
		"remove_external_id":     true,
		"use_updated_mapping":    true,
		"events_to_events": []any{
			map[string]any{"from": "Product Viewed", "to": "ViewContent"},
			map[string]any{"from": "Order Completed", "to": "Purchase"},
		},
		"blacklist_pii_properties": []any{
			map[string]any{"property": "email", "hash": true},
			map[string]any{"property": "phone", "hash": false},
		},
		"whitelist_pii_properties": []any{
			map[string]any{"property": "country"},
		},
		"event_filtering": map[string]any{
			"whitelist": []any{"Product Viewed", "Order Completed"},
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{"marketing"},
				},
			},
		},
	}
}

func validWebDeviceConfig() map[string]any {
	return map[string]any{
		"pixel_id":     "pixel-1",
		"access_token": "fbAccessToken",
		"use_native_sdk": map[string]any{
			"web": true,
		},
		"auto_config": map[string]any{
			"web": false,
		},
		"legacy_conversion_pixel_id": []any{
			map[string]any{"from": "Signup", "to": "1234567890"},
			map[string]any{"from": "Purchase", "to": "0987654321"},
		},
	}
}

func exampleYAMLConfig() map[string]any {
	return map[string]any{
		"pixel_id":               "123456789012345",
		"access_token":           "{{ .FACEBOOK_PIXEL_ACCESS_TOKEN }}",
		"standard_page_call":     true,
		"value_field_identifier": "properties.price",
		"advanced_mapping":       true,
		"limited_data_usage":     false,
		"test_destination":       true,
		"test_event_code":        "TEST12345",
		"remove_external_id":     false,
		"use_updated_mapping":    true,
		"events_to_events": []any{
			map[string]any{"from": "Product Viewed", "to": "ViewContent"},
			map[string]any{"from": "Order Completed", "to": "Purchase"},
		},
		"blacklist_pii_properties": []any{
			map[string]any{"property": "email", "hash": true},
		},
		"whitelist_pii_properties": []any{
			map[string]any{"property": "phone"},
		},
		"event_filtering": map[string]any{
			"whitelist": []any{"Product Viewed", "Order Completed"},
		},
		"use_native_sdk": map[string]any{
			"web": true,
		},
		"auto_config": map[string]any{
			"web": false,
		},
		"legacy_conversion_pixel_id": []any{
			map[string]any{"from": "Signup", "to": "123456789012345"},
		},
		"consent_management": map[string]any{
			"android_kotlin": []any{
				map[string]any{
					"provider":            "custom",
					"resolution_strategy": "and",
					"consents":            []any{"marketing"},
				},
			},
		},
	}
}
