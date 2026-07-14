package s3_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/s3"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(s3.NewDefinition()))

	registered, err := registry.Get("s3", 1)
	require.NoError(t, err)

	assert.Equal(t, "s3", registered.Type)
	assert.Equal(t, "S3", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"access_key_id", "access_key"}, registered.SecretKeys())

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

	byAPI, err := registry.GetByAPIType("S3", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestS3ConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(s3.NewDefinition()))
	registered, err := registry.Get("s3", 1)
	require.NoError(t, err)

	t.Run("missing bucket_name", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/bucket_name", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": "my-bucket",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid with secrets and consent", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name":   "my-bucket",
			"prefix":        "rudder/",
			"access_key_id": "AKIA...",
			"access_key":    "secret",
			"enable_sse":    true,
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

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"bucket_name":     "my-bucket",
			"role_based_auth": true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/role_based_auth", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"bucket_name": "my-bucket",
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
			"bucket_name": "my-bucket",
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

func TestS3ConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := s3.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal bucket only",
			LocalJSON: `{
				"bucket_name": "my-bucket"
			}`,
			APIJSON: `{
				"bucketName": "my-bucket"
			}`,
		},
		{
			Name: "full TF fields",
			LocalJSON: `{
				"bucket_name": "my-bucket",
				"prefix": "rudder/",
				"access_key_id": "AKIAEXAMPLE",
				"access_key": "secret-value",
				"enable_sse": true
			}`,
			APIJSON: `{
				"bucketName": "my-bucket",
				"prefix": "rudder/",
				"accessKeyID": "AKIAEXAMPLE",
				"accessKey": "secret-value",
				"enableSSE": true
			}`,
		},
		{
			Name: "consent for web",
			LocalJSON: `{
				"bucket_name": "my-bucket",
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
				"bucketName": "my-bucket",
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
				"bucket_name": "my-bucket",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"bucketName": "my-bucket",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
