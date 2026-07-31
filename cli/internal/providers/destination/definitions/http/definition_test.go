package http_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/http"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(http.NewDefinition()))

	registered, err := registry.Get("http", 1)
	require.NoError(t, err)

	assert.Equal(t, "http", registered.Type)
	assert.Equal(t, "HTTP", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"password", "bearer_token", "api_key_value"}, registered.SecretKeys())

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

	byAPI, err := registry.GetByAPIType("HTTP", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestHTTPConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(http.NewDefinition()))
	registered, err := registry.Get("http", 1)
	require.NoError(t, err)

	validMinimal := func() map[string]any {
		return map[string]any{
			"api_url": "https://example.com/webhook",
			"auth":    "noAuth",
			"method":  "POST",
			"format":  "JSON",
		}
	}

	t.Run("valid minimal", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(validMinimal())
		assert.Empty(t, errors)
	})

	t.Run("missing api_url", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		delete(cfg, "api_url")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_url", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing auth", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		delete(cfg, "auth")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/auth", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing method", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		delete(cfg, "method")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/method", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing format", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		delete(cfg, "format")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/format", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("invalid api_url is rejected", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		cfg["api_url"] = "ftp://example.com"
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_url", errors[0].Path)
	})

	t.Run("localhost api_url is rejected", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		cfg["api_url"] = "https://localhost:8080/webhook"
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_url", errors[0].Path)
	})

	t.Run("ngrok api_url is rejected", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		cfg["api_url"] = "https://myapp.ngrok.io/webhook"
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_url", errors[0].Path)
	})

	t.Run("invalid auth is rejected", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		cfg["auth"] = "oauth"
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/auth", errors[0].Path)
	})

	t.Run("invalid method is rejected", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		cfg["method"] = "HEAD"
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/method", errors[0].Path)
	})

	t.Run("invalid format is rejected", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		cfg["format"] = "CSV"
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/format", errors[0].Path)
	})

	t.Run("invalid max_batch_size is rejected", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		cfg["max_batch_size"] = "500"
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/max_batch_size", errors[0].Path)
	})

	t.Run("dynamic value satisfies enum", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		cfg["method"] = "{{ .HTTP_METHOD }}"
		errors := registered.ValidateConfig(cfg)
		assert.Empty(t, errors)
	})

	t.Run("secrets via var substitution", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		cfg["auth"] = "basicAuth"
		cfg["username"] = "svc-account"
		cfg["password"] = "{{ .HTTP_PASSWORD }}"
		errors := registered.ValidateConfig(cfg)
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_url":       "https://example.com/collect",
			"auth":          "apiKeyAuth",
			"api_key_name":  "X-Api-Key",
			"api_key_value": "secret-key-value",
			"xml_root_key":  "root",
			"method":        "PUT",
			"format":        "XML",
			"properties_mapping": []any{
				map[string]any{"to": "userId", "from": "$.userId"},
			},
			"query_params": []any{
				map[string]any{"to": "source", "from": "rudderstack"},
			},
			"headers": []any{
				map[string]any{"to": "X-Trace", "from": "$.messageId"},
			},
			"path_params": []any{
				map[string]any{"path": "$.userId"},
			},
			"is_batching_enabled": true,
			"max_batch_size":      "50",
			"event_filtering": map[string]any{
				"whitelist": []any{"Product Viewed", "Order Completed"},
			},
			"is_default_mapping": false,
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
		cfg := validMinimal()
		cfg["not_a_field"] = true
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		cfg["consent_management"] = map[string]any{
			"warehouse": []any{},
		}
		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'warehouse' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		cfg := validMinimal()
		cfg["consent_management"] = map[string]any{
			"ios_swift": []any{
				map[string]any{"provider": "unknown"},
			},
		}
		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/ios_swift/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestHTTPConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := http.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"api_url": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON"
			}`,
			APIJSON: `{
				"apiUrl": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON"
			}`,
		},
		{
			Name: "auth and secrets",
			LocalJSON: `{
				"api_url": "https://example.com/webhook",
				"auth": "apiKeyAuth",
				"username": "svc",
				"password": "pw",
				"bearer_token": "tok",
				"api_key_name": "X-Api-Key",
				"api_key_value": "val",
				"xml_root_key": "root",
				"method": "PUT",
				"format": "XML"
			}`,
			APIJSON: `{
				"apiUrl": "https://example.com/webhook",
				"auth": "apiKeyAuth",
				"username": "svc",
				"password": "pw",
				"bearerToken": "tok",
				"apiKeyName": "X-Api-Key",
				"apiKeyValue": "val",
				"xmlRootKey": "root",
				"method": "PUT",
				"format": "XML"
			}`,
		},
		{
			Name: "object arrays reshape",
			LocalJSON: `{
				"api_url": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON",
				"properties_mapping": [{"to": "userId", "from": "$.userId"}],
				"query_params": [{"to": "source", "from": "rudderstack"}],
				"headers": [{"to": "X-Trace", "from": "$.messageId"}],
				"path_params": [{"path": "$.userId"}]
			}`,
			APIJSON: `{
				"apiUrl": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON",
				"propertiesMapping": [{"to": "userId", "from": "$.userId"}],
				"queryParams": [{"to": "source", "from": "rudderstack"}],
				"headers": [{"to": "X-Trace", "from": "$.messageId"}],
				"pathParams": [{"path": "$.userId"}]
			}`,
		},
		{
			Name: "event filtering whitelist discriminator",
			LocalJSON: `{
				"api_url": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON",
				"event_filtering": {"whitelist": ["Product Viewed", "Order Completed"]}
			}`,
			APIJSON: `{
				"apiUrl": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON",
				"whitelistedEvents": [{"eventName": "Product Viewed"}, {"eventName": "Order Completed"}],
				"eventFilteringOption": "whitelistedEvents"
			}`,
		},
		{
			Name: "event filtering blacklist discriminator",
			LocalJSON: `{
				"api_url": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON",
				"event_filtering": {"blacklist": ["Internal Event"]}
			}`,
			APIJSON: `{
				"apiUrl": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON",
				"blacklistedEvents": [{"eventName": "Internal Event"}],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "batching and default mapping bools",
			LocalJSON: `{
				"api_url": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON",
				"is_batching_enabled": true,
				"max_batch_size": "50",
				"is_default_mapping": false
			}`,
			APIJSON: `{
				"apiUrl": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON",
				"isBatchingEnabled": true,
				"maxBatchSize": "50",
				"isDefaultMapping": false
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"api_url": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"apiUrl": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
