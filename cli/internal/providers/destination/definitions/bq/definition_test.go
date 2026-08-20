package bq_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/bq"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func registeredDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(bq.NewDefinition()))
	registered, err := registry.Get("bq", 1)
	require.NoError(t, err)
	return registered
}

func minimalConfig() map[string]any {
	return map[string]any{
		"project":        "rudder-cli-e2e",
		"bucket_name":    "rudder-cli-e2e-bq",
		"credentials":    `{"type":"service_account"}`,
		"sync_frequency": "180",
	}
}

func exampleConfig() map[string]any {
	cfg := copyConfig(minimalConfig())
	cfg["credentials"] = "{{ .BQ_CREDENTIALS }}"
	cfg["location"] = "US"
	cfg["prefix"] = "rudder/bq/"
	cfg["namespace"] = "rudder_e2e"
	cfg["partition_column"] = "loaded_at"
	cfg["partition_type"] = "day"
	cfg["skip_tracks_table"] = false
	cfg["skip_views"] = false
	cfg["skip_users_table"] = true
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
	require.NoError(t, registry.Register(bq.NewDefinition()))

	registered, err := registry.Get("bq", 1)
	require.NoError(t, err)

	assert.Equal(t, "bq", registered.Type)
	assert.Equal(t, "BQ", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"credentials"}, registered.SecretKeys())
	assert.Empty(t, registered.GatedKeyPaths())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web", "unity", "amp",
		"cloud", "react_native", "cloud_source", "flutter", "cordova", "shopify",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	byAPI, err := registry.GetByAPIType("BQ", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestBQConfigValidation(t *testing.T) {
	t.Parallel()

	registered := registeredDefinition(t)

	t.Run("required fields missing", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"project", "bucket_name", "credentials", "sync_frequency"} {
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
		cfg := exampleConfig()
		cfg["sync_frequency"] = "30"
		cfg["sync_start_at"] = "start when ready"
		cfg["exclude_window"] = map[string]any{"start_time": "not-a-clock", "end_time": "also-not-a-clock"}
		cfg["allow_users_context_traits"] = false
		cfg["underscore_divide_numbers"] = false
		cfg["consent_management"] = map[string]any{
			"android_kotlin": []any{
				map[string]any{
					"provider":            "custom",
					"resolution_strategy": "and",
					"consents":            []any{"analytics", "marketing"},
				},
			},
		}

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("schema enum values enforced", func(t *testing.T) {
		t.Parallel()

		for _, freq := range []string{"5", "10", "15", "30", "60", "180", "360", "720", "1440"} {
			cfg := copyConfig(minimalConfig())
			cfg["sync_frequency"] = freq
			assert.Empty(t, registered.ValidateConfig(cfg), freq)
		}

		cfg := copyConfig(minimalConfig())
		cfg["sync_frequency"] = "45"
		assertHasPath(t, registered.ValidateConfig(cfg), "/sync_frequency")

		for _, column := range []string{"_PARTITIONTIME", "loaded_at", "received_at", "timestamp", "sent_at", "original_timestamp"} {
			cfg := copyConfig(minimalConfig())
			cfg["partition_column"] = column
			assert.Empty(t, registered.ValidateConfig(cfg), column)
		}

		cfg = copyConfig(minimalConfig())
		cfg["partition_column"] = "message_id"
		assertHasPath(t, registered.ValidateConfig(cfg), "/partition_column")

		for _, partitionType := range []string{"hour", "day"} {
			cfg := copyConfig(minimalConfig())
			cfg["partition_type"] = partitionType
			assert.Empty(t, registered.ValidateConfig(cfg), partitionType)
		}

		cfg = copyConfig(minimalConfig())
		cfg["partition_type"] = "month"
		assertHasPath(t, registered.ValidateConfig(cfg), "/partition_type")
	})

	t.Run("exclude window child fields required when object is present", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["exclude_window"] = map[string]any{}

		errors := registered.ValidateConfig(cfg)
		assertHasPath(t, errors, "/exclude_window/start_time")
		assertHasPath(t, errors, "/exclude_window/end_time")
	})

	t.Run("pattern constraints reject invalid literals", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			field string
			value any
		}{
			{field: "project", value: "bad\nproject"},
			{field: "location", value: "bad\nlocation"},
			{field: "prefix", value: "bad\nprefix"},
			{field: "namespace", value: "pg_catalog"},
			{field: "bucket_name", value: "goog-rudder"},
			{field: "bucket_name", value: "rudder-google-bucket"},
			{field: "bucket_name", value: "192.168.0.1"},
			{field: "bucket_name", value: "rudder..bucket"},
			{field: "bucket_name", value: "BadBucket"},
		}

		for _, tc := range cases {
			cfg := copyConfig(minimalConfig())
			cfg[tc.field] = tc.value
			assertHasPath(t, registered.ValidateConfig(cfg), "/"+tc.field)
		}
	})

	// A reject pattern is only correct if it is also narrow enough: these values
	// resemble a blocked shape without being one, and would fail if a reject
	// branch were widened (goog -> goo, \.\. -> \., a 4-octet IP -> any dotted
	// digits, or the pg_ prefix matched anywhere rather than at the start).
	t.Run("near miss values are accepted", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			field string
			value any
		}{
			{field: "bucket_name", value: "gogle-rudder"},
			{field: "bucket_name", value: "rudder.bucket"},
			{field: "bucket_name", value: "1.2.3.4.5"},
			{field: "namespace", value: "pgx_events"},
		}

		for _, tc := range cases {
			cfg := copyConfig(minimalConfig())
			cfg[tc.field] = tc.value
			assert.Empty(t, registered.ValidateConfig(cfg), "%s=%v", tc.field, tc.value)
		}
	})

	t.Run("pattern fields accept ui templates", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["project"] = "{{ config.project || " + strings.Repeat("a", 150) + " }}"
		cfg["bucket_name"] = "{{ config.bucketName || rudder-google-bucket }}"
		cfg["namespace"] = "{{ config.namespace || pg_events }}"
		cfg["partition_column"] = "{{ config.partitionColumn || message_id }}"
		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("sync and exclude window times follow schema string shape", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["sync_start_at"] = "tomorrow morning"
		cfg["exclude_window"] = map[string]any{"start_time": "after lunch", "end_time": "before dinner"}

		assert.Empty(t, registered.ValidateConfig(cfg))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["not_a_field"] = true

		errors := registered.ValidateConfig(cfg)
		assertHasPath(t, errors, "/not_a_field")
	})

	t.Run("connection mode rejected as unknown key", func(t *testing.T) {
		t.Parallel()
		cfg := copyConfig(minimalConfig())
		cfg["connection_mode"] = map[string]any{"web": "cloud"}

		errors := registered.ValidateConfig(cfg)
		assertHasPath(t, errors, "/connection_mode")
	})

	t.Run("legacy consent blocks are not supported keys", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			cfg := copyConfig(minimalConfig())
			cfg[key] = map[string]any{"web": []any{}}

			errors := registered.ValidateConfig(cfg)
			assertHasPath(t, errors, "/"+key)
		}
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
			"cloud_source": []any{map[string]any{"provider": "unknown"}},
		}

		errors := registered.ValidateConfig(cfg)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/cloud_source/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestBQConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := bq.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"project": "rudder-cli-e2e",
				"bucket_name": "rudder-cli-e2e-bq",
				"credentials": "{\"type\":\"service_account\"}",
				"sync_frequency": "180"
			}`,
			APIJSON: `{
				"project": "rudder-cli-e2e",
				"bucketName": "rudder-cli-e2e-bq",
				"credentials": "{\"type\":\"service_account\"}",
				"syncFrequency": "180"
			}`,
		},
		{
			Name: "full warehouse config",
			LocalJSON: `{
				"project": "rudder-cli-e2e",
				"location": "US",
				"bucket_name": "rudder-cli-e2e-bq",
				"prefix": "rudder/bq/",
				"namespace": "rudder_e2e",
				"credentials": "{\"type\":\"service_account\"}",
				"sync_frequency": "30",
				"sync_start_at": "01:00",
				"exclude_window": {"start_time": "02:00", "end_time": "03:00"},
				"skip_tracks_table": false,
				"skip_views": true,
				"skip_users_table": true,
				"partition_column": "loaded_at",
				"partition_type": "hour",
				"json_paths": "context.traits",
				"cleanup_object_storage_files": false,
				"underscore_divide_numbers": false,
				"allow_users_context_traits": false
			}`,
			APIJSON: `{
				"project": "rudder-cli-e2e",
				"location": "US",
				"bucketName": "rudder-cli-e2e-bq",
				"prefix": "rudder/bq/",
				"namespace": "rudder_e2e",
				"credentials": "{\"type\":\"service_account\"}",
				"syncFrequency": "30",
				"syncStartAt": "01:00",
				"excludeWindow": {"excludeWindowStartTime": "02:00", "excludeWindowEndTime": "03:00"},
				"skipTracksTable": false,
				"skipViews": true,
				"skipUsersTable": true,
				"partitionColumn": "loaded_at",
				"partitionType": "hour",
				"jsonPaths": "context.traits",
				"cleanupObjectStorageFiles": false,
				"underscoreDivideNumbers": false,
				"allowUsersContextTraits": false
			}`,
		},
		{
			Name: "zero values survive without SkipZeroValue",
			LocalJSON: `{
				"project": "rudder-cli-e2e",
				"location": "",
				"bucket_name": "rudder-cli-e2e-bq",
				"prefix": "",
				"namespace": "",
				"credentials": "{\"type\":\"service_account\"}",
				"sync_frequency": "180",
				"sync_start_at": "",
				"skip_tracks_table": false,
				"skip_views": false,
				"skip_users_table": false,
				"json_paths": "",
				"cleanup_object_storage_files": false,
				"underscore_divide_numbers": false,
				"allow_users_context_traits": false
			}`,
			APIJSON: `{
				"project": "rudder-cli-e2e",
				"location": "",
				"bucketName": "rudder-cli-e2e-bq",
				"prefix": "",
				"namespace": "",
				"credentials": "{\"type\":\"service_account\"}",
				"syncFrequency": "180",
				"syncStartAt": "",
				"skipTracksTable": false,
				"skipViews": false,
				"skipUsersTable": false,
				"jsonPaths": "",
				"cleanupObjectStorageFiles": false,
				"underscoreDivideNumbers": false,
				"allowUsersContextTraits": false
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"project": "rudder-cli-e2e",
				"bucket_name": "rudder-cli-e2e-bq",
				"credentials": "{\"type\":\"service_account\"}",
				"sync_frequency": "180",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"cloud_source": [{"provider": "custom", "resolution_strategy": "or", "consents": ["analytics"]}]
				}
			}`,
			APIJSON: `{
				"project": "rudder-cli-e2e",
				"bucketName": "rudder-cli-e2e-bq",
				"credentials": "{\"type\":\"service_account\"}",
				"syncFrequency": "180",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"cloudSource": [{"provider": "custom", "resolutionStrategy": "or", "consents": [{"consent": "analytics"}]}]
				}
			}`,
		},
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
