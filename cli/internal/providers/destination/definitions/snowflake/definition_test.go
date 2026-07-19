package snowflake_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/snowflake"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(snowflake.NewDefinition()))

	registered, err := registry.Get("snowflake", 1)
	require.NoError(t, err)

	assert.Equal(t, "snowflake", registered.Type)
	assert.Equal(t, "SNOWFLAKE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"password", "private_key", "private_key_passphrase"}, registered.SecretKeys())

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
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")
	assert.Empty(t, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("SNOWFLAKE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestSnowflakeConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(snowflake.NewDefinition()))
	registered, err := registry.Get("snowflake", 1)
	require.NoError(t, err)

	minimalValid := map[string]any{
		"account":            "xy12345.us-east-1",
		"database":           "ANALYTICS",
		"warehouse":          "COMPUTE_WH",
		"user":               "rudder_user",
		"use_key_pair_auth":  false,
		"password":           "secret-password",
		"use_rudder_storage": true,
		"sync": map[string]any{
			"frequency": "180",
		},
	}

	t.Run("missing account", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		delete(cfg, "account")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/account", errors[0].Path)
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

	t.Run("missing warehouse", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		delete(cfg, "warehouse")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/warehouse", errors[0].Path)
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

	t.Run("use_rudder_storage required", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		delete(cfg, "use_rudder_storage")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/use_rudder_storage", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("use_key_pair_auth required", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		delete(cfg, "use_key_pair_auth")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/use_key_pair_auth", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("password required when use_key_pair_auth false", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		delete(cfg, "password")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/password", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("private_key required when use_key_pair_auth true", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["use_key_pair_auth"] = true
		delete(cfg, "password")
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/private_key", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal rudder storage password auth", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(copyConfig(minimalValid))
		assert.Empty(t, errors)
	})

	t.Run("valid key pair auth", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["use_key_pair_auth"] = true
		delete(cfg, "password")
		cfg["private_key"] = "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----"
		cfg["private_key_passphrase"] = "pass"
		errors := registered.ValidateConfig(cfg)
		assert.Empty(t, errors)
	})

	t.Run("valid s3 role based auth", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["use_rudder_storage"] = false
		cfg["s3"] = map[string]any{
			"bucket_name": "my-snowflake-bucket",
			"role_based_authentication": map[string]any{
				"i_am_role_arn": "arn:aws:iam::123456789012:role/SnowflakeAccess",
			},
		}
		errors := registered.ValidateConfig(cfg)
		assert.Empty(t, errors)
	})

	t.Run("valid s3 access key auth", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["use_rudder_storage"] = false
		cfg["s3"] = map[string]any{
			"bucket_name":   "my-snowflake-bucket",
			"access_key_id": "AKIAEXAMPLE",
			"access_key":    "secret-value",
			"enable_sse":    true,
		}
		errors := registered.ValidateConfig(cfg)
		assert.Empty(t, errors)
	})

	t.Run("s3 access keys excluded with role based auth", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["use_rudder_storage"] = false
		cfg["s3"] = map[string]any{
			"bucket_name":   "my-snowflake-bucket",
			"access_key_id": "AKIAEXAMPLE",
			"access_key":    "secret-value",
			"role_based_authentication": map[string]any{
				"i_am_role_arn": "arn:aws:iam::123456789012:role/SnowflakeAccess",
			},
		}
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		paths := map[string]bool{}
		for _, err := range errors {
			paths[err.Path] = true
		}
		assert.True(t, paths["/s3/access_key_id"] || paths["/s3/access_key"] || paths["/s3/role_based_authentication"])
	})

	t.Run("valid gcp storage", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["use_rudder_storage"] = false
		cfg["gcp"] = map[string]any{
			"bucket_name":         "my-gcs-bucket",
			"credentials":         `{"type":"service_account"}`,
			"storage_integration": "gcs_int",
		}
		errors := registered.ValidateConfig(cfg)
		assert.Empty(t, errors)
	})

	t.Run("gcp missing credentials", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["use_rudder_storage"] = false
		cfg["gcp"] = map[string]any{
			"bucket_name":         "my-gcs-bucket",
			"storage_integration": "gcs_int",
		}
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/gcp/credentials", errors[0].Path)
	})

	t.Run("valid azure storage", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["use_rudder_storage"] = false
		cfg["azure"] = map[string]any{
			"container_name":      "my-container",
			"account_name":        "mystorageaccount",
			"account_key":         "account-key",
			"storage_integration": "azure_int",
		}
		errors := registered.ValidateConfig(cfg)
		assert.Empty(t, errors)
	})

	t.Run("valid full config with consent", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalValid)
		cfg["role"] = "RUDDER_ROLE"
		cfg["namespace"] = "RUDDER"
		cfg["prefix"] = "rudder/"
		cfg["skip_tracks_table"] = true
		cfg["skip_users_table"] = false
		cfg["prefer_append"] = true
		cfg["manual_sync"] = false
		cfg["json_paths"] = "properties.cart"
		cfg["sync"] = map[string]any{
			"frequency":                 "60",
			"start_at":                  "10:00",
			"exclude_window_start_time": "11:00",
			"exclude_window_end_time":   "12:00",
		}
		cfg["consent_management"] = map[string]any{
			"web": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{"analytics"},
				},
			},
		}
		errors := registered.ValidateConfig(cfg)
		assert.Empty(t, errors)
	})

	t.Run("example yaml config", func(t *testing.T) {
		t.Parallel()
		// Exact config from the onboarding example YAML.
		errors := registered.ValidateConfig(map[string]any{
			"account":            "xy12345.us-east-1",
			"database":           "ANALYTICS",
			"warehouse":          "COMPUTE_WH",
			"user":               "rudder_user",
			"use_key_pair_auth":  false,
			"password":           "secret-password",
			"role":               "RUDDER_ROLE",
			"namespace":          "RUDDER",
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

func TestSnowflakeConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := snowflake.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal rudder storage",
			LocalJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"use_key_pair_auth": false,
				"password": "secret-password",
				"use_rudder_storage": true,
				"sync": {"frequency": "180"}
			}`,
			APIJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"useKeyPairAuth": false,
				"password": "secret-password",
				"useRudderStorage": true,
				"syncFrequency": "180"
			}`,
		},
		{
			Name: "key pair auth",
			LocalJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"use_key_pair_auth": true,
				"private_key": "-----BEGIN PRIVATE KEY-----\nKEY\n-----END PRIVATE KEY-----",
				"private_key_passphrase": "pass",
				"use_rudder_storage": true,
				"sync": {"frequency": "30"}
			}`,
			APIJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"useKeyPairAuth": true,
				"privateKey": "-----BEGIN PRIVATE KEY-----\nKEY\n-----END PRIVATE KEY-----",
				"privateKeyPassphrase": "pass",
				"useRudderStorage": true,
				"syncFrequency": "30"
			}`,
		},
		{
			Name: "s3 access key auth",
			LocalJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"use_key_pair_auth": false,
				"password": "secret-password",
				"use_rudder_storage": false,
				"sync": {"frequency": "60"},
				"prefix": "rudder/",
				"s3": {
					"bucket_name": "example-bucket",
					"access_key_id": "AKIAEXAMPLE",
					"access_key": "secret-value",
					"enable_sse": true
				}
			}`,
			APIJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"useKeyPairAuth": false,
				"password": "secret-password",
				"useRudderStorage": false,
				"syncFrequency": "60",
				"prefix": "rudder/",
				"cloudProvider": "AWS",
				"roleBasedAuth": false,
				"bucketName": "example-bucket",
				"accessKeyID": "AKIAEXAMPLE",
				"accessKey": "secret-value",
				"enableSSE": true
			}`,
		},
		{
			Name: "s3 role based auth",
			LocalJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"use_key_pair_auth": false,
				"password": "secret-password",
				"use_rudder_storage": false,
				"sync": {"frequency": "60"},
				"s3": {
					"bucket_name": "example-bucket",
					"role_based_authentication": {
						"i_am_role_arn": "arn:aws:iam::123456789012:role/SnowflakeAccess"
					},
					"storage_integration": "s3_int"
				}
			}`,
			APIJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"useKeyPairAuth": false,
				"password": "secret-password",
				"useRudderStorage": false,
				"syncFrequency": "60",
				"cloudProvider": "AWS",
				"roleBasedAuth": true,
				"bucketName": "example-bucket",
				"iamRoleARN": "arn:aws:iam::123456789012:role/SnowflakeAccess",
				"storageIntegration": "s3_int"
			}`,
		},
		{
			Name: "gcp storage",
			LocalJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"use_key_pair_auth": false,
				"password": "secret-password",
				"use_rudder_storage": false,
				"sync": {"frequency": "60"},
				"gcp": {
					"bucket_name": "example-gcs-bucket",
					"credentials": "{\"type\":\"service_account\"}",
					"storage_integration": "gcs_int"
				}
			}`,
			APIJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"useKeyPairAuth": false,
				"password": "secret-password",
				"useRudderStorage": false,
				"syncFrequency": "60",
				"cloudProvider": "GCP",
				"bucketName": "example-gcs-bucket",
				"credentials": "{\"type\":\"service_account\"}",
				"storageIntegration": "gcs_int"
			}`,
		},
		{
			Name: "azure storage",
			LocalJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"use_key_pair_auth": false,
				"password": "secret-password",
				"use_rudder_storage": false,
				"sync": {"frequency": "60"},
				"azure": {
					"container_name": "example-container",
					"account_name": "mystorageaccount",
					"account_key": "account-key",
					"storage_integration": "azure_int"
				}
			}`,
			APIJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"useKeyPairAuth": false,
				"password": "secret-password",
				"useRudderStorage": false,
				"syncFrequency": "60",
				"cloudProvider": "AZURE",
				"containerName": "example-container",
				"accountName": "mystorageaccount",
				"accountKey": "account-key",
				"storageIntegration": "azure_int"
			}`,
		},
		{
			Name: "sync exclude window reshape",
			LocalJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"use_key_pair_auth": false,
				"password": "secret-password",
				"use_rudder_storage": true,
				"sync": {
					"frequency": "60",
					"start_at": "10:00",
					"exclude_window_start_time": "11:00",
					"exclude_window_end_time": "12:00"
				}
			}`,
			APIJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"useKeyPairAuth": false,
				"password": "secret-password",
				"useRudderStorage": true,
				"syncFrequency": "60",
				"syncStartAt": "10:00",
				"excludeWindow": {
					"excludeWindowStartTime": "11:00",
					"excludeWindowEndTime": "12:00"
				}
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"use_key_pair_auth": false,
				"password": "secret-password",
				"use_rudder_storage": true,
				"sync": {"frequency": "180"},
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"account": "xy12345.us-east-1",
				"database": "ANALYTICS",
				"warehouse": "COMPUTE_WH",
				"user": "rudder_user",
				"useKeyPairAuth": false,
				"password": "secret-password",
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
			out[k] = copyConfig(nested)
			continue
		}
		out[k] = v
	}
	return out
}
