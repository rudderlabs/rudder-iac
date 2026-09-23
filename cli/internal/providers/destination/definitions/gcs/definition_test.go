package gcs_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/gcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(gcs.NewDefinition()))

	registered, err := registry.Get("gcs", 1)
	require.NoError(t, err)

	assert.Equal(t, "gcs", registered.Type)
	assert.Equal(t, "GCS", registered.APIType)
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

	byAPI, err := registry.GetByAPIType("GCS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestGCSConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(gcs.NewDefinition()))
	registered, err := registry.Get("gcs", 1)
	require.NoError(t, err)

	t.Run("missing bucket_name", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"prefix": "rudder/",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/bucket_name", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": "my-gcs-bucket",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid with prefix", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": "my-gcs-bucket",
			"prefix":      "rudder/",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid with credentials", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": "my-gcs-bucket",
			"credentials": `{"type":"service_account","project_id":"rudder-e2e"}`,
		})
		assert.Empty(t, errors)
	})

	t.Run("bucket_name rejects values over 100 characters", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": strings.Repeat("a", 101),
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/bucket_name", errors[0].Path)
	})

	t.Run("prefix rejects values over 100 characters", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": "my-gcs-bucket",
			"prefix":      strings.Repeat("a", 101),
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/prefix", errors[0].Path)
	})

	// `^(.{0,100})$` forbids line breaks as well as bounding length; a plain
	// max=100 would let these through.
	t.Run("rejects values containing line breaks", func(t *testing.T) {
		t.Parallel()

		for field, config := range map[string]map[string]any{
			"/bucket_name": {"bucket_name": "my\ngcs-bucket"},
			"/prefix":      {"bucket_name": "my-gcs-bucket", "prefix": "rudder\n/"},
		} {
			errors := registered.ValidateConfig(config)
			require.Len(t, errors, 1, "config %v must be rejected", config)
			assert.Equal(t, field, errors[0].Path)
		}
	})

	// Upstream's template branch carries no length cap, so a template longer than
	// the literal limit is still valid.
	t.Run("accepts ui templates regardless of length", func(t *testing.T) {
		t.Parallel()

		long := "{{ config.bucketName || " + strings.Repeat("x", 90) + " }}"
		require.Greater(t, len(long), 100)

		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": long,
			"prefix":      "{{ config.prefix || rudder/ }}",
		})
		assert.Empty(t, errors)
	})

	// env.VAR gets no escape hatch: it is judged as an ordinary literal, so a
	// short one passes on length alone while an over-long one is rejected —
	// unlike a template, which bypasses the pattern entirely.
	t.Run("deprecated env references get no template exemption", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": "env." + strings.Repeat("A", 101),
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/bucket_name", errors[0].Path)
	})

	t.Run("valid example config with var credentials", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": "rudder-cli-e2e-gcs",
			"prefix":      "rudder/gcs/",
			"credentials": "{{ .GCS_CREDENTIALS }}",
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
				"bucket_name": "my-gcs-bucket",
				"connection_mode": map[string]any{
					"web": mode,
				},
			})
			require.Len(t, errors, 1, mode)
			assert.Equal(t, "/connection_mode/web", errors[0].Path, mode)
			assert.Contains(t, errors[0].Message, "must be one of", mode)
		}
	})

	// connectionMode is a plain enum upstream — no template branch, unlike the
	// pattern-validated fields — so a template is rejected rather than passed
	// through as an opaque dynamic value.
	t.Run("connection_mode rejects a template", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": "my-gcs-bucket",
			"connection_mode": map[string]any{
				"web": "{{ .GCS_CONNECTION_MODE || cloud }}",
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/web", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	// An explicit empty string is a real value, not an absent key — it must be
	// rejected like any other invalid entry, since converter.Simple never skips
	// zero values and would otherwise send an empty connectionMode.web upstream.
	t.Run("connection_mode rejects an empty string", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": "my-gcs-bucket",
			"connection_mode": map[string]any{
				"web": "",
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/web", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	// A non-string value trips both the mapstructure decode error and
	// validateConnectionMode's own type check, so the same path can be reported
	// twice. consent_management's shape check has the same pre-existing behavior.
	t.Run("connection_mode rejects a non-string value", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": "my-gcs-bucket",
			"connection_mode": map[string]any{
				"web": true,
			},
		})
		require.NotEmpty(t, errors)
		for _, err := range errors {
			assert.Equal(t, "/connection_mode/web", err.Path)
		}
	})

	// The framework's generic source-type-scoped-key check (not this definition's
	// own validation) catches unsupported connection_mode source types in full
	// project validation, so ValidateConfig skips unknown source-type values.
	t.Run("connection_mode for an unsupported source type does not error here", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": "my-gcs-bucket",
			"connection_mode": map[string]any{
				"warehouse": "cloud",
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": "my-gcs-bucket",
			"not_a_field": true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": "my-gcs-bucket",
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
			"bucket_name": "my-gcs-bucket",
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

func TestGCSConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := gcs.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal bucket only",
			LocalJSON: `{
				"bucket_name": "my-gcs-bucket"
			}`,
			APIJSON: `{
				"bucketName": "my-gcs-bucket"
			}`,
		},
		{
			Name: "full TF fields with connection mode",
			LocalJSON: `{
				"bucket_name": "my-gcs-bucket",
				"prefix": "rudder/",
				"credentials": "{\"type\":\"service_account\",\"project_id\":\"rudder-e2e\"}",
				"connection_mode": {
					"web": "cloud",
					"android_kotlin": "cloud"
				}
			}`,
			APIJSON: `{
				"bucketName": "my-gcs-bucket",
				"prefix": "rudder/",
				"credentials": "{\"type\":\"service_account\",\"project_id\":\"rudder-e2e\"}",
				"connectionMode": {
					"web": "cloud",
					"androidKotlin": "cloud"
				}
			}`,
		},
		{
			// Zero values must survive conversion: filtering them out would drop
			// the key upstream and re-introduce it on every plan as a phantom diff.
			Name: "explicit zero values preserved",
			LocalJSON: `{
				"bucket_name": "my-gcs-bucket",
				"prefix": "",
				"credentials": ""
			}`,
			APIJSON: `{
				"bucketName": "my-gcs-bucket",
				"prefix": "",
				"credentials": ""
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"bucket_name": "my-gcs-bucket",
				"consent_management": {
					"android_kotlin": [
						{
							"provider": "oneTrust",
							"resolution_strategy": "and",
							"consents": ["analytics", "marketing"]
						}
					]
				}
			}`,
			APIJSON: `{
				"bucketName": "my-gcs-bucket",
				"consentManagement": {
					"androidKotlin": [
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
	})
}
