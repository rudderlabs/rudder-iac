package definitions

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

const definitionTestSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["api_secret"],
  "properties": {
    "api_secret": { "type": "string" },
    "connection_mode": {
      "type": "object",
      "properties": {
        "web": { "type": "string", "enum": ["cloud", "device", "hybrid"] },
        "android": { "type": "string", "enum": ["cloud", "device"] }
      }
    }
  }
}`

func TestNewRegisteredDefinition(t *testing.T) {
	t.Parallel()

	def := &DestinationDefinition{
		Type:    "GA4",
		Version: 1,
		Properties: []converter.ConfigProperty{
			converter.Simple("apiSecret", "api_secret"),
		},
		SecretKeys: []string{"api_secret"},
		Schema:     json.RawMessage(definitionTestSchema),
	}

	registered, err := newRegisteredDefinition(def)
	require.NoError(t, err)

	assert.Same(t, def, registered.DestinationDefinition)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"api_secret"}, registered.SecretKeys())
	assert.ElementsMatch(t, []string{"web", "android"}, registered.SupportedSourceTypes())

	modes, err := registered.ConnectionModes("web")
	require.NoError(t, err)
	assert.Equal(t, []string{"cloud", "device", "hybrid"}, modes)

	errors := registered.ValidateConfig(map[string]any{})
	require.NotEmpty(t, errors)
	assertConfigError(t, errors, "/api_secret", "Required property 'api_secret' is missing")
}

func assertConfigError(t *testing.T, errors []ConfigError, path, message string) {
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

func TestNewRegisteredDefinitionInvalidSchema(t *testing.T) {
	t.Parallel()

	_, err := newRegisteredDefinition(&DestinationDefinition{
		Type:    "GA4",
		Version: 1,
		Schema:  json.RawMessage(`{"type":`),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compiling schema")
}

func TestNewRegisteredDefinitionWithoutConnectionMode(t *testing.T) {
	t.Parallel()

	registered, err := newRegisteredDefinition(&DestinationDefinition{
		Type:    "WEBHOOK",
		Version: 1,
		Schema: json.RawMessage(`{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"properties": {
				"webhook_url": { "type": "string" }
			}
		}`),
	})
	require.NoError(t, err)
	assert.Nil(t, registered.SupportedSourceTypes())
	assert.False(t, registered.IsSourceTypeSupported("web"))
}
