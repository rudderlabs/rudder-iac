package firebase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/firebase"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(firebase.NewDefinition()))

	registered, err := registry.Get("firebase", 1)
	require.NoError(t, err)

	assert.Equal(t, "firebase", registered.Type)
	assert.Equal(t, "FIREBASE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "unity", "react_native", "flutter",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"device"}, modes, "source type %s", sourceType)
	}

	assert.NotContains(t, registered.SupportedSourceTypes(), "web")
	assert.NotContains(t, registered.SupportedSourceTypes(), "cloud")
	assert.NotContains(t, registered.SupportedSourceTypes(), "cordova")
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")

	assert.Empty(t, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("FIREBASE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestFirebaseConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(firebase.NewDefinition()))
	registered, err := registry.Get("firebase", 1)
	require.NoError(t, err)

	t.Run("valid empty config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{}))
	})

	t.Run("valid full example config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"connection_mode": map[string]any{
				"android":        "device",
				"android_kotlin": "device",
				"ios":            "device",
				"ios_swift":      "device",
				"unity":          "device",
				"react_native":   "device",
				"flutter":        "device",
			},
			"use_native_sdk": map[string]any{
				"android":        true,
				"android_kotlin": true,
				"ios":            true,
				"ios_swift":      true,
				"unity":          true,
				"react_native":   true,
				"flutter":        true,
			},
			"event_filtering": map[string]any{
				"whitelist": []any{"Product Viewed", "Order Completed"},
			},
			"consent_management": map[string]any{
				"android_kotlin": []any{
					map[string]any{
						"provider": "oneTrust",
						"consents": []any{"analytics"},
					},
				},
				"react_native": []any{
					map[string]any{
						"provider":            "custom",
						"resolution_strategy": "and",
						"consents":            []any{"marketing"},
					},
				},
			},
			"one_trust_cookie_categories": map[string]any{
				"android": []any{
					map[string]any{"one_trust_cookie_category": "C0002"},
				},
			},
			"ketch_consent_purposes": map[string]any{
				"react_native": []any{
					map[string]any{"purpose": "analytics"},
				},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("event filtering lists are mutually exclusive", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"event_filtering": map[string]any{
				"whitelist": []any{"Order Completed"},
				"blacklist": []any{"Product Viewed"},
			},
		})
		require.NotEmpty(t, errors)
	})

	t.Run("event names enforce single line pattern", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"event_filtering": map[string]any{
				"whitelist": []any{"line one\nline two"},
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/event_filtering/whitelist/0", errors[0].Path)

		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"event_filtering": map[string]any{
				"whitelist": []any{"{{ eventName || Product Viewed }}"},
			},
		}))
	})

	t.Run("connection mode accepts only device", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"connection_mode": map[string]any{
				"android": "cloud",
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/connection_mode/android", errors[0].Path)
	})

	t.Run("connection mode rejects dynamic values", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"connection_mode": map[string]any{
				"android": "{{ .FIREBASE_CONNECTION_MODE }}",
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/connection_mode/android", errors[0].Path)
	})

	t.Run("unsupported source keys rejected under source-scoped config", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name string
			key  string
			path string
		}{
			{name: "connection mode", key: "connection_mode", path: "/connection_mode/web"},
			{name: "use native sdk", key: "use_native_sdk", path: "/use_native_sdk/web"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(map[string]any{
					tc.key: map[string]any{
						"web": true,
					},
				})
				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
				assert.Contains(t, errors[0].Message, "unknown config field")
			})
		}
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"not_a_field": true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent management source rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"consent_management": map[string]any{
				"web": []any{},
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'web' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"consent_management": map[string]any{
				"android": []any{
					map[string]any{"provider": "unknown"},
				},
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/android/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})

	t.Run("unsupported legacy consent source keys rejected", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name       string
			key        string
			sourceType string
			path       string
		}{
			{name: "one trust android kotlin", key: "one_trust_cookie_categories", sourceType: "android_kotlin", path: "/one_trust_cookie_categories/android_kotlin"},
			{name: "one trust ios swift", key: "one_trust_cookie_categories", sourceType: "ios_swift", path: "/one_trust_cookie_categories/ios_swift"},
			{name: "ketch android kotlin", key: "ketch_consent_purposes", sourceType: "android_kotlin", path: "/ketch_consent_purposes/android_kotlin"},
			{name: "ketch ios swift", key: "ketch_consent_purposes", sourceType: "ios_swift", path: "/ketch_consent_purposes/ios_swift"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(map[string]any{
					tc.key: map[string]any{
						tc.sourceType: []any{},
					},
				})
				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
				assert.Contains(t, errors[0].Message, "unknown config field")
			})
		}
	})

	t.Run("legacy consent strings enforce single line pattern", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name   string
			config map[string]any
			path   string
		}{
			{
				name: "one trust",
				config: map[string]any{
					"one_trust_cookie_categories": map[string]any{
						"android": []any{map[string]any{"one_trust_cookie_category": "line one\nline two"}},
					},
				},
				path: "/one_trust_cookie_categories/android/0/one_trust_cookie_category",
			},
			{
				name: "ketch",
				config: map[string]any{
					"ketch_consent_purposes": map[string]any{
						"flutter": []any{map[string]any{"purpose": "line one\nline two"}},
					},
				},
				path: "/ketch_consent_purposes/flutter/0/purpose",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(tc.config)
				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})
}

func TestFirebaseConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := firebase.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name:      "empty config",
			LocalJSON: `{}`,
			APIJSON:   `{}`,
		},
		{
			Name: "event filtering whitelist",
			LocalJSON: `{
				"event_filtering": {
					"whitelist": ["Product Viewed", "Order Completed"]
				}
			}`,
			APIJSON: `{
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
				"event_filtering": {
					"blacklist": ["Screen Viewed"]
				}
			}`,
			APIJSON: `{
				"blacklistedEvents": [
					{"eventName": "Screen Viewed"}
				],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "connection mode source mapping",
			LocalJSON: `{
				"connection_mode": {
					"android": "device",
					"android_kotlin": "device",
					"ios": "device",
					"ios_swift": "device",
					"unity": "device",
					"react_native": "device",
					"flutter": "device"
				}
			}`,
			APIJSON: `{
				"connectionMode": {
					"android": "device",
					"androidKotlin": "device",
					"ios": "device",
					"iosSwift": "device",
					"unity": "device",
					"reactnative": "device",
					"flutter": "device"
				}
			}`,
		},
		{
			Name: "use native sdk source mapping",
			LocalJSON: `{
				"use_native_sdk": {
					"android": true,
					"android_kotlin": false,
					"ios": true,
					"ios_swift": false,
					"unity": true,
					"react_native": true,
					"flutter": false
				}
			}`,
			APIJSON: `{
				"useNativeSDK": {
					"android": true,
					"androidKotlin": false,
					"ios": true,
					"iosSwift": false,
					"unity": true,
					"reactnative": true,
					"flutter": false
				}
			}`,
		},
		{
			Name: "consent management source boundary mappings",
			LocalJSON: `{
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
		{
			Name: "one trust cookie categories source mapping",
			LocalJSON: `{
				"one_trust_cookie_categories": {
					"android": [{"one_trust_cookie_category": "C0001"}],
					"ios": [{"one_trust_cookie_category": "C0002"}],
					"unity": [{"one_trust_cookie_category": "C0003"}],
					"react_native": [{"one_trust_cookie_category": "C0004"}],
					"flutter": [{"one_trust_cookie_category": "C0005"}]
				}
			}`,
			APIJSON: `{
				"oneTrustCookieCategories": {
					"android": [{"oneTrustCookieCategory": "C0001"}],
					"ios": [{"oneTrustCookieCategory": "C0002"}],
					"unity": [{"oneTrustCookieCategory": "C0003"}],
					"reactnative": [{"oneTrustCookieCategory": "C0004"}],
					"flutter": [{"oneTrustCookieCategory": "C0005"}]
				}
			}`,
		},
		{
			Name: "ketch consent purposes source mapping",
			LocalJSON: `{
				"ketch_consent_purposes": {
					"android": [{"purpose": "analytics"}],
					"ios": [{"purpose": "marketing"}],
					"unity": [{"purpose": "functional"}],
					"react_native": [{"purpose": "personalization"}],
					"flutter": [{"purpose": "advertising"}]
				}
			}`,
			APIJSON: `{
				"ketchConsentPurposes": {
					"android": [{"purpose": "analytics"}],
					"ios": [{"purpose": "marketing"}],
					"unity": [{"purpose": "functional"}],
					"reactnative": [{"purpose": "personalization"}],
					"flutter": [{"purpose": "advertising"}]
				}
			}`,
		},
	})
}
