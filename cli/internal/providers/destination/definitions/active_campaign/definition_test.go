package activecampaign_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	activecampaign "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/active_campaign"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(activecampaign.NewDefinition()))

	registered, err := registry.Get("active_campaign", 1)
	require.NoError(t, err)

	assert.Equal(t, "active_campaign", registered.Type)
	assert.Equal(t, "ACTIVE_CAMPAIGN", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"api_key", "event_key"}, registered.SecretKeys())
	assert.Empty(t, registered.GatedKeyPaths())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web", "unity", "amp",
		"cloud", "warehouse", "react_native", "flutter", "cordova", "shopify",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		t.Run(sourceType+" connection modes", func(t *testing.T) {
			t.Parallel()

			modes, err := registered.ConnectionModes(sourceType)
			require.NoError(t, err)

			if sourceType == "web" {
				assert.Equal(t, []string{"cloud", "device", "hybrid"}, modes)
				return
			}
			assert.Equal(t, []string{"cloud"}, modes)
		})
	}

	byAPI, err := registry.GetByAPIType("ACTIVE_CAMPAIGN", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestActiveCampaignConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(activecampaign.NewDefinition()))
	registered, err := registry.Get("active_campaign", 1)
	require.NoError(t, err)

	t.Run("required fields", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"api_url", "api_key"} {
			t.Run("missing "+field, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				delete(config, field)

				errors := registered.ValidateConfig(config)

				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+field, errors[0].Path)
				assert.Contains(t, errors[0].Message, "required")
			})

			t.Run("empty "+field, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config[field] = ""

				errors := registered.ValidateConfig(config)

				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+field, errors[0].Path)
			})
		}
	})

	t.Run("api_url rejects invalid urls", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["api_url"] = "not-a-url"

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_url", errors[0].Path)
	})

	t.Run("api_url rejects ngrok urls", func(t *testing.T) {
		t.Parallel()

		for _, value := range []string{
			"https://account.ngrok.io",
			"https://account.ngrok.io.",
			"https://account.ngrok.io:443",
		} {
			t.Run(value, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config["api_url"] = value

				errors := registered.ValidateConfig(config)

				require.NotEmpty(t, errors)
				assert.Equal(t, "/api_url", errors[0].Path)
			})
		}
	})

	t.Run("api_url accepts active campaign urls", func(t *testing.T) {
		t.Parallel()

		for _, value := range []string{
			"https://accountname.api-us1.com",
			"accountname.api-us1.com",
			"https://account-ngrok.io",
		} {
			t.Run(value, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config["api_url"] = value

				assert.Empty(t, registered.ValidateConfig(config))
			})
		}
	})

	t.Run("single line fields reject values over maximum length", func(t *testing.T) {
		t.Parallel()

		for _, tc := range singleLineFieldCases(strings.Repeat("x", 101)) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(tc.config)

				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	t.Run("single line fields reject line breaks", func(t *testing.T) {
		t.Parallel()

		for _, tc := range singleLineFieldCases("line\nbreak") {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(tc.config)

				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	t.Run("pattern fields accept ui templates of any length", func(t *testing.T) {
		t.Parallel()
		longTemplate := "{{ config.value || " + strings.Repeat("x", 120) + " }}"

		for _, tc := range patternFieldCases(longTemplate) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				assert.Empty(t, registered.ValidateConfig(tc.config))
			})
		}
	})

	t.Run("deprecated env references get no template exemption", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["api_key"] = "env." + strings.Repeat("A", 101)

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/api_key", errors[0].Path)
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
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"api_url":   "https://accountname.api-us1.com",
			"api_key":   "{{ .ACTIVE_CAMPAIGN_API_KEY }}",
			"actid":     "2764X0567",
			"event_key": "{{ .ACTIVE_CAMPAIGN_EVENT_KEY }}",
			"consent_management": map[string]any{
				"web": []any{
					map[string]any{
						"provider":            "custom",
						"resolution_strategy": "and",
						"consents":            []any{"marketing"},
					},
				},
			},
		}))
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

	t.Run("connection mode rejected as unknown key", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["connection_mode"] = map[string]any{"web": "cloud"}

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/connection_mode", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("use native sdk rejected as unknown key", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["use_native_sdk"] = map[string]any{"web": true}

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/use_native_sdk", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("legacy consent blocks are not supported keys", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			t.Run(key, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config[key] = map[string]any{"web": []any{}}

				errors := registered.ValidateConfig(config)

				require.Len(t, errors, 1)
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
			"ios_swift": []any{
				map[string]any{"provider": "unknown"},
			},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/ios_swift/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})

	t.Run("duplicate consent provider rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["consent_management"] = map[string]any{
			"web": []any{
				map[string]any{"provider": "oneTrust"},
				map[string]any{"provider": "oneTrust"},
			},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web/1/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "only one consent entry")
	})
}

func TestActiveCampaignConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := activecampaign.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"api_url": "https://accountname.api-us1.com",
				"api_key": "active-campaign-api-key"
			}`,
			APIJSON: `{
				"apiUrl": "https://accountname.api-us1.com",
				"apiKey": "active-campaign-api-key"
			}`,
		},
		{
			Name: "full",
			LocalJSON: `{
				"api_url": "https://accountname.api-us1.com",
				"api_key": "active-campaign-api-key",
				"actid": "2764X0567",
				"event_key": "active-campaign-event-key"
			}`,
			APIJSON: `{
				"apiUrl": "https://accountname.api-us1.com",
				"apiKey": "active-campaign-api-key",
				"actid": "2764X0567",
				"eventKey": "active-campaign-event-key"
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"api_url": "https://accountname.api-us1.com",
				"api_key": "active-campaign-api-key",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"apiUrl": "https://accountname.api-us1.com",
				"apiKey": "active-campaign-api-key",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}

type fieldCase struct {
	name   string
	path   string
	config map[string]any
}

func singleLineFieldCases(value string) []fieldCase {
	return []fieldCase{
		{
			name: "api_key",
			path: "/api_key",
			config: map[string]any{
				"api_url": "https://accountname.api-us1.com",
				"api_key": value,
			},
		},
		{
			name: "actid",
			path: "/actid",
			config: map[string]any{
				"api_url": "https://accountname.api-us1.com",
				"api_key": "active-campaign-api-key",
				"actid":   value,
			},
		},
		{
			name: "event_key",
			path: "/event_key",
			config: map[string]any{
				"api_url":   "https://accountname.api-us1.com",
				"api_key":   "active-campaign-api-key",
				"event_key": value,
			},
		},
	}
}

func patternFieldCases(value string) []fieldCase {
	return []fieldCase{
		{
			name: "api_url",
			path: "/api_url",
			config: map[string]any{
				"api_url": value,
				"api_key": "active-campaign-api-key",
			},
		},
		{
			name: "api_key",
			path: "/api_key",
			config: map[string]any{
				"api_url": "https://accountname.api-us1.com",
				"api_key": value,
			},
		},
		{
			name: "actid",
			path: "/actid",
			config: map[string]any{
				"api_url": "https://accountname.api-us1.com",
				"api_key": "active-campaign-api-key",
				"actid":   value,
			},
		},
		{
			name: "event_key",
			path: "/event_key",
			config: map[string]any{
				"api_url":   "https://accountname.api-us1.com",
				"api_key":   "active-campaign-api-key",
				"event_key": value,
			},
		},
	}
}

func validMinimalConfig() map[string]any {
	return map[string]any{
		"api_url": "https://accountname.api-us1.com",
		"api_key": "active-campaign-api-key",
	}
}

func validFullConfig() map[string]any {
	return map[string]any{
		"api_url":   "https://accountname.api-us1.com",
		"api_key":   "active-campaign-api-key",
		"actid":     "2764X0567",
		"event_key": "active-campaign-event-key",
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider":            "custom",
					"resolution_strategy": "and",
					"consents":            []any{"marketing", "analytics"},
				},
			},
			"android_kotlin": []any{
				map[string]any{"provider": "oneTrust"},
			},
		},
	}
}
