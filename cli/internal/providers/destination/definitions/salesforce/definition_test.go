package salesforce_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/salesforce"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(salesforce.NewDefinition()))

	registered, err := registry.Get("salesforce", 1)
	require.NoError(t, err)

	assert.Equal(t, "salesforce", registered.Type)
	assert.Equal(t, "SALESFORCE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"password", "initial_access_token"}, registered.SecretKeys())
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

	byAPI, err := registry.GetByAPIType("SALESFORCE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestSalesforceConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(salesforce.NewDefinition()))
	registered, err := registry.Get("salesforce", 1)
	require.NoError(t, err)

	for _, field := range []string{"user_name", "password", "initial_access_token"} {
		t.Run("missing "+field, func(t *testing.T) {
			t.Parallel()

			config := validMinimalConfig()
			delete(config, field)

			errors := registered.ValidateConfig(config)
			require.NotEmpty(t, errors)
			assert.Equal(t, "/"+field, errors[0].Path)
			assert.Contains(t, errors[0].Message, "required")
		})
	}

	t.Run("required strings reject empty values", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"user_name", "password", "initial_access_token"} {
			t.Run(field, func(t *testing.T) {
				t.Parallel()

				config := validMinimalConfig()
				config[field] = ""

				errors := registered.ValidateConfig(config)
				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+field, errors[0].Path)
			})
		}
	})

	t.Run("required strings reject values over maximum length", func(t *testing.T) {
		t.Parallel()

		for _, tc := range patternFieldCases(strings.Repeat("x", 101)) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(tc.config)
				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	// The upstream constraint is `^(.{1,100})$` — a pattern, not just a
	// length limit, so it rejects line breaks as well as bounding length.
	t.Run("required strings reject line breaks", func(t *testing.T) {
		t.Parallel()

		for _, tc := range patternFieldCases("line\nbreak") {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(tc.config)
				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	// Upstream's template branch carries no length cap, so a template longer than
	// the literal limit is still valid.
	t.Run("required strings accept ui templates of any length", func(t *testing.T) {
		t.Parallel()

		long := "{{ config.salesforce.value || " + strings.Repeat("x", 120) + " }}"
		for _, tc := range patternFieldCases(long) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				assert.Empty(t, registered.ValidateConfig(tc.config))
			})
		}
	})

	// env.VAR gets no escape hatch: it is judged as an ordinary literal, so an
	// over-long one is rejected, unlike a UI template.
	t.Run("deprecated env references get no template exemption", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"user_name":            "env." + strings.Repeat("A", 101),
			"password":             "password-1",
			"initial_access_token": "token-1",
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/user_name", errors[0].Path)
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(validMinimalConfig()))
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(validFullConfig()))
	})

	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"user_name":            "rudder-cli@example.com",
			"password":             "salesforce-password",
			"initial_access_token": "salesforce-token",
			"map_properties":       true,
			"sandbox":              true,
			"use_contact_id":       false,
		})
		assert.Empty(t, errors)
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
			"android_kotlin": []any{
				map[string]any{"provider": "unknown"},
			},
		}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/android_kotlin/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestSalesforceConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := salesforce.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"user_name": "rudder-cli@example.com",
				"password": "password-1",
				"initial_access_token": "token-1"
			}`,
			APIJSON: `{
				"userName": "rudder-cli@example.com",
				"password": "password-1",
				"initialAccessToken": "token-1"
			}`,
		},
		{
			Name: "full",
			LocalJSON: `{
				"user_name": "rudder-cli@example.com",
				"password": "password-1",
				"initial_access_token": "token-1",
				"map_properties": true,
				"sandbox": true,
				"use_contact_id": true
			}`,
			APIJSON: `{
				"userName": "rudder-cli@example.com",
				"password": "password-1",
				"initialAccessToken": "token-1",
				"mapProperties": true,
				"sandbox": true,
				"useContactId": true
			}`,
		},
		{
			// Terraform skips false values for these optional booleans, but the CLI
			// preserves them to avoid non-converging diffs.
			Name: "explicit false booleans preserved",
			LocalJSON: `{
				"user_name": "rudder-cli@example.com",
				"password": "password-1",
				"initial_access_token": "token-1",
				"map_properties": false,
				"sandbox": false,
				"use_contact_id": false
			}`,
			APIJSON: `{
				"userName": "rudder-cli@example.com",
				"password": "password-1",
				"initialAccessToken": "token-1",
				"mapProperties": false,
				"sandbox": false,
				"useContactId": false
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"user_name": "rudder-cli@example.com",
				"password": "password-1",
				"initial_access_token": "token-1",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"userName": "rudder-cli@example.com",
				"password": "password-1",
				"initialAccessToken": "token-1",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}

type patternFieldCase struct {
	name   string
	path   string
	config map[string]any
}

func patternFieldCases(value string) []patternFieldCase {
	return []patternFieldCase{
		{
			name: "user_name",
			path: "/user_name",
			config: map[string]any{
				"user_name":            value,
				"password":             "password-1",
				"initial_access_token": "token-1",
			},
		},
		{
			name: "password",
			path: "/password",
			config: map[string]any{
				"user_name":            "rudder-cli@example.com",
				"password":             value,
				"initial_access_token": "token-1",
			},
		},
		{
			name: "initial_access_token",
			path: "/initial_access_token",
			config: map[string]any{
				"user_name":            "rudder-cli@example.com",
				"password":             "password-1",
				"initial_access_token": value,
			},
		},
	}
}

func validMinimalConfig() map[string]any {
	return map[string]any{
		"user_name":            "rudder-cli@example.com",
		"password":             "password-1",
		"initial_access_token": "token-1",
	}
}

func validFullConfig() map[string]any {
	return map[string]any{
		"user_name":            "rudder-cli@example.com",
		"password":             "password-1",
		"initial_access_token": "token-1",
		"map_properties":       true,
		"sandbox":              true,
		"use_contact_id":       false,
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
