package snowflake_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/snowflake"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func registeredDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(snowflake.NewDefinition()))
	registered, err := registry.Get("snowflake", 1)
	require.NoError(t, err)
	return registered
}

func minimalConfig() map[string]any {
	return map[string]any{
		"account":            "rudder-cli-e2e.us-east-1",
		"database":           "RUDDER_E2E",
		"warehouse":          "RUDDER_WH",
		"user":               "RUDDER_CLI_E2E",
		"use_key_pair_auth":  false,
		"password":           "s3cret",
		"sync_frequency":     "180",
		"use_rudder_storage": true,
	}
}

func copyConfig(src map[string]any) map[string]any {
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func assertHasPath(t *testing.T, errors []definitions.ConfigError, path string) {
	t.Helper()
	for _, err := range errors {
		if err.Path == path {
			return
		}
	}
	assert.Failf(t, "missing validation path", "expected path %s in %#v", path, errors)
}

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(snowflake.NewDefinition()))

	registered, err := registry.Get("snowflake", 1)
	require.NoError(t, err)

	assert.Equal(t, "snowflake", registered.Type)
	assert.Equal(t, "SNOWFLAKE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.GatedKeyPaths())

	// The full db-config secretKeys set. All eight are top-level, so none depend
	// on nested secret-path support.
	assert.Equal(t, []string{
		"password", "private_key", "private_key_passphrase",
		"access_key_id", "access_key", "account_key", "sas_token", "credentials",
	}, registered.SecretKeys())

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

	byAPI, err := registry.GetByAPIType("SNOWFLAKE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestSnowflakeConfigValidation(t *testing.T) {
	t.Parallel()

	registered := registeredDefinition(t)

	t.Run("required fields missing", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"account", "database", "warehouse", "user", "use_key_pair_auth", "sync_frequency", "use_rudder_storage"} {
			cfg := copyConfig(minimalConfig())
			delete(cfg, field)

			errors := registered.ValidateConfig(cfg)
			require.NotEmpty(t, errors, field)
			assertHasPath(t, errors, "/"+field)
		}
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(minimalConfig()))
	})

	t.Run("password required when key pair auth is off", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		delete(cfg, "password")

		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/password", errors[0].Path)
	})

	t.Run("private key required when key pair auth is on", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_key_pair_auth"] = true
		delete(cfg, "password")

		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/private_key", errors[0].Path)
	})

	t.Run("private key requires pem literal", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_key_pair_auth"] = true
		delete(cfg, "password")
		cfg["private_key"] = "raw-private-key"

		assertHasPath(t, registered.ValidateConfig(cfg), "/private_key")
	})

	// schema.json's PEM pattern is unanchored, so a key carrying the trailing
	// newline every .pem file ends with must still validate. Anchoring it would
	// reject a pasted key that the API accepts.
	t.Run("private key accepts surrounding whitespace", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{
			"-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n",
			"\n-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n",
			"-----BEGIN ENCRYPTED PRIVATE KEY-----\nabc\n-----END ENCRYPTED PRIVATE KEY-----\n",
		} {
			cfg := copyConfig(minimalConfig())
			cfg["use_key_pair_auth"] = true
			delete(cfg, "password")
			cfg["private_key"] = key

			assert.Empty(t, registered.ValidateConfig(cfg))
		}
	})

	t.Run("cloud provider required when rudder storage is off", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = false

		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/cloud_provider", errors[0].Path)
	})

	t.Run("cloud provider enum enforced", func(t *testing.T) {
		t.Parallel()

		for _, provider := range []string{"AWS", "GCP", "AZURE"} {
			cfg := copyConfig(minimalConfig())
			cfg["use_rudder_storage"] = false
			cfg["cloud_provider"] = provider
			switch provider {
			case "AWS":
				cfg["bucket_name"] = "rudder-bucket"
			case "GCP":
				cfg["bucket_name"] = "rudder-gcs"
				cfg["storage_integration"] = "RUDDER_GCS"
				cfg["credentials"] = "{}"
			case "AZURE":
				cfg["container_name"] = "rudder-logs"
				cfg["storage_integration"] = "RUDDER_AZURE"
				cfg["account_name"] = "rudderaccount"
			}
			assert.Empty(t, registered.ValidateConfig(cfg), provider)
		}

		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = false
		cfg["cloud_provider"] = "ORACLE"
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/cloud_provider", errors[0].Path)
	})

	t.Run("exclude window requires both boundaries when present", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["exclude_window"] = map[string]any{"start_time": "05:00"}

		assertHasPath(t, registered.ValidateConfig(cfg), "/exclude_window/end_time")
	})

	t.Run("aws storage requires bucket name", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = false
		cfg["cloud_provider"] = "AWS"

		assertHasPath(t, registered.ValidateConfig(cfg), "/bucket_name")
	})

	t.Run("aws omitted role selector uses backend default", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = false
		cfg["cloud_provider"] = "AWS"
		cfg["bucket_name"] = "rudder-bucket"

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("aws explicit role auth requires iam role arn", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = false
		cfg["cloud_provider"] = "AWS"
		cfg["bucket_name"] = "rudder-bucket"
		cfg["role_based_auth"] = true

		assertHasPath(t, registered.ValidateConfig(cfg), "/iam_role_arn")
	})

	t.Run("aws explicit key auth requires both access keys", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = false
		cfg["cloud_provider"] = "AWS"
		cfg["bucket_name"] = "rudder-bucket"
		cfg["role_based_auth"] = false
		cfg["access_key_id"] = "access-key-id"

		assertHasPath(t, registered.ValidateConfig(cfg), "/access_key")
	})

	t.Run("aws access key id template does not bypass literal pattern", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = false
		cfg["cloud_provider"] = "AWS"
		cfg["bucket_name"] = "rudder-bucket"
		cfg["role_based_auth"] = false
		cfg["access_key_id"] = "{{ config.key || " + strings.Repeat("a", 150) + " }}"
		cfg["access_key"] = "{{ config.secret || secret-access-key }}"

		assertHasPath(t, registered.ValidateConfig(cfg), "/access_key_id")
	})

	t.Run("gcp storage requires staging fields", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = false
		cfg["cloud_provider"] = "GCP"

		errors := registered.ValidateConfig(cfg)
		assertHasPath(t, errors, "/bucket_name")
		assertHasPath(t, errors, "/storage_integration")
		assertHasPath(t, errors, "/credentials")
	})

	t.Run("azure storage requires staging fields", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = false
		cfg["cloud_provider"] = "AZURE"

		errors := registered.ValidateConfig(cfg)
		assertHasPath(t, errors, "/container_name")
		assertHasPath(t, errors, "/storage_integration")
		assertHasPath(t, errors, "/account_name")
	})

	t.Run("azure omitted sas selector uses backend default", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = false
		cfg["cloud_provider"] = "AZURE"
		cfg["container_name"] = "azure-logs"
		cfg["storage_integration"] = "RUDDER_AZURE"
		cfg["account_name"] = "rudderaccount"

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("azure explicit account key auth requires account key", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = false
		cfg["cloud_provider"] = "AZURE"
		cfg["container_name"] = "azure-logs"
		cfg["storage_integration"] = "RUDDER_AZURE"
		cfg["account_name"] = "rudderaccount"
		cfg["use_sas_tokens"] = false

		assertHasPath(t, registered.ValidateConfig(cfg), "/account_key")
	})

	t.Run("azure explicit sas auth requires sas token", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = false
		cfg["cloud_provider"] = "AZURE"
		cfg["container_name"] = "azure-logs"
		cfg["storage_integration"] = "RUDDER_AZURE"
		cfg["account_name"] = "rudderaccount"
		cfg["use_sas_tokens"] = true

		assertHasPath(t, registered.ValidateConfig(cfg), "/sas_token")
	})

	// The full schema.json enum, including "10" which the definition originally omitted.
	t.Run("every schema sync frequency accepted", func(t *testing.T) {
		t.Parallel()
		for _, freq := range []string{"5", "10", "15", "30", "60", "180", "360", "720", "1440"} {
			cfg := copyConfig(minimalConfig())
			cfg["sync_frequency"] = freq
			assert.Empty(t, registered.ValidateConfig(cfg), freq)
		}

		cfg := copyConfig(minimalConfig())
		cfg["sync_frequency"] = "45"
		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/sync_frequency", errors[0].Path)
	})

	t.Run("namespace rejects reserved pg_ prefix", func(t *testing.T) {
		t.Parallel()
		for _, ns := range []string{"pg_catalog", "PG_x", "pG_x", "Pg_x"} {
			cfg := copyConfig(minimalConfig())
			cfg["namespace"] = ns
			errors := registered.ValidateConfig(cfg)
			require.NotEmpty(t, errors, ns)
			assert.Equal(t, "/namespace", errors[0].Path)
		}

		cfg := copyConfig(minimalConfig())
		cfg["namespace"] = "analytics_pg_data"
		assert.Empty(t, registered.ValidateConfig(cfg), "pg_ is only reserved as a prefix")
	})

	// The backend rejects a bad container name with a 400, so this must fail at
	// validate rather than at apply.
	t.Run("azure container name rules enforced", func(t *testing.T) {
		t.Parallel()

		for _, name := range []string{"ruddercliE2e", "ab", strings.Repeat("a", 64), "a--b", "-abc", "abc-"} {
			cfg := copyConfig(minimalConfig())
			cfg["container_name"] = name
			errors := registered.ValidateConfig(cfg)
			require.NotEmpty(t, errors, name)
			assert.Equal(t, "/container_name", errors[0].Path)
		}

		cfg := copyConfig(minimalConfig())
		cfg["container_name"] = "rudder-cli-e2e"
		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("single line fields reject line breaks", func(t *testing.T) {
		t.Parallel()
		for _, field := range []string{"account", "database", "warehouse", "user", "role", "prefix", "bucket_name"} {
			cfg := copyConfig(minimalConfig())
			cfg[field] = "bad\nvalue"
			errors := registered.ValidateConfig(cfg)
			require.NotEmpty(t, errors, field)
			assert.Equal(t, "/"+field, errors[0].Path)
		}
	})

	t.Run("pattern fields accept ui templates", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["account"] = "{{ config.account || " + strings.Repeat("a", 150) + " }}"
		cfg["namespace"] = "{{ config.namespace || rudder_events }}"
		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	// Upstream keeps every provider's keys in one flat object, so a config may
	// legitimately carry another provider's keys alongside the selected one.
	t.Run("cross provider keys accepted", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = false
		cfg["cloud_provider"] = "AZURE"
		cfg["bucket_name"] = "gcs"
		cfg["container_name"] = "azure-logs"
		cfg["storage_integration"] = "azure_int"
		cfg["account_name"] = "accountname"
		cfg["account_key"] = "key"
		cfg["role_based_auth"] = true
		cfg["iam_role_arn"] = ""

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["not_a_field"] = true

		errors := registered.ValidateConfig(cfg)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["consent_management"] = map[string]any{"warehouse": []any{}}

		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'warehouse' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["consent_management"] = map[string]any{
			"ios_swift": []any{map[string]any{"provider": "unknown"}},
		}

		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/ios_swift/0/provider", errors[0].Path)
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

func TestSnowflakeConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := snowflake.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal rudder storage",
			LocalJSON: `{
				"account": "rudder-cli-e2e.us-east-1",
				"database": "RUDDER_E2E",
				"warehouse": "RUDDER_WH",
				"user": "RUDDER_CLI_E2E",
				"use_key_pair_auth": false,
				"password": "s3cret",
				"sync_frequency": "180",
				"use_rudder_storage": true
			}`,
			APIJSON: `{
				"account": "rudder-cli-e2e.us-east-1",
				"database": "RUDDER_E2E",
				"warehouse": "RUDDER_WH",
				"user": "RUDDER_CLI_E2E",
				"useKeyPairAuth": false,
				"password": "s3cret",
				"syncFrequency": "180",
				"useRudderStorage": true
			}`,
		},
		{
			// Modelled on a real submitted payload: cloudProvider and roleBasedAuth
			// are first-class keys, every provider's keys travel together, and
			// unused ones are explicit empty strings rather than omitted.
			Name: "real world azure payload with cross provider keys",
			LocalJSON: `{
				"account": "qua-xxx-1",
				"database": "rudder",
				"warehouse": "rudder",
				"user": "rudder",
				"role": "",
				"use_key_pair_auth": false,
				"namespace": "",
				"sync_frequency": "180",
				"use_rudder_storage": false,
				"prefer_append": true,
				"skip_users_table": true,
				"skip_tracks_table": false,
				"json_paths": "",
				"password": "pass",
				"cloud_provider": "AZURE",
				"prefix": "",
				"cleanup_object_storage_files": false,
				"bucket_name": "gcs",
				"storage_integration": "azure_int",
				"role_based_auth": true,
				"enable_sse": false,
				"iam_role_arn": "",
				"private_key": "",
				"private_key_passphrase": "",
				"credentials": "",
				"container_name": "azure-logs",
				"account_name": "accountname",
				"use_sas_tokens": false,
				"account_key": "key"
			}`,
			APIJSON: `{
				"account": "qua-xxx-1",
				"database": "rudder",
				"warehouse": "rudder",
				"user": "rudder",
				"role": "",
				"useKeyPairAuth": false,
				"namespace": "",
				"syncFrequency": "180",
				"useRudderStorage": false,
				"preferAppend": true,
				"skipUsersTable": true,
				"skipTracksTable": false,
				"jsonPaths": "",
				"password": "pass",
				"cloudProvider": "AZURE",
				"prefix": "",
				"cleanupObjectStorageFiles": false,
				"bucketName": "gcs",
				"storageIntegration": "azure_int",
				"roleBasedAuth": true,
				"enableSSE": false,
				"iamRoleARN": "",
				"privateKey": "",
				"privateKeyPassphrase": "",
				"credentials": "",
				"containerName": "azure-logs",
				"accountName": "accountname",
				"useSASTokens": false,
				"accountKey": "key"
			}`,
		},
		{
			Name: "key pair auth with s3 role based auth",
			LocalJSON: `{
				"account": "rudder-cli-e2e.us-east-1",
				"database": "RUDDER_E2E",
				"warehouse": "RUDDER_WH",
				"user": "RUDDER_CLI_E2E",
				"use_key_pair_auth": true,
				"private_key": "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----",
				"private_key_passphrase": "phrase",
				"sync_frequency": "1440",
				"use_rudder_storage": false,
				"cloud_provider": "AWS",
				"bucket_name": "rudder-bucket",
				"storage_integration": "RUDDER_S3",
				"role_based_auth": true,
				"iam_role_arn": "arn:aws:iam::000000000000:role/rudder"
			}`,
			APIJSON: `{
				"account": "rudder-cli-e2e.us-east-1",
				"database": "RUDDER_E2E",
				"warehouse": "RUDDER_WH",
				"user": "RUDDER_CLI_E2E",
				"useKeyPairAuth": true,
				"privateKey": "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----",
				"privateKeyPassphrase": "phrase",
				"syncFrequency": "1440",
				"useRudderStorage": false,
				"cloudProvider": "AWS",
				"bucketName": "rudder-bucket",
				"storageIntegration": "RUDDER_S3",
				"roleBasedAuth": true,
				"iamRoleARN": "arn:aws:iam::000000000000:role/rudder"
			}`,
		},
		{
			Name: "gcp storage and exclude window reshape",
			LocalJSON: `{
				"account": "rudder-cli-e2e.us-east-1",
				"database": "RUDDER_E2E",
				"warehouse": "RUDDER_WH",
				"user": "RUDDER_CLI_E2E",
				"use_key_pair_auth": false,
				"password": "s3cret",
				"sync_frequency": "360",
				"sync_start_at": "02:00",
				"exclude_window": {
					"start_time": "05:00",
					"end_time": "06:00"
				},
				"use_rudder_storage": false,
				"cloud_provider": "GCP",
				"bucket_name": "rudder-gcs",
				"storage_integration": "RUDDER_GCS",
				"credentials": "{\"type\":\"service_account\"}"
			}`,
			APIJSON: `{
				"account": "rudder-cli-e2e.us-east-1",
				"database": "RUDDER_E2E",
				"warehouse": "RUDDER_WH",
				"user": "RUDDER_CLI_E2E",
				"useKeyPairAuth": false,
				"password": "s3cret",
				"syncFrequency": "360",
				"syncStartAt": "02:00",
				"excludeWindow": {
					"excludeWindowStartTime": "05:00",
					"excludeWindowEndTime": "06:00"
				},
				"useRudderStorage": false,
				"cloudProvider": "GCP",
				"bucketName": "rudder-gcs",
				"storageIntegration": "RUDDER_GCS",
				"credentials": "{\"type\":\"service_account\"}"
			}`,
		},
		{
			// Terraform passes SkipZeroValue on 21 mappings; porting it would drop
			// these keys from the payload and surface as a phantom diff every plan.
			Name: "zero values survive without SkipZeroValue",
			LocalJSON: `{
				"account": "rudder-cli-e2e.us-east-1",
				"database": "RUDDER_E2E",
				"warehouse": "RUDDER_WH",
				"user": "RUDDER_CLI_E2E",
				"use_key_pair_auth": false,
				"password": "",
				"role": "",
				"namespace": "",
				"json_paths": "",
				"prefix": "",
				"sync_frequency": "180",
				"use_rudder_storage": false,
				"cloud_provider": "AWS",
				"skip_tracks_table": false,
				"prefer_append": false,
				"enable_sse": false,
				"use_sas_tokens": false,
				"underscore_divide_numbers": false,
				"allow_users_context_traits": false
			}`,
			APIJSON: `{
				"account": "rudder-cli-e2e.us-east-1",
				"database": "RUDDER_E2E",
				"warehouse": "RUDDER_WH",
				"user": "RUDDER_CLI_E2E",
				"useKeyPairAuth": false,
				"password": "",
				"role": "",
				"namespace": "",
				"jsonPaths": "",
				"prefix": "",
				"syncFrequency": "180",
				"useRudderStorage": false,
				"cloudProvider": "AWS",
				"skipTracksTable": false,
				"preferAppend": false,
				"enableSSE": false,
				"useSASTokens": false,
				"underscoreDivideNumbers": false,
				"allowUsersContextTraits": false
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"account": "rudder-cli-e2e.us-east-1",
				"database": "RUDDER_E2E",
				"warehouse": "RUDDER_WH",
				"user": "RUDDER_CLI_E2E",
				"use_key_pair_auth": false,
				"password": "s3cret",
				"sync_frequency": "180",
				"use_rudder_storage": true,
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"account": "rudder-cli-e2e.us-east-1",
				"database": "RUDDER_E2E",
				"warehouse": "RUDDER_WH",
				"user": "RUDDER_CLI_E2E",
				"useKeyPairAuth": false,
				"password": "s3cret",
				"syncFrequency": "180",
				"useRudderStorage": true,
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
