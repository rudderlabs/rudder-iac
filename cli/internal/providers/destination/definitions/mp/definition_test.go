package mp_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/mp"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(mp.NewDefinition()))

	registered, err := registry.Get("mp", 1)
	require.NoError(t, err)

	assert.Equal(t, "mp", registered.Type)
	assert.Equal(t, "MP", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"token", "gdpr_api_token", "service_account_secret"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web", "unity",
		"cloud", "react_native", "flutter", "cordova"}
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
		"session_replay_percentage/web": {"web"},
	}, registered.GatedKeyPaths())

	assert.Equal(t, map[string]any{
		"user_deletion_api":                  "engage",
		"strict_mode":                        false,
		"ignore_dnt":                         false,
		"use_user_defined_page_event_name":   false,
		"user_defined_page_event_template":   "Viewed {{ category }} {{ name }} page",
		"use_user_defined_screen_event_name": false,
		"user_defined_screen_event_template": "Viewed {{ category }} {{ name }} screen",
		"drop_traits_in_track_event":         false,
		"people":                             false,
		"set_all_traits_by_default":          false,
		"consolidated_page_calls":            true,
		"track_categorized_pages":            false,
		"track_named_pages":                  false,
		"cross_subdomain_cookie":             false,
		"persistence_type":                   "cookie",
		"secure_cookie":                      false,
		"use_new_mapping":                    false,
	}, registered.ConfigDefaults())

	byAPI, err := registry.GetByAPIType("MP", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestMPApplyDefaults(t *testing.T) {
	t.Parallel()

	registered := registeredMPDefinition(t)

	t.Run("fills defaults omitted by the spec", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, map[string]any{
			"token":                              "mp_project_token",
			"data_residency":                     "us",
			"identity_merge_api":                 "original",
			"user_deletion_api":                  "engage",
			"strict_mode":                        false,
			"ignore_dnt":                         false,
			"use_user_defined_page_event_name":   false,
			"user_defined_page_event_template":   "Viewed {{ category }} {{ name }} page",
			"use_user_defined_screen_event_name": false,
			"user_defined_screen_event_template": "Viewed {{ category }} {{ name }} screen",
			"drop_traits_in_track_event":         false,
			"people":                             false,
			"set_all_traits_by_default":          false,
			"consolidated_page_calls":            true,
			"track_categorized_pages":            false,
			"track_named_pages":                  false,
			"cross_subdomain_cookie":             false,
			"persistence_type":                   "cookie",
			"secure_cookie":                      false,
			"use_new_mapping":                    false,
		}, registered.ApplyDefaults(validMinimalConfig()))
	})

	t.Run("keeps values the spec sets", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["people"] = true
		config["use_new_mapping"] = true
		config["persistence_type"] = "localStorage"

		defaults := registered.ApplyDefaults(config)
		assert.Equal(t, true, defaults["people"])
		assert.Equal(t, true, defaults["use_new_mapping"])
		assert.Equal(t, "localStorage", defaults["persistence_type"])
	})
}

