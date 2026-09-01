package gtm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/gtm"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(gtm.NewDefinition()))

	registered, err := registry.Get("gtm", 1)
	require.NoError(t, err)

	assert.Equal(t, "gtm", registered.Type)
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

func TestGTMConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(gtm.NewDefinition()))
	registered, err := registry.Get("gtm", 1)
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
			"container_id":        "GTM-XXXXXXX",
			"server_url":          "https://gtm.example.com",
			"environment_id":      "env-5",
			"authorization_token": "gtmAuthToken",
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
		assert.Contains(t, errors[0].Message, "must be a domain URL")
	})

	t.Run("server_url accepts empty value", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"container_id": "GTM-XXXXXXX",
			"server_url":   "",
		})
		assert.Empty(t, errors)
	})

	t.Run("server_url rejects ngrok tunnel", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"container_id": "GTM-XXXXXXX",
			"server_url":   "https://abcd.ngrok.io",
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/server_url", errors[0].Path)
	})

	t.Run("server_url accepts a scheme-less host with a path", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"container_id": "GTM-XXXXXXX",
			"server_url":   "sgtm.example.com/collect",
		})
		assert.Empty(t, errors)
	})

	t.Run("server_url rejects a bare host", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"container_id": "GTM-XXXXXXX",
			"server_url":   "localhost",
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/server_url", errors[0].Path)
	})

	t.Run("server_url accepts template", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"container_id": "GTM-XXXXXXX",
			"server_url":   "{{ .gtm.serverUrl || \"https://sgtm.example.com\" }}",
		})
		assert.Empty(t, errors)
	})

	t.Run("container_id rejects line break", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"container_id": "GTM-XXX\nYYY",
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/container_id", errors[0].Path)
	})

	t.Run("container_id accepts template", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"container_id": "{{ .gtm.containerID || \"GTM-XXXXXXX\" }}",
		})
		assert.Empty(t, errors)
	})

	t.Run("environment_id rejects line break", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"container_id":   "GTM-XXXXXXX",
			"environment_id": "env\n5",
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/environment_id", errors[0].Path)
	})

	t.Run("authorization_token rejects line break", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"container_id":        "GTM-XXXXXXX",
			"authorization_token": "token\nvalue",
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/authorization_token", errors[0].Path)
	})

	t.Run("whitelist and blacklist are mutually exclusive", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"container_id": "GTM-XXXXXXX",
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
	// connection_mode legality is per source type, taken from this definition's
	// own ConnectionModes map rather than a shared enum.
	t.Run("connection_mode accepts a supported mode", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"connection_mode": map[string]any{"web": "device"},
		})

		for _, err := range errors {
			assert.NotEqual(t, "/connection_mode/web", err.Path)
		}
	})

	t.Run("connection_mode rejects an unsupported mode", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"connection_mode": map[string]any{"web": "cloud"},
		})

		var found bool
		for _, err := range errors {
			if err.Path == "/connection_mode/web" {
				found = true
				assert.Contains(t, err.Message, "must be one of")
			}
		}
		assert.True(t, found, "expected /connection_mode/web to be rejected")
	})

}

func TestGTMConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := gtm.NewDefinition()
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
			Name: "full config with whitelist",
			LocalJSON: `{
				"container_id": "GTM-XXXXXXX",
				"server_url": "https://gtm.example.com",
				"environment_id": "env-5",
				"authorization_token": "gtmAuthToken",
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
				"environmentID": "env-5",
				"authorizationToken": "gtmAuthToken",
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
