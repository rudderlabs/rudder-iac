package rs_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/rs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func registeredDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(rs.NewDefinition()))
	registered, err := registry.Get("rs", 1)
	require.NoError(t, err)
	return registered
}

func minimalPasswordConfig() map[string]any {
	return map[string]any{
		"use_iam_for_auth":   false,
		"host":               "redshift.example.com",
		"port":               "5439",
		"database":           "analytics",
		"user":               "rudder",
		"password":           "secret",
		"sync_frequency":     "180",
		"use_rudder_storage": true,
	}
}

func exampleConfig() map[string]any {
	cfg := copyConfig(minimalPasswordConfig())
	cfg["password"] = "{{ .RS_PASSWORD }}"
	cfg["namespace"] = "rudder_events"
	cfg["skip_tracks_table"] = false
	cfg["skip_users_table"] = true
	cfg["prefer_append"] = true
	cfg["json_paths"] = "context.traits,properties.metadata"
	return cfg
}

func fullConfig() map[string]any {
	cfg := copyConfig(minimalPasswordConfig())
	cfg["use_iam_for_auth"] = true
	cfg["iam_role_arn_for_auth"] = "arn:aws:iam::123456789012:role/RedshiftAccess"
	cfg["cluster_region"] = "us-east-1"
	cfg["use_serverless"] = false
	cfg["cluster_id"] = "rudder-redshift-cluster"
	cfg["workgroup_name"] = "rudder-redshift-workgroup"
	cfg["namespace"] = "rudder_events"
	cfg["use_ssh"] = true
	cfg["ssh_host"] = "bastion.example.com"
	cfg["ssh_port"] = "22"
	cfg["ssh_user"] = "rudder"
	cfg["ssh_public_key"] = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDrudder"
	cfg["sync_frequency"] = "10"
	cfg["sync_start_at"] = "01:00"
	cfg["exclude_window"] = map[string]any{"start_time": "02:00", "end_time": "03:00"}
	cfg["skip_tracks_table"] = false
	cfg["skip_users_table"] = true
	cfg["prefer_append"] = true
	cfg["json_paths"] = "context.traits"
	cfg["underscore_divide_numbers"] = false
	cfg["allow_users_context_traits"] = false
	cfg["use_rudder_storage"] = false
	cfg["bucket_name"] = "rudder-redshift-staging"
	cfg["iam_role_arn"] = "arn:aws:iam::123456789012:role/RudderRedshiftStorage"
	cfg["role_based_auth"] = true
	cfg["access_key_id"] = "AKIAEXAMPLE"
	cfg["access_key"] = "secret-access-key"
	cfg["prefix"] = "rudder/redshift"
	cfg["enable_sse"] = true
	cfg["cleanup_object_storage_files"] = false
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
	require.NoError(t, registry.Register(rs.NewDefinition()))

	registered, err := registry.Get("rs", 1)
	require.NoError(t, err)

	assert.Equal(t, "rs", registered.Type)
	assert.Equal(t, "RS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"password", "access_key_id", "access_key"}, registered.SecretKeys())
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

	byAPI, err := registry.GetByAPIType("RS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestRSConfigValidation(t *testing.T) {
	t.Parallel()

	registered := registeredDefinition(t)

	t.Run("required fields missing", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"use_iam_for_auth", "database", "user", "sync_frequency", "use_rudder_storage"} {
			cfg := copyConfig(minimalPasswordConfig())
			delete(cfg, field)

			assertHasPath(t, registered.ValidateConfig(cfg), "/"+field)
		}
	})

	t.Run("password auth fields required when iam auth is off", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"host", "port", "password"} {
			cfg := copyConfig(minimalPasswordConfig())
			delete(cfg, field)

			assertHasPath(t, registered.ValidateConfig(cfg), "/"+field)
		}
	})

	t.Run("valid password auth", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(minimalPasswordConfig()))
	})

	t.Run("validated example config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(exampleConfig()))
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		cfg := fullConfig()
		cfg["consent_management"] = map[string]any{
			"cloud": []any{map[string]any{
				"provider":            "custom",
				"resolution_strategy": "or",
				"consents":            []any{"analytics", "marketing"},
			}},
		}

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("iam cluster auth requires provisioned cluster fields", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalPasswordConfig())
		cfg["use_iam_for_auth"] = true
		cfg["use_serverless"] = false
		delete(cfg, "host")
		delete(cfg, "port")
		delete(cfg, "password")

		errors := registered.ValidateConfig(cfg)
		assertHasPath(t, errors, "/iam_role_arn_for_auth")
		assertHasPath(t, errors, "/cluster_region")
		assertHasPath(t, errors, "/cluster_id")
	})

	t.Run("valid iam provisioned cluster auth", func(t *testing.T) {
		t.Parallel()

		cfg := validIAMClusterConfig()
		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("iam serverless auth requires workgroup", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalPasswordConfig())
		cfg["use_iam_for_auth"] = true
		cfg["iam_role_arn_for_auth"] = "arn:aws:iam::123456789012:role/RedshiftAccess"
		cfg["cluster_region"] = "us-east-1"
		cfg["use_serverless"] = true
		delete(cfg, "host")
		delete(cfg, "port")
		delete(cfg, "password")

		assertHasPath(t, registered.ValidateConfig(cfg), "/workgroup_name")
	})

	t.Run("valid iam serverless auth", func(t *testing.T) {
		t.Parallel()

		cfg := validIAMServerlessConfig()
		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("rudder storage tolerates stale custom storage fields", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalPasswordConfig())
		cfg["use_rudder_storage"] = true
		cfg["role_based_auth"] = true
		cfg["bucket_name"] = "rudder-redshift-staging"
		cfg["prefix"] = "stale-prefix"
		cfg["cleanup_object_storage_files"] = false

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("custom storage role auth requires iam role", func(t *testing.T) {
		t.Parallel()

		cfg := validCustomRoleStorageConfig()
		delete(cfg, "iam_role_arn")

		assertHasPath(t, registered.ValidateConfig(cfg), "/iam_role_arn")
	})

	t.Run("valid custom storage role auth", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(validCustomRoleStorageConfig()))
	})

	t.Run("valid custom storage key auth", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(validCustomKeyStorageConfig()))
	})

	t.Run("custom storage key auth tolerates missing write-only secrets on import", func(t *testing.T) {
		t.Parallel()

		cfg := validCustomKeyStorageConfig()
		delete(cfg, "access_key_id")
		delete(cfg, "access_key")

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("custom storage key auth validates present key shapes", func(t *testing.T) {
		t.Parallel()

		cfg := validCustomKeyStorageConfig()
		cfg["access_key_id"] = "bad\nkey"

		assertHasPath(t, registered.ValidateConfig(cfg), "/access_key_id")
	})

	t.Run("ssh disabled does not require tunnel fields", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalPasswordConfig())
		cfg["use_ssh"] = false

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("ssh enabled requires tunnel fields", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalPasswordConfig())
		cfg["use_ssh"] = true

		errors := registered.ValidateConfig(cfg)
		assertHasPath(t, errors, "/ssh_host")
		assertHasPath(t, errors, "/ssh_port")
		assertHasPath(t, errors, "/ssh_user")
		assertHasPath(t, errors, "/ssh_public_key")
	})

	t.Run("valid ssh config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(validSSHConfig()))
	})

	t.Run("sync frequency follows schema enum", func(t *testing.T) {
		t.Parallel()

		for _, freq := range []string{"5", "10", "15", "30", "60", "180", "360", "720", "1440"} {
			cfg := copyConfig(minimalPasswordConfig())
			cfg["sync_frequency"] = freq
			assert.Empty(t, registered.ValidateConfig(cfg), freq)
		}

		cfg := copyConfig(minimalPasswordConfig())
		cfg["sync_frequency"] = "45"
		assertHasPath(t, registered.ValidateConfig(cfg), "/sync_frequency")
	})

	t.Run("exclude window child fields required when object is present", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalPasswordConfig())
		cfg["exclude_window"] = map[string]any{"start_time": "01:00"}

		assertHasPath(t, registered.ValidateConfig(cfg), "/exclude_window/end_time")
	})

	t.Run("sync and exclude window times follow schema string shape", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalPasswordConfig())
		cfg["sync_start_at"] = "after the hour"
		cfg["exclude_window"] = map[string]any{"start_time": "after lunch", "end_time": "before dinner"}

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("legacy sync block is rejected", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalPasswordConfig())
		delete(cfg, "sync_frequency")
		cfg["sync"] = map[string]any{"frequency": "180"}

		errors := registered.ValidateConfig(cfg)
		assertHasPath(t, errors, "/sync")
		assertHasPath(t, errors, "/sync_frequency")
	})

	t.Run("pattern constraints reject invalid literals", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			field string
			value any
			cfg   map[string]any
		}{
			{field: "host", value: "demo.ngrok.io", cfg: minimalPasswordConfig()},
			{field: "port", value: "bad\nport", cfg: minimalPasswordConfig()},
			{field: "database", value: "bad\ndatabase", cfg: minimalPasswordConfig()},
			{field: "user", value: "bad\nuser", cfg: minimalPasswordConfig()},
			{field: "namespace", value: "pg_catalog", cfg: minimalPasswordConfig()},
			{field: "bucket_name", value: "xn--bucket", cfg: validCustomRoleStorageConfig()},
			{field: "bucket_name", value: "rudder..bucket", cfg: validCustomRoleStorageConfig()},
			{field: "bucket_name", value: "192.168.0.1", cfg: validCustomRoleStorageConfig()},
			{field: "bucket_name", value: "BadBucket", cfg: validCustomRoleStorageConfig()},
			{field: "prefix", value: "bad prefix", cfg: validCustomRoleStorageConfig()},
			{field: "cluster_region", value: "bad\nregion", cfg: validIAMClusterConfig()},
			{field: "workgroup_name", value: "bad\nworkgroup", cfg: validIAMServerlessConfig()},
			{field: "ssh_host", value: "bad\nhost", cfg: validSSHConfig()},
			{field: "ssh_port", value: "bad\nport", cfg: validSSHConfig()},
			{field: "ssh_user", value: "bad\nuser", cfg: validSSHConfig()},
			{field: "iam_role_arn_for_auth", value: "bad\narn", cfg: validIAMClusterConfig()},
			{field: "cluster_id", value: "bad\ncluster", cfg: validIAMClusterConfig()},
			{field: "iam_role_arn", value: "bad\narn", cfg: validCustomRoleStorageConfig()},
			{field: "access_key_id", value: "bad\nkey", cfg: validCustomKeyStorageConfig()},
			{field: "access_key", value: "bad\nsecret", cfg: validCustomKeyStorageConfig()},
			// A reject pattern must match broadly: an end-anchored version would let
			// a trailing-dot FQDN or a host:port suffix past the ngrok block.
			{field: "host", value: "demo.ngrok.io.", cfg: minimalPasswordConfig()},
			{field: "host", value: "demo.ngrok.io:5439", cfg: minimalPasswordConfig()},
		}

		for _, tc := range cases {
			cfg := copyConfig(tc.cfg)
			cfg[tc.field] = tc.value
			assertHasPath(t, registered.ValidateConfig(cfg), "/"+tc.field)
		}
	})

	t.Run("near miss values are accepted", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			field string
			value any
			cfg   map[string]any
		}{
			{field: "host", value: "redshift-ngrok.example.com", cfg: minimalPasswordConfig()},
			{field: "namespace", value: "pgx_events", cfg: minimalPasswordConfig()},
			{field: "bucket_name", value: "rudder.bucket", cfg: validCustomRoleStorageConfig()},
			{field: "bucket_name", value: "1.2.3.4.5", cfg: validCustomRoleStorageConfig()},
			{field: "prefix", value: "rudder/redshift", cfg: validCustomRoleStorageConfig()},
		}

		for _, tc := range cases {
			cfg := copyConfig(tc.cfg)
			cfg[tc.field] = tc.value
			assert.Empty(t, registered.ValidateConfig(cfg), "%s=%v", tc.field, tc.value)
		}
	})

	t.Run("pattern fields accept ui templates", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalPasswordConfig())
		cfg["host"] = "{{ config.host || " + strings.Repeat("a", 300) + " }}"
		cfg["database"] = "{{ config.database || " + strings.Repeat("a", 150) + " }}"
		cfg["namespace"] = "{{ config.namespace || pg_events }}"
		cfg["bucket_name"] = "{{ config.bucketName || Bad_Bucket_Name }}"
		cfg["prefix"] = "{{ config.prefix || bad prefix }}"

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalPasswordConfig())
		cfg["not_a_field"] = true

		assertHasPath(t, registered.ValidateConfig(cfg), "/not_a_field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalPasswordConfig())
		cfg["consent_management"] = map[string]any{"warehouse": []any{}}

		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'warehouse' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalPasswordConfig())
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

