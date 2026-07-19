package zendesk_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/zendesk"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(zendesk.NewDefinition()))

	registered, err := registry.Get("zendesk", 1)
	require.NoError(t, err)

	assert.Equal(t, "zendesk", registered.Type)
	assert.Equal(t, "ZENDESK", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"api_token"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "react_native", "flutter", "cordova", "cloud",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	assert.NotContains(t, registered.SupportedSourceTypes(), "amp")
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")

	byAPI, err := registry.GetByAPIType("ZENDESK", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestZendeskConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(zendesk.NewDefinition()))
	registered, err := registry.Get("zendesk", 1)
	require.NoError(t, err)

	t.Run("missing email", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_token": "zendesk-api-token",
			"domain":    "mycompany",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/email", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing api_token", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"email":  "admin@example.com",
			"domain": "mycompany",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_token", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing domain", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"email":     "admin@example.com",
			"api_token": "zendesk-api-token",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/domain", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"email":     "admin@example.com",
			"api_token": "zendesk-api-token",
			"domain":    "mycompany",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"email":                            "admin@example.com",
			"api_token":                        "zendesk-api-token",
			"domain":                           "mycompany",
			"create_users_as_verified":         true,
			"send_group_calls_without_user_id": true,
			"remove_users_from_organization":   false,
			"search_by_external_id":            true,
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
			"email":                    "admin@example.com",
			"api_token":                "zendesk-api-token",
			"domain":                   "mycompany",
			"create_users_as_verified": true,
			"search_by_external_id":    true,
		})
		assert.Empty(t, errors)
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"email":       "admin@example.com",
			"api_token":   "zendesk-api-token",
			"domain":      "mycompany",
			"source_name": "rudder",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/source_name", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"email":     "admin@example.com",
			"api_token": "zendesk-api-token",
			"domain":    "mycompany",
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
			"email":     "admin@example.com",
			"api_token": "zendesk-api-token",
			"domain":    "mycompany",
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

func TestZendeskConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := zendesk.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal required fields",
			LocalJSON: `{
				"email": "admin@example.com",
				"api_token": "zendesk-api-token",
				"domain": "mycompany"
			}`,
			APIJSON: `{
				"email": "admin@example.com",
				"apiToken": "zendesk-api-token",
				"domain": "mycompany"
			}`,
		},
		{
			Name: "full TF fields",
			LocalJSON: `{
				"email": "admin@example.com",
				"api_token": "zendesk-api-token",
				"domain": "mycompany",
				"create_users_as_verified": true,
				"send_group_calls_without_user_id": true,
				"remove_users_from_organization": true,
				"search_by_external_id": true
			}`,
			APIJSON: `{
				"email": "admin@example.com",
				"apiToken": "zendesk-api-token",
				"domain": "mycompany",
				"createUsersAsVerified": true,
				"sendGroupCallsWithoutUserId": true,
				"removeUsersFromOrganization": true,
				"searchByExternalId": true
			}`,
		},
		{
			Name: "consent for web",
			LocalJSON: `{
				"email": "admin@example.com",
				"api_token": "zendesk-api-token",
				"domain": "mycompany",
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
				"email": "admin@example.com",
				"apiToken": "zendesk-api-token",
				"domain": "mycompany",
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
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"email": "admin@example.com",
				"api_token": "zendesk-api-token",
				"domain": "mycompany",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"email": "admin@example.com",
				"apiToken": "zendesk-api-token",
				"domain": "mycompany",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
