package sentry_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/sentry"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(sentry.NewDefinition()))

	registered, err := registry.Get("sentry", 1)
	require.NoError(t, err)

	assert.Equal(t, "sentry", registered.Type)
	assert.Equal(t, "SENTRY", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys())

	expectedSourceTypes := []string{"web"}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	modes, err := registered.ConnectionModes("web")
	require.NoError(t, err)
	assert.Equal(t, []string{"device"}, modes)

	assert.NotContains(t, registered.SupportedSourceTypes(), "amp")
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")

	assert.Empty(t, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("SENTRY", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestSentryConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(sentry.NewDefinition()))
	registered, err := registry.Get("sentry", 1)
	require.NoError(t, err)

	t.Run("missing dsn", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/dsn", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("dsn too long rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dsn": strings.Repeat("a", 301),
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/dsn", errors[0].Path)
		assert.Contains(t, errors[0].Message, "300")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dsn": "https://public@o0.ingest.sentry.io/0",
		})
		assert.Empty(t, errors)
	})

	t.Run("ignore_errors item too long rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dsn":           "https://public@o0.ingest.sentry.io/0",
			"ignore_errors": []any{strings.Repeat("e", 101)},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/ignore_errors/0", errors[0].Path)
		assert.Contains(t, errors[0].Message, "100")
	})

	t.Run("allow_urls with ngrok rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dsn":        "https://public@o0.ingest.sentry.io/0",
			"allow_urls": []any{"https://example.ngrok.io/app"},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/allow_urls/0", errors[0].Path)
		assert.Contains(t, errors[0].Message, "ngrok")
	})

	t.Run("deny_urls with ngrok rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dsn":       "https://public@o0.ingest.sentry.io/0",
			"deny_urls": []any{"https://evil.ngrok.io"},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/deny_urls/0", errors[0].Path)
		assert.Contains(t, errors[0].Message, "ngrok")
	})

	t.Run("event_filtering whitelist item too long rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dsn": "https://public@o0.ingest.sentry.io/0",
			"event_filtering": map[string]any{
				"whitelist": []any{strings.Repeat("e", 101)},
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/event_filtering/whitelist/0", errors[0].Path)
		assert.Contains(t, errors[0].Message, "100")
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dsn":                     "https://public@o0.ingest.sentry.io/0",
			"environment":             "production",
			"custom_version_property": "app.version",
			"release":                 "1.2.3",
			"server_name":             "api-1",
			"logger":                  "rudderstack",
			"debug_mode":              false,
			"ignore_errors":           []any{"NetworkError", "ResizeObserver"},
			"include_paths":           []any{"https://app.example.com"},
			"allow_urls":              []any{"https://app.example.com"},
			"deny_urls":               []any{"https://cdn.example.com"},
			"event_filtering": map[string]any{
				"whitelist": []any{"Error Captured", "Exception Thrown"},
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

	t.Run("example yaml config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dsn":                     "https://public@o0.ingest.sentry.io/0",
			"environment":             "production",
			"custom_version_property": "app.version",
			"release":                 "1.2.3",
			"server_name":             "web-1",
			"logger":                  "rudderstack",
			"debug_mode":              false,
			"ignore_errors":           []any{"NetworkError"},
			"include_paths":           []any{"https://app.example.com"},
			"allow_urls":              []any{"https://app.example.com"},
			"deny_urls":               []any{"https://cdn.example.com"},
			"event_filtering": map[string]any{
				"whitelist": []any{"Error Captured"},
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

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dsn":         "https://public@o0.ingest.sentry.io/0",
			"not_a_field": true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dsn": "https://public@o0.ingest.sentry.io/0",
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
			"dsn": "https://public@o0.ingest.sentry.io/0",
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

func TestSentryConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := sentry.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal dsn",
			LocalJSON: `{
				"dsn": "https://public@o0.ingest.sentry.io/0"
			}`,
			APIJSON: `{
				"dsn": "https://public@o0.ingest.sentry.io/0"
			}`,
		},
		{
			Name: "full fields",
			LocalJSON: `{
				"dsn": "https://public@o0.ingest.sentry.io/0",
				"environment": "production",
				"custom_version_property": "app.version",
				"release": "1.2.3",
				"server_name": "api-1",
				"logger": "rudderstack",
				"debug_mode": true,
				"ignore_errors": ["NetworkError", "ResizeObserver"],
				"include_paths": ["https://app.example.com"],
				"allow_urls": ["https://app.example.com"],
				"deny_urls": ["https://cdn.example.com"],
				"event_filtering": {
					"whitelist": ["Error Captured", "Exception Thrown"]
				},
				"use_native_sdk": {"web": true}
			}`,
			APIJSON: `{
				"dsn": "https://public@o0.ingest.sentry.io/0",
				"environment": "production",
				"customVersionProperty": "app.version",
				"release": "1.2.3",
				"serverName": "api-1",
				"logger": "rudderstack",
				"debugMode": true,
				"ignoreErrors": [
					{"ignoreErrors": "NetworkError"},
					{"ignoreErrors": "ResizeObserver"}
				],
				"includePaths": [
					{"includePaths": "https://app.example.com"}
				],
				"allowUrls": [
					{"allowUrls": "https://app.example.com"}
				],
				"denyUrls": [
					{"denyUrls": "https://cdn.example.com"}
				],
				"eventFilteringOption": "whitelistedEvents",
				"whitelistedEvents": [
					{"eventName": "Error Captured"},
					{"eventName": "Exception Thrown"}
				],
				"useNativeSDK": {"web": true}
			}`,
		},
		{
			Name: "event filtering blacklist",
			LocalJSON: `{
				"dsn": "https://public@o0.ingest.sentry.io/0",
				"event_filtering": {
					"blacklist": ["Application Opened"]
				}
			}`,
			APIJSON: `{
				"dsn": "https://public@o0.ingest.sentry.io/0",
				"eventFilteringOption": "blacklistedEvents",
				"blacklistedEvents": [
					{"eventName": "Application Opened"}
				]
			}`,
		},
		{
			Name: "consent for web",
			LocalJSON: `{
				"dsn": "https://public@o0.ingest.sentry.io/0",
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
				"dsn": "https://public@o0.ingest.sentry.io/0",
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
