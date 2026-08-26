package kafka_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/kafka"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(kafka.NewDefinition()))

	registered, err := registry.Get("kafka", 1)
	require.NoError(t, err)

	assert.Equal(t, "kafka", registered.Type)
	assert.Equal(t, "KAFKA", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"password"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "cloud", "react_native",
		"flutter", "cordova"}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	assert.Empty(t, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("KAFKA", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestKafkaConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(kafka.NewDefinition()))
	registered, err := registry.Get("kafka", 1)
	require.NoError(t, err)

	minimalConfig := func() map[string]any {
		return map[string]any{
			"host_name": "broker1.example.com, broker2.example.com",
			"port":      "9092",
			"topic":     "rudder-cli-events",
		}
	}

	assertError := func(t *testing.T, errors []definitions.ConfigError, path string, message string) {
		t.Helper()
		for _, err := range errors {
			if err.Path == path && strings.Contains(err.Message, message) {
				return
			}
		}
		require.Failf(t, "expected validation error", "path %q containing %q not found in %#v", path, message, errors)
	}

	for _, field := range []string{"host_name", "port", "topic"} {
		field := field
		t.Run("missing "+field, func(t *testing.T) {
			t.Parallel()
			config := minimalConfig()
			delete(config, field)

			errors := registered.ValidateConfig(config)

			require.NotEmpty(t, errors)
			assert.Equal(t, "/"+field, errors[0].Path)
			assert.Contains(t, errors[0].Message, "required")
		})
	}

	for _, tc := range []struct {
		name  string
		field string
		value string
	}{
		{name: "host starts with hyphen", field: "host_name", value: "-broker.example.com"},
		{name: "port is out of range", field: "port", value: "65536"},
		{name: "topic contains invalid slash", field: "topic", value: "bad/topic"},
		{name: "username contains space", field: "username", value: "bad user"},
		{name: "password contains line break", field: "password", value: "bad\npassword"},
		{name: "ssh host starts with hyphen", field: "ssh_host", value: "-ssh.example.com"},
		{name: "ssh port is zero", field: "ssh_port", value: "0"},
		{name: "ssh user contains space", field: "ssh_user", value: "bad user"},
		{name: "ssh public key has unsupported type", field: "ssh_public_key", value: "ssh-ecdsa AAAAC3NzaC1lZDI1NTE5AAAAIExample"},
	} {
		tc := tc
		t.Run("invalid literal "+tc.name, func(t *testing.T) {
			t.Parallel()
			config := minimalConfig()
			config[tc.field] = tc.value

			errors := registered.ValidateConfig(config)

			require.NotEmpty(t, errors)
			assertError(t, errors, "/"+tc.field, "not valid")
		})
	}

	for _, field := range []string{"host_name", "port", "topic", "username", "ssh_host", "ssh_port", "ssh_user", "ssh_public_key"} {
		field := field
		t.Run("template accepted for "+field, func(t *testing.T) {
			t.Parallel()
			config := minimalConfig()
			config[field] = "{{ config." + field + " || fallback }}"
			if strings.HasPrefix(field, "ssh_") {
				config["use_ssh"] = true
				config["ssh_host"] = "ssh.example.com"
				config["ssh_port"] = "22"
				config["ssh_user"] = "rudder"
				config["ssh_public_key"] = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKey"
				config[field] = "{{ config." + field + " || fallback }}"
			}
			if field == "username" {
				config["ssl_enabled"] = true
				config["use_sasl"] = true
				config["sasl_type"] = "plain"
			}

			errors := registered.ValidateConfig(config)

			assert.Empty(t, errors)
		})
	}

	t.Run("sasl_type rejects unsupported value", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["sasl_type"] = "oauthbearer"

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assertError(t, errors, "/sasl_type", "must be one of")
	})

	t.Run("sasl fields required when ssl and sasl are enabled", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["ssl_enabled"] = true
		config["use_sasl"] = true

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 2)
		assertError(t, errors, "/sasl_type", "required")
		assertError(t, errors, "/username", "required")
	})

	t.Run("sasl fields not required when ssl is disabled", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["ssl_enabled"] = false
		config["use_sasl"] = true

		errors := registered.ValidateConfig(config)

		assert.Empty(t, errors)
	})

	t.Run("ssh fields required when ssh tunnel is enabled", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["use_ssh"] = true

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 4)
		assertError(t, errors, "/ssh_host", "required")
		assertError(t, errors, "/ssh_port", "required")
		assertError(t, errors, "/ssh_user", "required")
		assertError(t, errors, "/ssh_public_key", "required")
	})

	t.Run("avro schemas required when avro conversion is enabled", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["convert_to_avro"] = true

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assertError(t, errors, "/avro_schemas", "required")
	})

	// schema.json requires both fields inside the convertToAvro:true branch.
	t.Run("avro schema entries require id and schema", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			entry map[string]any
			path  string
		}{
			{map[string]any{"schema_id": "event-value"}, "/avro_schemas/0/schema"},
			{map[string]any{"schema": "{}"}, "/avro_schemas/0/schema_id"},
		} {
			config := minimalConfig()
			config["convert_to_avro"] = true
			config["avro_schemas"] = []any{tc.entry}

			assertError(t, registered.ValidateConfig(config), tc.path, "required")
		}
	})

	// Both mapping value fields are pattern-validated; cover the reject side.
	t.Run("topic mapping values reject line breaks", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			field string
			entry map[string]any
			path  string
		}{
			{"event_type_to_topic_map", map[string]any{"from": "identify", "to": "bad\ntopic"}, "/event_type_to_topic_map/0/to"},
			{"event_to_topic_map", map[string]any{"from": "bad\nevent", "to": "topic"}, "/event_to_topic_map/0/from"},
		} {
			config := minimalConfig()
			config[tc.field] = []any{tc.entry}

			assertError(t, registered.ValidateConfig(config), tc.path, "")
		}
	})

	t.Run("event type mapping rejects unsupported type", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["event_type_to_topic_map"] = []any{map[string]any{"from": "track", "to": "tracks"}}

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assertError(t, errors, "/event_type_to_topic_map/0/from", "must be one of")
	})

	t.Run("topic mapping rejects line breaks", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["event_to_topic_map"] = []any{map[string]any{"from": "Signed Up", "to": "bad\ntopic"}}

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assertError(t, errors, "/event_to_topic_map/0/to", "not valid")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(minimalConfig())

		assert.Empty(t, errors)
	})

	t.Run("valid example config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"host_name":   "broker1.example.com",
			"port":        "9092",
			"topic":       "rudder-cli-events",
			"ssl_enabled": true,
			"use_sasl":    true,
			"sasl_type":   "plain",
			"username":    "rudder_user",
			"password":    "kafkaPasswordXXXXXXXXXXXX",
		})

		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["ssl_enabled"] = true
		config["ca_certificate"] = "-----BEGIN CERTIFICATE-----\nnot-a-real-cert\n-----END CERTIFICATE-----"
		config["use_sasl"] = true
		config["sasl_type"] = "sha512"
		config["username"] = "rudder_user"
		config["password"] = "kafkaPasswordXXXXXXXXXXXX"
		config["convert_to_avro"] = true
		config["avro_schemas"] = []any{map[string]any{"schema_id": "event-value", "schema": `{"type":"record","name":"Event"}`}}
		config["embed_avro_schema_id"] = true
		config["enable_multi_topic"] = true
		config["event_type_to_topic_map"] = []any{map[string]any{"from": "identify", "to": "identifies"}}
		config["event_to_topic_map"] = []any{map[string]any{"from": "Signed Up", "to": "signups"}}
		config["use_ssh"] = true
		config["ssh_host"] = "ssh.example.com"
		config["ssh_port"] = "22"
		config["ssh_user"] = "rudder"
		config["ssh_public_key"] = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKey"
		config["consent_management"] = map[string]any{
			"android_kotlin": []any{
				map[string]any{
					"provider":            "custom",
					"resolution_strategy": "and",
					"consents":            []any{"analytics", "marketing"},
				},
			},
		}

		errors := registered.ValidateConfig(config)

		assert.Empty(t, errors)
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["not_a_field"] = true

		errors := registered.ValidateConfig(config)

		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["consent_management"] = map[string]any{
			"cloud_source": []any{},
		}

		errors := registered.ValidateConfig(config)

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/cloud_source", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'cloud_source' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		config := minimalConfig()
		config["consent_management"] = map[string]any{
			"ios_swift": []any{
				map[string]any{"provider": "unknown"},
			},
		}

		errors := registered.ValidateConfig(config)

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

func TestKafkaConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := kafka.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"host_name": "broker1.example.com",
				"port": "9092",
				"topic": "rudder-cli-events"
			}`,
			APIJSON: `{
				"hostName": "broker1.example.com",
				"port": "9092",
				"topic": "rudder-cli-events"
			}`,
		},
		{
			Name: "full config",
			LocalJSON: `{
				"host_name": "broker1.example.com, broker2.example.com",
				"port": "9092",
				"topic": "rudder-cli-events",
				"ssl_enabled": true,
				"ca_certificate": "certificate-body",
				"use_sasl": true,
				"sasl_type": "sha512",
				"username": "rudder_user",
				"password": "kafkaPasswordXXXXXXXXXXXX",
				"convert_to_avro": true,
				"avro_schemas": [
					{"schema_id": "event-value", "schema": "{\"type\":\"record\",\"name\":\"Event\"}"}
				],
				"embed_avro_schema_id": true,
				"enable_multi_topic": true,
				"event_type_to_topic_map": [
					{"from": "identify", "to": "identifies"}
				],
				"event_to_topic_map": [
					{"from": "Signed Up", "to": "signups"}
				],
				"use_ssh": true,
				"ssh_host": "ssh.example.com",
				"ssh_port": "22",
				"ssh_user": "rudder",
				"ssh_public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKey"
			}`,
			APIJSON: `{
				"hostName": "broker1.example.com, broker2.example.com",
				"port": "9092",
				"topic": "rudder-cli-events",
				"sslEnabled": true,
				"caCertificate": "certificate-body",
				"useSASL": true,
				"saslType": "sha512",
				"username": "rudder_user",
				"password": "kafkaPasswordXXXXXXXXXXXX",
				"convertToAvro": true,
				"avroSchemas": [
					{"schemaId": "event-value", "schema": "{\"type\":\"record\",\"name\":\"Event\"}"}
				],
				"embedAvroSchemaID": true,
				"enableMultiTopic": true,
				"eventTypeToTopicMap": [
					{"from": "identify", "to": "identifies"}
				],
				"eventToTopicMap": [
					{"from": "Signed Up", "to": "signups"}
				],
				"useSSH": true,
				"sshHost": "ssh.example.com",
				"sshPort": "22",
				"sshUser": "rudder",
				"sshPublicKey": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKey"
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"host_name": "broker1.example.com",
				"port": "9092",
				"topic": "rudder-cli-events",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"hostName": "broker1.example.com",
				"port": "9092",
				"topic": "rudder-cli-events",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}

