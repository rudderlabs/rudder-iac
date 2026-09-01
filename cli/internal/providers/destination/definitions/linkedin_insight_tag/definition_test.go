package linkedininsighttag_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	linkedininsighttag "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/linkedin_insight_tag"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(linkedininsighttag.NewDefinition()))

	registered, err := registry.Get("linkedin_insight_tag", 1)
	require.NoError(t, err)

	assert.Equal(t, "linkedin_insight_tag", registered.Type)
	assert.Equal(t, "LINKEDIN_INSIGHT_TAG", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{}, registered.SecretKeys())
	assert.Equal(t, []string{"web"}, registered.SupportedSourceTypes())

	modes, err := registered.ConnectionModes("web")
	require.NoError(t, err)
	assert.Equal(t, []string{"device"}, modes)

	assert.Empty(t, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("LINKEDIN_INSIGHT_TAG", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestLinkedinInsightTagConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(linkedininsighttag.NewDefinition()))
	registered, err := registry.Get("linkedin_insight_tag", 1)
	require.NoError(t, err)

	t.Run("missing partner_id", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/partner_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("empty partner_id rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"partner_id": "",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/partner_id", errors[0].Path)
	})

	t.Run("event mapping strings reject values over maximum length", func(t *testing.T) {
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

	t.Run("event mapping strings reject line breaks", func(t *testing.T) {
		t.Parallel()

		for _, tc := range patternFieldCases("line one\nline two") {
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
	t.Run("event mapping strings accept ui templates of any length", func(t *testing.T) {
		t.Parallel()

		long := "{{ config.value || " + strings.Repeat("x", 200) + " }}"
		for _, tc := range patternFieldCases(long) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				assert.Empty(t, registered.ValidateConfig(tc.config))
			})
		}
	})

	t.Run("event filtering rejects whitelist and blacklist together", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"partner_id": "12345",
			"event_filtering": map[string]any{
				"whitelist": []any{"Product Viewed"},
				"blacklist": []any{"Order Completed"},
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/event_filtering/whitelist", errors[0].Path)
		assert.Contains(t, errors[0].Message, "cannot be specified together")
	})

	t.Run("use_native_sdk rejects unsupported source keys", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"partner_id": "12345",
			"use_native_sdk": map[string]any{
				"android": true,
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/use_native_sdk/android", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("connection_mode is not a supported key", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"partner_id": "12345",
			"connection_mode": map[string]any{
				"web": "device",
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/connection_mode", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("legacy consent keys rejected as config", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			errors := registered.ValidateConfig(map[string]any{
				"partner_id": "12345",
				key:          map[string]any{"web": []any{"C0001"}},
			})
			require.NotEmpty(t, errors)
			assert.Equal(t, "/"+key, errors[0].Path)
			assert.Contains(t, errors[0].Message, "unknown config field")
		}
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(validMinimalConfig()))
	})

	t.Run("valid example config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(validExampleConfig()))
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(validFullConfig()))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"partner_id":  "12345",
			"not_a_field": true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent management source rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"partner_id": "12345",
			"consent_management": map[string]any{
				"android_kotlin": []any{},
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/android_kotlin", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'android_kotlin' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"partner_id": "12345",
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

	t.Run("custom consent provider requires resolution strategy", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"partner_id": "12345",
			"consent_management": map[string]any{
				"web": []any{
					map[string]any{"provider": "custom"},
				},
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web/0/resolution_strategy", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})
}

func TestLinkedinInsightTagConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := linkedininsighttag.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal partner id only",
			LocalJSON: `{
				"partner_id": "12345"
			}`,
			APIJSON: `{
				"partnerId": "12345"
			}`,
		},
		{
			Name: "full fields with whitelist reshape",
			LocalJSON: `{
				"partner_id": "12345",
				"event_to_conversion_id_map": [
					{"from": "Order Completed", "to": "123456789"}
				],
				"event_filtering": {
					"whitelist": ["Order Completed", "Product Viewed"]
				},
				"use_native_sdk": {"web": true},
				"consent_management": {
					"web": [{
						"provider": "custom",
						"resolution_strategy": "and",
						"consents": ["marketing"]
					}]
				}
			}`,
			APIJSON: `{
				"partnerId": "12345",
				"eventToConversionIdMap": [
					{"from": "Order Completed", "to": "123456789"}
				],
				"whitelistedEvents": [
					{"eventName": "Order Completed"},
					{"eventName": "Product Viewed"}
				],
				"eventFilteringOption": "whitelistedEvents",
				"useNativeSDK": {"web": true},
				"consentManagement": {
					"web": [{
						"provider": "custom",
						"resolutionStrategy": "and",
						"consents": [{"consent": "marketing"}]
					}]
				}
			}`,
		},
		{
			Name: "blacklist reshape",
			LocalJSON: `{
				"partner_id": "12345",
				"event_filtering": {
					"blacklist": ["Page Viewed"]
				}
			}`,
			APIJSON: `{
				"partnerId": "12345",
				"blacklistedEvents": [
					{"eventName": "Page Viewed"}
				],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
	})
}

func validMinimalConfig() map[string]any {
	return map[string]any{
		"partner_id": "12345",
	}
}

func validExampleConfig() map[string]any {
	return map[string]any{
		"partner_id": "12345",
		"event_to_conversion_id_map": []any{
			map[string]any{"from": "Order Completed", "to": "123456789"},
		},
		"event_filtering": map[string]any{
			"whitelist": []any{"Order Completed", "Product Viewed"},
		},
		"use_native_sdk": map[string]any{
			"web": true,
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider":            "custom",
					"resolution_strategy": "and",
					"consents":            []any{"marketing"},
				},
			},
		},
	}
}

func validFullConfig() map[string]any {
	return map[string]any{
		"partner_id": "12345",
		"event_to_conversion_id_map": []any{
			map[string]any{"from": "Order Completed", "to": "123456789"},
			map[string]any{"from": "Product Viewed", "to": "987654321"},
		},
		"event_filtering": map[string]any{
			"blacklist": []any{"Page Viewed"},
		},
		"use_native_sdk": map[string]any{
			"web": false,
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{"C0002"},
				},
			},
		},
	}
}

func patternFieldCases(value string) []struct {
	name   string
	config map[string]any
	path   string
} {
	return []struct {
		name   string
		config map[string]any
		path   string
	}{
		{
			name: "event conversion from",
			config: map[string]any{
				"partner_id": "12345",
				"event_to_conversion_id_map": []any{
					map[string]any{"from": value, "to": "123456789"},
				},
			},
			path: "/event_to_conversion_id_map/0/from",
		},
		{
			name: "event conversion to",
			config: map[string]any{
				"partner_id": "12345",
				"event_to_conversion_id_map": []any{
					map[string]any{"from": "Order Completed", "to": value},
				},
			},
			path: "/event_to_conversion_id_map/0/to",
		},
		{
			name: "event filtering whitelist",
			config: map[string]any{
				"partner_id": "12345",
				"event_filtering": map[string]any{
					"whitelist": []any{value},
				},
			},
			path: "/event_filtering/whitelist/0",
		},
		{
			name: "event filtering blacklist",
			config: map[string]any{
				"partner_id": "12345",
				"event_filtering": map[string]any{
					"blacklist": []any{value},
				},
			},
			path: "/event_filtering/blacklist/0",
		},
	}
}