func TestMPConfigValidation(t *testing.T) {
	t.Parallel()

	registered := registeredMPDefinition(t)

	t.Run("missing token", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		delete(config, "token")
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/token", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing data_residency", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		delete(config, "data_residency")
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/data_residency", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing identity_merge_api", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		delete(config, "identity_merge_api")
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/identity_merge_api", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("invalid enums rejected", func(t *testing.T) {
		t.Parallel()

		for key, value := range map[string]any{
			"data_residency":     "apac",
			"identity_merge_api": "classic",
			"user_deletion_api":  "delete",
			"persistence_type":   "sessionStorage",
		} {
			t.Run(key, func(t *testing.T) {
				t.Parallel()

				config := validMinimalConfig()
				config[key] = value
				errors := registered.ValidateConfig(config)
				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+key, errors[0].Path)
				assert.Contains(t, errors[0].Message, "must be one of")
			})
		}
	})

	t.Run("token pattern accepts templates and rejects invalid literals", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["token"] = "{{ secrets.mixpanel_token || fallback }}"
		assert.Empty(t, registered.ValidateConfig(config))

		config = validMinimalConfig()
		config["token"] = "bad\ntoken"
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/token", errors[0].Path)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(validFullConfig())
		assert.Empty(t, errors)
	})

	t.Run("example yaml config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"token":              "mp_project_token",
			"data_residency":     "us",
			"identity_merge_api": "original",
			"people":             true,
			"super_properties":   []any{"plan", "workspace_id"},
			"event_filtering": map[string]any{
				"whitelist": []any{"Product Viewed", "Order Completed"},
			},
			"connection_mode": map[string]any{
				"web": "cloud",
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("gdpr_api_token required for task user deletion", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["user_deletion_api"] = "task"
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/gdpr_api_token", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("page template required when custom page event name is enabled", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["use_user_defined_page_event_name"] = true
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/user_defined_page_event_template", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("screen template required when custom screen event name is enabled", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["use_user_defined_screen_event_name"] = true
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/user_defined_screen_event_template", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("session replay percentage pattern", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["session_replay_percentage"] = map[string]any{"web": "101"}
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/session_replay_percentage/web", errors[0].Path)
		assert.Contains(t, errors[0].Message, "0 to 100")

		config = validMinimalConfig()
		config["session_replay_percentage"] = map[string]any{"web": "{{ config.replay || 25 }}"}
		assert.Empty(t, registered.ValidateConfig(config))
	})

	t.Run("array item patterns reject newlines", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["super_properties"] = []any{"plan\nname"}
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/super_properties/0", errors[0].Path)
	})

	t.Run("connection mode validates per source type", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["connection_mode"] = map[string]any{"web": "hybrid"}
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/connection_mode/web", errors[0].Path)

		config = validMinimalConfig()
		config["connection_mode"] = map[string]any{"web": "device", "android": "cloud"}
		assert.Empty(t, registered.ValidateConfig(config))
	})

	t.Run("event filtering whitelist and blacklist are exclusive", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["event_filtering"] = map[string]any{
			"whitelist": []any{"Product Viewed"},
			"blacklist": []any{"Page Viewed"},
		}
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/event_filtering/whitelist", errors[0].Path)
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

	t.Run("legacy consent include keys rejected", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			t.Run(key, func(t *testing.T) {
				t.Parallel()

				config := validMinimalConfig()
				config[key] = map[string]any{"web": []any{}}
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

func TestMPSecretKeysWrapSensitiveValues(t *testing.T) {
	t.Parallel()

	registered := registeredMPDefinition(t)
	config := secret.WrapKnownSecrets(map[string]any{
		"token":                     "raw-token",
		"gdpr_api_token":            "raw-gdpr-token",
		"service_account_secret":    "raw-service-secret",
		"service_account_user_name": "visible-user",
	}, registered.SecretKeys())

	for _, key := range []string{"token", "gdpr_api_token", "service_account_secret"} {
		wrapped, ok := config[key].(*secret.String)
		require.True(t, ok, "expected %s to be wrapped as a secret", key)
		assert.NotContains(t, wrapped.String(), "raw-")
	}
	assert.Equal(t, "visible-user", config["service_account_user_name"])
}

func TestMPConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := mp.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal required fields",
			LocalJSON: `{
				"token": "mp_project_token",
				"data_residency": "us",
				"identity_merge_api": "original"
			}`,
			APIJSON: `{
				"token": "mp_project_token",
				"dataResidency": "us",
				"identityMergeApi": "original"
			}`,
		},
		{
			Name: "full fields",
			LocalJSON: `{
				"token": "mp_project_token",
				"data_residency": "eu",
				"identity_merge_api": "simplified",
				"service_account_user_name": "service-user",
				"service_account_secret": "service-secret",
				"project_id": "123456",
				"user_deletion_api": "task",
				"gdpr_api_token": "gdpr-token",
				"strict_mode": true,
				"ignore_dnt": true,
				"use_user_defined_page_event_name": true,
				"user_defined_page_event_template": "Viewed {{ category }} {{ name }} page",
				"use_user_defined_screen_event_name": true,
				"user_defined_screen_event_template": "Viewed {{ category }} {{ name }} screen",
				"drop_traits_in_track_event": true,
				"people": true,
				"set_all_traits_by_default": true,
				"super_properties": ["plan", "workspace_id"],
				"set_once_properties": ["first_plan"],
				"union_properties": ["roles"],
				"append_properties": ["segments"],
				"people_properties": ["email", "name"],
				"event_increments": ["Signed Up"],
				"prop_increments": ["cart_value"],
				"group_key_settings": ["company"],
				"consolidated_page_calls": false,
				"track_categorized_pages": true,
				"track_named_pages": true,
				"source_name": "Rudder Web",
				"session_replay_percentage": {"web": "25"},
				"cross_subdomain_cookie": true,
				"persistence_type": "localStorage",
				"persistence_name": "mp_cookie",
				"secure_cookie": true,
				"event_filtering": {"whitelist": ["Product Viewed", "Order Completed"]},
				"use_native_sdk": {"web": true},
				"use_new_mapping": true,
				"connection_mode": {"web": "device", "android": "cloud"}
			}`,
			APIJSON: `{
				"token": "mp_project_token",
				"dataResidency": "eu",
				"identityMergeApi": "simplified",
				"serviceAccountUserName": "service-user",
				"serviceAccountSecret": "service-secret",
				"projectId": "123456",
				"userDeletionApi": "task",
				"gdprApiToken": "gdpr-token",
				"strictMode": true,
				"ignoreDnt": true,
				"useUserDefinedPageEventName": true,
				"userDefinedPageEventTemplate": "Viewed {{ category }} {{ name }} page",
				"useUserDefinedScreenEventName": true,
				"userDefinedScreenEventTemplate": "Viewed {{ category }} {{ name }} screen",
				"dropTraitsInTrackEvent": true,
				"people": true,
				"setAllTraitsByDefault": true,
				"superProperties": [{"property": "plan"}, {"property": "workspace_id"}],
				"setOnceProperties": [{"property": "first_plan"}],
				"unionProperties": [{"property": "roles"}],
				"appendProperties": [{"property": "segments"}],
				"peopleProperties": [{"property": "email"}, {"property": "name"}],
				"eventIncrements": [{"property": "Signed Up"}],
				"propIncrements": [{"property": "cart_value"}],
				"groupKeySettings": [{"groupKey": "company"}],
				"consolidatedPageCalls": false,
				"trackCategorizedPages": true,
				"trackNamedPages": true,
				"sourceName": "Rudder Web",
				"sessionReplayPercentage": {"web": "25"},
				"crossSubdomainCookie": true,
				"persistenceType": "localStorage",
				"persistenceName": "mp_cookie",
				"secureCookie": true,
				"eventFilteringOption": "whitelistedEvents",
				"whitelistedEvents": [{"eventName": "Product Viewed"}, {"eventName": "Order Completed"}],
				"useNativeSDK": {"web": true},
				"useNewMapping": true,
				"connectionMode": {"web": "device", "android": "cloud"}
			}`,
		},
		{
			Name: "event filtering blacklist",
			LocalJSON: `{
				"token": "mp_project_token",
				"data_residency": "us",
				"identity_merge_api": "original",
				"event_filtering": {
					"blacklist": ["Internal Event"]
				}
			}`,
			APIJSON: `{
				"token": "mp_project_token",
				"dataResidency": "us",
				"identityMergeApi": "original",
				"eventFilteringOption": "blacklistedEvents",
				"blacklistedEvents": [{"eventName": "Internal Event"}]
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"token": "mp_project_token",
				"data_residency": "us",
				"identity_merge_api": "original",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"token": "mp_project_token",
				"dataResidency": "us",
				"identityMergeApi": "original",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}

func registeredMPDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(mp.NewDefinition()))
	registered, err := registry.Get("mp", 1)
	require.NoError(t, err)
	return registered
}

func validMinimalConfig() map[string]any {
	return map[string]any{
		"token":              "mp_project_token",
		"data_residency":     "us",
		"identity_merge_api": "original",
	}
}

func validFullConfig() map[string]any {
	return map[string]any{
		"token":                              "mp_project_token",
		"data_residency":                     "eu",
		"identity_merge_api":                 "simplified",
		"service_account_user_name":          "service-user",
		"service_account_secret":             "service-secret",
		"project_id":                         "123456",
		"user_deletion_api":                  "task",
		"gdpr_api_token":                     "gdpr-token",
		"strict_mode":                        true,
		"ignore_dnt":                         true,
		"use_user_defined_page_event_name":   true,
		"user_defined_page_event_template":   "Viewed {{ category }} {{ name }} page",
		"use_user_defined_screen_event_name": true,
		"user_defined_screen_event_template": "Viewed {{ category }} {{ name }} screen",
		"drop_traits_in_track_event":         true,
		"people":                             true,
		"set_all_traits_by_default":          true,
		"super_properties":                   []any{"plan", "workspace_id"},
		"set_once_properties":                []any{"first_plan"},
		"union_properties":                   []any{"roles"},
		"append_properties":                  []any{"segments"},
		"people_properties":                  []any{"email", "name"},
		"event_increments":                   []any{"Signed Up"},
		"prop_increments":                    []any{"cart_value"},
		"group_key_settings":                 []any{"company"},
		"consolidated_page_calls":            false,
		"track_categorized_pages":            true,
		"track_named_pages":                  true,
		"source_name":                        "Rudder Web",
		"session_replay_percentage": map[string]any{
			"web": "25",
		},
		"cross_subdomain_cookie": true,
		"persistence_type":       "localStorage",
		"persistence_name":       "mp_cookie",
		"secure_cookie":          true,
		"event_filtering": map[string]any{
			"blacklist": []any{"Internal Event"},
		},
		"use_native_sdk": map[string]any{
			"web": true,
		},
		"use_new_mapping": true,
		"connection_mode": map[string]any{
			"web":     "device",
			"android": "cloud",
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{"provider": "oneTrust", "consents": []any{"analytics"}},
			},
		},
	}
}