func TestKafkaRemoteReadExportRoundTrip(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(kafka.NewDefinition()))
	registered, err := registry.Get("kafka", 1)
	require.NoError(t, err)

	apiConfig := map[string]any{
		"hostName":          "broker1.example.com",
		"port":              "9092",
		"topic":             "rudder-cli-events",
		"sslEnabled":        true,
		"caCertificate":     "certificate-body",
		"useSASL":           true,
		"saslType":          "sha256",
		"username":          "rudder_user",
		"password":          "kafkaPasswordXXXXXXXXXXXX",
		"useSSH":            true,
		"sshHost":           "ssh.example.com",
		"sshPort":           "22",
		"sshUser":           "rudder",
		"sshPublicKey":      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKey",
		"convertToAvro":     true,
		"avroSchemas":       []any{map[string]any{"schemaId": "event-value", "schema": `{"type":"record","name":"Event"}`}},
		"embedAvroSchemaID": true,
		"enableMultiTopic":  true,
		"eventTypeToTopicMap": []any{
			map[string]any{"from": "identify", "to": "identifies"},
		},
		"eventToTopicMap": []any{
			map[string]any{"from": "Signed Up", "to": "signups"},
		},
	}
	expectedLocal := map[string]any{
		"host_name":            "broker1.example.com",
		"port":                 "9092",
		"topic":                "rudder-cli-events",
		"ssl_enabled":          true,
		"ca_certificate":       "certificate-body",
		"use_sasl":             true,
		"sasl_type":            "sha256",
		"username":             "rudder_user",
		"password":             "kafkaPasswordXXXXXXXXXXXX",
		"use_ssh":              true,
		"ssh_host":             "ssh.example.com",
		"ssh_port":             "22",
		"ssh_user":             "rudder",
		"ssh_public_key":       "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKey",
		"convert_to_avro":      true,
		"avro_schemas":         []any{map[string]any{"schema_id": "event-value", "schema": `{"type":"record","name":"Event"}`}},
		"embed_avro_schema_id": true,
		"enable_multi_topic":   true,
		"event_type_to_topic_map": []any{
			map[string]any{"from": "identify", "to": "identifies"},
		},
		"event_to_topic_map": []any{
			map[string]any{"from": "Signed Up", "to": "signups"},
		},
	}

	local, err := registered.APIToLocal(apiConfig)
	require.NoError(t, err)
	assert.Equal(t, expectedLocal, local)

	roundTripped, err := registered.LocalToAPI(local)
	require.NoError(t, err)
	assert.Equal(t, apiConfig, roundTripped)
}

func TestKafkaDestinationSpecExtraction(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(kafka.NewDefinition()))

	h := destination.NewHandler(nil, registry)
	extracted, err := h.Impl.ExtractResourcesFromSpec("destinations/kafka.yaml", &destination.DestinationSpec{
		ID:                "kafka-production",
		DisplayName:       "Kafka Production",
		Type:              "kafka",
		Enabled:           true,
		DefinitionVersion: 1,
		Config: map[string]any{
			"host_name": "broker1.example.com",
			"port":      "9092",
			"topic":     "rudder-cli-events",
			"password":  "kafkaPasswordXXXXXXXXXXXX",
		},
	})
	require.NoError(t, err)

	resource := extracted["kafka-production"]
	require.NotNil(t, resource)
	assert.Equal(t, "kafka-production", resource.ID)
	assert.Equal(t, "Kafka Production", resource.DisplayName)
	assert.Equal(t, "kafka", resource.Type)
	assert.True(t, resource.Enabled)
	assert.Equal(t, int64(1), resource.DefinitionVersion)
	assert.Equal(t, "broker1.example.com", resource.Config["host_name"])

	password, ok := resource.Config["password"].(*secret.String)
	require.True(t, ok)
	assert.False(t, password.IsUnknown())
	assert.Equal(t, "kafkaPasswordXXXXXXXXXXXX", password.Reveal())
}
