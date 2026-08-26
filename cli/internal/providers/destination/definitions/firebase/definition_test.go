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

	t.Run("connection mode rejected as config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"connection_mode": map[string]any{
				"android": "device",
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/connection_mode", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported source keys rejected under source-scoped config", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name string
			key  string
			path string
		}{
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

	// The backend migrates these into consentManagement and never returns them,
	// so modelling them would make every plan diff.
	t.Run("legacy consent keys rejected as config", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			errors := registered.ValidateConfig(map[string]any{
				key: map[string]any{"android": []any{"C0001"}},
			})
			require.NotEmpty(t, errors)
			assert.Equal(t, "/"+key, errors[0].Path)
			assert.Contains(t, errors[0].Message, "unknown config field")
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
	})
}
