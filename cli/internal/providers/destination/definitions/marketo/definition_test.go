package marketo_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/marketo"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(marketo.NewDefinition()))

	registered, err := registry.Get("marketo", 1)
	require.NoError(t, err)

	assert.Equal(t, "marketo", registered.Type)
	assert.Equal(t, "MARKETO", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"client_secret"}, registered.SecretKeys())
	assert.Empty(t, registered.GatedKeyPaths())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web", "unity",
		"cloud", "react_native", "flutter", "cordova",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	byAPI, err := registry.GetByAPIType("MARKETO", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestMarketoConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(marketo.NewDefinition()))
	registered, err := registry.Get("marketo", 1)
	require.NoError(t, err)

	t.Run("required fields", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"account_id", "client_id", "client_secret"} {
			t.Run("missing "+field, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				delete(config, field)

				errors := registered.ValidateConfig(config)

				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+field, errors[0].Path)
				assert.Contains(t, errors[0].Message, "required")
			})

			t.Run("empty "+field, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config[field] = ""

				errors := registered.ValidateConfig(config)

				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+field, errors[0].Path)
			})
		}
	})

	t.Run("credential fields reject values over maximum length", func(t *testing.T) {
		t.Parallel()

		for _, tc := range credentialFieldCases(strings.Repeat("x", 101)) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(tc.config)

				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	// The upstream constraint is ^(.{1,100})$ after stripping template/env branches,
	// so it rejects line breaks in addition to bounding literal length.
	t.Run("credential fields reject line breaks", func(t *testing.T) {
		t.Parallel()

		for _, tc := range credentialFieldCases("line\nbreak") {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				errors := registered.ValidateConfig(tc.config)

				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	// Upstream templates are exempt from literal length and line-break constraints.
	t.Run("credential fields accept ui templates of any length", func(t *testing.T) {
		t.Parallel()
		long := "{{ config.value || " + strings.Repeat("x", 120) + " }}"
		require.Greater(t, len(long), 100)

		for _, tc := range credentialFieldCases(long) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				assert.Empty(t, registered.ValidateConfig(tc.config))
			})
		}
	})

	t.Run("deprecated env references get no template exemption", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["account_id"] = "env." + strings.Repeat("A", 101)

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/account_id", errors[0].Path)
	})

	// schema.json declares these nested fields as plain strings with no
	// constraint; terraform marks them Required. Validation follows schema.json,
	// so a partially filled mapping row is accepted rather than rejected — which
	// keeps such a row importable from a remote config.
	t.Run("partially filled mapping rows are accepted", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name   string
			config map[string]any
		}{
			{
				name: "rudder event missing activity id",
				config: withConfig(validMinimalConfig(), "rudder_events_mapping", []any{
					map[string]any{"event": "Product Viewed", "marketo_primarykey": "email"},
				}),
			},
			{
				name: "lead trait missing to",
				config: withConfig(validMinimalConfig(), "lead_trait_mapping", []any{
					map[string]any{"from": "email"},
				}),
			},
			{
				name: "custom activity property missing from",
				config: withConfig(validMinimalConfig(), "custom_activity_property_map", []any{
					map[string]any{"to": "Product Name"},
				}),
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				assert.Empty(t, registered.ValidateConfig(tc.config))
			})
		}
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
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"account_id":             "123-ABC-456",
			"client_id":              "marketo-client-id",
			"client_secret":          "{{ .MARKETO_CLIENT_SECRET }}",
			"track_anonymous_events": false,
			"create_if_not_exist":    true,
			"rudder_events_mapping": []any{
				map[string]any{"event": "Product Viewed", "marketo_primarykey": "email", "marketo_activity_id": "100001"},
				map[string]any{"event": "Order Completed", "marketo_primarykey": "email", "marketo_activity_id": "100002"},
			},
			"lead_trait_mapping": []any{
				map[string]any{"from": "email", "to": "Email"},
				map[string]any{"from": "firstName", "to": "First Name"},
			},
			"custom_activity_property_map": []any{
				map[string]any{"from": "properties.product_id", "to": "Product ID"},
			},
		}))
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

func TestMarketoConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := marketo.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"account_id": "account-1",
				"client_id": "client-1",
				"client_secret": "secret-1"
			}`,
			APIJSON: `{
				"accountId": "account-1",
				"clientId": "client-1",
				"clientSecret": "secret-1"
			}`,
		},
		{
			Name: "full",
			LocalJSON: `{
				"account_id": "account-1",
				"client_id": "client-1",
				"client_secret": "secret-1",
				"track_anonymous_events": true,
				"create_if_not_exist": false,
				"rudder_events_mapping": [
					{"event": "Product Viewed", "marketo_primarykey": "email", "marketo_activity_id": "100001"},
					{"event": "Order Completed", "marketo_primarykey": "email", "marketo_activity_id": "100002"}
				],
				"lead_trait_mapping": [
					{"from": "email", "to": "Email"},
					{"from": "firstName", "to": "First Name"}
				],
				"custom_activity_property_map": [
					{"from": "properties.product_id", "to": "Product ID"},
					{"from": "properties.price", "to": "Price"}
				]
			}`,
			APIJSON: `{
				"accountId": "account-1",
				"clientId": "client-1",
				"clientSecret": "secret-1",
				"trackAnonymousEvents": true,
				"createIfNotExist": false,
				"rudderEventsMapping": [
					{"event": "Product Viewed", "marketoPrimarykey": "email", "marketoActivityId": "100001"},
					{"event": "Order Completed", "marketoPrimarykey": "email", "marketoActivityId": "100002"}
				],
				"leadTraitMapping": [
					{"from": "email", "to": "Email"},
					{"from": "firstName", "to": "First Name"}
				],
				"customActivityPropertyMap": [
					{"from": "properties.product_id", "to": "Product ID"},
					{"from": "properties.price", "to": "Price"}
				]
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"account_id": "account-1",
				"client_id": "client-1",
				"client_secret": "secret-1",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"accountId": "account-1",
				"clientId": "client-1",
				"clientSecret": "secret-1",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}

type credentialFieldCase struct {
	name   string
	path   string
	config map[string]any
}

func credentialFieldCases(value string) []credentialFieldCase {
	return []credentialFieldCase{
		{
			name: "account_id",
			path: "/account_id",
			config: map[string]any{
				"account_id":    value,
				"client_id":     "client-1",
				"client_secret": "secret-1",
			},
		},
		{
			name: "client_id",
			path: "/client_id",
			config: map[string]any{
				"account_id":    "account-1",
				"client_id":     value,
				"client_secret": "secret-1",
			},
		},
		{
			name: "client_secret",
			path: "/client_secret",
			config: map[string]any{
				"account_id":    "account-1",
				"client_id":     "client-1",
				"client_secret": value,
			},
		},
	}
}

func validMinimalConfig() map[string]any {
	return map[string]any{
		"account_id":    "account-1",
		"client_id":     "client-1",
		"client_secret": "secret-1",
	}
}

func validFullConfig() map[string]any {
	return map[string]any{
		"account_id":             "account-1",
		"client_id":              "client-1",
		"client_secret":          "secret-1",
		"track_anonymous_events": true,
		"create_if_not_exist":    false,
		"rudder_events_mapping": []any{
			map[string]any{"event": "Product Viewed", "marketo_primarykey": "email", "marketo_activity_id": "100001"},
			map[string]any{"event": "Order Completed", "marketo_primarykey": "email", "marketo_activity_id": "100002"},
		},
		"lead_trait_mapping": []any{
			map[string]any{"from": "email", "to": "Email"},
			map[string]any{"from": "firstName", "to": "First Name"},
		},
		"custom_activity_property_map": []any{
			map[string]any{"from": "properties.product_id", "to": "Product ID"},
			map[string]any{"from": "properties.price", "to": "Price"},
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{"marketing"},
				},
			},
		},
	}
}

func withConfig(config map[string]any, key string, value any) map[string]any {
	config[key] = value
	return config
}
