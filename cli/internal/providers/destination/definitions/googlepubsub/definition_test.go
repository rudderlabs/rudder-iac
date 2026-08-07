package googlepubsub_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/googlepubsub"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googlepubsub.NewDefinition()))

	registered, err := registry.Get("googlepubsub", 1)
	require.NoError(t, err)

	assert.Equal(t, "googlepubsub", registered.Type)
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

	// Upstream lists these, but the CLI event-stream provider does not own them.
	assert.NotContains(t, registered.SupportedSourceTypes(), "amp")
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")

	byAPI, err := registry.GetByAPIType("GOOGLEPUBSUB", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestGooglePubSubConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googlepubsub.NewDefinition()))
	registered, err := registry.Get("googlepubsub", 1)
	require.NoError(t, err)

	minimalValid := map[string]any{
		"project_id":  "my-gcp-project",
		"credentials": `{"type":"service_account"}`,
	}

	t.Run("missing project_id", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		delete(cfg, "project_id")
		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/project_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing credentials", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		delete(cfg, "credentials")
		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/credentials", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("project_id longer than 100 chars rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["project_id"] = strings.Repeat("a", 101)
		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/project_id", errors[0].Path)
	})

	t.Run("event_to_topic_map value longer than 100 chars rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["event_to_topic_map"] = []any{
			map[string]any{"from": "Product Viewed", "to": strings.Repeat("t", 101)},
		}
		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/event_to_topic_map/0/to", errors[0].Path)
	})

	t.Run("event_to_attribute_map value longer than 100 chars rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["event_to_attribute_map"] = []any{
			map[string]any{"from": strings.Repeat("f", 101), "to": "attribute"},
		}
		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/event_to_attribute_map/0/from", errors[0].Path)
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(copyConfig(minimalValid))
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"project_id":  "my-gcp-project",
			"credentials": `{"type":"service_account"}`,
			"event_to_topic_map": []any{
				map[string]any{"from": "Product Viewed", "to": "product-events"},
				map[string]any{"from": "Order Completed", "to": "order-events"},
			},
			"event_to_attribute_map": []any{
				map[string]any{"from": "userId", "to": "user_id"},
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
			"project_id":  "my-gcp-project",
			"credentials": "{{ .GOOGLE_PUBSUB_CREDENTIALS }}",
			"event_to_topic_map": []any{
				map[string]any{"from": "Product Viewed", "to": "product-events"},
			},
			"event_to_attribute_map": []any{
				map[string]any{"from": "userId", "to": "user_id"},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["not_a_field"] = true
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
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
		cfg := copyConfig(minimalValid)
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

func TestGooglePubSubConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := googlepubsub.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"project_id": "my-gcp-project",
				"credentials": "{\"type\":\"service_account\"}"
			}`,
			APIJSON: `{
				"projectId": "my-gcp-project",
				"credentials": "{\"type\":\"service_account\"}"
			}`,
		},
		{
			Name: "event to topic map reshape",
			LocalJSON: `{
				"project_id": "my-gcp-project",
				"credentials": "creds",
				"event_to_topic_map": [
					{"from": "Product Viewed", "to": "product-events"},
					{"from": "Order Completed", "to": "order-events"}
				]
			}`,
			APIJSON: `{
				"projectId": "my-gcp-project",
				"credentials": "creds",
				"eventToTopicMap": [
					{"from": "Product Viewed", "to": "product-events"},
					{"from": "Order Completed", "to": "order-events"}
				]
			}`,
		},
		{
			Name: "event to attribute map maps to pluralized API key",
			LocalJSON: `{
				"project_id": "my-gcp-project",
				"credentials": "creds",
				"event_to_attribute_map": [
					{"from": "userId", "to": "user_id"}
				]
			}`,
			APIJSON: `{
				"projectId": "my-gcp-project",
				"credentials": "creds",
				"eventToAttributesMap": [
					{"from": "userId", "to": "user_id"}
				]
			}`,
		},
		{
			Name: "consent for web",
			LocalJSON: `{
				"project_id": "my-gcp-project",
				"credentials": "creds",
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
				"projectId": "my-gcp-project",
				"credentials": "creds",
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
				"project_id": "my-gcp-project",
				"credentials": "creds",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"projectId": "my-gcp-project",
				"credentials": "creds",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}

func copyConfig(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
