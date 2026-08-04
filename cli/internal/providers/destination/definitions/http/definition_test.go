package http_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	httpdest "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/http"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(httpdest.NewDefinition()))

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
	assert.Empty(t, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("HTTP", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestHTTPConfigValidation(t *testing.T) {
	t.Parallel()

	registered := registeredHTTPDefinition(t)

	for _, tc := range []struct {
		name string
		key  string
		path string
	}{
		{name: "missing api_url", key: "api_url", path: "/api_url"},
		{name: "missing auth", key: "auth", path: "/auth"},
		{name: "missing method", key: "method", path: "/method"},
		{name: "missing format", key: "format", path: "/format"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := validMinimalConfig()
			delete(config, tc.key)

			errors := registered.ValidateConfig(config)
			require.NotEmpty(t, errors)
			assert.Equal(t, tc.path, errors[0].Path)
			assert.Contains(t, errors[0].Message, "required")
		})
	}

	t.Run("valid noAuth minimal config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(validMinimalConfig())
		assert.Empty(t, errors)
	})

	t.Run("validated example config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(exampleConfig())
		assert.Empty(t, errors)
	})

	t.Run("invalid api_url rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["api_url"] = "http://localhost:8080/webhook"

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_url", errors[0].Path)
		assert.Contains(t, errors[0].Message, "public http(s) domain URL")
	})

	t.Run("ngrok api_url rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["api_url"] = "https://demo.ngrok.io/webhook"

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_url", errors[0].Path)
	})

	t.Run("invalid auth rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["auth"] = "oauth"

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/auth", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("invalid method rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["method"] = "OPTIONS"

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/method", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("invalid format rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["format"] = "CSV"

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/format", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("valid basicAuth", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["auth"] = "basicAuth"
		config["username"] = "rudder"
		config["password"] = "secret"

		errors := registered.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("basicAuth requires username and password", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["auth"] = "basicAuth"

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 2)
		assertValidationPaths(t, errors, "/username", "/password")
	})

	t.Run("valid bearerTokenAuth", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["auth"] = "bearerTokenAuth"
		config["bearer_token"] = "bearer-token"

		errors := registered.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("bearerTokenAuth requires bearer_token", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["auth"] = "bearerTokenAuth"

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/bearer_token", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid apiKeyAuth", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["auth"] = "apiKeyAuth"
		config["api_key_name"] = "X-Api-Key"
		config["api_key_value"] = "api-key-value"

		errors := registered.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("apiKeyAuth requires api key name and value", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["auth"] = "apiKeyAuth"

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 2)
		assertValidationPaths(t, errors, "/api_key_name", "/api_key_value")
	})

	t.Run("api_key_name rejects whitespace", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["auth"] = "apiKeyAuth"
		config["api_key_name"] = "X Api Key"
		config["api_key_value"] = "api-key-value"

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_key_name", errors[0].Path)
	})

	t.Run("batching enabled requires string max_batch_size", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["is_batching_enabled"] = true
		config["max_batch_size"] = "100"

		errors := registered.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("batching enabled requires max_batch_size", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["is_batching_enabled"] = true

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/max_batch_size", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("numeric max_batch_size rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["is_batching_enabled"] = true
		config["max_batch_size"] = 100

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/max_batch_size", errors[0].Path)
		assert.Contains(t, errors[0].Message, "expected type 'string'")
	})

	t.Run("max_batch_size range rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["is_batching_enabled"] = true
		config["max_batch_size"] = "101"

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/max_batch_size", errors[0].Path)
	})

	t.Run("valid XML with xml_root_key", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["format"] = "XML"
		config["xml_root_key"] = "rudderEvent"

		errors := registered.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("xml_root_key length rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["format"] = "XML"
		config["xml_root_key"] = stringOfLength(101)

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/xml_root_key", errors[0].Path)
	})

	t.Run("valid mapping arrays", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["is_default_mapping"] = false
		config["properties_mapping"] = []any{map[string]any{"to": "$.traits.email", "from": "$.context.traits.email"}}
		config["query_params"] = []any{map[string]any{"to": "plan", "from": "$.context.plan"}}
		config["headers"] = []any{map[string]any{"to": "X-Source", "from": "rudder/web"}}
		config["path_params"] = []any{map[string]any{"path": "$.userId"}}

		errors := registered.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("invalid properties_mapping nested field rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["properties_mapping"] = []any{map[string]any{"to": "not-json-path", "from": "$.userId"}}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/properties_mapping/0/to", errors[0].Path)
	})

	t.Run("invalid query_params nested field rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["query_params"] = []any{map[string]any{"to": "$.bad", "from": "value"}}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/query_params/0/to", errors[0].Path)
	})

	t.Run("invalid headers nested field rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["headers"] = []any{map[string]any{"to": "X Header", "from": "value"}}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/headers/0/to", errors[0].Path)
	})

	t.Run("invalid path_params nested field rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["path_params"] = []any{map[string]any{"path": "bad path"}}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/path_params/0/path", errors[0].Path)
	})

	t.Run("valid event filtering whitelist", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["event_filtering"] = map[string]any{"whitelist": []any{"Order Completed", "Product Viewed"}}

		errors := registered.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("event filtering rejects whitelist and blacklist together", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["event_filtering"] = map[string]any{
			"whitelist": []any{"Order Completed"},
			"blacklist": []any{"Product Viewed"},
		}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assertValidationPaths(t, errors, "/event_filtering/whitelist", "/event_filtering/blacklist")
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

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["consent_management"] = map[string]any{"warehouse": []any{}}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'warehouse' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["consent_management"] = map[string]any{
			"web": []any{map[string]any{"provider": "unknown"}},
		}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestHTTPConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := httpdest.NewDefinition()
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
			Name: "basic auth",
			LocalJSON: `{
				"api_url": "https://example.com/webhook",
				"auth": "basicAuth",
				"username": "rudder",
				"password": "secret",
				"method": "POST",
				"format": "JSON"
			}`,
			APIJSON: `{
				"apiUrl": "https://example.com/webhook",
				"auth": "basicAuth",
				"username": "rudder",
				"password": "secret",
				"method": "POST",
				"format": "JSON"
			}`,
		},
		{
			Name: "bearer auth",
			LocalJSON: `{
				"api_url": "https://example.com/webhook",
				"auth": "bearerTokenAuth",
				"bearer_token": "bearer-token",
				"method": "PUT",
				"format": "JSON"
			}`,
			APIJSON: `{
				"apiUrl": "https://example.com/webhook",
				"auth": "bearerTokenAuth",
				"bearerToken": "bearer-token",
				"method": "PUT",
				"format": "JSON"
			}`,
		},
		{
			Name: "api key auth",
			LocalJSON: `{
				"api_url": "https://example.com/webhook",
				"auth": "apiKeyAuth",
				"api_key_name": "X-Api-Key",
				"api_key_value": "api-key-value",
				"method": "PATCH",
				"format": "JSON"
			}`,
			APIJSON: `{
				"apiUrl": "https://example.com/webhook",
				"auth": "apiKeyAuth",
				"apiKeyName": "X-Api-Key",
				"apiKeyValue": "api-key-value",
				"method": "PATCH",
				"format": "JSON"
			}`,
		},
		{
			Name: "mapping arrays and batching",
			LocalJSON: `{
				"api_url": "https://example.com/webhook",
				"auth": "noAuth",
				"xml_root_key": "rudderEvent",
				"method": "POST",
				"format": "XML",
				"properties_mapping": [{"to": "$.traits.email", "from": "$.context.traits.email"}],
				"query_params": [{"to": "plan", "from": "$.context.plan"}],
				"headers": [{"to": "X-Source", "from": "rudder/web"}],
				"path_params": [{"path": "$.userId"}],
				"is_batching_enabled": true,
				"max_batch_size": "25",
				"is_default_mapping": false
			}`,
			APIJSON: `{
				"apiUrl": "https://example.com/webhook",
				"auth": "noAuth",
				"xmlRootKey": "rudderEvent",
				"method": "POST",
				"format": "XML",
				"propertiesMapping": [{"to": "$.traits.email", "from": "$.context.traits.email"}],
				"queryParams": [{"to": "plan", "from": "$.context.plan"}],
				"headers": [{"to": "X-Source", "from": "rudder/web"}],
				"pathParams": [{"path": "$.userId"}],
				"isBatchingEnabled": true,
				"maxBatchSize": "25",
				"isDefaultMapping": false
			}`,
		},
		{
			Name: "event filtering whitelist",
			LocalJSON: `{
				"api_url": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON",
				"event_filtering": {"whitelist": ["Order Completed", "Product Viewed"]}
			}`,
			APIJSON: `{
				"apiUrl": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON",
				"whitelistedEvents": [{"eventName": "Order Completed"}, {"eventName": "Product Viewed"}],
				"eventFilteringOption": "whitelistedEvents"
			}`,
		},
		{
			Name: "event filtering blacklist",
			LocalJSON: `{
				"api_url": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON",
				"event_filtering": {"blacklist": ["Order Cancelled"]}
			}`,
			APIJSON: `{
				"apiUrl": "https://example.com/webhook",
				"auth": "noAuth",
				"method": "POST",
				"format": "JSON",
				"blacklistedEvents": [{"eventName": "Order Cancelled"}],
				"eventFilteringOption": "blacklistedEvents"
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

func TestHTTPSecretKeysUseLocalConfigKeys(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(httpdest.NewDefinition()))

	h := destination.NewHandler(nil, registry)
	extracted, err := h.Impl.ExtractResourcesFromSpec("destinations/http.yaml", &destination.DestinationSpec{
		ID:                "http-production",
		DisplayName:       "HTTP Production",
		Type:              "http",
		Enabled:           true,
		DefinitionVersion: 1,
		Config: map[string]any{
			"api_url":       "https://example.com/webhook",
			"auth":          "apiKeyAuth",
			"method":        "POST",
			"format":        "JSON",
			"password":      "password-value",
			"bearer_token":  "bearer-token-value",
			"api_key_name":  "X-Api-Key",
			"api_key_value": "api-key-secret-value",
		},
	})
	require.NoError(t, err)

	config := extracted["http-production"].Config
	assertWrappedSecret(t, config, "password", "password-value")
	assertWrappedSecret(t, config, "bearer_token", "bearer-token-value")
	assertWrappedSecret(t, config, "api_key_value", "api-key-secret-value")
	assert.Equal(t, "X-Api-Key", config["api_key_name"])
}

func TestHTTPRemoteSecretsAreUnknownAndRedacted(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(httpdest.NewDefinition()))

	h := destination.NewHandler(nil, registry)
	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID:         "dst-http",
		ExternalID: "http-production",
		Name:       "HTTP Production",
		Type:       "HTTP",
		Version:    1,
		IsEnabled:  true,
		Config: []byte(`{
			"apiUrl": "https://example.com/webhook",
			"auth": "apiKeyAuth",
			"method": "POST",
			"format": "JSON",
			"bearerToken": "",
			"apiKeyName": "X-Api-Key",
			"apiKeyValue": ""
		}`),
	}}

	resource, _, err := h.Impl.MapRemoteToState(remote, urnResolver{})
	require.NoError(t, err)

	bearerToken := requireSecret(t, resource.Config, "bearer_token")
	apiKeyValue := requireSecret(t, resource.Config, "api_key_value")
	assert.True(t, bearerToken.IsUnknown())
	assert.True(t, apiKeyValue.IsUnknown())
	assert.NotContains(t, fmt.Sprintf("%v %v", bearerToken, apiKeyValue), "api-key")
	assert.NotContains(t, mustJSON(t, resource.Config), "api-key")
}

func registeredHTTPDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(httpdest.NewDefinition()))
	registered, err := registry.Get("http", 1)
	require.NoError(t, err)
	return registered
}

func validMinimalConfig() map[string]any {
	return map[string]any{
		"api_url": "https://example.com/webhook",
		"auth":    "noAuth",
		"method":  "POST",
		"format":  "JSON",
	}
}

func exampleConfig() map[string]any {
	return map[string]any{
		"api_url":             "https://webhooks.example.com/rudder/events",
		"auth":                "apiKeyAuth",
		"api_key_name":        "X-Api-Key",
		"api_key_value":       "{{ .HTTP_API_KEY_VALUE }}",
		"method":              "POST",
		"format":              "JSON",
		"is_default_mapping":  false,
		"properties_mapping":  []any{map[string]any{"to": "$.email", "from": "$.context.traits.email"}},
		"query_params":        []any{map[string]any{"to": "source", "from": "rudder"}},
		"headers":             []any{map[string]any{"to": "X-Rudder-Source", "from": "rudder-cli"}},
		"path_params":         []any{map[string]any{"path": "$.userId"}},
		"is_batching_enabled": true,
		"max_batch_size":      "25",
		"event_filtering":     map[string]any{"whitelist": []any{"Order Completed", "Product Viewed"}},
		"consent_management":  map[string]any{"web": []any{map[string]any{"provider": "oneTrust", "consents": []any{"analytics"}}}},
	}
}

func assertValidationPaths(t *testing.T, errors []definitions.ConfigError, paths ...string) {
	t.Helper()

	byPath := map[string]struct{}{}
	for _, err := range errors {
		byPath[err.Path] = struct{}{}
	}
	for _, path := range paths {
		assert.Contains(t, byPath, path)
	}
}

func assertWrappedSecret(t *testing.T, config map[string]any, key string, want string) {
	t.Helper()

	s := requireSecret(t, config, key)
	assert.False(t, s.IsUnknown())
	assert.Equal(t, want, s.Reveal())
	assert.NotContains(t, fmt.Sprintf("%v", s), want)
}

func requireSecret(t *testing.T, config map[string]any, key string) *secret.String {
	t.Helper()

	v, ok := config[key]
	require.True(t, ok, "missing secret key %q", key)
	s, ok := v.(*secret.String)
	require.True(t, ok, "key %q: expected *secret.String, got %T", key, v)
	require.NotNil(t, s)
	return s
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()

	b, err := json.Marshal(value)
	require.NoError(t, err)
	return string(b)
}

func stringOfLength(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'x'
	}
	return string(b)
}

type urnResolver struct{}

func (urnResolver) GetURNByID(string, string) (string, error) {
	return "", resources.ErrRemoteResourceExternalIdNotFound
}
