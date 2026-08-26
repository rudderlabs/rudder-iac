package slack_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/slack"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(slack.NewDefinition()))

	registered, err := registry.Get("slack", 1)
	require.NoError(t, err)

	assert.Equal(t, "slack", registered.Type)
	assert.Equal(t, "SLACK", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys())
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

	byAPI, err := registry.GetByAPIType("SLACK", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestSlackConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(slack.NewDefinition()))
	registered, err := registry.Get("slack", 1)
	require.NoError(t, err)

	t.Run("missing webhook_url", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{})

		require.NotEmpty(t, errors)
		assert.Equal(t, "/webhook_url", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("webhook_url rejects invalid literals", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name  string
			value string
		}{
			{name: "ngrok URL", value: "https://rudder.ngrok.io/slack"},
			{name: "line break", value: "https://hooks.slack.com/services/T000/B000\nsecret"},
			{name: "over 100 characters", value: strings.Repeat("a", 101)},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(map[string]any{"webhook_url": tc.value})

				require.Len(t, errors, 1)
				assert.Equal(t, "/webhook_url", errors[0].Path)
			})
		}
	})

	t.Run("template fields reject invalid literals", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name   string
			config map[string]any
			path   string
		}{
			{
				name:   "identify template line break",
				config: withConfig(validMinimalConfig(), "identify_template", "hello\nworld"),
				path:   "/identify_template",
			},
			{
				name: "event channel name over 100 characters",
				config: withConfig(validMinimalConfig(), "event_channel_settings", []any{
					map[string]any{"name": strings.Repeat("a", 101), "channel": "alerts", "regex": false},
				}),
				path: "/event_channel_settings/0/name",
			},
			{
				name: "event channel line break",
				config: withConfig(validMinimalConfig(), "event_channel_settings", []any{
					map[string]any{"name": "Order Completed", "channel": "alerts\nprod", "regex": false},
				}),
				path: "/event_channel_settings/0/channel",
			},
			{
				name: "event template over 1000 characters",
				config: withConfig(validMinimalConfig(), "event_template_settings", []any{
					map[string]any{"name": "Order Completed", "template": strings.Repeat("a", 1001), "regex": false},
				}),
				path: "/event_template_settings/0/template",
			},
			{
				name:   "whitelisted trait line break",
				config: withConfig(validMinimalConfig(), "whitelisted_trait_settings", []any{"email", "bad\ntrait"}),
				path:   "/whitelisted_trait_settings/1",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(tc.config)

				require.Len(t, errors, 1)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	t.Run("template fields accept UI templates", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"webhook_url":       "{{ config.webhookUrl || https://hooks.slack.com/services/T000/B000/example }}",
			"identify_template": "{{ message.traits || " + strings.Repeat("x", 1100) + " }}",
			"event_channel_settings": []any{
				map[string]any{
					"name":    "{{ message.event || " + strings.Repeat("x", 150) + " }}",
					"channel": "{{ message.channel || " + strings.Repeat("x", 150) + " }}",
					"regex":   true,
				},
			},
			"event_template_settings": []any{
				map[string]any{
					"name":     "{{ message.event || Product Viewed }}",
					"template": "{{ message.properties || " + strings.Repeat("x", 1100) + " }}",
					"regex":    false,
				},
			},
			"whitelisted_trait_settings": []any{"{{ message.trait || " + strings.Repeat("x", 150) + " }}"},
		})

		assert.Empty(t, errors)
	})

	t.Run("deprecated env references get no template exemption", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["webhook_url"] = "env." + strings.Repeat("A", 101)

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/webhook_url", errors[0].Path)
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
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"webhook_url":       "https://hooks.slack.com/services/T000/B000/example",
			"identify_template": "New user: {{ user.traits.email }}",
			"event_channel_settings": []any{
				map[string]any{"name": "Order Completed", "channel": "orders", "regex": false},
			},
			"event_template_settings": []any{
				map[string]any{"name": "Product Viewed", "template": "Viewed {{ event.product }}", "regex": false},
			},
			"whitelisted_trait_settings": []any{"email", "first_name"},
		}))
	})

	t.Run("partially filled mapping rows are accepted", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(withConfig(validMinimalConfig(), "event_channel_settings", []any{
			map[string]any{"name": "Order Completed"},
		})))
		assert.Empty(t, registered.ValidateConfig(withConfig(validMinimalConfig(), "event_template_settings", []any{
			map[string]any{"template": "Order {{ id }}"},
		})))
	})

	// These three keys exist in schema.json but have no terraform mapping. They
	// are modelled anyway so a CLI apply does not erase values set elsewhere.
	t.Run("keys absent from terraform are still validated", func(t *testing.T) {
		t.Parallel()

		t.Run("incoming_webhooks_type rejects values outside the enum", func(t *testing.T) {
			t.Parallel()
			errors := registered.ValidateConfig(withConfig(validMinimalConfig(), "incoming_webhooks_type", "classic"))
			require.Len(t, errors, 1)
			assert.Equal(t, "/incoming_webhooks_type", errors[0].Path)
		})

		t.Run("incoming_webhooks_type accepts both enum values", func(t *testing.T) {
			t.Parallel()
			for _, v := range []string{"legacy", "modern"} {
				assert.Empty(t, registered.ValidateConfig(withConfig(validMinimalConfig(), "incoming_webhooks_type", v)))
			}
		})

		t.Run("event channel webhook rejects ngrok", func(t *testing.T) {
			t.Parallel()
			errors := registered.ValidateConfig(withConfig(validMinimalConfig(), "event_channel_settings", []any{
				map[string]any{"name": "Order Completed", "webhook": "https://evil.ngrok.io/hook"},
			}))
			require.Len(t, errors, 1)
			assert.Equal(t, "/event_channel_settings/0/webhook", errors[0].Path)
		})

		t.Run("deny_list_of_events rejects over-long entries", func(t *testing.T) {
			t.Parallel()
			errors := registered.ValidateConfig(withConfig(validMinimalConfig(), "deny_list_of_events", []any{strings.Repeat("e", 101)}))
			require.Len(t, errors, 1)
			assert.Equal(t, "/deny_list_of_events/0", errors[0].Path)
		})
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
	// connection_mode legality is per source type, taken from this definition's
	// own ConnectionModes map rather than a shared enum.
	t.Run("connection_mode accepts a supported mode", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"connection_mode": map[string]any{"web": "cloud"},
		})

		for _, err := range errors {
			assert.NotEqual(t, "/connection_mode/web", err.Path)
		}
	})

	t.Run("connection_mode rejects an unsupported mode", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"connection_mode": map[string]any{"web": "device"},
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

func TestSlackConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := slack.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"webhook_url": "https://hooks.slack.com/services/T000/B000/example"
			}`,
			APIJSON: `{
				"webhookUrl": "https://hooks.slack.com/services/T000/B000/example"
			}`,
		},
		{
			// Covers the three keys schema.json declares but terraform omits:
			// incomingWebhooksType, eventChannelWebhook, and denyListOfEvents.
			Name: "keys absent from terraform round trip",
			LocalJSON: `{
				"webhook_url": "https://hooks.slack.com/services/T000/B000/example",
				"incoming_webhooks_type": "modern",
				"event_channel_settings": [
					{"name": "Order Completed", "channel": "orders", "webhook": "https://hooks.slack.com/services/T000/B111/orders", "regex": false}
				],
				"deny_list_of_events": ["Heartbeat", "Debug Ping"]
			}`,
			APIJSON: `{
				"webhookUrl": "https://hooks.slack.com/services/T000/B000/example",
				"incomingWebhooksType": "modern",
				"eventChannelSettings": [
					{"eventName": "Order Completed", "eventChannel": "orders", "eventChannelWebhook": "https://hooks.slack.com/services/T000/B111/orders", "eventRegex": false}
				],
				"denyListOfEvents": [
					{"eventName": "Heartbeat"},
					{"eventName": "Debug Ping"}
				]
			}`,
		},
		{
			Name: "full",
			LocalJSON: `{
				"webhook_url": "https://hooks.slack.com/services/T000/B000/example",
				"identify_template": "Identify user",
				"event_channel_settings": [
					{"name": "Order Completed", "channel": "orders", "regex": false},
					{"name": "^Cart", "channel": "commerce", "regex": true}
				],
				"event_template_settings": [
					{"name": "Product Viewed", "template": "Viewed a product", "regex": false},
					{"name": "^Checkout", "template": "Checkout update", "regex": true}
				],
				"whitelisted_trait_settings": ["email", "first_name"]
			}`,
			APIJSON: `{
				"webhookUrl": "https://hooks.slack.com/services/T000/B000/example",
				"identifyTemplate": "Identify user",
				"eventChannelSettings": [
					{"eventName": "Order Completed", "eventChannel": "orders", "eventRegex": false},
					{"eventName": "^Cart", "eventChannel": "commerce", "eventRegex": true}
				],
				"eventTemplateSettings": [
					{"eventName": "Product Viewed", "eventTemplate": "Viewed a product", "eventRegex": false},
					{"eventName": "^Checkout", "eventTemplate": "Checkout update", "eventRegex": true}
				],
				"whitelistedTraitsSettings": [
					{"trait": "email"},
					{"trait": "first_name"}
				]
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"webhook_url": "https://hooks.slack.com/services/T000/B000/example",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"react_native": [{"provider": "iubenda"}],
					"warehouse": [{"provider": "ketch"}]
				}
			}`,
			APIJSON: `{
				"webhookUrl": "https://hooks.slack.com/services/T000/B000/example",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"reactnative": [{"provider": "iubenda"}],
					"warehouse": [{"provider": "ketch"}]
				}
			}`,
		},
	})
}

func validMinimalConfig() map[string]any {
	return map[string]any{
		"webhook_url": "https://hooks.slack.com/services/T000/B000/example",
	}
}

func validFullConfig() map[string]any {
	return map[string]any{
		"webhook_url":            "https://hooks.slack.com/services/T000/B000/example",
		"incoming_webhooks_type": "modern",
		"identify_template":      "Identify user",
		"event_channel_settings": []any{
			map[string]any{"name": "Order Completed", "channel": "orders", "webhook": "https://hooks.slack.com/services/T000/B111/orders", "regex": false},
			map[string]any{"name": "^Cart", "channel": "commerce", "regex": true},
		},
		"event_template_settings": []any{
			map[string]any{"name": "Product Viewed", "template": "Viewed a product", "regex": false},
			map[string]any{"name": "^Checkout", "template": "Checkout update", "regex": true},
		},
		"whitelisted_trait_settings": []any{"email", "first_name"},
		"deny_list_of_events":        []any{"Heartbeat", "Debug Ping"},
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

func withConfig(config map[string]any, key string, value any) map[string]any {
	config[key] = value
	return config
}
