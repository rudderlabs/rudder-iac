package s3datalake_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	s3datalake "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/s3_datalake"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func registeredDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(s3datalake.NewDefinition()))
	registered, err := registry.Get("s3_datalake", 1)
	require.NoError(t, err)
	return registered
}

func minimalRoleConfig() map[string]any {
	return map[string]any{
		"bucket_name":     "rudder-s3-datalake",
		"use_glue":        false,
		"role_based_auth": true,
		"iam_role_arn":    "arn:aws:iam::123456789012:role/S3DatalakeAccess",
		"sync_frequency":  "180",
	}
}

func validKeyConfig() map[string]any {
	cfg := copyConfig(minimalRoleConfig())
	cfg["role_based_auth"] = false
	delete(cfg, "iam_role_arn")
	cfg["access_key_id"] = "AKIAS3DATALAKE"
	cfg["access_key"] = "secret-access-key"
	return cfg
}

func exampleConfig() map[string]any {
	cfg := copyConfig(minimalRoleConfig())
	cfg["use_glue"] = true
	cfg["region"] = "us-east-1"
	cfg["namespace"] = "analytics"
	cfg["prefix"] = "rudder/"
	cfg["sync_frequency"] = "60"
	cfg["sync_start_at"] = "10:00"
	return cfg
}

