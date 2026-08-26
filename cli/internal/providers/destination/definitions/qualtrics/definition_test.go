package qualtrics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/qualtrics"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(qualtrics.NewDefinition()))

	registered, err := registry.Get("qualtrics", 1)
	require.NoError(t, err)

	assert.Equal(t, "qualtrics", registered.Type)
	assert.Equal(t, "QUALTRICS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys())

	expectedSourceTypes := []string{"web", "android", "ios"}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	expectedModes := map[string][]string{
		"web":     {"device"},
		"android": {"device"},
		"ios":     {"device"},
	}
	for sourceType, want := range expectedModes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, want, modes, "source type %s", sourceType)
	}

	assert.NotContains(t, registered.SupportedSourceTypes(), "android_kotlin")
	assert.NotContains(t, registered.SupportedSourceTypes(), "ios_swift")
	assert.NotContains(t, registered.SupportedSourceTypes(), "cloud")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")

	assert.Equal(t, map[string][]string{
		"enable_generic_page_title/web": {"web"},
	}, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("QUALTRICS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestQualtricsConfigValidation(t *testing.T) {
	t.Parallel()

	registered := registeredQualtricsDefinition(t)

	for _, tc := range []struct {
		name string
		key  string
		path string
	}{
		{name: "missing project_id", key: "project_id", path: "/project_id"},
		{name: "missing brand_id", key: "brand_id", path: "/brand_id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := minimalConfig()
			delete(config, tc.key)

			errors := registered.ValidateConfig(config)
			require.NotEmpty(t, errors)
			assert.Equal(t, tc.path, errors[0].Path)
			assert.Contains(t, errors[0].Message, "required")
		})
	}

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(minimalConfig())
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(fullConfig())
		assert.Empty(t, errors)
	})

	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(exampleConfig())
		assert.Empty(t, errors)
	})

	t.Run("event filtering lists are mutually exclusive", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["event_filtering"] = map[string]any{
			"whitelist": []any{"Order Completed"},
			"blacklist": []any{"Product Viewed"},
		}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assertValidationPaths(t, errors, "/event_filtering/whitelist", "/event_filtering/blacklist")
	})

	t.Run("event names enforce single line pattern", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["event_filtering"] = map[string]any{
			"whitelist": []any{"line one\nline two"},
		}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/event_filtering/whitelist/0", errors[0].Path)
	})

	t.Run("event names accept templates", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["event_filtering"] = map[string]any{
			"whitelist": []any{"{{ eventName || Product Viewed }}"},
		}

		errors := registered.ValidateConfig(config)
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

	t.Run("flat enable_generic_page_title rejected", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["enable_generic_page_title"] = true

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/enable_generic_page_title", errors[0].Path)
	})

	t.Run("connection mode rejected as config", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["connection_mode"] = map[string]any{"web": "device"}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/connection_mode", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("legacy consent keys rejected as config", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			config := minimalConfig()
			config[key] = map[string]any{"web": []any{"C0001"}}

			errors := registered.ValidateConfig(config)
			require.NotEmpty(t, errors, key)
			assert.Equal(t, "/"+key, errors[0].Path)
			assert.Contains(t, errors[0].Message, "unknown config field")
		}
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["consent_management"] = map[string]any{
			"android_kotlin": []any{},
		}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/android_kotlin", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'android_kotlin' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
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

	t.Run("custom consent provider requires resolution strategy", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["consent_management"] = map[string]any{
			"ios": []any{
				map[string]any{"provider": "custom"},
			},
		}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/ios/0/resolution_strategy", errors[0].Path)
	})
}

func TestQualtricsConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := qualtrics.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal required config",
			LocalJSON: `{
				"project_id": "ZN_blw7XXXTWxCGung",
				"brand_id": "examplebrand"
			}`,
			APIJSON: `{
				"projectId": "ZN_blw7XXXTWxCGung",
				"brandId": "examplebrand"
			}`,
		},
		{
			Name: "full config",
			LocalJSON: `{
				"project_id": "ZN_blw7XXXTWxCGung",
				"brand_id": "examplebrand",
				"enable_generic_page_title": {"web": true},
				"use_native_sdk": {"web": true, "android": false, "ios": true},
				"event_filtering": {
					"whitelist": ["Anonymous Page Visit", "Product Viewed"]
				},
				"consent_management": {
					"web": [{"provider": "oneTrust", "consents": ["C0002"]}],
					"android": [{"provider": "custom", "resolution_strategy": "or", "consents": ["marketing"]}]
				}
			}`,
			APIJSON: `{
				"projectId": "ZN_blw7XXXTWxCGung",
				"brandId": "examplebrand",
				"enableGenericPageTitle": {"web": true},
				"useNativeSDK": {"web": true, "android": false, "ios": true},
				"whitelistedEvents": [
					{"eventName": "Anonymous Page Visit"},
					{"eventName": "Product Viewed"}
				],
				"eventFilteringOption": "whitelistedEvents",
				"consentManagement": {
					"web": [{"provider": "oneTrust", "consents": [{"consent": "C0002"}]}],
					"android": [{"provider": "custom", "resolutionStrategy": "or", "consents": [{"consent": "marketing"}]}]
				}
			}`,
		},
		{
			Name: "event filtering blacklist",
			LocalJSON: `{
				"event_filtering": {
					"blacklist": ["Screen Viewed"]
				}
			}`,
			APIJSON: `{
				"blacklistedEvents": [
					{"eventName": "Screen Viewed"}
				],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "source scoped sdk and page title",
			LocalJSON: `{
				"enable_generic_page_title": {"web": false},
				"use_native_sdk": {"web": true, "android": true, "ios": false}
			}`,
			APIJSON: `{
				"enableGenericPageTitle": {"web": false},
				"useNativeSDK": {"web": true, "android": true, "ios": false}
			}`,
		},
		{
			Name: "consent source mappings",
			LocalJSON: `{
				"consent_management": {
					"web": [{"provider": "oneTrust"}],
					"android": [{"provider": "ketch"}],
					"ios": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"consentManagement": {
					"web": [{"provider": "oneTrust"}],
					"android": [{"provider": "ketch"}],
					"ios": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}

func registeredQualtricsDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(qualtrics.NewDefinition()))
	registered, err := registry.Get("qualtrics", 1)
	require.NoError(t, err)
	return registered
}

func minimalConfig() map[string]any {
	return map[string]any{
		"project_id": "ZN_blw7XXXTWxCGung",
		"brand_id":   "examplebrand",
	}
}

func fullConfig() map[string]any {
	return map[string]any{
		"project_id":                "ZN_blw7XXXTWxCGung",
		"brand_id":                  "examplebrand",
		"enable_generic_page_title": map[string]any{"web": true},
		"use_native_sdk": map[string]any{
			"web":     true,
			"android": true,
			"ios":     true,
		},
		"event_filtering": map[string]any{
			"whitelist": []any{"Product Viewed", "Order Completed"},
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{"analytics"},
				},
			},
			"android": []any{
				map[string]any{
					"provider":            "custom",
					"resolution_strategy": "and",
					"consents":            []any{"marketing"},
				},
			},
			"ios": []any{
				map[string]any{
					"provider": "ketch",
					"consents": []any{"mobile"},
				},
			},
		},
	}
}

func exampleConfig() map[string]any {
	return map[string]any{
		"project_id":                "ZN_blw7XXXTWxCGung",
		"brand_id":                  "examplebrand",
		"enable_generic_page_title": map[string]any{"web": true},
		"use_native_sdk": map[string]any{
			"web":     true,
			"android": true,
			"ios":     true,
		},
		"event_filtering": map[string]any{
			"whitelist": []any{"Anonymous Page Visit", "Product Viewed"},
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{"C0002"},
				},
			},
			"android": []any{
				map[string]any{
					"provider":            "custom",
					"resolution_strategy": "or",
					"consents":            []any{"marketing"},
				},
			},
		},
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
