package adjust_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/adjust"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(adjust.NewDefinition()))

	registered, err := registry.Get("adjust", 1)
	require.NoError(t, err)

	assert.Equal(t, "adjust", registered.Type)
	assert.Equal(t, "ADJ", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"app_token"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift",
		"unity", "react_native", "flutter", "cordova", "cloud",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	expectedModes := map[string][]string{
		"android":        {"cloud", "device"},
		"android_kotlin": {"cloud", "device"},
		"ios":            {"cloud", "device"},
		"ios_swift":      {"cloud", "device"},
		"unity":          {"cloud", "device"},
		"react_native":   {"cloud"},
		"flutter":        {"cloud", "device"},
		"cordova":        {"cloud"},
		"cloud":          {"cloud"},
	}
	for sourceType, want := range expectedModes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, want, modes, sourceType)
	}

	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")

	assert.Equal(t, map[string][]string{
		"enable_install_attribution_tracking/android": {"android", "android_kotlin"},
		"enable_install_attribution_tracking/ios":     {"ios", "ios_swift"},
	}, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("ADJ", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestAdjustConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(adjust.NewDefinition()))
	registered, err := registry.Get("adjust", 1)
	require.NoError(t, err)

	t.Run("missing app_token", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/app_token", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"app_token": "abc123",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"app_token":   "abc123",
			"delay":       "5",
			"environment": true,
			"custom_mappings": []any{
				map[string]any{"from": "Product Purchased", "to": "tok1"},
			},
			"partner_param_keys": []any{
				map[string]any{"from": "userId", "to": "user_id"},
			},
			"enable_install_attribution_tracking": map[string]any{
				"android": true,
				"ios":     true,
			},
			"event_filtering_whitelist": []any{"Purchase", "Signup"},
			"consent_management": map[string]any{
				"android": []any{
					map[string]any{
						"provider": "oneTrust",
						"consents": []any{"analytics"},
					},
				},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("example yaml config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"app_token":   "YOUR_ADJUST_APP_TOKEN",
			"delay":       "5",
			"environment": true,
			"custom_mappings": []any{
				map[string]any{"from": "Product Purchased", "to": "abc123"},
				map[string]any{"from": "Signup", "to": "def456"},
			},
			"partner_param_keys": []any{
				map[string]any{"from": "userId", "to": "user_id"},
			},
			"enable_install_attribution_tracking": map[string]any{
				"android": true,
				"ios":     true,
			},
			"event_filtering_whitelist": []any{"Product Purchased", "Signup"},
			"consent_management": map[string]any{
				"android": []any{
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
			"app_token":   "abc123",
			"not_a_field": true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"app_token": "abc123",
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
			"app_token": "abc123",
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

func TestAdjustConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := adjust.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal app token",
			LocalJSON: `{
				"app_token": "abc123"
			}`,
			APIJSON: `{
				"appToken": "abc123"
			}`,
		},
		{
			Name: "full fields",
			LocalJSON: `{
				"app_token": "abc123",
				"delay": "5",
				"environment": true,
				"custom_mappings": [
					{"from": "Product Purchased", "to": "tok1"},
					{"from": "Signup", "to": "tok2"}
				],
				"partner_param_keys": [
					{"from": "userId", "to": "user_id"}
				],
				"enable_install_attribution_tracking": {
					"android": true,
					"ios": true
				},
				"event_filtering_whitelist": ["one", "two"]
			}`,
			APIJSON: `{
				"appToken": "abc123",
				"delay": "5",
				"environment": true,
				"customMappings": [
					{"from": "Product Purchased", "to": "tok1"},
					{"from": "Signup", "to": "tok2"}
				],
				"partnerParamKeys": [
					{"from": "userId", "to": "user_id"}
				],
				"enableInstallAttributionTracking": {
					"android": true,
					"ios": true
				},
				"eventFilteringOption": "whitelistedEvents",
				"whitelistedEvents": [
					{"eventName": "one"},
					{"eventName": "two"}
				]
			}`,
		},
		{
			Name: "event filtering blacklist",
			LocalJSON: `{
				"app_token": "abc123",
				"event_filtering_blacklist": ["noise"]
			}`,
			APIJSON: `{
				"appToken": "abc123",
				"eventFilteringOption": "blacklistedEvents",
				"blacklistedEvents": [
					{"eventName": "noise"}
				]
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"app_token": "abc123",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"appToken": "abc123",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
