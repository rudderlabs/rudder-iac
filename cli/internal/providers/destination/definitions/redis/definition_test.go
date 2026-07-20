package redis_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/redis"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(redis.NewDefinition()))

	registered, err := registry.Get("redis", 1)
	require.NoError(t, err)

	assert.Equal(t, "redis", registered.Type)
	assert.Equal(t, "REDIS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"password", "ca_certificate"}, registered.SecretKeys())

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

	byAPI, err := registry.GetByAPIType("REDIS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestRedisConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(redis.NewDefinition()))
	registered, err := registry.Get("redis", 1)
	require.NoError(t, err)

	t.Run("missing address", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/address", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("address too long rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"address": strings.Repeat("a", 101),
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/address", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be at most 100 characters")
	})

	t.Run("address with ngrok rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"address": "redis.ngrok.io:6379",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/address", errors[0].Path)
		assert.Contains(t, errors[0].Message, ".ngrok.io")
	})

	t.Run("prefix too long rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"address": "redis.example.com:6379",
			"prefix":  strings.Repeat("p", 101),
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/prefix", errors[0].Path)
		assert.Contains(t, errors[0].Message, "less than or equal to 100")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"address": "redis.example.com:6379",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid example config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"address":        "redis.example.com:6379",
			"password":       "s3cret",
			"cluster_mode":   true,
			"secure":         true,
			"prefix":         "rudder",
			"database":       "0",
			"ca_certificate": "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
			"skip_verify":    false,
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

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"address":        "redis.example.com:6379",
			"password":       "s3cret",
			"cluster_mode":   false,
			"secure":         true,
			"prefix":         "rudder",
			"database":       "1",
			"ca_certificate": "cert-data",
			"skip_verify":    true,
		})
		assert.Empty(t, errors)
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"address":     "redis.example.com:6379",
			"not_a_field": true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"address": "redis.example.com:6379",
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
			"address": "redis.example.com:6379",
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

func TestRedisConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := redis.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal address only",
			LocalJSON: `{
				"address": "redis.example.com:6379"
			}`,
			APIJSON: `{
				"address": "redis.example.com:6379"
			}`,
		},
		{
			Name: "full TF fields",
			LocalJSON: `{
				"address": "redis.example.com:6379",
				"password": "s3cret",
				"cluster_mode": false,
				"secure": true,
				"prefix": "rudder",
				"database": "1",
				"ca_certificate": "cert-data",
				"skip_verify": true
			}`,
			APIJSON: `{
				"address": "redis.example.com:6379",
				"password": "s3cret",
				"clusterMode": false,
				"secure": true,
				"prefix": "rudder",
				"database": "1",
				"caCertificate": "cert-data",
				"skipVerify": true
			}`,
		},
		{
			Name: "booleans false written without SkipZeroValue",
			LocalJSON: `{
				"address": "redis.example.com:6379",
				"cluster_mode": false,
				"secure": false,
				"skip_verify": false
			}`,
			APIJSON: `{
				"address": "redis.example.com:6379",
				"clusterMode": false,
				"secure": false,
				"skipVerify": false
			}`,
		},
		{
			Name: "consent for web",
			LocalJSON: `{
				"address": "redis.example.com:6379",
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
				"address": "redis.example.com:6379",
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
				"address": "redis.example.com:6379",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"address": "redis.example.com:6379",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