func copyConfig(src map[string]any) map[string]any {
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(s3datalake.NewDefinition()))

	registered, err := registry.Get("s3_datalake", 1)
	require.NoError(t, err)

	assert.Equal(t, "s3_datalake", registered.Type)
	assert.Equal(t, "S3_DATALAKE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.GatedKeyPaths())
	assert.Equal(t, []string{"password", "access_key_id", "access_key"}, registered.SecretKeys())

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

	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")

	byAPI, err := registry.GetByAPIType("S3_DATALAKE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestS3DatalakeConfigValidation(t *testing.T) {
	t.Parallel()

	registered := registeredDefinition(t)

	t.Run("required fields missing", func(t *testing.T) {
		t.Parallel()

		// schema.json requires only bucketName; role_based_auth is also required
		// because it selects between two credential shapes, so absent must differ
		// from false.
		for _, field := range []string{"bucket_name", "role_based_auth"} {
			cfg := copyConfig(minimalRoleConfig())
			delete(cfg, field)

			assertHasPath(t, registered.ValidateConfig(cfg), "/"+field)
		}
	})

	// use_glue only gates region in one direction (required_if=UseGlue true), so
	// absent and false behave alike and schema.json's default:false settles it.
	t.Run("use_glue is optional", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalRoleConfig())
		delete(cfg, "use_glue")
		assert.Empty(t, registered.ValidateConfig(cfg))

		assert.Equal(t, false, registered.ApplyDefaults(cfg)["use_glue"])
	})

	t.Run("sync frequency is optional", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalRoleConfig())
		delete(cfg, "sync_frequency")
		assert.Empty(t, registered.ValidateConfig(cfg), "schema.json does not require syncFrequency")
	})

	t.Run("valid minimal role based auth", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(minimalRoleConfig()))
	})

	t.Run("valid access key auth", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(validKeyConfig()))
	})

	t.Run("validated example config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(exampleConfig()))
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(validKeyConfig())
		cfg["password"] = "legacy-secret"
		cfg["use_glue"] = true
		cfg["region"] = "us-east-1"
		cfg["prefix"] = "rudder/"
		cfg["namespace"] = "analytics"
		cfg["enable_sse"] = true
		cfg["sync_frequency"] = "30"
		cfg["sync_start_at"] = "10:00"
		cfg["skip_tracks_table"] = false
		cfg["skip_users_table"] = true
		cfg["time_window_layout"] = "dt=2006-01-02"
		cfg["underscore_divide_numbers"] = false
		cfg["cleanup_object_storage_files"] = true
		cfg["allow_users_context_traits"] = false
		cfg["consent_management"] = map[string]any{
			"cloud": []any{map[string]any{"provider": "oneTrust", "consents": []any{"analytics"}}},
		}

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("region required when use_glue true", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalRoleConfig())
		cfg["use_glue"] = true

		assertHasPath(t, registered.ValidateConfig(cfg), "/region")
	})

	t.Run("region optional when use_glue false", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalRoleConfig())
		cfg["use_glue"] = false
		delete(cfg, "region")

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("iam_role_arn required when role_based_auth true", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalRoleConfig())
		delete(cfg, "iam_role_arn")

		assertHasPath(t, registered.ValidateConfig(cfg), "/iam_role_arn")
	})

	t.Run("access keys required when role_based_auth false", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalRoleConfig())
		cfg["role_based_auth"] = false
		delete(cfg, "iam_role_arn")

		errors := registered.ValidateConfig(cfg)
		assertHasPath(t, errors, "/access_key_id")
		assertHasPath(t, errors, "/access_key")
	})

	t.Run("opposite auth mode keys stay optional for import compatibility", func(t *testing.T) {
		t.Parallel()
		roleCfg := copyConfig(minimalRoleConfig())
		roleCfg["access_key_id"] = "stale-key-id"
		roleCfg["access_key"] = "stale-secret-key"
		assert.Empty(t, registered.ValidateConfig(roleCfg))

		keyCfg := validKeyConfig()
		keyCfg["iam_role_arn"] = "arn:aws:iam::123456789012:role/StaleRole"
		assert.Empty(t, registered.ValidateConfig(keyCfg))
	})

	t.Run("sync frequency enum enforced", func(t *testing.T) {
		t.Parallel()

		for _, frequency := range []string{"5", "10", "15", "30", "60", "180", "360", "720", "1440"} {
			cfg := copyConfig(minimalRoleConfig())
			cfg["sync_frequency"] = frequency
			assert.Empty(t, registered.ValidateConfig(cfg), frequency)
		}

		cfg := copyConfig(minimalRoleConfig())
		cfg["sync_frequency"] = "45"
		assertHasPath(t, registered.ValidateConfig(cfg), "/sync_frequency")
	})

	t.Run("bucket name pattern rejects invalid literals", func(t *testing.T) {
		t.Parallel()

		for _, bucketName := range []string{"BadBucket", "xn--bucket", "bucket..name", "192.168.0.1", "a", "bad_bucket"} {
			cfg := copyConfig(minimalRoleConfig())
			cfg["bucket_name"] = bucketName
			assertHasPath(t, registered.ValidateConfig(cfg), "/bucket_name")
		}
	})

	t.Run("namespace rejects reserved prefix and line breaks", func(t *testing.T) {
		t.Parallel()

		for _, namespace := range []string{"pg_catalog", "PG_catalog", "analytics\nwarehouse", strings.Repeat("a", 65)} {
			cfg := copyConfig(minimalRoleConfig())
			cfg["namespace"] = namespace
			assertHasPath(t, registered.ValidateConfig(cfg), "/namespace")
		}
	})

	// schema.json constrains these inside the roleBasedAuth:false branch, so a
	// multiline literal must be rejected locally rather than at apply.
	t.Run("access keys reject line breaks", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"access_key_id", "access_key"} {
			cfg := copyConfig(validKeyConfig())
			cfg[field] = "AKIA\nBROKEN"
			assertHasPath(t, registered.ValidateConfig(cfg), "/"+field)
		}
	})

	t.Run("iam role arn rejects invalid literals", func(t *testing.T) {
		t.Parallel()

		for _, value := range []string{"arn:aws:iam::123456789012:role/" + strings.Repeat("a", 120), "arn\nbroken"} {
			cfg := copyConfig(minimalRoleConfig())
			cfg["iam_role_arn"] = value
			assertHasPath(t, registered.ValidateConfig(cfg), "/iam_role_arn")
		}
	})

	t.Run("pattern fields accept ui templates", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalRoleConfig())
		cfg["bucket_name"] = "{{ config.bucketName || rudder-s3-datalake }}"
		cfg["namespace"] = "{{ config.namespace || pg_events }}"
		cfg["iam_role_arn"] = "{{ config.iamRoleARN || " + strings.Repeat("a", 150) + " }}"
		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalRoleConfig())
		cfg["not_a_field"] = true

		assertHasPath(t, registered.ValidateConfig(cfg), "/not_a_field")
	})

	// Legacy consent blocks are declared in schema.json includeKeys, but the backend
	// migrates them into consentManagement. Model only the canonical block so apply
	// does not repeatedly re-send a key the backend drops.
	t.Run("legacy consent blocks are not supported keys", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			cfg := copyConfig(minimalRoleConfig())
			cfg[key] = map[string]any{"web": []any{}}

			errors := registered.ValidateConfig(cfg)
			require.Len(t, errors, 1, key)
			assert.Equal(t, "/"+key, errors[0].Path)
			assert.Contains(t, errors[0].Message, "unknown config field")
		}
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalRoleConfig())
		cfg["consent_management"] = map[string]any{"warehouse": []any{}}

		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'warehouse' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalRoleConfig())
		cfg["consent_management"] = map[string]any{
			"cloud": []any{map[string]any{"provider": "unknown"}},
		}

		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/cloud/0/provider", errors[0].Path)
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

