package bqstream_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/bqstream"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(bqstream.NewDefinition()))

	registered, err := registry.Get("bqstream", 1)
	require.NoError(t, err)

	assert.Equal(t, "bqstream", registered.Type)
	assert.Equal(t, "BQSTREAM", registered.APIType)
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

	byAPI, err := registry.GetByAPIType("BQSTREAM", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestBQStreamConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(bqstream.NewDefinition()))
	registered, err := registry.Get("bqstream", 1)
	require.NoError(t, err)

	minimalValid := map[string]any{
		"project_id":  "my-gcp-project",
		"dataset_id":  "my_dataset",
		"table_id":    "my_table",
		"credentials": `{"type":"service_account"}`,
	}

	t.Run("missing project_id", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dataset_id":  "my_dataset",
			"table_id":    "my_table",
			"credentials": `{"type":"service_account"}`,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/project_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing dataset_id", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"project_id":  "my-gcp-project",
			"table_id":    "my_table",
			"credentials": `{"type":"service_account"}`,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/dataset_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing table_id", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"project_id":  "my-gcp-project",
			"dataset_id":  "my_dataset",
			"credentials": `{"type":"service_account"}`,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/table_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing credentials", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"project_id": "my-gcp-project",
			"dataset_id": "my_dataset",
			"table_id":   "my_table",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/credentials", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(minimalValid)
		assert.Empty(t, errors)
	})

	t.Run("valid full config with example values", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"project_id":  "my-gcp-project",
			"dataset_id":  "my_dataset",
			"table_id":    "my_table",
			"insert_id":   "messageId",
			"credentials": `{"type":"service_account","project_id":"my-gcp-project"}`,
			"connection_mode": map[string]any{
				"web":            "cloud",
				"android_kotlin": "cloud",
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

	t.Run("connection_mode rejects non-cloud values", func(t *testing.T) {
		t.Parallel()
		for _, mode := range []string{"device", "hybrid"} {
			errors := registered.ValidateConfig(map[string]any{
				"project_id":  "my-gcp-project",
				"dataset_id":  "my_dataset",
				"table_id":    "my_table",
				"credentials": `{"type":"service_account"}`,
				"connection_mode": map[string]any{
					"web": mode,
				},
			})
			require.Len(t, errors, 1, mode)
			assert.Equal(t, "/connection_mode/web", errors[0].Path, mode)
			assert.Contains(t, errors[0].Message, "must be one of", mode)
		}
	})

	t.Run("connection_mode rejects a template", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"project_id":  "my-gcp-project",
			"dataset_id":  "my_dataset",
			"table_id":    "my_table",
			"credentials": `{"type":"service_account"}`,
			"connection_mode": map[string]any{
				"web": "{{ .BQSTREAM_CONNECTION_MODE || cloud }}",
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/web", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("connection_mode rejects an empty string", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"project_id":  "my-gcp-project",
			"dataset_id":  "my_dataset",
			"table_id":    "my_table",
			"credentials": `{"type":"service_account"}`,
			"connection_mode": map[string]any{
				"web": "",
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/web", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("connection_mode rejects a non-string value", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"project_id":  "my-gcp-project",
			"dataset_id":  "my_dataset",
			"table_id":    "my_table",
			"credentials": `{"type":"service_account"}`,
			"connection_mode": map[string]any{
				"web": true,
			},
		})
		require.NotEmpty(t, errors)
		for _, err := range errors {
			assert.Equal(t, "/connection_mode/web", err.Path)
		}
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		cfg := map[string]any{
			"project_id":  "my-gcp-project",
			"dataset_id":  "my_dataset",
			"table_id":    "my_table",
			"credentials": `{"type":"service_account"}`,
			"not_a_field": true,
		}
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"project_id":  "my-gcp-project",
			"dataset_id":  "my_dataset",
			"table_id":    "my_table",
			"credentials": `{"type":"service_account"}`,
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
			"project_id":  "my-gcp-project",
			"dataset_id":  "my_dataset",
			"table_id":    "my_table",
			"credentials": `{"type":"service_account"}`,
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

func TestBQStreamConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := bqstream.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal required fields",
			LocalJSON: `{
				"project_id": "my-gcp-project",
				"dataset_id": "my_dataset",
				"table_id": "my_table",
				"credentials": "{\"type\":\"service_account\"}"
			}`,
			APIJSON: `{
				"projectId": "my-gcp-project",
				"datasetId": "my_dataset",
				"tableId": "my_table",
				"credentials": "{\"type\":\"service_account\"}"
			}`,
		},
		{
			Name: "full TF fields with connection mode",
			LocalJSON: `{
				"project_id": "my-gcp-project",
				"dataset_id": "my_dataset",
				"table_id": "my_table",
				"insert_id": "messageId",
				"credentials": "{\"type\":\"service_account\"}",
				"connection_mode": {
					"web": "cloud",
					"android_kotlin": "cloud"
				}
			}`,
			APIJSON: `{
				"projectId": "my-gcp-project",
				"datasetId": "my_dataset",
				"tableId": "my_table",
				"insertId": "messageId",
				"credentials": "{\"type\":\"service_account\"}",
				"connectionMode": {
					"web": "cloud",
					"androidKotlin": "cloud"
				}
			}`,
		},
		{
			// Terraform passes SkipZeroValue on insertId; porting it would drop the
			// key from the payload and surface as a phantom diff on every plan.
			Name: "empty insert id survives without SkipZeroValue",
			LocalJSON: `{
				"project_id": "my-gcp-project",
				"dataset_id": "my_dataset",
				"table_id": "my_table",
				"insert_id": "",
				"credentials": "{\"type\":\"service_account\"}"
			}`,
			APIJSON: `{
				"projectId": "my-gcp-project",
				"datasetId": "my_dataset",
				"tableId": "my_table",
				"insertId": "",
				"credentials": "{\"type\":\"service_account\"}"
			}`,
		},
		{
			Name: "consent for web",
			LocalJSON: `{
				"project_id": "my-gcp-project",
				"dataset_id": "my_dataset",
				"table_id": "my_table",
				"credentials": "{\"type\":\"service_account\"}",
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
				"datasetId": "my_dataset",
				"tableId": "my_table",
				"credentials": "{\"type\":\"service_account\"}",
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
				"dataset_id": "my_dataset",
				"table_id": "my_table",
				"credentials": "{\"type\":\"service_account\"}",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"projectId": "my-gcp-project",
				"datasetId": "my_dataset",
				"tableId": "my_table",
				"credentials": "{\"type\":\"service_account\"}",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
