package googlepubsub_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	googlepubsub "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/google_pubsub"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googlepubsub.NewDefinition()))

	registered, err := registry.Get("google_pubsub", 1)
	require.NoError(t, err)

	assert.Equal(t, "google_pubsub", registered.Type)
	assert.Equal(t, "GOOGLEPUBSUB", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"credentials"}, registered.SecretKeys())

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

	byAPI, err := registry.GetByAPIType("GOOGLEPUBSUB", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestGooglePubSubConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googlepubsub.NewDefinition()))
	registered, err := registry.Get("google_pubsub", 1)
	require.NoError(t, err)

	minimalConfig := func() map[string]any {
		return map[string]any{
			"project_id":  "rudder-cli-e2e",
			"credentials": `{"type":"service_account","project_id":"rudder-cli-e2e"}`,
		}
	}

	t.Run("missing project_id", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		delete(config, "project_id")

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/project_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing credentials", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		delete(config, "credentials")

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/credentials", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("empty credentials rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["credentials"] = ""

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/credentials", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(minimalConfig())

		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"project_id":  "rudder-cli-e2e",
			"credentials": `{"type":"service_account","project_id":"rudder-cli-e2e"}`,
			"event_to_topic_map": []any{
				map[string]any{"from": "Product Viewed", "to": "product-events"},
				map[string]any{"from": "Order Completed", "to": "order-events"},
			},
			"event_to_attribute_map": []any{
				map[string]any{"from": "context.traits.plan", "to": "plan"},
				map[string]any{"from": "properties.currency", "to": "currency"},
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

	t.Run("project_id rejects values over 100 characters", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["project_id"] = strings.Repeat("a", 101)

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/project_id", errors[0].Path)
	})

	t.Run("project_id rejects line breaks", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["project_id"] = "rudder\nproject"

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/project_id", errors[0].Path)
	})

	t.Run("project_id accepts ui template regardless of length", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["project_id"] = "{{ config.projectId || " + strings.Repeat("x", 90) + " }}"
		require.Greater(t, len(config["project_id"].(string)), 100)

		errors := registered.ValidateConfig(config)

		assert.Empty(t, errors)
	})

	t.Run("deprecated env references get no template exemption", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["project_id"] = "env." + strings.Repeat("A", 101)

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/project_id", errors[0].Path)
	})

	t.Run("mapping fields reject invalid literals", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name   string
			config map[string]any
			path   string
		}{
			{
				name: "topic from over 100 characters",
				config: map[string]any{
					"project_id":         "rudder-cli-e2e",
					"credentials":        "secret",
					"event_to_topic_map": []any{map[string]any{"from": strings.Repeat("a", 101), "to": "topic"}},
				},
				path: "/event_to_topic_map/0/from",
			},
			{
				name: "topic to has line break",
				config: map[string]any{
					"project_id":         "rudder-cli-e2e",
					"credentials":        "secret",
					"event_to_topic_map": []any{map[string]any{"from": "event", "to": "bad\ntopic"}},
				},
				path: "/event_to_topic_map/0/to",
			},
			{
				name: "attribute from over 100 characters",
				config: map[string]any{
					"project_id":             "rudder-cli-e2e",
					"credentials":            "secret",
					"event_to_attribute_map": []any{map[string]any{"from": strings.Repeat("a", 101), "to": "attribute"}},
				},
				path: "/event_to_attribute_map/0/from",
			},
			{
				name: "attribute to has line break",
				config: map[string]any{
					"project_id":             "rudder-cli-e2e",
					"credentials":            "secret",
					"event_to_attribute_map": []any{map[string]any{"from": "property", "to": "bad\nattribute"}},
				},
				path: "/event_to_attribute_map/0/to",
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

	t.Run("mapping fields accept ui templates", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"project_id":  "rudder-cli-e2e",
			"credentials": "secret",
			"event_to_topic_map": []any{
				map[string]any{"from": "{{ event.name || Product Viewed }}", "to": "{{ topic.name || product-events }}"},
			},
			"event_to_attribute_map": []any{
				map[string]any{"from": "{{ event.property || plan }}", "to": "{{ attribute.name || plan }}"},
			},
		})

		assert.Empty(t, errors)
	})

	t.Run("valid example config with var credentials", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"project_id":  "rudder-cli-e2e",
			"credentials": "{{ .GOOGLE_PUBSUB_CREDENTIALS }}",
			"event_to_topic_map": []any{
				map[string]any{"from": "Product Viewed", "to": "product-events"},
				map[string]any{"from": "Order Completed", "to": "order-events"},
			},
			"event_to_attribute_map": []any{
				map[string]any{"from": "context.traits.plan", "to": "plan"},
				map[string]any{"from": "properties.currency", "to": "currency"},
			},
		})

		assert.Empty(t, errors)
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["not_a_field"] = true

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
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
		config := minimalConfig()
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
}

func TestGooglePubSubConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := googlepubsub.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal config",
			LocalJSON: `{
				"project_id": "rudder-cli-e2e",
				"credentials": "{\"type\":\"service_account\",\"project_id\":\"rudder-cli-e2e\"}"
			}`,
			APIJSON: `{
				"projectId": "rudder-cli-e2e",
				"credentials": "{\"type\":\"service_account\",\"project_id\":\"rudder-cli-e2e\"}"
			}`,
		},
		{
			Name: "full mappings",
			LocalJSON: `{
				"project_id": "rudder-cli-e2e",
				"credentials": "secret-value",
				"event_to_topic_map": [
					{"from": "Product Viewed", "to": "product-events"},
					{"from": "Order Completed", "to": "order-events"}
				],
				"event_to_attribute_map": [
					{"from": "context.traits.plan", "to": "plan"},
					{"from": "properties.currency", "to": "currency"}
				]
			}`,
			APIJSON: `{
				"projectId": "rudder-cli-e2e",
				"credentials": "secret-value",
				"eventToTopicMap": [
					{"from": "Product Viewed", "to": "product-events"},
					{"from": "Order Completed", "to": "order-events"}
				],
				"eventToAttributesMap": [
					{"from": "context.traits.plan", "to": "plan"},
					{"from": "properties.currency", "to": "currency"}
				]
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"project_id": "rudder-cli-e2e",
				"credentials": "secret-value",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"projectId": "rudder-cli-e2e",
				"credentials": "secret-value",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
