package rs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/rs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(rs.NewDefinition()))

	registered, err := registry.Get("rs", 1)
	require.NoError(t, err)

	assert.Equal(t, "rs", registered.Type)
	assert.Equal(t, "RS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"password", "access_key_id", "access_key"}, registered.SecretKeys())

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
	assert.NotContains(t, registered.SupportedSourceTypes(), "cloud_source")

	byAPI, err := registry.GetByAPIType("RS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestRSConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(rs.NewDefinition()))
	registered, err := registry.Get("rs", 1)
	require.NoError(t, err)

	minimalValid := map[string]any{
		"host":               "example.redshift.amazonaws.com",
		"port":               "5439",
		"database":           "analytics",
		"user":               "rudder",
		"password":           "secret",
		"use_rudder_storage": true,
		"sync": map[string]any{
			"frequency": "180",
		},
	}

	t.Run("missing host", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		delete(cfg, "host")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/host", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing port", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		delete(cfg, "port")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/port", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing database", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		delete(cfg, "database")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/database", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing user", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		delete(cfg, "user")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/user", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing password", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		delete(cfg, "password")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/password", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing use_rudder_storage", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		delete(cfg, "use_rudder_storage")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/use_rudder_storage", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing sync frequency", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["sync"] = map[string]any{}
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/sync/frequency", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("invalid sync frequency", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["sync"] = map[string]any{"frequency": "10"}
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/sync/frequency", errors[0].Path)
	})

	t.Run("bucket_name required when use_rudder_storage false", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["use_rudder_storage"] = false
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/bucket_name", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal rudder storage", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(copyConfig(minimalValid))
		assert.Empty(t, errors)
	})

	t.Run("valid custom s3 storage", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["use_rudder_storage"] = false
		cfg["bucket_name"] = "my-redshift-bucket"
		cfg["access_key_id"] = "AKIAEXAMPLE"
		cfg["access_key"] = "secret-value"
		errors := registered.ValidateConfig(cfg)
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"host":               "example.redshift.amazonaws.com",
			"port":               "5439",
			"database":           "analytics",
			"user":               "rudder",
			"password":           "secret",
			"namespace":          "rudder_events",
			"enable_sse":         true,
			"use_rudder_storage": false,
			"bucket_name":        "my-redshift-bucket",
			"access_key_id":      "AKIAEXAMPLE",
			"access_key":         "secret-value",
			"sync": map[string]any{
				"frequency":                 "30",
				"start_at":                  "10:00",
				"exclude_window_start_time": "11:00",
				"exclude_window_end_time":   "12:00",
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
			"host":               "example.redshift.amazonaws.com",
			"port":               "5439",
			"database":           "analytics",
			"user":               "rudder",
			"password":           "{{ .RS_PASSWORD }}",
			"namespace":          "rudder_events",
			"use_rudder_storage": true,
			"sync": map[string]any{
				"frequency": "180",
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

func TestRSConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := rs.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal rudder storage",
			LocalJSON: `{
				"host": "example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"use_rudder_storage": true,
				"sync": {"frequency": "30"}
			}`,
			APIJSON: `{
				"host": "example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"useRudderStorage": true,
				"syncFrequency": "30"
			}`,
		},
		{
			Name: "full sync and s3 fields",
			LocalJSON: `{
				"host": "example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"namespace": "example-namespace",
				"enable_sse": true,
				"use_rudder_storage": false,
				"bucket_name": "some-bucket-name",
				"access_key_id": "some-access-key-id",
				"access_key": "some-access-key",
				"sync": {
					"frequency": "30",
					"start_at": "10:00",
					"exclude_window_start_time": "11:00",
					"exclude_window_end_time": "12:00"
				}
			}`,
			APIJSON: `{
				"host": "example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"namespace": "example-namespace",
				"enableSSE": true,
				"useRudderStorage": false,
				"bucketName": "some-bucket-name",
				"accessKeyID": "some-access-key-id",
				"accessKey": "some-access-key",
				"syncFrequency": "30",
				"syncStartAt": "10:00",
				"excludeWindow": {
					"excludeWindowStartTime": "11:00",
					"excludeWindowEndTime": "12:00"
				}
			}`,
		},
		{
			Name: "exclude window reshape",
			LocalJSON: `{
				"host": "example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"use_rudder_storage": true,
				"sync": {
					"frequency": "180",
					"exclude_window_start_time": "01:00",
					"exclude_window_end_time": "02:00"
				}
			}`,
			APIJSON: `{
				"host": "example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"useRudderStorage": true,
				"syncFrequency": "180",
				"excludeWindow": {
					"excludeWindowStartTime": "01:00",
					"excludeWindowEndTime": "02:00"
				}
			}`,
		},
		{
			Name: "consent for web",
			LocalJSON: `{
				"host": "example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"use_rudder_storage": true,
				"sync": {"frequency": "180"},
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
				"host": "example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"useRudderStorage": true,
				"syncFrequency": "180",
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
				"host": "example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"use_rudder_storage": true,
				"sync": {"frequency": "180"},
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"host": "example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"useRudderStorage": true,
				"syncFrequency": "180",
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
		if nested, ok := v.(map[string]any); ok {
			cloned := make(map[string]any, len(nested))
			for nk, nv := range nested {
				cloned[nk] = nv
			}
			out[k] = cloned
			continue
		}
		out[k] = v
	}
	return out
}
