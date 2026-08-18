package webhook_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/webhook"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(webhook.NewDefinition()))

	registered, err := registry.Get("webhook", 1)
	require.NoError(t, err)

	assert.Equal(t, "webhook", registered.Type)
	assert.Equal(t, "WEBHOOK", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"headers.to"}, registered.SecretKeys())
	assert.Empty(t, registered.GatedKeyPaths())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web", "unity", "amp",
		"cloud", "warehouse", "react_native", "flutter", "cordova", "shopify",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	byAPI, err := registry.GetByAPIType("WEBHOOK", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestWebhookConfigValidation(t *testing.T) {
	t.Parallel()

	registered := registeredWebhookDefinition(t)

	t.Run("missing webhook_url", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{})

		require.NotEmpty(t, errors)
		assert.Equal(t, "/webhook_url", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("webhook_url rejects invalid literals", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name  string
			value string
		}{
			{name: "missing protocol", value: "webhooks.example.com/rudder"},
			{name: "localhost", value: "https://localhost:8080/rudder"},
			{name: "localhost subdomain", value: "https://api.localhost/rudder"},
			{name: "ngrok", value: "https://rudder.ngrok.io/rudder"},
			{name: "deprecated env reference", value: "env.WEBHOOK_URL"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				config := validMinimalConfig()
				config["webhook_url"] = tc.value

				errors := registered.ValidateConfig(config)
				require.NotEmpty(t, errors)
				assert.Equal(t, "/webhook_url", errors[0].Path)
			})
		}
	})

	t.Run("webhook_url accepts UI template", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["webhook_url"] = `{{ .WEBHOOK_URL || "https://webhooks.example.com/rudder" }}`

		assert.Empty(t, registered.ValidateConfig(config))
	})

	t.Run("webhook_method validates schema enum", func(t *testing.T) {
		t.Parallel()

		for _, method := range []string{"POST", "PUT", "PATCH", "GET", "DELETE"} {
			config := validMinimalConfig()
			config["webhook_method"] = method
			assert.Empty(t, registered.ValidateConfig(config), method)
		}

		config := validMinimalConfig()
		config["webhook_method"] = "OPTIONS"
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/webhook_method", errors[0].Path)
	})

	t.Run("headers validate nested single line patterns", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["headers"] = []any{
			map[string]any{"from": "X-Api-Key", "to": "{{ .WEBHOOK_HEADER_VALUE || safe-dummy-value }}"},
			map[string]any{"from": "X-Trace", "to": "trace-value"},
		}
		assert.Empty(t, registered.ValidateConfig(config))

		config = validMinimalConfig()
		config["headers"] = []any{map[string]any{"from": "X-Bad\nHeader", "to": "value"}}
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/headers/0/from", errors[0].Path)

		config = validMinimalConfig()
		config["headers"] = []any{map[string]any{"from": "X-Api-Key", "to": stringOfLength(1001)}}
		errors = registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/headers/0/to", errors[0].Path)
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(validMinimalConfig()))
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(validFullConfig()))
	})

	t.Run("valid example config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(exampleConfig()))
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
		config["consent_management"] = map[string]any{"cloud_source": []any{}}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/cloud_source", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'cloud_source' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["consent_management"] = map[string]any{
			"android_kotlin": []any{map[string]any{"provider": "unknown"}},
		}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/android_kotlin/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestWebhookConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := webhook.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"webhook_url": "https://webhooks.example.com/rudder"
			}`,
			APIJSON: `{
				"webhookUrl": "https://webhooks.example.com/rudder"
			}`,
		},
		{
			Name: "full",
			LocalJSON: `{
				"webhook_url": "https://webhooks.example.com/rudder",
				"webhook_method": "PATCH",
				"headers": [
					{"from": "X-Api-Key", "to": "{{ .WEBHOOK_HEADER_VALUE }}"},
					{"from": "X-Trace", "to": "rudder-cli"}
				]
			}`,
			APIJSON: `{
				"webhookUrl": "https://webhooks.example.com/rudder",
				"webhookMethod": "PATCH",
				"headers": [
					{"from": "X-Api-Key", "to": "{{ .WEBHOOK_HEADER_VALUE }}"},
					{"from": "X-Trace", "to": "rudder-cli"}
				]
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"webhook_url": "https://webhooks.example.com/rudder",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}],
					"warehouse": [{"provider": "custom", "resolution_strategy": "or", "consents": ["analytics"]}]
				}
			}`,
			APIJSON: `{
				"webhookUrl": "https://webhooks.example.com/rudder",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}],
					"warehouse": [{"provider": "custom", "resolutionStrategy": "or", "consents": [{"consent": "analytics"}]}]
				}
			}`,
		},
	})
}

func TestWebhookHeaderSecretsAreWrappedRevealedAndMasked(t *testing.T) {
	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(webhook.NewDefinition()))
	registered, err := registry.Get("webhook", 1)
	require.NoError(t, err)

	h := destination.NewHandler(nil, registry)
	extracted, err := h.Impl.ExtractResourcesFromSpec("destinations/webhook.yaml", &destination.DestinationSpec{
		ID:                "webhook-production",
		DisplayName:       "Webhook Production",
		Type:              "webhook",
		Enabled:           true,
		DefinitionVersion: 1,
		Config: map[string]any{
			"webhook_url":    "https://webhooks.example.com/rudder",
			"webhook_method": "POST",
			"headers": []any{
				map[string]any{"from": "X-Api-Key", "to": "safe-dummy-secret-value"},
			},
		},
	})
	require.NoError(t, err)

	config := extracted["webhook-production"].Config
	headers, ok := config["headers"].([]any)
	require.True(t, ok)
	header := headers[0].(map[string]any)
	wrapped, ok := header["to"].(*secret.String)
	require.True(t, ok)
	assert.Equal(t, "safe-dummy-secret-value", wrapped.Reveal())
	assert.Equal(t, "X-Api-Key", header["from"])

	apiConfig, err := registered.LocalToAPI(secret.RevealSecrets(config, []string{"headers.to"}))
	require.NoError(t, err)
	assert.Equal(t, []any{map[string]any{"from": "X-Api-Key", "to": "safe-dummy-secret-value"}}, apiConfig["headers"])

	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID:         "dst-webhook",
		ExternalID: "webhook-production",
		Name:       "Webhook Production",
		Type:       "WEBHOOK",
		Version:    1,
		IsEnabled:  true,
		Config:     []byte(`{"webhookUrl":"https://webhooks.example.com/rudder","headers":[{"from":"X-Api-Key","to":""}]}`),
	}}
	resource, _, err := h.Impl.MapRemoteToState(remote, nil)
	require.NoError(t, err)
	remoteHeaders := resource.Config["headers"].([]any)
	remoteSecret, ok := remoteHeaders[0].(map[string]any)["to"].(*secret.String)
	require.True(t, ok)
	assert.True(t, remoteSecret.IsUnknown())

	enableVarSubstitution(t)
	entities, _, err := h.Impl.FormatForExport(map[string]*destination.RemoteDestination{"webhook-production": remote}, nil, nil)
	require.NoError(t, err)
	require.Len(t, entities, 1)

	payload, err := json.Marshal(entities[0].Content)
	require.NoError(t, err)
	assert.Contains(t, string(payload), "{{ .WEBHOOK_PRODUCTION_HEADERS_0_TO }}")
	assert.NotContains(t, string(payload), "safe-dummy-secret-value")
}

func registeredWebhookDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(webhook.NewDefinition()))
	registered, err := registry.Get("webhook", 1)
	require.NoError(t, err)
	return registered
}

func validMinimalConfig() map[string]any {
	return map[string]any{
		"webhook_url": "https://webhooks.example.com/rudder",
	}
}

func validFullConfig() map[string]any {
	return map[string]any{
		"webhook_url":    "https://webhooks.example.com/rudder",
		"webhook_method": "DELETE",
		"headers": []any{
			map[string]any{"from": "X-Api-Key", "to": "{{ .WEBHOOK_HEADER_VALUE || safe-dummy-value }}"},
			map[string]any{"from": "X-Trace", "to": "rudder-cli"},
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{"analytics"},
				},
			},
		},
	}
}

func exampleConfig() map[string]any {
	return map[string]any{
		"webhook_url":    "https://webhooks.example.com/rudder/events",
		"webhook_method": "POST",
		"headers": []any{
			map[string]any{"from": "X-Rudder-Token", "to": "{{ .WEBHOOK_HEADER_TOKEN || safe-dummy-token }}"},
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{"analytics"},
				},
			},
		},
	}
}

func stringOfLength(n int) string {
	return fmt.Sprintf("%*s", n, "")
}

func enableVarSubstitution(t *testing.T) {
	t.Helper()
	prevExp, prevFlag := viper.Get("experimental"), viper.Get("flags.enableVarSubstitution")
	viper.Set("experimental", true)
	viper.Set("flags.enableVarSubstitution", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.enableVarSubstitution", prevFlag)
	})
}
