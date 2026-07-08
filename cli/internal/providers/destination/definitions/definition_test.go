package definitions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func TestNewRegisteredDefinition(t *testing.T) {
	t.Parallel()

	def := GA4TestDefinition()
	def.Properties = []converter.ConfigProperty{
		converter.Simple("apiSecret", "api_secret"),
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
	assertConfigError(t, errors, "/api_secret", "'api_secret' is required")
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

func TestLocalSourceTypeKeys(t *testing.T) {
	t.Parallel()

	def := GA4TestDefinition()
	def.SourceTypes = []string{"web", "reactNative", "amp"}

	registered, err := newRegisteredDefinition(def)
	require.NoError(t, err)
	assert.Equal(t, []string{"web", "react_native", "amp"}, registered.LocalSourceTypeKeys())
}

func TestLocalSourceTypeKeysEmpty(t *testing.T) {
	t.Parallel()

	registered, err := newRegisteredDefinition(WebhookTestDefinitionWithoutConnectionMode())
	require.NoError(t, err)
	assert.Nil(t, registered.LocalSourceTypeKeys())
}

func TestNewRegisteredDefinitionMissingNewConfig(t *testing.T) {
	t.Parallel()

	_, err := newRegisteredDefinition(&DestinationDefinition{
		Type:    "GA4",
		Version: 1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NewConfig is required")
}

func TestNewRegisteredDefinitionWithoutConnectionMode(t *testing.T) {
	t.Parallel()

	registered, err := newRegisteredDefinition(WebhookTestDefinitionWithoutConnectionMode())
	require.NoError(t, err)
	assert.Nil(t, registered.SupportedSourceTypes())
	assert.False(t, registered.IsSourceTypeSupported("web"))
}