func TestRSModelsCurrentDefaultConfigKeys(t *testing.T) {
	t.Parallel()

	registered := registeredDefinition(t)
	apiConfig, err := registered.LocalToAPI(fullConfig())
	require.NoError(t, err)

	expectedAPIKeys := []string{
		"host", "port", "database", "user", "useIAMForAuth", "password",
		"iamRoleARNForAuth", "clusterId", "clusterRegion", "useServerless", "workgroupName",
		"bucketName", "iamRoleARN", "roleBasedAuth", "accessKeyID", "accessKey", "prefix",
		"namespace", "useSSH", "sshHost", "sshPort", "skipTracksTable", "skipUsersTable",
		"sshUser", "sshPublicKey", "syncFrequency", "syncStartAt", "enableSSE", "preferAppend",
		"excludeWindow", "jsonPaths", "useRudderStorage", "underscoreDivideNumbers",
		"cleanupObjectStorageFiles", "allowUsersContextTraits",
	}
	slices.Sort(expectedAPIKeys)

	actualAPIKeys := make([]string, 0, len(apiConfig))
	for key := range apiConfig {
		actualAPIKeys = append(actualAPIKeys, key)
	}
	slices.Sort(actualAPIKeys)

	assert.Equal(t, expectedAPIKeys, actualAPIKeys)
}

func TestRSConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := rs.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "password auth rudder storage",
			LocalJSON: `{
				"use_iam_for_auth": false,
				"host": "redshift.example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"sync_frequency": "180",
				"use_rudder_storage": true
			}`,
			APIJSON: `{
				"useIAMForAuth": false,
				"host": "redshift.example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"syncFrequency": "180",
				"useRudderStorage": true
			}`,
		},
		{
			Name: "iam provisioned cluster",
			LocalJSON: `{
				"use_iam_for_auth": true,
				"iam_role_arn_for_auth": "arn:aws:iam::123456789012:role/RedshiftAccess",
				"cluster_region": "us-east-1",
				"use_serverless": false,
				"cluster_id": "rudder-redshift-cluster",
				"database": "analytics",
				"user": "rudder",
				"sync_frequency": "180",
				"use_rudder_storage": true
			}`,
			APIJSON: `{
				"useIAMForAuth": true,
				"iamRoleARNForAuth": "arn:aws:iam::123456789012:role/RedshiftAccess",
				"clusterRegion": "us-east-1",
				"useServerless": false,
				"clusterId": "rudder-redshift-cluster",
				"database": "analytics",
				"user": "rudder",
				"syncFrequency": "180",
				"useRudderStorage": true
			}`,
		},
		{
			Name: "iam serverless",
			LocalJSON: `{
				"use_iam_for_auth": true,
				"iam_role_arn_for_auth": "arn:aws:iam::123456789012:role/RedshiftAccess",
				"cluster_region": "us-east-1",
				"use_serverless": true,
				"workgroup_name": "rudder-redshift-workgroup",
				"database": "analytics",
				"user": "rudder",
				"sync_frequency": "180",
				"use_rudder_storage": true
			}`,
			APIJSON: `{
				"useIAMForAuth": true,
				"iamRoleARNForAuth": "arn:aws:iam::123456789012:role/RedshiftAccess",
				"clusterRegion": "us-east-1",
				"useServerless": true,
				"workgroupName": "rudder-redshift-workgroup",
				"database": "analytics",
				"user": "rudder",
				"syncFrequency": "180",
				"useRudderStorage": true
			}`,
		},
		{
			Name: "custom storage key auth and advanced flags",
			LocalJSON: `{
				"use_iam_for_auth": false,
				"host": "redshift.example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"namespace": "rudder_events",
				"sync_frequency": "10",
				"sync_start_at": "01:00",
				"exclude_window": {"start_time": "02:00", "end_time": "03:00"},
				"skip_tracks_table": false,
				"skip_users_table": true,
				"prefer_append": true,
				"json_paths": "context.traits",
				"underscore_divide_numbers": false,
				"allow_users_context_traits": false,
				"use_rudder_storage": false,
				"bucket_name": "rudder-redshift-staging",
				"role_based_auth": false,
				"access_key_id": "AKIAEXAMPLE",
				"access_key": "secret-access-key",
				"prefix": "rudder/redshift",
				"enable_sse": true,
				"cleanup_object_storage_files": false
			}`,
			APIJSON: `{
				"useIAMForAuth": false,
				"host": "redshift.example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"namespace": "rudder_events",
				"syncFrequency": "10",
				"syncStartAt": "01:00",
				"excludeWindow": {"excludeWindowStartTime": "02:00", "excludeWindowEndTime": "03:00"},
				"skipTracksTable": false,
				"skipUsersTable": true,
				"preferAppend": true,
				"jsonPaths": "context.traits",
				"underscoreDivideNumbers": false,
				"allowUsersContextTraits": false,
				"useRudderStorage": false,
				"bucketName": "rudder-redshift-staging",
				"roleBasedAuth": false,
				"accessKeyID": "AKIAEXAMPLE",
				"accessKey": "secret-access-key",
				"prefix": "rudder/redshift",
				"enableSSE": true,
				"cleanupObjectStorageFiles": false
			}`,
		},
		{
			Name: "custom storage role auth",
			LocalJSON: `{
				"use_iam_for_auth": false,
				"host": "redshift.example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"sync_frequency": "180",
				"use_rudder_storage": false,
				"bucket_name": "rudder-redshift-staging",
				"role_based_auth": true,
				"iam_role_arn": "arn:aws:iam::123456789012:role/RudderRedshiftStorage"
			}`,
			APIJSON: `{
				"useIAMForAuth": false,
				"host": "redshift.example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"syncFrequency": "180",
				"useRudderStorage": false,
				"bucketName": "rudder-redshift-staging",
				"roleBasedAuth": true,
				"iamRoleARN": "arn:aws:iam::123456789012:role/RudderRedshiftStorage"
			}`,
		},
		{
			Name: "ssh and consent source boundary mappings",
			LocalJSON: `{
				"use_iam_for_auth": false,
				"host": "redshift.example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"use_ssh": true,
				"ssh_host": "bastion.example.com",
				"ssh_port": "22",
				"ssh_user": "rudder",
				"ssh_public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDrudder",
				"sync_frequency": "180",
				"use_rudder_storage": true,
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"react_native": [{"provider": "custom", "resolution_strategy": "or", "consents": ["analytics"]}]
				}
			}`,
			APIJSON: `{
				"useIAMForAuth": false,
				"host": "redshift.example.com",
				"port": "5439",
				"database": "analytics",
				"user": "rudder",
				"password": "secret",
				"useSSH": true,
				"sshHost": "bastion.example.com",
				"sshPort": "22",
				"sshUser": "rudder",
				"sshPublicKey": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDrudder",
				"syncFrequency": "180",
				"useRudderStorage": true,
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"reactnative": [{"provider": "custom", "resolutionStrategy": "or", "consents": [{"consent": "analytics"}]}]
				}
			}`,
		},
	})
}

