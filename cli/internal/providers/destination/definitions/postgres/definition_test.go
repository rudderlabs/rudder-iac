package postgres_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/postgres"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func registeredDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(postgres.NewDefinition()))
	registered, err := registry.Get("postgres", 1)
	require.NoError(t, err)
	return registered
}

func minimalConfig() map[string]any {
	return map[string]any{
		"host":               "postgres.example.com",
		"database":           "rudder_events",
		"user":               "rudder",
		"password":           "s3cret",
		"port":               "5432",
		"ssl_mode":           "disable",
		"sync_frequency":     "180",
		"use_rudder_storage": true,
	}
}

func exampleConfig() map[string]any {
	cfg := copyConfig(minimalConfig())
	cfg["namespace"] = "analytics"
	cfg["skip_tracks_table"] = false
	cfg["skip_users_table"] = true
	cfg["prefer_append"] = true
	cfg["json_paths"] = "context.traits,properties.metadata"
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
	require.NoError(t, registry.Register(postgres.NewDefinition()))

	registered, err := registry.Get("postgres", 1)
	require.NoError(t, err)

	assert.Equal(t, "postgres", registered.Type)
	assert.Equal(t, "POSTGRES", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.GatedKeyPaths())
	assert.Equal(t, []string{
		"password",
		"access_key_id",
		"access_key",
		"account_key",
		"sas_token",
		"secret_access_key",
		"credentials",
	}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web", "unity", "cloud", "react_native", "flutter", "cordova"}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	byAPI, err := registry.GetByAPIType("POSTGRES", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestPostgresConfigValidation(t *testing.T) {
	t.Parallel()

	registered := registeredDefinition(t)

	t.Run("required fields missing", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"host", "database", "user", "password", "port", "ssl_mode", "sync_frequency", "use_rudder_storage"} {
			cfg := copyConfig(minimalConfig())
			delete(cfg, field)

			errors := registered.ValidateConfig(cfg)
			assertHasPath(t, errors, "/"+field)
		}
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(minimalConfig()))
	})

	t.Run("validated example config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(exampleConfig()))
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["namespace"] = "analytics"
		cfg["use_ssh"] = true
		cfg["ssh_host"] = "bastion.example.com"
		cfg["ssh_port"] = "22"
		cfg["ssh_user"] = "rudder"
		cfg["ssh_public_key"] = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDrudder"
		cfg["ssl_mode"] = "verify-ca"
		cfg["client_key"] = "-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----"
		cfg["client_cert"] = "-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----"
		cfg["server_ca"] = "-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----"
		cfg["sync_start_at"] = "01:00"
		cfg["exclude_window"] = map[string]any{"start_time": "02:00", "end_time": "03:00"}
		cfg["skip_tracks_table"] = false
		cfg["skip_users_table"] = true
		cfg["prefer_append"] = true
		cfg["json_paths"] = "context.traits"
		cfg["allow_users_context_traits"] = false
		cfg["underscore_divide_numbers"] = false
		cfg["use_rudder_storage"] = false
		cfg["bucket_provider"] = "S3"
		cfg["bucket_name"] = "rudder-postgres-staging"
		cfg["cleanup_object_storage_files"] = false
		cfg["role_based_auth"] = true
		cfg["iam_role_arn"] = "arn:aws:iam::123456789012:role/RudderPostgres"
		cfg["consent_management"] = map[string]any{
			"android_kotlin": []any{map[string]any{"provider": "oneTrust"}},
		}

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("enum values enforced", func(t *testing.T) {
		t.Parallel()

		for _, mode := range []string{"disable", "require", "verify-ca"} {
			cfg := copyConfig(minimalConfig())
			cfg["ssl_mode"] = mode
			if mode == "verify-ca" {
				cfg["client_key"] = "-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----"
				cfg["client_cert"] = "-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----"
				cfg["server_ca"] = "-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----"
			}
			assert.Empty(t, registered.ValidateConfig(cfg), mode)
		}

		cfg := copyConfig(minimalConfig())
		cfg["ssl_mode"] = "prefer"
		assertHasPath(t, registered.ValidateConfig(cfg), "/ssl_mode")

		for _, freq := range []string{"5", "10", "15", "30", "60", "180", "360", "720", "1440"} {
			cfg := copyConfig(minimalConfig())
			cfg["sync_frequency"] = freq
			assert.Empty(t, registered.ValidateConfig(cfg), freq)
		}

		cfg = copyConfig(minimalConfig())
		cfg["sync_frequency"] = "45"
		assertHasPath(t, registered.ValidateConfig(cfg), "/sync_frequency")
	})

	t.Run("ssh fields required when ssh is enabled", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_ssh"] = true

		errors := registered.ValidateConfig(cfg)
		assertHasPath(t, errors, "/ssh_host")
		assertHasPath(t, errors, "/ssh_port")
		assertHasPath(t, errors, "/ssh_user")
		assertHasPath(t, errors, "/ssh_public_key")
	})

	t.Run("verify ca certificate fields required", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["ssl_mode"] = "verify-ca"

		errors := registered.ValidateConfig(cfg)
		assertHasPath(t, errors, "/client_key")
		assertHasPath(t, errors, "/client_cert")
		assertHasPath(t, errors, "/server_ca")

		// schema.json declares no pattern for the TLS material, so any non-empty
		// value satisfies the conditional requirement.
		cfg["client_key"] = "-----BEGIN PRIVATE KEY-----\nkey\n-----END PRIVATE KEY-----"
		cfg["client_cert"] = "-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----"
		cfg["server_ca"] = "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----"
		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("object storage provider required when rudder storage is off", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = false

		assertHasPath(t, registered.ValidateConfig(cfg), "/bucket_provider")
	})

	t.Run("object storage provider enum enforced", func(t *testing.T) {
		t.Parallel()
		cfg := validS3KeyConfig()
		cfg["bucket_provider"] = "R2"

		assertHasPath(t, registered.ValidateConfig(cfg), "/bucket_provider")
	})

	t.Run("every object storage provider config is accepted", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, registered.ValidateConfig(validS3KeyConfig()))
		assert.Empty(t, registered.ValidateConfig(validS3RoleConfig()))
		assert.Empty(t, registered.ValidateConfig(validGCSConfig()))
		assert.Empty(t, registered.ValidateConfig(validAzureKeyConfig()))
		assert.Empty(t, registered.ValidateConfig(validAzureSASConfig()))
		assert.Empty(t, registered.ValidateConfig(validMINIOConfig()))
	})

	t.Run("per provider keys required for the selected provider", func(t *testing.T) {
		t.Parallel()

		cases := map[string][]string{
			"S3":         {"/bucket_name"},
			"GCS":        {"/bucket_name", "/credentials"},
			"AZURE_BLOB": {"/container_name", "/account_name"},
			"MINIO":      {"/bucket_name", "/end_point", "/access_key_id", "/secret_access_key", "/use_ssl"},
		}

		for provider, want := range cases {
			cfg := copyConfig(minimalConfig())
			cfg["use_rudder_storage"] = false
			cfg["bucket_provider"] = provider

			paths := make([]string, 0)
			for _, e := range registered.ValidateConfig(cfg) {
				paths = append(paths, e.Path)
			}
			assert.ElementsMatch(t, want, paths, "provider %q", provider)
		}
	})

	// Upstream keeps every provider's keys in one flat object, so carrying another
	// provider's keys alongside the selected one stays valid.
	t.Run("keys belonging to other providers stay optional", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"role_based_auth", "access_key", "credentials", "use_sas_tokens", "account_key"} {
			cfg := validS3KeyConfig()
			delete(cfg, key)
			assert.Empty(t, registered.ValidateConfig(cfg), "%q is not required for S3", key)
		}
	})

	// bucket_name is the only key required for a subset of providers, so it is
	// tagged required_unless rather than required_if. This pins the exemption that
	// inversion encodes: schema.json's AZURE_BLOB branch does not list bucketName.
	t.Run("azure blob is exempt from bucket_name", func(t *testing.T) {
		t.Parallel()

		cfg := validAzureKeyConfig()
		delete(cfg, "bucket_name")
		assert.Empty(t, registered.ValidateConfig(cfg))

		// The same config under any other provider does require it.
		for _, provider := range []string{"S3", "GCS", "MINIO"} {
			cfg := validAzureKeyConfig()
			delete(cfg, "bucket_name")
			cfg["bucket_provider"] = provider
			assertHasPath(t, registered.ValidateConfig(cfg), "/bucket_name")
		}
	})

	// required_unless states the rule as an exemption, so a config with no usable
	// provider falls outside every exemption and reports bucket_name too. That is
	// extra detail on a config already rejected for its provider, never a
	// rejection of a config the API would accept.
	t.Run("missing or invalid provider also reports bucket_name", func(t *testing.T) {
		t.Parallel()

		for _, provider := range []any{nil, "R2"} {
			cfg := copyConfig(minimalConfig())
			cfg["use_rudder_storage"] = false
			if provider != nil {
				cfg["bucket_provider"] = provider
			}

			errors := registered.ValidateConfig(cfg)
			assertHasPath(t, errors, "/bucket_provider")
			assertHasPath(t, errors, "/bucket_name")
		}
	})

	t.Run("minio accepts use_ssl set to false", func(t *testing.T) {
		t.Parallel()

		// schema.json requires the key to be present, not to be true.
		cfg := validMINIOConfig()
		cfg["use_ssl"] = false
		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("stale bucket provider does not require keys when rudder storage is on", func(t *testing.T) {
		t.Parallel()

		// Cleared config keys persist upstream, so a leftover bucket_provider must
		// not resurrect per-provider requirements once rudder storage is back on.
		cfg := copyConfig(minimalConfig())
		cfg["use_rudder_storage"] = true
		cfg["bucket_provider"] = "MINIO"

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("pattern constraints reject invalid literals", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			field string
			value any
			cfg   map[string]any
		}{
			{field: "host", value: "demo.ngrok.io", cfg: minimalConfig()},
			{field: "database", value: "bad\nvalue", cfg: minimalConfig()},
			{field: "namespace", value: "pg_catalog", cfg: minimalConfig()},
			{field: "bucket_name", value: "bad\nbucket", cfg: validS3KeyConfig()},
			{field: "container_name", value: "bad\ncontainer", cfg: validAzureKeyConfig()},
			{field: "end_point", value: "bad\nendpoint", cfg: validMINIOConfig()},
		}

		for _, tc := range cases {
			cfg := copyConfig(tc.cfg)
			cfg[tc.field] = tc.value
			errors := registered.ValidateConfig(cfg)
			assertHasPath(t, errors, "/"+tc.field)
		}
	})

	t.Run("host rejects ngrok tunnels", func(t *testing.T) {
		t.Parallel()

		// Matches schema.json's guard exactly. The trailing-dot FQDN and host:port
		// shapes are what an end-anchored pattern would let through; the last two
		// are rejected only because schema.json leaves its dots unescaped, so they
		// match any character.
		for _, host := range []string{
			"demo.ngrok.io",
			"a.b.ngrok.io",
			"demo.ngrok.io.",
			"demo.ngrok.io:5432",
			"myngrok.iohost.com",
			"xngrokzio.com",
		} {
			cfg := copyConfig(minimalConfig())
			cfg["host"] = host
			assertHasPath(t, registered.ValidateConfig(cfg), "/host")
		}

		for _, host := range []string{"postgres.example.com", "analytics.internal"} {
			cfg := copyConfig(minimalConfig())
			cfg["host"] = host
			assert.Empty(t, registered.ValidateConfig(cfg), host)
		}
	})

	t.Run("pattern fields accept ui templates", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["host"] = "{{ config.host || " + strings.Repeat("a", 250) + " }}"
		cfg["namespace"] = "{{ config.namespace || pg_events }}"
		cfg["database"] = "{{ config.database || " + strings.Repeat("a", 150) + " }}"
		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["not_a_field"] = true

		errors := registered.ValidateConfig(cfg)
		assertHasPath(t, errors, "/not_a_field")
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

func validS3KeyConfig() map[string]any {
	cfg := copyConfig(minimalConfig())
	cfg["use_rudder_storage"] = false
	cfg["bucket_provider"] = "S3"
	cfg["bucket_name"] = "rudder-postgres-staging"
	cfg["role_based_auth"] = false
	cfg["access_key_id"] = "access-key-id"
	cfg["access_key"] = "secret-access-key"
	return cfg
}

func validS3RoleConfig() map[string]any {
	cfg := copyConfig(minimalConfig())
	cfg["use_rudder_storage"] = false
	cfg["bucket_provider"] = "S3"
	cfg["bucket_name"] = "rudder-postgres-staging"
	cfg["role_based_auth"] = true
	cfg["iam_role_arn"] = "arn:aws:iam::123456789012:role/RudderPostgres"
	return cfg
}

func validGCSConfig() map[string]any {
	cfg := copyConfig(minimalConfig())
	cfg["use_rudder_storage"] = false
	cfg["bucket_provider"] = "GCS"
	cfg["bucket_name"] = "rudder-postgres-gcs"
	cfg["credentials"] = `{"type":"service_account"}`
	return cfg
}

func validAzureKeyConfig() map[string]any {
	cfg := copyConfig(minimalConfig())
	cfg["use_rudder_storage"] = false
	cfg["bucket_provider"] = "AZURE_BLOB"
	cfg["container_name"] = "rudder-postgres"
	cfg["account_name"] = "rudderaccount"
	cfg["use_sas_tokens"] = false
	cfg["account_key"] = "account-key"
	return cfg
}

func validAzureSASConfig() map[string]any {
	cfg := copyConfig(minimalConfig())
	cfg["use_rudder_storage"] = false
	cfg["bucket_provider"] = "AZURE_BLOB"
	cfg["container_name"] = "rudder-postgres"
	cfg["account_name"] = "rudderaccount"
	cfg["use_sas_tokens"] = true
	cfg["sas_token"] = "sas-token"
	return cfg
}

func validMINIOConfig() map[string]any {
	cfg := copyConfig(minimalConfig())
	cfg["use_rudder_storage"] = false
	cfg["bucket_provider"] = "MINIO"
	cfg["bucket_name"] = "rudder-postgres-minio"
	cfg["end_point"] = "minio.example.com:9000"
	cfg["access_key_id"] = "access-key-id"
	cfg["secret_access_key"] = "secret-access-key"
	cfg["use_ssl"] = true
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

func TestPostgresConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := postgres.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal rudder storage",
			LocalJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"ssl_mode": "disable",
				"sync_frequency": "180",
				"use_rudder_storage": true
			}`,
			APIJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"sslMode": "disable",
				"syncFrequency": "180",
				"useRudderStorage": true
			}`,
		},
		{
			Name: "s3 key based storage and advanced flags",
			LocalJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"namespace": "analytics",
				"ssl_mode": "require",
				"sync_frequency": "30",
				"sync_start_at": "01:00",
				"exclude_window": {"start_time": "02:00", "end_time": "03:00"},
				"skip_tracks_table": false,
				"skip_users_table": true,
				"prefer_append": true,
				"json_paths": "context.traits",
				"allow_users_context_traits": false,
				"underscore_divide_numbers": false,
				"use_rudder_storage": false,
				"bucket_provider": "S3",
				"bucket_name": "rudder-postgres-staging",
				"cleanup_object_storage_files": false,
				"role_based_auth": false,
				"access_key_id": "access-key-id",
				"access_key": "secret-access-key"
			}`,
			APIJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"namespace": "analytics",
				"sslMode": "require",
				"syncFrequency": "30",
				"syncStartAt": "01:00",
				"excludeWindow": {"excludeWindowStartTime": "02:00", "excludeWindowEndTime": "03:00"},
				"skipTracksTable": false,
				"skipUsersTable": true,
				"preferAppend": true,
				"jsonPaths": "context.traits",
				"allowUsersContextTraits": false,
				"underscoreDivideNumbers": false,
				"useRudderStorage": false,
				"bucketProvider": "S3",
				"bucketName": "rudder-postgres-staging",
				"cleanupObjectStorageFiles": false,
				"roleBasedAuth": false,
				"accessKeyID": "access-key-id",
				"accessKey": "secret-access-key"
			}`,
		},
		{
			Name: "s3 role based storage",
			LocalJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"ssl_mode": "disable",
				"sync_frequency": "180",
				"use_rudder_storage": false,
				"bucket_provider": "S3",
				"bucket_name": "rudder-postgres-staging",
				"role_based_auth": true,
				"iam_role_arn": "arn:aws:iam::123456789012:role/RudderPostgres"
			}`,
			APIJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"sslMode": "disable",
				"syncFrequency": "180",
				"useRudderStorage": false,
				"bucketProvider": "S3",
				"bucketName": "rudder-postgres-staging",
				"roleBasedAuth": true,
				"iamRoleARN": "arn:aws:iam::123456789012:role/RudderPostgres"
			}`,
		},
		{
			Name: "gcs storage",
			LocalJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"ssl_mode": "disable",
				"sync_frequency": "180",
				"use_rudder_storage": false,
				"bucket_provider": "GCS",
				"bucket_name": "rudder-postgres-gcs",
				"credentials": "{\"type\":\"service_account\"}"
			}`,
			APIJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"sslMode": "disable",
				"syncFrequency": "180",
				"useRudderStorage": false,
				"bucketProvider": "GCS",
				"bucketName": "rudder-postgres-gcs",
				"credentials": "{\"type\":\"service_account\"}"
			}`,
		},
		{
			Name: "azure sas storage",
			LocalJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"ssl_mode": "disable",
				"sync_frequency": "180",
				"use_rudder_storage": false,
				"bucket_provider": "AZURE_BLOB",
				"container_name": "rudder-postgres",
				"account_name": "rudderaccount",
				"use_sas_tokens": true,
				"sas_token": "sas-token"
			}`,
			APIJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"sslMode": "disable",
				"syncFrequency": "180",
				"useRudderStorage": false,
				"bucketProvider": "AZURE_BLOB",
				"containerName": "rudder-postgres",
				"accountName": "rudderaccount",
				"useSASTokens": true,
				"sasToken": "sas-token"
			}`,
		},
		{
			Name: "azure key storage",
			LocalJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"ssl_mode": "disable",
				"sync_frequency": "180",
				"use_rudder_storage": false,
				"bucket_provider": "AZURE_BLOB",
				"container_name": "rudder-postgres",
				"account_name": "rudderaccount",
				"use_sas_tokens": false,
				"account_key": "account-key"
			}`,
			APIJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"sslMode": "disable",
				"syncFrequency": "180",
				"useRudderStorage": false,
				"bucketProvider": "AZURE_BLOB",
				"containerName": "rudder-postgres",
				"accountName": "rudderaccount",
				"useSASTokens": false,
				"accountKey": "account-key"
			}`,
		},
		{
			Name: "minio storage",
			LocalJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"ssl_mode": "disable",
				"sync_frequency": "180",
				"use_rudder_storage": false,
				"bucket_provider": "MINIO",
				"bucket_name": "rudder-postgres-minio",
				"end_point": "minio.example.com:9000",
				"access_key_id": "access-key-id",
				"secret_access_key": "secret-access-key",
				"use_ssl": true
			}`,
			APIJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"sslMode": "disable",
				"syncFrequency": "180",
				"useRudderStorage": false,
				"bucketProvider": "MINIO",
				"bucketName": "rudder-postgres-minio",
				"endPoint": "minio.example.com:9000",
				"accessKeyID": "access-key-id",
				"secretAccessKey": "secret-access-key",
				"useSSL": true
			}`,
		},
		{
			Name: "ssh and verify ca options",
			LocalJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"use_ssh": true,
				"ssh_host": "bastion.example.com",
				"ssh_port": "22",
				"ssh_user": "rudder",
				"ssh_public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDrudder",
				"ssl_mode": "verify-ca",
				"client_key": "-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----",
				"client_cert": "-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----",
				"server_ca": "-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----",
				"sync_frequency": "180",
				"use_rudder_storage": true
			}`,
			APIJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"useSSH": true,
				"sshHost": "bastion.example.com",
				"sshPort": "22",
				"sshUser": "rudder",
				"sshPublicKey": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDrudder",
				"sslMode": "verify-ca",
				"clientKey": "-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----",
				"clientCert": "-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----",
				"serverCA": "-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----",
				"syncFrequency": "180",
				"useRudderStorage": true
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"ssl_mode": "disable",
				"sync_frequency": "180",
				"use_rudder_storage": true,
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"host": "postgres.example.com",
				"database": "rudder_events",
				"user": "rudder",
				"password": "s3cret",
				"port": "5432",
				"sslMode": "disable",
				"syncFrequency": "180",
				"useRudderStorage": true,
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
