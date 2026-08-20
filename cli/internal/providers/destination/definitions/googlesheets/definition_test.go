package googlesheets_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	googlesheets "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/googlesheets"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googlesheets.NewDefinition()))

	registered, err := registry.Get("googlesheets", 1)
	require.NoError(t, err)

	assert.Equal(t, "googlesheets", registered.Type)
	assert.Equal(t, "GOOGLESHEETS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"credentials"}, registered.SecretKeys())
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

	byAPI, err := registry.GetByAPIType("GOOGLESHEETS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestGoogleSheetsConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googlesheets.NewDefinition()))
	registered, err := registry.Get("googlesheets", 1)
	require.NoError(t, err)

	t.Run("required fields", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"credentials", "sheet_id", "sheet_name", "event_key_map"} {
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
	})

	t.Run("empty required strings rejected", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"credentials", "sheet_id", "sheet_name"} {
			t.Run(field, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config[field] = ""

				errors := registered.ValidateConfig(config)

				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+field, errors[0].Path)
				assert.Contains(t, errors[0].Message, "required")
			})
		}
	})

	// schema.json declares from/to as unconstrained strings, so a partially filled
	// mapping row must validate — otherwise importing a remote config that has one
	// produces a spec the CLI rejects. Mirrors the marketo definition.
	t.Run("sparse event mapping rows are accepted", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name  string
			entry map[string]any
		}{
			{name: "missing from", entry: map[string]any{"to": "Product Name"}},
			{name: "missing to", entry: map[string]any{"from": "properties.product_name"}},
			{name: "empty values", entry: map[string]any{"from": "", "to": ""}},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				cfg := withConfig(validMinimalConfig(), "event_key_map", []any{tc.entry})
				assert.Empty(t, registered.ValidateConfig(cfg))
			})
		}
	})

	t.Run("schema wildcard fields accept line breaks", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["sheet_name"] = "Sheet\n1"
		config["event_key_map"] = []any{map[string]any{"from": "properties\n.email", "to": "Email\nAddress"}}

		assert.Empty(t, registered.ValidateConfig(config))
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
			"credentials": "{{ .GOOGLE_SHEETS_CREDENTIALS }}",
			"sheet_id":    "13N0gXX9Be_2gR2afax2G4j6hXXXCOgmDcCRgopTc905",
			"sheet_name":  "Rudder Events",
			"event_key_map": []any{
				map[string]any{"from": "userId", "to": "User ID"},
				map[string]any{"from": "event", "to": "Event Name"},
				map[string]any{"from": "properties.revenue", "to": "Revenue"},
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

	t.Run("legacy consent blocks are not supported keys", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			t.Run(key, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config[key] = map[string]any{"web": []any{}}

				errors := registered.ValidateConfig(config)

				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+key, errors[0].Path)
				assert.Contains(t, errors[0].Message, "unknown config field")
			})
		}
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
}

func TestGoogleSheetsConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := googlesheets.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal config",
			LocalJSON: `{
				"credentials": "{\"type\":\"service_account\",\"project_id\":\"rudder-cli-e2e\"}",
				"sheet_id": "13N0gXX9Be_2gR2afax2G4j6hXXXCOgmDcCRgopTc905",
				"sheet_name": "Rudder Events",
				"event_key_map": [
					{"from": "userId", "to": "User ID"}
				]
			}`,
			APIJSON: `{
				"credentials": "{\"type\":\"service_account\",\"project_id\":\"rudder-cli-e2e\"}",
				"sheetId": "13N0gXX9Be_2gR2afax2G4j6hXXXCOgmDcCRgopTc905",
				"sheetName": "Rudder Events",
				"eventKeyMap": [
					{"from": "userId", "to": "User ID"}
				]
			}`,
		},
		{
			Name: "full mappings",
			LocalJSON: `{
				"credentials": "secret-value",
				"sheet_id": "sheet-123",
				"sheet_name": "Sheet1",
				"event_key_map": [
					{"from": "properties.product_name", "to": "Product Name"},
					{"from": "properties.revenue", "to": "Revenue"}
				]
			}`,
			APIJSON: `{
				"credentials": "secret-value",
				"sheetId": "sheet-123",
				"sheetName": "Sheet1",
				"eventKeyMap": [
					{"from": "properties.product_name", "to": "Product Name"},
					{"from": "properties.revenue", "to": "Revenue"}
				]
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"credentials": "secret-value",
				"sheet_id": "sheet-123",
				"sheet_name": "Sheet1",
				"event_key_map": [
					{"from": "event", "to": "Event Name"}
				],
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"react_native": [{"provider": "iubenda"}],
					"warehouse": [{"provider": "ketch"}]
				}
			}`,
			APIJSON: `{
				"credentials": "secret-value",
				"sheetId": "sheet-123",
				"sheetName": "Sheet1",
				"eventKeyMap": [
					{"from": "event", "to": "Event Name"}
				],
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
		"credentials": "secret-value",
		"sheet_id":    "sheet-123",
		"sheet_name":  "Sheet1",
		"event_key_map": []any{
			map[string]any{"from": "userId", "to": "User ID"},
		},
	}
}

func validFullConfig() map[string]any {
	return map[string]any{
		"credentials": "secret-value",
		"sheet_id":    "sheet-123",
		"sheet_name":  "Sheet1",
		"event_key_map": []any{
			map[string]any{"from": "userId", "to": "User ID"},
			map[string]any{"from": "properties.product_name", "to": "Product Name"},
			map[string]any{"from": "properties.revenue", "to": "Revenue"},
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