func assertHasPath(t *testing.T, errors []definitions.ConfigError, path string) {
	t.Helper()

	for _, err := range errors {
		if err.Path == path {
			return
		}
	}
	assert.Failf(t, "expected validation path", "path %s not found in %#v", path, errors)
}

func TestS3DatalakeConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := s3datalake.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal role auth",
			LocalJSON: `{
				"bucket_name": "rudder-s3-datalake",
				"use_glue": false,
				"role_based_auth": true,
				"iam_role_arn": "arn:aws:iam::123456789012:role/S3DatalakeAccess",
				"sync_frequency": "180"
			}`,
			APIJSON: `{
				"bucketName": "rudder-s3-datalake",
				"useGlue": false,
				"roleBasedAuth": true,
				"iamRoleARN": "arn:aws:iam::123456789012:role/S3DatalakeAccess",
				"syncFrequency": "180"
			}`,
		},
		{
			Name: "access key auth",
			LocalJSON: `{
				"bucket_name": "rudder-s3-datalake",
				"use_glue": false,
				"role_based_auth": false,
				"access_key_id": "AKIAS3DATALAKE",
				"access_key": "secret-access-key",
				"sync_frequency": "30"
			}`,
			APIJSON: `{
				"bucketName": "rudder-s3-datalake",
				"useGlue": false,
				"roleBasedAuth": false,
				"accessKeyID": "AKIAS3DATALAKE",
				"accessKey": "secret-access-key",
				"syncFrequency": "30"
			}`,
		},
		{
			Name: "glue region and flat sync settings",
			LocalJSON: `{
				"bucket_name": "rudder-s3-datalake",
				"use_glue": true,
				"region": "us-east-1",
				"role_based_auth": true,
				"iam_role_arn": "arn:aws:iam::123456789012:role/S3DatalakeAccess",
				"sync_frequency": "60",
				"sync_start_at": "10:00"
			}`,
			APIJSON: `{
				"bucketName": "rudder-s3-datalake",
				"useGlue": true,
				"region": "us-east-1",
				"roleBasedAuth": true,
				"iamRoleARN": "arn:aws:iam::123456789012:role/S3DatalakeAccess",
				"syncFrequency": "60",
				"syncStartAt": "10:00"
			}`,
		},
		{
			Name: "full mapped and schema only fields",
			LocalJSON: `{
				"bucket_name": "rudder-s3-datalake",
				"use_glue": true,
				"region": "us-east-1",
				"prefix": "rudder/",
				"namespace": "analytics",
				"role_based_auth": false,
				"access_key_id": "AKIAS3DATALAKE",
				"access_key": "secret-access-key",
				"password": "legacy-secret",
				"enable_sse": true,
				"sync_frequency": "30",
				"sync_start_at": "10:00",
				"skip_tracks_table": false,
				"skip_users_table": true,
				"time_window_layout": "dt=2006-01-02",
				"underscore_divide_numbers": false,
				"cleanup_object_storage_files": true,
				"allow_users_context_traits": false
			}`,
			APIJSON: `{
				"bucketName": "rudder-s3-datalake",
				"useGlue": true,
				"region": "us-east-1",
				"prefix": "rudder/",
				"namespace": "analytics",
				"roleBasedAuth": false,
				"accessKeyID": "AKIAS3DATALAKE",
				"accessKey": "secret-access-key",
				"password": "legacy-secret",
				"enableSSE": true,
				"syncFrequency": "30",
				"syncStartAt": "10:00",
				"skipTracksTable": false,
				"skipUsersTable": true,
				"timeWindowLayout": "dt=2006-01-02",
				"underscoreDivideNumbers": false,
				"cleanupObjectStorageFiles": true,
				"allowUsersContextTraits": false
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"bucket_name": "rudder-s3-datalake",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"bucketName": "rudder-s3-datalake",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
