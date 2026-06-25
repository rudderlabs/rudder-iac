package definitions_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

const testSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["api_secret", "types_of_client"],
  "properties": {
    "api_secret": { "type": "string" },
    "types_of_client": { "type": "string", "enum": ["gtag", "firebase"] },
    "measurement_id": { "type": "string" },
    "debug_mode": { "type": "boolean" },
    "connection_mode": {
      "type": "object",
      "properties": {
        "web": { "type": "string", "enum": ["cloud", "device", "hybrid"] },
        "android": { "type": "string", "enum": ["cloud", "device"] }
      },
      "additionalProperties": false
    }
  },
  "additionalProperties": false,
  "allOf": [
    {
      "if": {
        "properties": { "types_of_client": { "const": "gtag" } },
        "required": ["types_of_client"]
      },
      "then": { "required": ["measurement_id"] }
    }
  ]
}`

func TestCompileSchema(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := &definitions.DestinationDefinition{
		Type:    "GA4",
		Version: 1,
		Schema:  json.RawMessage(testSchema),
	}

	require.NoError(t, registry.Register(def))
}

func TestCompileSchemaInvalidJSON(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := &definitions.DestinationDefinition{
		Type:    "GA4",
		Version: 1,
		Schema:  json.RawMessage(`{"type":`),
	}

	err := registry.Register(def)
	require.Error(t, err)
}

func TestValidateConfigCollectsAllErrors(t *testing.T) {
	t.Parallel()

	registered := registerTestDefinition(t)

	errors := registered.ValidateConfig(map[string]any{
		"types_of_client": "gtag",
		"debug_mode":      "not-a-boolean",
		"unknown_field":   "value",
	})

	assertConfigError(t, errors, "/api_secret", "Required property 'api_secret' is missing")
	assertConfigError(t, errors, "/measurement_id", "Required property 'measurement_id' is missing")
	assertConfigError(t, errors, "/debug_mode", "Value is string but should be boolean")
	assertConfigError(t, errors, "/unknown_field", "No values are allowed because the schema is set to 'false'")
}

func TestValidateConfigConditionalRequired(t *testing.T) {
	t.Parallel()

	registered := registerTestDefinition(t)

	errors := registered.ValidateConfig(map[string]any{
		"api_secret":      "secret",
		"types_of_client": "gtag",
	})
	assertConfigError(t, errors, "/measurement_id", "Required property 'measurement_id' is missing")

	errors = registered.ValidateConfig(map[string]any{
		"api_secret":      "secret",
		"types_of_client": "gtag",
		"measurement_id":  "G-123",
	})
	assert.Empty(t, errors)
}

func TestValidateConfigConnectionModeEnum(t *testing.T) {
	t.Parallel()

	registered := registerTestDefinition(t)

	errors := registered.ValidateConfig(map[string]any{
		"api_secret":      "secret",
		"types_of_client": "gtag",
		"measurement_id":  "G-123",
		"connection_mode": map[string]any{
			"web": "invalid-mode",
		},
	})
	assertConfigError(t, errors, "/connection_mode/web", "Value invalid-mode should be one of the allowed values: cloud, device, hybrid")
}

func registerTestDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	def := &definitions.DestinationDefinition{
		Type:    "GA4",
		Version: 1,
		Properties: []converter.ConfigProperty{
			converter.Simple("apiSecret", "api_secret"),
		},
		Schema: json.RawMessage(testSchema),
	}
	require.NoError(t, registry.Register(def))

	registered, err := registry.Get("GA4", 1)
	require.NoError(t, err)
	return registered
}

func assertConfigError(t *testing.T, errors []definitions.ConfigError, path, message string) {
	t.Helper()

	for _, err := range errors {
		if err.Path != path {
			continue
		}
		assert.Equal(t, message, err.Message)
		return
	}

	t.Fatalf("expected validation error at %q with message %q, got %#v", path, message, errors)
}
