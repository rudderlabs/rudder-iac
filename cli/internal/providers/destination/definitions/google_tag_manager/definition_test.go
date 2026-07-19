package googletagmanager_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	googletagmanager "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/google_tag_manager"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googletagmanager.NewDefinition()))

	registered, err := registry.Get("google_tag_manager", 1)
	require.NoError(t, err)

	assert.Equal(t, "google_tag_manager", registered.Type)
	assert.Equal(t, "GTM", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{}, registered.SecretKeys())

	expectedSourceTypes := []string{"web"}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	modes, err := registered.ConnectionModes("web")
	require.NoError(t, err)
	assert.Equal(t, []string{"device"}, modes)

	assert.Empty(t, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("GTM", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestGoogleTagManagerConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googletagmanager.NewDefinition()))
	registered, err := registry.Get("google_tag_manager", 1)
	require.NoError(t, err)

	t.Run("missing container_id", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/container_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"container_id": "GTM-XXXXXXX",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"container_id": "GTM-XXXXXXX",
			"server_url":   "https://gtm.example.com",
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
			"container_id": "GTM-ABC1234",
			"server_url":   "https://gtm.example.com",
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

	t.Run("invalid server_url rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"container_id": "GTM-XXXXXXX",
			"server_url":   "not-a-url",
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/server_url", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be a valid URL")
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"container_id": "GTM-XXXXXXX",
			"not_a_field":  true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"container_id": "GTM-XXXXXXX",
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
			"container_id": "GTM-XXXXXXX",
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

func TestGoogleTagManagerConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := googletagmanager.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal container only",
			LocalJSON: `{
				"container_id": "GTM-XXXXXXX"
			}`,
			APIJSON: `{
				"containerID": "GTM-XXXXXXX"
			}`,
		},
		{
			Name: "full TF fields with whitelist",
			LocalJSON: `{
				"container_id": "GTM-XXXXXXX",
				"server_url": "https://gtm.example.com",
				"event_filtering": {
					"whitelist": ["Product Viewed", "Order Completed"]
				},
				"use_native_sdk": {
					"web": true
				}
			}`,
			APIJSON: `{
				"containerID": "GTM-XXXXXXX",
				"serverUrl": "https://gtm.example.com",
				"whitelistedEvents": [
					{"eventName": "Product Viewed"},
					{"eventName": "Order Completed"}
				],
				"eventFilteringOption": "whitelistedEvents",
				"useNativeSDK": {
					"web": true
				}
			}`,
		},
		{
			Name: "event filtering blacklist reshape",
			LocalJSON: `{
				"container_id": "GTM-XXXXXXX",
				"event_filtering": {
					"blacklist": ["Application Opened"]
				}
			}`,
			APIJSON: `{
				"containerID": "GTM-XXXXXXX",
				"blacklistedEvents": [
					{"eventName": "Application Opened"}
				],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "consent for web",
			LocalJSON: `{
				"container_id": "GTM-XXXXXXX",
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
				"containerID": "GTM-XXXXXXX",
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
