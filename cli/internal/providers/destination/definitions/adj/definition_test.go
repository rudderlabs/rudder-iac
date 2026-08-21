package adj_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/adj"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(adj.NewDefinition()))

	registered, err := registry.Get("adj", 1)
	require.NoError(t, err)

	assert.Equal(t, "adj", registered.Type)
	assert.Equal(t, "ADJ", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys(), "db-config declares no secretKeys")

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
	require.NoError(t, registry.Register(adj.NewDefinition()))
	registered, err := registry.Get("adj", 1)
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
			"partner_params_keys": []any{
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
			"partner_params_keys": []any{
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

	t.Run("pattern constraints reject invalid literals", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			field string
			value any
			path  string
		}{
			{"app_token", "bad\ntoken", "/app_token"},
			{"delay", "bad\ndelay", "/delay"},
			{"custom_mappings", []any{map[string]any{"from": "bad\nfrom", "to": "x"}}, "/custom_mappings/0/from"},
			{"partner_params_keys", []any{map[string]any{"from": "x", "to": "bad\nto"}}, "/partner_params_keys/0/to"},
			{"event_filtering_blacklist", []any{"bad\nevent"}, "/event_filtering_blacklist/0"},
			{"event_filtering_whitelist", []any{"bad\nevent"}, "/event_filtering_whitelist/0"},
		}

		for _, tc := range cases {
			cfg := map[string]any{"app_token": "token"}
			cfg[tc.field] = tc.value

			errors := registered.ValidateConfig(cfg)
			require.NotEmpty(t, errors, tc.field)
			assert.Equal(t, tc.path, errors[0].Path)
		}
	})

	// Only the nested mapping and event-filter patterns carry a template branch
	// upstream; app_token and delay do not, so a template must be rejected there.
	t.Run("templates accepted only where schema allows them", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"app_token":                 "token",
			"custom_mappings":           []any{map[string]any{"from": "{{ config.from || evt }}", "to": "abc"}},
			"partner_params_keys":       []any{map[string]any{"from": "userId", "to": "{{ config.to || user_id }}"}},
			"event_filtering_blacklist": []any{"{{ config.event || Password Reset }}"},
		}))

		for _, field := range []string{"app_token", "delay"} {
			cfg := map[string]any{"app_token": "token"}
			cfg[field] = "{{ config.x || " + strings.Repeat("a", 150) + " }}"

			errors := registered.ValidateConfig(cfg)
			require.NotEmpty(t, errors, field)
			assert.Equal(t, "/"+field, errors[0].Path)
		}
	})

	t.Run("legacy consent blocks are not supported keys", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			cfg := map[string]any{"app_token": "token", key: map[string]any{}}

			errors := registered.ValidateConfig(cfg)
			require.NotEmpty(t, errors, key)
			assert.Equal(t, "/"+key, errors[0].Path)
		}
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

	def := adj.NewDefinition()
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
				"partner_params_keys": [
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
				"partnerParamsKeys": [
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
