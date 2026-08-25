package ga4_test

import (
	"maps"
	"strings"
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
		"capture_page_view/web":               {"web"},
		"debug_view/web":                      {"web"},
		"override_client_and_session_ids/web": {"web"},
		"extend_page_view_params/web":         {"web"},
		"use_native_sdk_to_send/web":          {"web"},
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

	// schema.json requires the G- prefix on the gtag measurement ID; the branch
	// previously enforced only max=100, which accepted any string.
	t.Run("measurement_id must carry the G- prefix", func(t *testing.T) {
		t.Parallel()

		for _, id := range []string{"XXXXXXXXXX", "g-lowercase", "G" + strings.Repeat("x", 120)} {
			errors := registered.ValidateConfig(map[string]any{
				"api_secret":     "secret",
				"client_type":    "gtag",
				"measurement_id": id,
			})
			require.NotEmpty(t, errors, id)
			assert.Equal(t, "/measurement_id", errors[0].Path)
		}

		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"api_secret":     "secret",
			"client_type":    "gtag",
			"measurement_id": "G-XXXXXXXXXX",
		}))
	})

	// schema.json constrains sdkBaseUrl under typesOfClient=gtag >
	// connectionMode.web=device; the branch left it entirely unvalidated.
	// schema.json only constrains sdk_base_url's format under
	// client_type=gtag AND connection_mode.web=device — see
	// ga4SDKBaseURLConditional, which now that connection_mode is real config
	// (DEX-708) reads both siblings instead of applying the pattern always.
	t.Run("sdk_base_url pattern enforced when gtag and device", func(t *testing.T) {
		t.Parallel()

		for _, url := range []string{"nodots", "https://foo.ngrok.io", "foo.ngrok.io/gtm"} {
			errors := registered.ValidateConfig(map[string]any{
				"api_secret":      "secret",
				"client_type":     "gtag",
				"measurement_id":  "G-XXXXXXXXXX",
				"connection_mode": map[string]any{"web": "device"},
				"sdk_base_url":    url,
			})
			require.NotEmpty(t, errors, url)
			assert.Equal(t, "/sdk_base_url", errors[0].Path)
		}

		for _, url := range []string{"https://www.googletagmanager.com", "www.googletagmanager.com", ""} {
			assert.Empty(t, registered.ValidateConfig(map[string]any{
				"api_secret":      "secret",
				"client_type":     "gtag",
				"measurement_id":  "G-XXXXXXXXXX",
				"connection_mode": map[string]any{"web": "device"},
				"sdk_base_url":    url,
			}), url)
		}
	})

	t.Run("sdk_base_url unconstrained outside gtag+device", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name   string
			config map[string]any
		}{
			{
				name: "connection_mode.web not device",
				config: map[string]any{
					"connection_mode": map[string]any{"web": "cloud"},
				},
			},
			{
				name:   "connection_mode not set at all",
				config: map[string]any{},
			},
		}
		for _, tc := range cases {
			config := map[string]any{
				"api_secret":     "secret",
				"client_type":    "gtag",
				"measurement_id": "G-XXXXXXXXXX",
				"sdk_base_url":   "not a domain url with spaces!!",
			}
			maps.Copy(config, tc.config)
			assert.Empty(t, registered.ValidateConfig(config), tc.name)
		}

		// client_type=firebase never satisfies the gtag half of the condition.
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"api_secret":      "secret",
			"client_type":     "firebase",
			"firebase_app_id": "1:123:android:abc",
			"connection_mode": map[string]any{"web": "device"},
			"sdk_base_url":    "not a domain url with spaces!!",
		}))
	})

	// Terraform maps blockPageViewEvent and sendUserId, but neither appears in
	// schema.json or db-config defaultConfig, so they are not part of the surface.
	t.Run("terraform only keys rejected", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"block_page_view_event", "send_user_id"} {
			errors := registered.ValidateConfig(map[string]any{
				"api_secret":     "secret",
				"client_type":    "gtag",
				"measurement_id": "G-XXXXXXXXXX",
				key:              true,
			})
			require.NotEmpty(t, errors, key)
			assert.Equal(t, "/"+key, errors[0].Path)
			assert.Contains(t, errors[0].Message, "unknown config field")
		}
	})

	t.Run("event filtering lists are mutually exclusive", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"api_secret":     "secret",
			"client_type":    "gtag",
			"measurement_id": "G-XXXXXXXXXX",
			"event_filtering": map[string]any{
				"whitelist": []any{"Order Completed"},
				"blacklist": []any{"Page Viewed"},
			},
		})
		require.NotEmpty(t, errors)
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
			"connection_mode": map[string]any{
				"web":     "hybrid",
				"android": "device",
				"unity":   "cloud",
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
			"extend_page_view_params": map[string]any{
				"web": true,
			},
			"use_native_sdk_to_send": map[string]any{
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

	t.Run("connection_mode value validated per source type", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":     "secret",
			"client_type":    "gtag",
			"measurement_id": "G-XXXXXXXXXX",
			"connection_mode": map[string]any{
				"web":   "hybrid",
				"unity": "hybrid", // unity only allows cloud
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/unity", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("connection_mode accepts a template", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":     "secret",
			"client_type":    "gtag",
			"measurement_id": "G-XXXXXXXXXX",
			"connection_mode": map[string]any{
				"unity": "{{ .GA4_CONNECTION_MODE || cloud }}",
			},
		})
		assert.Empty(t, errors)
	})

	// The framework's generic source-type-scoped-key check (not this
	// definition's own validation) is what catches this — connection_mode was
	// reserved for it before any destination modeled a config field for it.
	// See rule_spec_syntax_valid_test.go's coverage of this path.
	t.Run("connection_mode for an unsupported source type does not error here", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_secret":     "secret",
			"client_type":    "gtag",
			"measurement_id": "G-XXXXXXXXXX",
			"connection_mode": map[string]any{
				"warehouse": "cloud",
			},
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
				"connection_mode": {
					"web": "hybrid",
					"android": "device"
				},
				"capture_page_view": {"web": "rs"},
				"debug_view": {"web": true},
				"override_client_and_session_ids": {"web": true},
				"extend_page_view_params": {"web": true},
				"use_native_sdk_to_send": {"web": false}
			}`,
			APIJSON: `{
				"apiSecret": "secret",
				"typesOfClient": "gtag",
				"measurementId": "G-XXXXXXXXXX",
				"firebaseAppId": "1:123:android:abc",
				"debugMode": true,
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
				"connectionMode": {
					"web": "hybrid",
					"android": "device"
				},
				"capturePageView": {"web": "rs"},
				"debugView": {"web": true},
				"overrideClientAndSessionId": {"web": true},
				"extendPageViewParams": {"web": true},
				"useNativeSDKToSend": {"web": false}
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
