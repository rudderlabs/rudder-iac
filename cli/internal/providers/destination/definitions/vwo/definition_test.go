package vwo_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/vwo"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(vwo.NewDefinition()))

	registered, err := registry.Get("vwo", 1)
	require.NoError(t, err)

	assert.Equal(t, "vwo", registered.Type)
	assert.Equal(t, "VWO", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys())

	expectedSourceTypes := []string{"web"}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	modes, err := registered.ConnectionModes("web")
	require.NoError(t, err)
	assert.Equal(t, []string{"device"}, modes)

	assert.Empty(t, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("VWO", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestVWOConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(vwo.NewDefinition()))
	registered, err := registry.Get("vwo", 1)
	require.NoError(t, err)

	t.Run("missing account_id", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/account_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id": "410057",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id":               "410057",
			"is_spa":                   true,
			"send_experiment_track":    true,
			"send_experiment_identify": false,
			"library_tolerance":        "2500",
			"settings_tolerance":       "2000",
			"use_existing_jquery":      true,
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
			"account_id":               "410057",
			"is_spa":                   true,
			"send_experiment_track":    true,
			"send_experiment_identify": true,
			"library_tolerance":        "2500",
			"settings_tolerance":       "2000",
			"use_existing_jquery":      true,
			"event_filtering": map[string]any{
				"whitelist": []any{"Product Viewed", "Order Completed"},
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

	t.Run("account_id rejects line break", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id": "410\n057",
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/account_id", errors[0].Path)
	})

	t.Run("account_id rejects overlength literal", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id": strings.Repeat("1", 101),
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/account_id", errors[0].Path)
	})

	t.Run("account_id accepts template", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id": "{{ .vwo.accountID || \"410057\" }}",
		})
		assert.Empty(t, errors)
	})

	t.Run("library_tolerance rejects line break", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id":        "410057",
			"library_tolerance": "25\n00",
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/library_tolerance", errors[0].Path)
	})

	t.Run("library_tolerance accepts template", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id":        "410057",
			"library_tolerance": "{{ .vwo.libraryTolerance || \"2500\" }}",
		})
		assert.Empty(t, errors)
	})

	t.Run("settings_tolerance rejects overlength literal", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id":         "410057",
			"settings_tolerance": strings.Repeat("2", 101),
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/settings_tolerance", errors[0].Path)
	})

	t.Run("settings_tolerance accepts template", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id":         "410057",
			"settings_tolerance": "{{ .vwo.settingsTolerance || \"2000\" }}",
		})
		assert.Empty(t, errors)
	})

	t.Run("event filtering entry rejects line break", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id": "410057",
			"event_filtering": map[string]any{
				"whitelist": []any{"Product\nViewed"},
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/event_filtering/whitelist/0", errors[0].Path)
	})

	t.Run("event filtering entry rejects overlength literal", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id": "410057",
			"event_filtering": map[string]any{
				"blacklist": []any{strings.Repeat("x", 101)},
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/event_filtering/blacklist/0", errors[0].Path)
	})

	t.Run("event filtering entry accepts template", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id": "410057",
			"event_filtering": map[string]any{
				"whitelist": []any{"{{ .events.productViewed || \"Product Viewed\" }}"},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("whitelist and blacklist are mutually exclusive", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id": "410057",
			"event_filtering": map[string]any{
				"whitelist": []any{"Product Viewed"},
				"blacklist": []any{"Application Opened"},
			},
		})
		require.NotEmpty(t, errors)
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id":  "410057",
			"not_a_field": true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("legacy consent key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id": "410057",
			"one_trust_cookie_categories": map[string]any{
				"web": []any{"analytics"},
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/one_trust_cookie_categories", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"account_id": "410057",
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
			"account_id": "410057",
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

func TestVWOConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := vwo.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal account only",
			LocalJSON: `{
				"account_id": "410057"
			}`,
			APIJSON: `{
				"accountId": "410057"
			}`,
		},
		{
			Name: "full config with whitelist",
			LocalJSON: `{
				"account_id": "410057",
				"is_spa": true,
				"send_experiment_track": true,
				"send_experiment_identify": false,
				"library_tolerance": "2500",
				"settings_tolerance": "2000",
				"use_existing_jquery": true,
				"event_filtering": {
					"whitelist": ["Product Viewed", "Order Completed"]
				},
				"use_native_sdk": {
					"web": true
				}
			}`,
			APIJSON: `{
				"accountId": "410057",
				"isSPA": true,
				"sendExperimentTrack": true,
				"sendExperimentIdentify": false,
				"libraryTolerance": "2500",
				"settingsTolerance": "2000",
				"useExistingJquery": true,
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
				"account_id": "410057",
				"event_filtering": {
					"blacklist": ["Application Opened"]
				}
			}`,
			APIJSON: `{
				"accountId": "410057",
				"blacklistedEvents": [
					{"eventName": "Application Opened"}
				],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "use native sdk web",
			LocalJSON: `{
				"account_id": "410057",
				"use_native_sdk": {
					"web": false
				}
			}`,
			APIJSON: `{
				"accountId": "410057",
				"useNativeSDK": {
					"web": false
				}
			}`,
		},
		{
			Name: "consent for web",
			LocalJSON: `{
				"account_id": "410057",
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
				"accountId": "410057",
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
