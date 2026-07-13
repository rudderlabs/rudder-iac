package definitions_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
)

func TestRegisterDefinition(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(definitions.GA4TestDefinition()))
}

func TestRegisterDefinitionMissingNewConfig(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	err := registry.Register(&definitions.DestinationDefinition{
		Type:    "GA4",
		Version: 1,
	})
	require.Error(t, err)
}

func TestValidateConfigCollectsAllErrors(t *testing.T) {
	t.Parallel()

	registered := registerTestDefinition(t)

	errors := registered.ValidateConfig(map[string]any{
		"types_of_client": "gtag",
		"unknown_field":   "value",
	})

	assertConfigError(t, errors, "/unknown_field", "unknown config field \"unknown_field\"")
	assertConfigError(t, errors, "/api_secret", "'api_secret' is required")
	assertConfigError(t, errors, "/measurement_id", "'measurement_id' is required when 'types_of_client' is gtag")
}

func TestValidateConfigConditionalRequired(t *testing.T) {
	t.Parallel()

	registered := registerTestDefinition(t)

	errors := registered.ValidateConfig(map[string]any{
		"api_secret":      "secret",
		"types_of_client": "gtag",
	})
	assertConfigError(t, errors, "/measurement_id", "'measurement_id' is required when 'types_of_client' is gtag")

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
	assertConfigError(
		t,
		errors,
		"/connection_mode/web",
		"'web' must be one of [cloud device hybrid] or a dynamic config value (env.VAR, {{ path || fallback }}, or {{ .VAR }})",
	)
}

func TestValidateConfigDecodeTypeError(t *testing.T) {
	t.Parallel()

	registered := registerTestDefinition(t)

	errors := registered.ValidateConfig(map[string]any{
		"api_secret":      "secret",
		"types_of_client": "gtag",
		"measurement_id":  "G-123",
		"debug_mode":      "not-a-boolean",
	})
	require.Len(t, errors, 1)
	assert.Contains(t, errors[0].Message, "debug_mode")
}

func registerTestDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(definitions.GA4TestDefinition()))

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
