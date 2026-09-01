package ga_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/ga"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(ga.NewDefinition()))

	registered, err := registry.Get("ga", 1)
	require.NoError(t, err)

	assert.Equal(t, "ga", registered.Type)
	assert.Equal(t, "GA", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "cloud", "react_native", "flutter", "cordova",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

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
		"domain/web":                          {"web"},
		"named_tracker/web":                   {"web"},
		"optimize/web":                        {"web"},
		"reset_custom_dimensions_on_page/web": {"web"},
		"sample_rate/web":                     {"web"},
		"set_all_mapped_props/web":            {"web"},
		"site_speed_sample_rate/web":          {"web"},
		"track_categorized_pages/web":         {"web"},
		"track_named_pages/web":               {"web"},
		"use_google_amp_client_id/web":        {"web"},
		"use_rich_event_names/web":            {"web"},
	}, registered.GatedKeyPaths())

	assert.NotContains(t, registered.SupportedSourceTypes(), "amp")
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")

	// schema.json defaults these eight; the backend injects them on create but
	// not update, so declaring them keeps a spec that omits them from diffing
	// forever. enableServerSideIdentify and eventFilteringOption are absent by
	// design: both are Discriminator-derived and have no local key to tag.
	assert.Equal(t, map[string]any{
		"double_click":              false,
		"enhanced_link_attribution": false,
		"include_search":            false,
		"disable_md5":               false,
		"anonymize_ip":              false,
		"enhanced_ecommerce":        false,
		"non_interaction":           false,
		"send_user_id":              false,
	}, registered.ApplyDefaults(map[string]any{}))

	// An explicit value wins, including one equal to the default.
	assert.Equal(t, true, registered.ApplyDefaults(map[string]any{
		"anonymize_ip": true,
	})["anonymize_ip"])

	byAPI, err := registry.GetByAPIType("GA", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestGoogleAnalyticsConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(ga.NewDefinition()))
	registered, err := registry.Get("ga", 1)
	require.NoError(t, err)

	t.Run("missing tracking_id", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/tracking_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("invalid tracking_id rejected", func(t *testing.T) {
		t.Parallel()

		for _, id := range []string{"G-XXXXXXXXXX", "ua-123-1", "UA-123", "UA-123-" + strings.Repeat("1", 101)} {
			errors := registered.ValidateConfig(map[string]any{"tracking_id": id})
			require.NotEmpty(t, errors, id)
			assert.Equal(t, "/tracking_id", errors[0].Path)
		}
	})

	t.Run("tracking_id accepts ui templates", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"tracking_id": "{{ config.ga_tracking_id || UA-123456-1 }}",
		})
		assert.Empty(t, errors)
	})

	t.Run("event filtering lists are mutually exclusive", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"tracking_id": "UA-123456-1",
			"event_filtering": map[string]any{
				"whitelist": []any{"Signed Up"},
				"blacklist": []any{"Signed Out"},
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/event_filtering/whitelist", errors[0].Path)
		assert.Contains(t, errors[0].Message, "cannot be specified together")
	})

	t.Run("server side identify fields require each other", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"tracking_id": "UA-123456-1",
			"server_side_identify": map[string]any{
				"event_category": "identify",
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/server_side_identify/event_action", errors[0].Path)
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
				name: "sample_rate",
				key:  "sample_rate",
				config: map[string]any{
					"ios": "100",
				},
				path: "/sample_rate/ios",
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

	// schema.json declares connectionMode for this destination, so it is real
	// config: db-config allows web cloud+device and everything else cloud only.
	t.Run("connection_mode value validated per source type", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"tracking_id":     "UA-123456-1",
			"connection_mode": map[string]any{"web": "device"},
		}))

		errors := registered.ValidateConfig(map[string]any{
			"tracking_id": "UA-123456-1",
			"connection_mode": map[string]any{
				"web":     "device",
				"android": "device", // android is cloud-only
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/android", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
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

	t.Run("string fields accept ui templates", func(t *testing.T) {
		t.Parallel()

		longTemplate := "{{ config.value || " + strings.Repeat("x", 200) + " }}"
		for _, tc := range patternFieldCases(longTemplate) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				assert.Empty(t, registered.ValidateConfig(tc.config))
			})
		}
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, registered.ValidateConfig(validMinimalConfig()))
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, registered.ValidateConfig(validFullConfig()))
	})

	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, registered.ValidateConfig(exampleYAMLConfig()))
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
					"web": []any{map[string]any{"consent": "analytics"}},
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

func TestGoogleAnalyticsConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := ga.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"tracking_id": "UA-123456-1"
			}`,
			APIJSON: `{
				"trackingID": "UA-123456-1"
			}`,
		},
		{
			Name: "full",
			LocalJSON: `{
				"tracking_id": "UA-123456-1",
				"rudder_delete_account_id": "delete-account-1",
				"double_click": true,
				"enhanced_link_attribution": true,
				"include_search": true,
				"server_side_identify": {
					"event_category": "User",
					"event_action": "Identify"
				},
				"disable_md5": true,
				"anonymize_ip": true,
				"enhanced_ecommerce": true,
				"non_interaction": true,
				"send_user_id": true,
				"event_filtering": {
					"blacklist": ["Signed Out", "Viewed Admin"]
				},
				"use_native_sdk": {"web": true},
				"track_categorized_pages": {"web": true},
				"track_named_pages": {"web": true},
				"use_rich_event_names": {"web": true},
				"sample_rate": {"web": "100"},
				"site_speed_sample_rate": {"web": "1"},
				"reset_custom_dimensions_on_page": {"web": ["dimension1", "dimension2"]},
				"set_all_mapped_props": {"web": true},
				"domain": {"web": "auto"},
				"optimize": {"web": "GTM-ABCDE"},
				"use_google_amp_client_id": {"web": true},
				"named_tracker": {"web": true},
				"dimensions": [{"from": "user_type", "to": "dimension1"}],
				"metrics": [{"from": "ltv", "to": "metric1"}],
				"content_groupings": [{"from": "section", "to": "contentGroup1"}],
				"custom_mappings": [{"from": "plan", "to": "dimension2"}],
				"consent_management": {
					"android_kotlin": [{"provider": "ketch", "consents": ["analytics"]}],
					"react_native": [{"provider": "custom", "resolution_strategy": "and", "consents": ["marketing"]}]
				}
			}`,
			APIJSON: `{
				"trackingID": "UA-123456-1",
				"rudderDeleteAccountId": "delete-account-1",
				"doubleClick": true,
				"enhancedLinkAttribution": true,
				"includeSearch": true,
				"serverSideIdentifyEventCategory": "User",
				"serverSideIdentifyEventAction": "Identify",
				"enableServerSideIdentify": true,
				"disableMd5": true,
				"anonymizeIp": true,
				"enhancedEcommerce": true,
				"nonInteraction": true,
				"sendUserId": true,
				"blacklistedEvents": [
					{"eventName": "Signed Out"},
					{"eventName": "Viewed Admin"}
				],
				"eventFilteringOption": "blacklistedEvents",
				"useNativeSDK": {"web": true},
				"trackCategorizedPages": {"web": true},
				"trackNamedPages": {"web": true},
				"useRichEventNames": {"web": true},
				"sampleRate": {"web": "100"},
				"siteSpeedSampleRate": {"web": "1"},
				"resetCustomDimensionsOnPage": {
					"web": [
						{"resetCustomDimensionsOnPage": "dimension1"},
						{"resetCustomDimensionsOnPage": "dimension2"}
					]
				},
				"setAllMappedProps": {"web": true},
				"domain": {"web": "auto"},
				"optimize": {"web": "GTM-ABCDE"},
				"useGoogleAmpClientId": {"web": true},
				"namedTracker": {"web": true},
				"dimensions": [{"from": "user_type", "to": "dimension1"}],
				"metrics": [{"from": "ltv", "to": "metric1"}],
				"contentGroupings": [{"from": "section", "to": "contentGroup1"}],
				"customMappings": [{"from": "plan", "to": "dimension2"}],
				"consentManagement": {
					"androidKotlin": [{"provider": "ketch", "consents": [{"consent": "analytics"}]}],
					"reactnative": [{"provider": "custom", "resolutionStrategy": "and", "consents": [{"consent": "marketing"}]}]
				}
			}`,
		},
		{
			Name: "whitelist event filtering",
			LocalJSON: `{
				"tracking_id": "UA-123456-1",
				"event_filtering": {
					"whitelist": ["Signed Up"]
				}
			}`,
			APIJSON: `{
				"trackingID": "UA-123456-1",
				"whitelistedEvents": [{"eventName": "Signed Up"}],
				"eventFilteringOption": "whitelistedEvents"
			}`,
		},
	})
}

// Import keeps every stored value, including a list the selector does not
// currently point at. Gating APIToLocal on the selector would emit a spec that
// silently drops the other list, which the next apply then erases upstream.
func TestGoogleAnalyticsAPIToLocalKeepsUnselectedValues(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(ga.NewDefinition()))
	registered, err := registry.Get("ga", 1)
	require.NoError(t, err)

	local, err := registered.APIToLocal(map[string]any{
		"trackingID":                      "UA-123456-1",
		"whitelistedEvents":               []any{map[string]any{"eventName": "Order Completed"}},
		"blacklistedEvents":               []any{map[string]any{"eventName": "Application Opened"}},
		"eventFilteringOption":            "whitelistedEvents",
		"enableServerSideIdentify":        false,
		"serverSideIdentifyEventCategory": "All",
		"serverSideIdentifyEventAction":   "User Enriched",
	})
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"tracking_id": "UA-123456-1",
		"event_filtering": map[string]any{
			"whitelist": []any{"Order Completed"},
			"blacklist": []any{"Application Opened"},
		},
		"server_side_identify": map[string]any{
			"event_category": "All",
			"event_action":   "User Enriched",
		},
	}, local)
}

func validMinimalConfig() map[string]any {
	return map[string]any{
		"tracking_id": "UA-123456-1",
	}
}

func validFullConfig() map[string]any {
	return map[string]any{
		"tracking_id":               "UA-123456-1",
		"rudder_delete_account_id":  "delete-account-1",
		"double_click":              true,
		"enhanced_link_attribution": true,
		"include_search":            true,
		"server_side_identify": map[string]any{
			"event_category": "User",
			"event_action":   "Identify",
		},
		"disable_md5":        true,
		"anonymize_ip":       true,
		"enhanced_ecommerce": true,
		"non_interaction":    true,
		"send_user_id":       true,
		"event_filtering": map[string]any{
			"whitelist": []any{"Signed Up", "Order Completed"},
		},
		"use_native_sdk": map[string]any{
			"web": true,
		},
		"track_categorized_pages": map[string]any{
			"web": true,
		},
		"track_named_pages": map[string]any{
			"web": true,
		},
		"use_rich_event_names": map[string]any{
			"web": true,
		},
		"sample_rate": map[string]any{
			"web": "100",
		},
		"site_speed_sample_rate": map[string]any{
			"web": "1",
		},
		"reset_custom_dimensions_on_page": map[string]any{
			"web": []any{"dimension1", "dimension2"},
		},
		"set_all_mapped_props": map[string]any{
			"web": true,
		},
		"domain": map[string]any{
			"web": "auto",
		},
		"optimize": map[string]any{
			"web": "GTM-ABCDE",
		},
		"use_google_amp_client_id": map[string]any{
			"web": true,
		},
		"named_tracker": map[string]any{
			"web": true,
		},
		"dimensions": []any{
			map[string]any{"from": "user_type", "to": "dimension1"},
		},
		"metrics": []any{
			map[string]any{"from": "ltv", "to": "metric1"},
		},
		"content_groupings": []any{
			map[string]any{"from": "section", "to": "contentGroup1"},
		},
		"custom_mappings": []any{
			map[string]any{"from": "plan", "to": "dimension2"},
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{"provider": "oneTrust", "consents": []any{"analytics"}},
			},
		},
	}
}

func exampleYAMLConfig() map[string]any {
	return map[string]any{
		"tracking_id": "UA-123456-1",
		"sample_rate": map[string]any{
			"web": "100",
		},
		"site_speed_sample_rate": map[string]any{
			"web": "1",
		},
		"domain": map[string]any{
			"web": "auto",
		},
	}
}

type patternCase struct {
	name   string
	config map[string]any
	path   string
}

func patternFieldCases(value string) []patternCase {
	return []patternCase{
		{
			name: "server_side_identify.event_category",
			config: map[string]any{
				"tracking_id": "UA-123456-1",
				"server_side_identify": map[string]any{
					"event_category": value,
					"event_action":   "Identify",
				},
			},
			path: "/server_side_identify/event_category",
		},
		{
			name: "event_filtering.whitelist",
			config: map[string]any{
				"tracking_id": "UA-123456-1",
				"event_filtering": map[string]any{
					"whitelist": []any{value},
				},
			},
			path: "/event_filtering/whitelist/0",
		},
		{
			name: "sample_rate.web",
			config: map[string]any{
				"tracking_id": "UA-123456-1",
				"sample_rate": map[string]any{
					"web": value,
				},
			},
			path: "/sample_rate/web",
		},
		{
			name: "reset_custom_dimensions_on_page.web",
			config: map[string]any{
				"tracking_id": "UA-123456-1",
				"reset_custom_dimensions_on_page": map[string]any{
					"web": []any{value},
				},
			},
			path: "/reset_custom_dimensions_on_page/web/0",
		},
		{
			name: "dimensions.from",
			config: map[string]any{
				"tracking_id": "UA-123456-1",
				"dimensions": []any{
					map[string]any{"from": value, "to": "dimension1"},
				},
			},
			path: "/dimensions/0/from",
		},
	}
}
