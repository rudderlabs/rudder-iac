package googleadwordsofflineconversions_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	googleadwordsofflineconversions "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/google_adwords_offline_conversions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googleadwordsofflineconversions.NewDefinition()))

	registered, err := registry.Get("google_adwords_offline_conversions", 1)
	require.NoError(t, err)

	assert.Equal(t, "google_adwords_offline_conversions", registered.Type)
	assert.Equal(t, "GOOGLE_ADWORDS_OFFLINE_CONVERSIONS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys())
	assert.Empty(t, registered.GatedKeyPaths())
	assert.Equal(t, map[string]any{
		"sub_account":             false,
		"user_identifier_source":  "none",
		"conversion_environment":  "none",
		"default_user_identifier": "email",
		"hash_user_identifier":    true,
		"validate_only":           false,
	}, registered.ConfigDefaults())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web", "unity",
		"cloud", "react_native", "flutter", "cordova",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())
	assert.NotContains(t, registered.SupportedSourceTypes(), "amp")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	byAPI, err := registry.GetByAPIType("GOOGLE_ADWORDS_OFFLINE_CONVERSIONS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestGoogleAdwordsOfflineConversionsApplyDefaults(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googleadwordsofflineconversions.NewDefinition()))
	registered, err := registry.Get("google_adwords_offline_conversions", 1)
	require.NoError(t, err)

	t.Run("fills defaults omitted by the spec", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, map[string]any{
			"rudder_account_id":       "account-1",
			"customer_id":             "1234567890",
			"sub_account":             false,
			"user_identifier_source":  "none",
			"conversion_environment":  "none",
			"default_user_identifier": "email",
			"hash_user_identifier":    true,
			"validate_only":           false,
		}, registered.ApplyDefaults(validMinimalConfig()))
	})

	t.Run("keeps values the spec sets", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["sub_account"] = true
		config["login_customer_id"] = "9876543210"
		config["user_identifier_source"] = "FIRST_PARTY"
		config["conversion_environment"] = "WEB"
		config["default_user_identifier"] = "phone"
		config["hash_user_identifier"] = false
		config["validate_only"] = true

		assert.Equal(t, map[string]any{
			"rudder_account_id":       "account-1",
			"customer_id":             "1234567890",
			"sub_account":             true,
			"login_customer_id":       "9876543210",
			"user_identifier_source":  "FIRST_PARTY",
			"conversion_environment":  "WEB",
			"default_user_identifier": "phone",
			"hash_user_identifier":    false,
			"validate_only":           true,
		}, registered.ApplyDefaults(config))
	})
}

func TestGoogleAdwordsOfflineConversionsConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googleadwordsofflineconversions.NewDefinition()))
	registered, err := registry.Get("google_adwords_offline_conversions", 1)
	require.NoError(t, err)

	t.Run("missing rudder_account_id", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		delete(config, "rudder_account_id")

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/rudder_account_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing customer_id", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		delete(config, "customer_id")

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/customer_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("login_customer_id required for sub accounts", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["sub_account"] = true

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/login_customer_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("login_customer_id optional without sub account", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["sub_account"] = false

		errors := registered.ValidateConfig(config)

		assert.Empty(t, errors)
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(validMinimalConfig())

		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(validFullConfig())

		assert.Empty(t, errors)
	})

	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id":       "google-ads-account-1",
			"customer_id":             "1234567890",
			"sub_account":             true,
			"login_customer_id":       "0987654321",
			"validate_only":           true,
			"hash_user_identifier":    true,
			"user_identifier_source":  "FIRST_PARTY",
			"conversion_environment":  "WEB",
			"default_user_identifier": "email",
			"events_to_offline_conversions_type_mapping": []any{
				map[string]any{"from": "Lead Form Submitted", "to": "click"},
				map[string]any{"from": "Phone Call Completed", "to": "call"},
			},
			"events_to_conversions_names_mapping": []any{
				map[string]any{"from": "Lead Form Submitted", "to": "Website Lead"},
			},
			"custom_variables": []any{
				map[string]any{"from": "plan", "to": "plan_type"},
			},
			"connection_mode": map[string]any{
				"web":   "cloud",
				"cloud": "cloud",
			},
			"consent_management": map[string]any{
				"web": []any{
					map[string]any{
						"provider":            "custom",
						"resolution_strategy": "and",
						"consents":            []any{"ads_storage", "analytics_storage"},
					},
				},
			},
		})

		assert.Empty(t, errors)
	})

	t.Run("string fields reject invalid literals", func(t *testing.T) {
		t.Parallel()

		for _, tc := range patternFieldCases(strings.Repeat("x", 101)) {
			t.Run(tc.name+" over 100 characters", func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(tc.config)

				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}

		for _, tc := range patternFieldCases("line\nbreak") {
			t.Run(tc.name+" with line break", func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(tc.config)

				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	t.Run("string fields accept ui templates", func(t *testing.T) {
		t.Parallel()

		long := "{{ config.value || " + strings.Repeat("x", 150) + " }}"
		for _, tc := range patternFieldCases(long) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(tc.config)

				assert.Empty(t, errors)
			})
		}
	})

	t.Run("invalid offline conversion type rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["events_to_offline_conversions_type_mapping"] = []any{
			map[string]any{"from": "Signup", "to": "purchase"},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/events_to_offline_conversions_type_mapping/0/to", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("empty offline conversion type follows schema optionality", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["events_to_offline_conversions_type_mapping"] = []any{
			map[string]any{"from": "Signup", "to": ""},
		}

		errors := registered.ValidateConfig(config)

		assert.Empty(t, errors)
	})

	t.Run("invalid enum fields rejected", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name  string
			field string
			value string
		}{
			{name: "user identifier source", field: "user_identifier_source", value: "SECOND_PARTY"},
			{name: "conversion environment", field: "conversion_environment", value: "MOBILE_WEB"},
			{name: "default user identifier", field: "default_user_identifier", value: "address"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config[tc.field] = tc.value

				errors := registered.ValidateConfig(config)

				require.Len(t, errors, 1)
				assert.Equal(t, "/"+tc.field, errors[0].Path)
				assert.Contains(t, errors[0].Message, "must be one of")
			})
		}
	})

	t.Run("enum fields reject dynamic values", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"user_identifier_source", "conversion_environment", "default_user_identifier"} {
			t.Run(field, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config[field] = "{{ config.value || none }}"

				errors := registered.ValidateConfig(config)

				require.Len(t, errors, 1)
				assert.Equal(t, "/"+field, errors[0].Path)
			})
		}
	})

	t.Run("connection_mode accepts a supported mode", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["connection_mode"] = map[string]any{"web": "cloud"}

		errors := registered.ValidateConfig(config)

		assert.Empty(t, errors)
	})

	t.Run("connection_mode rejects an unsupported mode", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["connection_mode"] = map[string]any{"web": "device"}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/web", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
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

	t.Run("legacy consent include key rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["one_trust_cookie_categories"] = map[string]any{}

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/one_trust_cookie_categories", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["consent_management"] = map[string]any{
			"warehouse": []any{},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'warehouse' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["consent_management"] = map[string]any{
			"android_kotlin": []any{
				map[string]any{"provider": "unknown"},
			},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/android_kotlin/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestGoogleAdwordsOfflineConversionsConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := googleadwordsofflineconversions.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"rudder_account_id": "account-1",
				"customer_id": "1234567890"
			}`,
			APIJSON: `{
				"rudderAccountId": "account-1",
				"customerId": "1234567890"
			}`,
		},
		{
			Name: "full scalar fields",
			LocalJSON: `{
				"rudder_account_id": "account-1",
				"customer_id": "1234567890",
				"sub_account": true,
				"login_customer_id": "0987654321",
				"user_identifier_source": "FIRST_PARTY",
				"conversion_environment": "WEB",
				"default_user_identifier": "phone",
				"hash_user_identifier": false,
				"validate_only": true,
				"connection_mode": {
					"web": "cloud",
					"android_kotlin": "cloud"
				}
			}`,
			APIJSON: `{
				"rudderAccountId": "account-1",
				"customerId": "1234567890",
				"subAccount": true,
				"loginCustomerId": "0987654321",
				"UserIdentifierSource": "FIRST_PARTY",
				"conversionEnvironment": "WEB",
				"defaultUserIdentifier": "phone",
				"hashUserIdentifier": false,
				"validateOnly": true,
				"connectionMode": {
					"web": "cloud",
					"androidKotlin": "cloud"
				}
			}`,
		},
		{
			Name: "array mappings",
			LocalJSON: `{
				"rudder_account_id": "account-1",
				"customer_id": "1234567890",
				"events_to_offline_conversions_type_mapping": [
					{"from": "Lead Form Submitted", "to": "click"},
					{"from": "Phone Call Completed", "to": "call"}
				],
				"events_to_conversions_names_mapping": [
					{"from": "Lead Form Submitted", "to": "Website Lead"}
				],
				"custom_variables": [
					{"from": "plan", "to": "plan_type"},
					{"from": "region", "to": "sales_region"}
				]
			}`,
			APIJSON: `{
				"rudderAccountId": "account-1",
				"customerId": "1234567890",
				"eventsToOfflineConversionsTypeMapping": [
					{"from": "Lead Form Submitted", "to": "click"},
					{"from": "Phone Call Completed", "to": "call"}
				],
				"eventsToConversionsNamesMapping": [
					{"from": "Lead Form Submitted", "to": "Website Lead"}
				],
				"customVariables": [
					{"from": "plan", "to": "plan_type"},
					{"from": "region", "to": "sales_region"}
				]
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"rudder_account_id": "account-1",
				"customer_id": "1234567890",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}],
					"cloud": [{"provider": "custom", "resolution_strategy": "and", "consents": ["ads_storage"]}]
				}
			}`,
			APIJSON: `{
				"rudderAccountId": "account-1",
				"customerId": "1234567890",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}],
					"cloud": [{"provider": "custom", "resolutionStrategy": "and", "consents": [{"consent": "ads_storage"}]}]
				}
			}`,
		},
	})
}

type patternFieldCase struct {
	name   string
	path   string
	config map[string]any
}

func patternFieldCases(value string) []patternFieldCase {
	return []patternFieldCase{
		{
			name: "customer_id",
			path: "/customer_id",
			config: map[string]any{
				"rudder_account_id": "account-1",
				"customer_id":       value,
			},
		},
		{
			name: "login_customer_id",
			path: "/login_customer_id",
			config: map[string]any{
				"rudder_account_id": "account-1",
				"customer_id":       "1234567890",
				"sub_account":       true,
				"login_customer_id": value,
			},
		},
		{
			name: "offline conversion mapping from",
			path: "/events_to_offline_conversions_type_mapping/0/from",
			config: map[string]any{
				"rudder_account_id": "account-1",
				"customer_id":       "1234567890",
				"events_to_offline_conversions_type_mapping": []any{
					map[string]any{"from": value, "to": "click"},
				},
			},
		},
		{
			name: "conversion name mapping from",
			path: "/events_to_conversions_names_mapping/0/from",
			config: map[string]any{
				"rudder_account_id": "account-1",
				"customer_id":       "1234567890",
				"events_to_conversions_names_mapping": []any{
					map[string]any{"from": value, "to": "Website Lead"},
				},
			},
		},
		{
			name: "conversion name mapping to",
			path: "/events_to_conversions_names_mapping/0/to",
			config: map[string]any{
				"rudder_account_id": "account-1",
				"customer_id":       "1234567890",
				"events_to_conversions_names_mapping": []any{
					map[string]any{"from": "Lead Form Submitted", "to": value},
				},
			},
		},
		{
			name: "custom variable from",
			path: "/custom_variables/0/from",
			config: map[string]any{
				"rudder_account_id": "account-1",
				"customer_id":       "1234567890",
				"custom_variables": []any{
					map[string]any{"from": value, "to": "plan_type"},
				},
			},
		},
		{
			name: "custom variable to",
			path: "/custom_variables/0/to",
			config: map[string]any{
				"rudder_account_id": "account-1",
				"customer_id":       "1234567890",
				"custom_variables": []any{
					map[string]any{"from": "plan", "to": value},
				},
			},
		},
	}
}

func validMinimalConfig() map[string]any {
	return map[string]any{
		"rudder_account_id": "account-1",
		"customer_id":       "1234567890",
	}
}

func validFullConfig() map[string]any {
	return map[string]any{
		"rudder_account_id":       "account-1",
		"customer_id":             "1234567890",
		"sub_account":             true,
		"login_customer_id":       "0987654321",
		"user_identifier_source":  "FIRST_PARTY",
		"conversion_environment":  "WEB",
		"default_user_identifier": "phone",
		"hash_user_identifier":    false,
		"validate_only":           true,
		"events_to_offline_conversions_type_mapping": []any{
			map[string]any{"from": "Lead Form Submitted", "to": "click"},
			map[string]any{"from": "Phone Call Completed", "to": "call"},
		},
		"events_to_conversions_names_mapping": []any{
			map[string]any{"from": "Lead Form Submitted", "to": "Website Lead"},
			map[string]any{"from": "Order Completed", "to": "Purchase"},
		},
		"custom_variables": []any{
			map[string]any{"from": "plan", "to": "plan_type"},
			map[string]any{"from": "region", "to": "sales_region"},
		},
		"connection_mode": map[string]any{"web": "cloud"},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider":            "custom",
					"resolution_strategy": "and",
					"consents":            []any{"ads_storage"},
				},
			},
		},
	}
}
