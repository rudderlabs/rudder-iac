package postgres_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/postgres"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
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
		out[k] = copyConfigValue(v)
	}
	return out
}

func copyConfigValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return copyConfig(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = copyConfigValue(item)
		}
		return out
	default:
		return value
	}
}

func setStorage(cfg map[string]any, section string, key string, value any) {
	storage, ok := cfg[section].(map[string]any)
	if !ok {
		storage = map[string]any{}
		cfg[section] = storage
	}
	storage[key] = value
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
		"s3.access_key",
		"azure.account_key",
		"azure.sas_token",
		"gcs.credentials",
		"minio.secret_access_key",
	}, registered.SecretKeys())

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
		cfg["ssh"] = map[string]any{
			"host":       "bastion.example.com",
			"port":       "22",
			"user":       "rudder",
			"public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDrudder",
		}
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
		setStorage(cfg, "s3", "role_based_auth", true)
		setStorage(cfg, "s3", "iam_role_arn", "arn:aws:iam::123456789012:role/RudderPostgres")
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
		assertHasPath(t, errors, "/ssh/host")
		assertHasPath(t, errors, "/ssh/port")
		assertHasPath(t, errors, "/ssh/user")
		assertHasPath(t, errors, "/ssh/public_key")
	})

	// Postgres now follows Kafka's and Redshift's nested SSH local shape; flat SSH
	// tunnel keys remain unknown rather than becoming an alias layer.
	t.Run("legacy flat ssh key is rejected", func(t *testing.T) {
		t.Parallel()

		cfg := copyConfig(minimalConfig())
		cfg["ssh_host"] = "bastion.example.com"

		assertHasPath(t, registered.ValidateConfig(cfg), "/ssh_host")
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
			"S3":         {"/bucket_name", "/access_key_id", "/s3/access_key"},
			"GCS":        {"/bucket_name", "/gcs/credentials"},
			"AZURE_BLOB": {"/azure/container_name", "/azure/account_name", "/azure/account_key"},
			"MINIO":      {"/bucket_name", "/access_key_id", "/minio/end_point", "/minio/secret_access_key", "/minio/use_ssl"},
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

		for _, tc := range []struct {
			section string
			key     string
			value   any
		}{
			{section: "gcs", key: "credentials", value: "stale"},
			{section: "azure", key: "use_sas_tokens", value: true},
			{section: "azure", key: "account_key", value: "stale"},
			{section: "minio", key: "secret_access_key", value: "stale"},
		} {
			cfg := validS3KeyConfig()
			setStorage(cfg, tc.section, tc.key, tc.value)
			assert.Empty(t, registered.ValidateConfig(cfg), "%s.%s should be accepted while S3 is selected", tc.section, tc.key)
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

	t.Run("s3 explicit role auth requires iam role arn", func(t *testing.T) {
		t.Parallel()

		cfg := validS3RoleConfig()
		delete(cfg["s3"].(map[string]any), "iam_role_arn")

		assertHasPath(t, registered.ValidateConfig(cfg), "/s3/iam_role_arn")
	})

	t.Run("s3 explicit key auth requires both access keys", func(t *testing.T) {
		t.Parallel()

		cfg := validS3KeyConfig()
		delete(cfg, "access_key_id")
		assertHasPath(t, registered.ValidateConfig(cfg), "/access_key_id")

		cfg = validS3KeyConfig()
		delete(cfg["s3"].(map[string]any), "access_key")
		assertHasPath(t, registered.ValidateConfig(cfg), "/s3/access_key")
	})

	t.Run("s3 omitted role selector defaults to key auth", func(t *testing.T) {
		t.Parallel()

		cfg := validS3KeyConfig()
		delete(cfg["s3"].(map[string]any), "role_based_auth")
		assert.Empty(t, registered.ValidateConfig(cfg))

		api, err := registered.LocalToAPI(registered.ApplyDefaults(cfg))
		require.NoError(t, err)
		assert.NotContains(t, api, "roleBasedAuth", "schema.json declares no default for the selector")
	})

	t.Run("azure explicit account key auth requires account key", func(t *testing.T) {
		t.Parallel()

		cfg := validAzureKeyConfig()
		delete(cfg["azure"].(map[string]any), "account_key")

		assertHasPath(t, registered.ValidateConfig(cfg), "/azure/account_key")
	})

	t.Run("azure omitted sas selector defaults to account key auth", func(t *testing.T) {
		t.Parallel()

		cfg := validAzureKeyConfig()
		delete(cfg["azure"].(map[string]any), "use_sas_tokens")
		assert.Empty(t, registered.ValidateConfig(cfg))

		api, err := registered.LocalToAPI(registered.ApplyDefaults(cfg))
		require.NoError(t, err)
		assert.NotContains(t, api, "useSASTokens", "schema.json declares no default for the selector")
	})

	t.Run("azure explicit sas auth requires sas token", func(t *testing.T) {
		t.Parallel()

		cfg := validAzureSASConfig()
		delete(cfg["azure"].(map[string]any), "sas_token")

		assertHasPath(t, registered.ValidateConfig(cfg), "/azure/sas_token")
	})

	t.Run("minio accepts use_ssl set to false", func(t *testing.T) {
		t.Parallel()

		// schema.json requires the key to be present, not to be true.
		cfg := validMINIOConfig()
		setStorage(cfg, "minio", "use_ssl", false)
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
			section string
			field   string
			value   any
			cfg     map[string]any
		}{
			{field: "host", value: "demo.ngrok.io", cfg: minimalConfig()},
			{field: "database", value: "bad\nvalue", cfg: minimalConfig()},
			{field: "namespace", value: "pg_catalog", cfg: minimalConfig()},
			{field: "bucket_name", value: "bad\nbucket", cfg: validS3KeyConfig()},
			{section: "azure", field: "container_name", value: "bad\ncontainer", cfg: validAzureKeyConfig()},
			{section: "minio", field: "end_point", value: "bad\nendpoint", cfg: validMINIOConfig()},
		}

		for _, tc := range cases {
			cfg := copyConfig(tc.cfg)
			if tc.section == "" {
				cfg[tc.field] = tc.value
				assertHasPath(t, registered.ValidateConfig(cfg), "/"+tc.field)
				continue
			}

			setStorage(cfg, tc.section, tc.field, tc.value)
			assertHasPath(t, registered.ValidateConfig(cfg), "/"+tc.section+"/"+tc.field)
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

	// No postgres property declares a {{ … || … }} branch in schema.json — every
	// pattern is (^env[.]…)|<real> — so template text is an ordinary literal and
	// must meet the real constraint rather than bypassing it.
	t.Run("pattern fields measure template text", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			field string
			value string
		}{
			{field: "host", value: "{{ config.host || " + strings.Repeat("a", 250) + " }}"},
			{field: "database", value: "{{ config.database || " + strings.Repeat("a", 150) + " }}"},
			{field: "user", value: "{{ config.user || " + strings.Repeat("a", 150) + " }}"},
			// namespace's reject pattern fires on the pg_ prefix in the fallback.
			{field: "namespace", value: "{{ config.namespace || " + strings.Repeat("a", 100) + " }}"},
		} {
			cfg := copyConfig(minimalConfig())
			cfg[tc.field] = tc.value

			errors := registered.ValidateConfig(cfg)
			require.NotEmpty(t, errors, tc.field)
			assert.Equal(t, "/"+tc.field, errors[0].Path)
		}
	})

	// Enum fields reject templates outright, since the shape excludes them.
	t.Run("enum fields reject dynamic values", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			field string
			value string
		}{
			{field: "ssl_mode", value: `{{ config.sslMode || disable }}`},
			{field: "ssl_mode", value: "env.POSTGRES_SSL_MODE"},
			{field: "bucket_provider", value: `{{ config.bucketProvider || S3 }}`},
			{field: "sync_frequency", value: "env.POSTGRES_SYNC_FREQUENCY"},
		} {
			cfg := copyConfig(minimalConfig())
			cfg["use_rudder_storage"] = false
			cfg[tc.field] = tc.value

			errors := registered.ValidateConfig(cfg)
			require.NotEmpty(t, errors, "%s=%s", tc.field, tc.value)
			assert.Equal(t, "/"+tc.field, errors[0].Path, tc.value)
		}
	})

	// schema.json states a different bucketName rule per bucketProvider branch:
	// S3 bans xn-- and consecutive dots, GCS bans goog/google and allows
	// underscores, MinIO carries only the IP-address rule.
	t.Run("bucket_name follows the provider's own rules", func(t *testing.T) {
		t.Parallel()

		cfg := func(provider, bucket string) map[string]any {
			c := copyConfig(minimalConfig())
			c["use_rudder_storage"] = false
			c["bucket_provider"] = provider
			c["bucket_name"] = bucket
			switch provider {
			case "S3":
				setStorage(c, "s3", "role_based_auth", true)
				setStorage(c, "s3", "iam_role_arn", "arn:aws:iam::123456789012:role/rudder")
			case "GCS":
				setStorage(c, "gcs", "credentials", "{}")
			case "MINIO":
				setStorage(c, "minio", "end_point", "minio.example.com:9000")
				c["access_key_id"] = "minio-access"
				setStorage(c, "minio", "secret_access_key", "minio-secret")
				setStorage(c, "minio", "use_ssl", true)
			}
			return c
		}

		// Underscore is legal for GCS only; xn-- and consecutive dots are banned
		// on S3 but not MinIO.
		assert.Empty(t, registered.ValidateConfig(cfg("GCS", "rudder_bucket")), "gcs underscore")
		assertHasPath(t, registered.ValidateConfig(cfg("S3", "rudder_bucket")), "/bucket_name")
		assertHasPath(t, registered.ValidateConfig(cfg("S3", "xn--bucket")), "/bucket_name")
		assertHasPath(t, registered.ValidateConfig(cfg("GCS", "googbucket")), "/bucket_name")
		assertHasPath(t, registered.ValidateConfig(cfg("GCS", "my-google-bucket")), "/bucket_name")
		assert.Empty(t, registered.ValidateConfig(cfg("MINIO", "xn--bucket")), "minio has no xn-- rule")

		// Shared across all three providers.
		for _, provider := range []string{"S3", "GCS", "MINIO"} {
			assert.Empty(t, registered.ValidateConfig(cfg(provider, "rudder-bucket")), provider)
			assertHasPath(t, registered.ValidateConfig(cfg(provider, "192.168.0.1")), "/bucket_name")
			assertHasPath(t, registered.ValidateConfig(cfg(provider, "Rudder-Bucket")), "/bucket_name")
		}
	})

	t.Run("container_name follows azure naming rules", func(t *testing.T) {
		t.Parallel()

		cfg := func(container string) map[string]any {
			c := copyConfig(minimalConfig())
			c["use_rudder_storage"] = false
			c["bucket_provider"] = "AZURE_BLOB"
			setStorage(c, "azure", "account_name", "rudderaccount")
			setStorage(c, "azure", "use_sas_tokens", false)
			setStorage(c, "azure", "account_key", "account-key")
			setStorage(c, "azure", "container_name", container)
			return c
		}

		assert.Empty(t, registered.ValidateConfig(cfg("rudder-logs")))
		for _, invalid := range []string{"ab", "Rudder-Logs", "rudder--logs", "rudder_logs", strings.Repeat("a", 64)} {
			assertHasPath(t, registered.ValidateConfig(cfg(invalid)), "/azure/container_name")
		}
	})

	t.Run("end_point rejects ngrok hosts", func(t *testing.T) {
		t.Parallel()

		cfg := func(endpoint string) map[string]any {
			c := copyConfig(minimalConfig())
			c["use_rudder_storage"] = false
			c["bucket_provider"] = "MINIO"
			c["bucket_name"] = "rudder-bucket"
			c["access_key_id"] = "minio-access"
			setStorage(c, "minio", "secret_access_key", "minio-secret")
			setStorage(c, "minio", "use_ssl", true)
			setStorage(c, "minio", "end_point", endpoint)
			return c
		}

		assert.Empty(t, registered.ValidateConfig(cfg("minio.example.com:9000")))
		assertHasPath(t, registered.ValidateConfig(cfg("https://foo.ngrok.io")), "/minio/end_point")
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
	setStorage(cfg, "s3", "role_based_auth", false)
	cfg["access_key_id"] = "access-key-id"
	setStorage(cfg, "s3", "access_key", "secret-access-key")
	return cfg
}

func validS3RoleConfig() map[string]any {
	cfg := copyConfig(minimalConfig())
	cfg["use_rudder_storage"] = false
	cfg["bucket_provider"] = "S3"
	cfg["bucket_name"] = "rudder-postgres-staging"
	setStorage(cfg, "s3", "role_based_auth", true)
	setStorage(cfg, "s3", "iam_role_arn", "arn:aws:iam::123456789012:role/RudderPostgres")
	return cfg
}

func validGCSConfig() map[string]any {
	cfg := copyConfig(minimalConfig())
	cfg["use_rudder_storage"] = false
	cfg["bucket_provider"] = "GCS"
	cfg["bucket_name"] = "rudder-postgres-gcs"
	setStorage(cfg, "gcs", "credentials", `{"type":"service_account"}`)
	return cfg
}

func validAzureKeyConfig() map[string]any {
	cfg := copyConfig(minimalConfig())
	cfg["use_rudder_storage"] = false
	cfg["bucket_provider"] = "AZURE_BLOB"
	setStorage(cfg, "azure", "container_name", "rudder-postgres")
	setStorage(cfg, "azure", "account_name", "rudderaccount")
	setStorage(cfg, "azure", "use_sas_tokens", false)
	setStorage(cfg, "azure", "account_key", "account-key")
	return cfg
}

func validAzureSASConfig() map[string]any {
	cfg := copyConfig(minimalConfig())
	cfg["use_rudder_storage"] = false
	cfg["bucket_provider"] = "AZURE_BLOB"
	setStorage(cfg, "azure", "container_name", "rudder-postgres")
	setStorage(cfg, "azure", "account_name", "rudderaccount")
	setStorage(cfg, "azure", "use_sas_tokens", true)
	setStorage(cfg, "azure", "sas_token", "sas-token")
	return cfg
}

func validMINIOConfig() map[string]any {
	cfg := copyConfig(minimalConfig())
	cfg["use_rudder_storage"] = false
	cfg["bucket_provider"] = "MINIO"
	cfg["bucket_name"] = "rudder-postgres-minio"
	setStorage(cfg, "minio", "end_point", "minio.example.com:9000")
	cfg["access_key_id"] = "access-key-id"
	setStorage(cfg, "minio", "secret_access_key", "secret-access-key")
	setStorage(cfg, "minio", "use_ssl", true)
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

func TestPostgresNestedSecretKeysWrapAndMask(t *testing.T) {
	t.Parallel()

	registered := registeredDefinition(t)
	config := secret.WrapKnownSecrets(map[string]any{
		"password":      "database-password",
		"access_key_id": "shared-key-id",
		"s3": map[string]any{
			"access_key": "s3-key-secret",
		},
		"gcs": map[string]any{
			"credentials": "gcs-credentials",
		},
		"azure": map[string]any{
			"account_key": "azure-account-key",
			"sas_token":   "azure-sas-token",
		},
		"minio": map[string]any{
			"secret_access_key": "minio-key-secret",
		},
	}, registered.SecretKeys())

	assertSecretValue(t, config, "password", "database-password")
	assertSecretValue(t, config, "access_key_id", "shared-key-id")
	assertNestedSecretValue(t, config, "s3", "access_key", "s3-key-secret")
	assertNestedSecretValue(t, config, "gcs", "credentials", "gcs-credentials")
	assertNestedSecretValue(t, config, "azure", "account_key", "azure-account-key")
	assertNestedSecretValue(t, config, "azure", "sas_token", "azure-sas-token")
	assertNestedSecretValue(t, config, "minio", "secret_access_key", "minio-key-secret")

	revealed := secret.RevealSecrets(config, registered.SecretKeys())
	assert.Equal(t, "shared-key-id", revealed["access_key_id"])

	require.NoError(t, secret.MaskSecrets(revealed, "postgres-prod", registered.SecretKeys()))
	assert.Equal(t, "{{ .POSTGRES_PROD_PASSWORD }}", revealed["password"])
	assert.Equal(t, "{{ .POSTGRES_PROD_ACCESS_KEY_ID }}", revealed["access_key_id"])
	assert.Equal(t, "{{ .POSTGRES_PROD_GCS_CREDENTIALS }}", revealed["gcs"].(map[string]any)["credentials"])
	assert.Equal(t, "{{ .POSTGRES_PROD_AZURE_ACCOUNT_KEY }}", revealed["azure"].(map[string]any)["account_key"])
	assert.Equal(t, "{{ .POSTGRES_PROD_MINIO_SECRET_ACCESS_KEY }}", revealed["minio"].(map[string]any)["secret_access_key"])
}

func assertSecretValue(t *testing.T, config map[string]any, key string, want string) {
	t.Helper()

	value, ok := config[key].(*secret.String)
	require.True(t, ok, "expected %s to be wrapped as a secret", key)
	assert.Equal(t, want, value.Reveal())
}

func assertNestedSecretValue(t *testing.T, config map[string]any, section string, key string, want string) {
	t.Helper()

	storage, ok := config[section].(map[string]any)
	require.True(t, ok, "missing %s storage block", section)
	assertSecretValue(t, storage, key, want)
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
				"access_key_id": "access-key-id",
				"cleanup_object_storage_files": false,
				"s3": {
					"role_based_auth": false,
					"access_key": "secret-access-key"
				}
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
				"s3": {
					"role_based_auth": true,
					"iam_role_arn": "arn:aws:iam::123456789012:role/RudderPostgres"
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
				"gcs": {"credentials": "{\"type\":\"service_account\"}"}
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
				"azure": {
					"container_name": "rudder-postgres",
					"account_name": "rudderaccount",
					"use_sas_tokens": true,
					"sas_token": "sas-token"
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
				"azure": {
					"container_name": "rudder-postgres",
					"account_name": "rudderaccount",
					"use_sas_tokens": false,
					"account_key": "account-key"
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
				"access_key_id": "access-key-id",
				"minio": {
					"end_point": "minio.example.com:9000",
					"secret_access_key": "secret-access-key",
					"use_ssl": true
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
			Name: "inactive provider storage keys round trip",
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
				"access_key_id": "access-key-id",
				"s3": {
					"role_based_auth": false,
					"access_key": "secret-access-key"
				},
				"gcs": {"credentials": "{\"type\":\"service_account\"}"},
				"azure": {
					"account_name": "stalerudder",
					"account_key": "stale-account-key",
					"use_sas_tokens": false
				},
				"minio": {
					"end_point": "minio.example.com:9000",
					"secret_access_key": "stale-secret-access-key",
					"use_ssl": false
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
				"useRudderStorage": false,
				"bucketProvider": "S3",
				"bucketName": "rudder-postgres-staging",
				"roleBasedAuth": false,
				"accessKeyID": "access-key-id",
				"accessKey": "secret-access-key",
				"credentials": "{\"type\":\"service_account\"}",
				"accountName": "stalerudder",
				"accountKey": "stale-account-key",
				"useSASTokens": false,
				"endPoint": "minio.example.com:9000",
				"secretAccessKey": "stale-secret-access-key",
				"useSSL": false
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
				"ssh": {
					"host": "bastion.example.com",
					"port": "22",
					"user": "rudder",
					"public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDrudder"
				},
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
