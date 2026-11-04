package customaudience_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	customaudience "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/custom_audience"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func register(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(customaudience.NewDefinition()))

	registered, err := registry.Get("custom_audience", 1)
	require.NoError(t, err)
	return registered
}

func action() map[string]any {
	return map[string]any{
		"request_body": `{"schema": ["EMAIL_SHA256"], "data": []}`,
		"method":       "POST",
		"endpoint":     "/audiences/members",
		"batch_size":   100,
	}
}

// validConfig is the minimal config every subtest starts from.
func validConfig() map[string]any {
	return map[string]any{
		"base_url": "https://api.example.com",
		"actions": map[string]any{
			"insert": action(),
			"update": map[string]any{"use_insert_config": true},
			"delete": action(),
		},
	}
}

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(customaudience.NewDefinition()))

	registered, err := registry.Get("custom_audience", 1)
	require.NoError(t, err)

	assert.Equal(t, "custom_audience", registered.Type)
	assert.Equal(t, "CUSTOM_AUDIENCE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"basic_auth_password", "bearer_token", "api_key_value"}, registered.SecretKeys())

	assert.Equal(t, []string{"warehouse"}, registered.SupportedSourceTypes())

	modes, err := registered.ConnectionModes("warehouse")
	require.NoError(t, err)
	assert.Equal(t, []string{"cloud"}, modes)

	assert.Equal(t, []string{"mirror"}, registered.SyncBehaviours())
	assert.False(t, registered.SupportsVisualMapper())
	assert.True(t, registered.SupportsVisualMapperV2())

	byAPI, err := registry.GetByAPIType("CUSTOM_AUDIENCE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestCustomAudienceConfigValidation(t *testing.T) {
	t.Parallel()

	registered := register(t)

	t.Run("missing base_url", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		delete(config, "base_url")

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/base_url", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing actions", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		delete(config, "actions")

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/actions", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	for _, call := range []string{"insert", "update", "delete"} {
		t.Run("missing actions."+call, func(t *testing.T) {
			t.Parallel()
			config := validConfig()
			delete(config["actions"].(map[string]any), call)

			errors := registered.ValidateConfig(config)
			require.Len(t, errors, 1)
			assert.Equal(t, "/actions/"+call, errors[0].Path)
			assert.Contains(t, errors[0].Message, "required")
		})
	}

	for _, key := range []string{"request_body", "method", "endpoint", "batch_size"} {
		t.Run("missing actions.insert."+key, func(t *testing.T) {
			t.Parallel()
			config := validConfig()
			delete(config["actions"].(map[string]any)["insert"].(map[string]any), key)

			errors := registered.ValidateConfig(config)
			require.Len(t, errors, 1)
			assert.Equal(t, "/actions/insert/"+key, errors[0].Path)
			assert.Contains(t, errors[0].Message, "required")
		})
	}

	// schema.json's if/then/else on the update block, read from the other side:
	// use_insert_config true exempts the four keys, anything else requires them.
	t.Run("update requires its own keys unless use_insert_config is true", func(t *testing.T) {
		t.Parallel()

		config := validConfig()
		config["actions"].(map[string]any)["update"] = map[string]any{"use_insert_config": false}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 4)
		for _, err := range errors {
			assert.Contains(t, err.Path, "/actions/update/")
			assert.Contains(t, err.Message, "required")
		}

		config["actions"].(map[string]any)["update"] = map[string]any{"use_insert_config": true}
		assert.Empty(t, registered.ValidateConfig(config))

		update := action()
		update["use_insert_config"] = false
		config["actions"].(map[string]any)["update"] = update
		assert.Empty(t, registered.ValidateConfig(config))
	})

	// An absent use_insert_config falls outside the exemption, as it should:
	// the else branch requires the four whenever the flag is not true.
	t.Run("update with no use_insert_config requires its own keys", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["actions"].(map[string]any)["update"] = map[string]any{}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 4)
	})

	t.Run("invalid method rejected", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["actions"].(map[string]any)["insert"].(map[string]any)["method"] = "FETCH"

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/actions/insert/method", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("batch_size outside 1..5000 rejected", func(t *testing.T) {
		t.Parallel()
		for _, size := range []int{0, 5001} {
			config := validConfig()
			config["actions"].(map[string]any)["insert"].(map[string]any)["batch_size"] = size

			errors := registered.ValidateConfig(config)
			require.Len(t, errors, 1, size)
			assert.Equal(t, "/actions/insert/batch_size", errors[0].Path)
		}
	})

	// The pattern's reject half: upstream's two negative lookaheads, which RE2
	// cannot express directly.
	t.Run("base_url rejects localhost and ngrok hosts", func(t *testing.T) {
		t.Parallel()
		for _, value := range []string{
			"http://localhost:8080",
			"https://api.localhost",
			"https://abc123.ngrok.io",
			"https://abc123.ngrok.io.",
			"https://abc123.ngrok.io:8443",
			// Upstream's lookahead is anchored right after the scheme, so a
			// host merely *starting* with "localhost" is blocked too. Kept
			// faithful rather than narrowed: parity with upstream was checked
			// by running both regexes over this host list.
			"https://localhost.example.com",
			"ftp://api.example.com",
			"api.example.com",
		} {
			config := validConfig()
			config["base_url"] = value

			errors := registered.ValidateConfig(config)
			require.Len(t, errors, 1, value)
			assert.Equal(t, "/base_url", errors[0].Path)
		}
	})

	// A host that merely resembles a blocked one must pass.
	t.Run("base_url accepts public hosts", func(t *testing.T) {
		t.Parallel()
		for _, value := range []string{
			"https://api.example.com",
			"https://api.example.com:8443",
			"https://api.example.com/audiences",
			"https://notngrok.io",
			"https://ngrok.io.example.com",
		} {
			config := validConfig()
			config["base_url"] = value
			assert.Empty(t, registered.ValidateConfig(config), value)
		}
	})

	// schema.json's allOf: an unauthenticated endpoint needs no account.
	t.Run("rudder_account_id required only when authentication_enabled is true", func(t *testing.T) {
		t.Parallel()

		config := validConfig()
		config["authentication_enabled"] = true

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/rudder_account_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")

		config["rudder_account_id"] = "custom-activation-account"
		assert.Empty(t, registered.ValidateConfig(config))

		config = validConfig()
		config["authentication_enabled"] = false
		assert.Empty(t, registered.ValidateConfig(config))
	})

	t.Run("header with an empty key rejected", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["headers"] = []any{map[string]any{"key": "", "value": "v"}}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/headers/0/key", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("invalid field hash_type rejected", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["actions"].(map[string]any)["insert"].(map[string]any)["fields"] = []any{
			map[string]any{
				"name": "email", "is_required": true, "hash_type": "SHA1", "is_custom": false,
			},
		}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/actions/insert/fields/0/hash_type", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(validConfig()))
	})

	// The spec shipped in the README.
	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"base_url":               "https://api.example.com",
			"authentication_enabled": true,
			"rudder_account_id":      "custom-activation-account",
			"headers": []any{
				map[string]any{"key": "X-Audience-Source", "value": "rudderstack"},
			},
			"actions": map[string]any{
				"insert": map[string]any{
					"request_body": `{"schema": ["EMAIL_SHA256"], "data": []}`,
					"method":       "POST",
					"endpoint":     "/audiences/members",
					"batch_size":   100,
					"fields": []any{
						map[string]any{
							"name": "email", "is_required": true, "hash_type": "SHA256", "is_custom": false,
						},
					},
				},
				"update": map[string]any{"use_insert_config": true},
				"delete": map[string]any{
					"request_body": `{"schema": ["EMAIL_SHA256"], "data": []}`,
					"method":       "DELETE",
					"endpoint":     "/audiences/members",
					"batch_size":   100,
				},
			},
			"connection_mode": map[string]any{"warehouse": "cloud"},
		}))
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["not_a_field"] = true

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unknown nested key rejected", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["actions"].(map[string]any)["insert"].(map[string]any)["timeout"] = 30

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/actions/insert/timeout", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	// db-config declares no per-source-type destConfig lists at all.
	t.Run("consent_management rejected as unknown", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["consent_management"] = map[string]any{"warehouse": []any{}}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/consent_management", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})
}

func TestCustomAudienceConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := customaudience.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal, update reusing the insert call",
			LocalJSON: `{
				"base_url": "https://api.example.com",
				"actions": {
					"insert": {
						"request_body": "{\"email\": \"x\"}",
						"method": "POST",
						"endpoint": "/audiences/members",
						"batch_size": 100
					},
					"update": {"use_insert_config": true},
					"delete": {
						"request_body": "{\"email\": \"x\"}",
						"method": "DELETE",
						"endpoint": "/audiences/members",
						"batch_size": 100
					}
				}
			}`,
			APIJSON: `{
				"baseUrl": "https://api.example.com",
				"actions": {
					"insert": {
						"requestBody": "{\"email\": \"x\"}",
						"method": "POST",
						"endpoint": "/audiences/members",
						"batchSize": 100
					},
					"update": {"useInsertConfig": true},
					"delete": {
						"requestBody": "{\"email\": \"x\"}",
						"method": "DELETE",
						"endpoint": "/audiences/members",
						"batchSize": 100
					}
				}
			}`,
		},
		{
			Name: "full with headers, fields and an independent update call",
			LocalJSON: `{
				"rudder_account_id": "custom-activation-account",
				"base_url": "https://api.example.com",
				"authentication_enabled": true,
				"basic_auth_user_name": "svc",
				"basic_auth_password": "secret",
				"bearer_token": "token",
				"api_key_name": "X-API-Key",
				"api_key_value": "key",
				"headers": [{"key": "X-Audience-Source", "value": "rudderstack"}],
				"actions": {
					"insert": {
						"request_body": "{\"email\": \"x\"}",
						"method": "POST",
						"endpoint": "/members",
						"batch_size": 500,
						"fields": [
							{"name": "email", "is_required": true, "hash_type": "SHA256", "is_custom": false},
							{"name": "plan", "is_required": false, "hash_type": "NONE", "is_custom": true}
						]
					},
					"update": {
						"use_insert_config": false,
						"request_body": "{\"email\": \"x\"}",
						"method": "PATCH",
						"endpoint": "/members",
						"batch_size": 250,
						"fields": [
							{"name": "email", "is_required": true, "hash_type": "MD5", "is_custom": false}
						]
					},
					"delete": {
						"request_body": "{\"email\": \"x\"}",
						"method": "DELETE",
						"endpoint": "/members",
						"batch_size": 100
					}
				},
				"connection_mode": {"warehouse": "cloud"}
			}`,
			APIJSON: `{
				"rudderAccountId": "custom-activation-account",
				"baseUrl": "https://api.example.com",
				"authenticationEnabled": true,
				"basicAuthUserName": "svc",
				"basicAuthPassword": "secret",
				"bearerToken": "token",
				"apiKeyName": "X-API-Key",
				"apiKeyValue": "key",
				"headers": [{"key": "X-Audience-Source", "value": "rudderstack"}],
				"actions": {
					"insert": {
						"requestBody": "{\"email\": \"x\"}",
						"method": "POST",
						"endpoint": "/members",
						"batchSize": 500,
						"fields": [
							{"name": "email", "isRequired": true, "hashType": "SHA256", "isCustom": false},
							{"name": "plan", "isRequired": false, "hashType": "NONE", "isCustom": true}
						]
					},
					"update": {
						"useInsertConfig": false,
						"requestBody": "{\"email\": \"x\"}",
						"method": "PATCH",
						"endpoint": "/members",
						"batchSize": 250,
						"fields": [
							{"name": "email", "isRequired": true, "hashType": "MD5", "isCustom": false}
						]
					},
					"delete": {
						"requestBody": "{\"email\": \"x\"}",
						"method": "DELETE",
						"endpoint": "/members",
						"batchSize": 100
					}
				},
				"connectionMode": {"warehouse": "cloud"}
			}`,
		},
	})
}
