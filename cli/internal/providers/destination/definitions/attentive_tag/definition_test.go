package attentivetag_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	attentivetag "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/attentive_tag"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(attentivetag.NewDefinition()))

	registered, err := registry.Get("attentive_tag", 1)
	require.NoError(t, err)

	assert.Equal(t, "attentive_tag", registered.Type)
	assert.Equal(t, "ATTENTIVE_TAG", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"api_key"}, registered.SecretKeys())

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

	byAPI, err := registry.GetByAPIType("ATTENTIVE_TAG", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestAttentiveTagConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(attentivetag.NewDefinition()))
	registered, err := registry.Get("attentive_tag", 1)
	require.NoError(t, err)

	t.Run("missing api_key", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_key", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key": "test-api-key",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key":                  "test-api-key",
			"sign_up_source_id":        "12345",
			"enable_new_identify_flow": true,
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
			"api_key":                  "your-attentive-api-key",
			"sign_up_source_id":        "123456",
			"enable_new_identify_flow": true,
		})
		assert.Empty(t, errors)
	})

	t.Run("sign_up_source_id rejects non-digits", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key":           "test-api-key",
			"sign_up_source_id": "abc123",
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/sign_up_source_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must contain only digits")
	})

	t.Run("sign_up_source_id rejects dynamic values", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name  string
			value string
		}{
			{name: "env reference", value: "env.SIGN_UP_SOURCE_ID"},
			{name: "ui template", value: "{{ config.signUpSourceId || 123 }}"},
			{name: "iac variable", value: "{{ .SIGN_UP_SOURCE_ID }}"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				errors := registered.ValidateConfig(map[string]any{
					"api_key":           "test-api-key",
					"sign_up_source_id": tc.value,
				})
				require.Len(t, errors, 1)
				assert.Equal(t, "/sign_up_source_id", errors[0].Path)
				assert.Contains(t, errors[0].Message, "must contain only digits")
			})
		}
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key":     "test-api-key",
			"not_a_field": true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"api_key": "test-api-key",
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
			"api_key": "test-api-key",
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

func TestAttentiveTagConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := attentivetag.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal api key only",
			LocalJSON: `{
				"api_key": "test-api-key"
			}`,
			APIJSON: `{
				"apiKey": "test-api-key"
			}`,
		},
		{
			Name: "full config",
			LocalJSON: `{
				"api_key": "test-api-key",
				"sign_up_source_id": "12345",
				"enable_new_identify_flow": true
			}`,
			APIJSON: `{
				"apiKey": "test-api-key",
				"signUpSourceId": "12345",
				"enableNewIdentifyFlow": true
			}`,
		},
		{
			Name: "consent for web",
			LocalJSON: `{
				"api_key": "test-api-key",
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
				"apiKey": "test-api-key",
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
				"api_key": "test-api-key",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"apiKey": "test-api-key",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