func validIAMClusterConfig() map[string]any {
	cfg := copyConfig(minimalPasswordConfig())
	cfg["use_iam_for_auth"] = true
	cfg["iam_role_arn_for_auth"] = "arn:aws:iam::123456789012:role/RedshiftAccess"
	cfg["cluster_region"] = "us-east-1"
	cfg["use_serverless"] = false
	cfg["cluster_id"] = "rudder-redshift-cluster"
	delete(cfg, "host")
	delete(cfg, "port")
	delete(cfg, "password")
	return cfg
}

func validIAMServerlessConfig() map[string]any {
	cfg := copyConfig(minimalPasswordConfig())
	cfg["use_iam_for_auth"] = true
	cfg["iam_role_arn_for_auth"] = "arn:aws:iam::123456789012:role/RedshiftAccess"
	cfg["cluster_region"] = "us-east-1"
	cfg["use_serverless"] = true
	cfg["workgroup_name"] = "rudder-redshift-workgroup"
	delete(cfg, "host")
	delete(cfg, "port")
	delete(cfg, "password")
	return cfg
}

func validCustomRoleStorageConfig() map[string]any {
	cfg := copyConfig(minimalPasswordConfig())
	cfg["use_rudder_storage"] = false
	cfg["bucket_name"] = "rudder-redshift-staging"
	cfg["role_based_auth"] = true
	cfg["iam_role_arn"] = "arn:aws:iam::123456789012:role/RudderRedshiftStorage"
	return cfg
}

func validCustomKeyStorageConfig() map[string]any {
	cfg := copyConfig(minimalPasswordConfig())
	cfg["use_rudder_storage"] = false
	cfg["bucket_name"] = "rudder-redshift-staging"
	cfg["role_based_auth"] = false
	cfg["access_key_id"] = "AKIAEXAMPLE"
	cfg["access_key"] = "secret-access-key"
	return cfg
}

func validSSHConfig() map[string]any {
	cfg := copyConfig(minimalPasswordConfig())
	cfg["use_ssh"] = true
	cfg["ssh_host"] = "bastion.example.com"
	cfg["ssh_port"] = "22"
	cfg["ssh_user"] = "rudder"
	cfg["ssh_public_key"] = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDrudder"
	return cfg
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
