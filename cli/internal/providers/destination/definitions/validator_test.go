package definitions_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
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

func TestValidateConfigRejectsUnsupportedConsentSource(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "TEST",
		Version:     1,
		SourceTypes: []string{"web"},
		ConnectionModes: map[string][]string{
			"web": {"cloud"},
		},
		NewConfig: func() any {
			return &struct {
				ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
			}{}
		},
	}))
	registered, err := registry.Get("TEST", 1)
	require.NoError(t, err)

	errors := registered.ValidateConfig(map[string]any{
		"consent_management": map[string]any{
			"ios": []any{},
		},
	})

	assertConfigError(
		t,
		errors,
		"/consent_management/ios",
		"source type 'ios' is not supported by destination type 'TEST'; supported source types: web",
	)
}

func TestValidateConfigAppliesCanonicalConsentValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "TEST",
		Version:     1,
		SourceTypes: []string{"web"},
		ConnectionModes: map[string][]string{
			"web": {"cloud"},
		},
		NewConfig: func() any {
			return &struct {
				ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
			}{}
		},
	}))
	registered, err := registry.Get("TEST", 1)
	require.NoError(t, err)

	errors := registered.ValidateConfig(map[string]any{
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{"provider": "unknown"},
			},
		},
	})

	assertConfigError(
		t,
		errors,
		"/consent_management/web/0/provider",
		"'provider' must be one of [custom iubenda ketch oneTrust]",
	)
}

func TestValidateConfigReplacesCanonicalConsentValidationForSource(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "TEST",
		Version:     1,
		SourceTypes: []string{"ios_swift"},
		ConnectionModes: map[string][]string{
			"ios_swift": {"cloud"},
		},
		ConsentValidationOverrides: map[string]common.ConsentValidator{
			"ios_swift": func(_ []common.ConsentEntry) []common.ConsentValidationError {
				return nil
			},
		},
		NewConfig: func() any {
			return &struct {
				ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
			}{}
		},
	}))
	registered, err := registry.Get("TEST", 1)
	require.NoError(t, err)

	errors := registered.ValidateConfig(map[string]any{
		"consent_management": map[string]any{
			"ios_swift": []any{
				map[string]any{"provider": "replacement-specific-provider"},
			},
		},
	})

	assert.Empty(t, errors)
}

func TestValidateConfigKeepsCanonicalConsentValidationForSiblingSource(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "TEST",
		Version:     1,
		SourceTypes: []string{"ios_swift", "web"},
		ConnectionModes: map[string][]string{
			"ios_swift": {"cloud"},
			"web":       {"cloud"},
		},
		ConsentValidationOverrides: map[string]common.ConsentValidator{
			"ios_swift": func(_ []common.ConsentEntry) []common.ConsentValidationError {
				return nil
			},
		},
		NewConfig: func() any {
			return &struct {
				ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
			}{}
		},
	}))
	registered, err := registry.Get("TEST", 1)
	require.NoError(t, err)

	errors := registered.ValidateConfig(map[string]any{
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{"provider": "unknown"},
			},
		},
	})

	assertConfigError(
		t,
		errors,
		"/consent_management/web/0/provider",
		"'provider' must be one of [custom iubenda ketch oneTrust]",
	)
}

func TestValidateConfigReportsMalformedConsentValuePath(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "TEST",
		Version:     1,
		SourceTypes: []string{"web"},
		ConnectionModes: map[string][]string{
			"web": {"cloud"},
		},
		NewConfig: func() any {
			return &struct {
				ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
			}{}
		},
	}))
	registered, err := registry.Get("TEST", 1)
	require.NoError(t, err)

	errors := registered.ValidateConfig(map[string]any{
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{"provider": 42},
			},
		},
	})

	require.NotEmpty(t, errors)
	assert.Equal(t, "/consent_management/web/0/provider", errors[0].Path, "errors: %#v", errors)
}

func TestValidateConfigRejectsNullConsentManagement(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "TEST",
		Version:     1,
		SourceTypes: []string{"web"},
		ConnectionModes: map[string][]string{
			"web": {"cloud"},
		},
		NewConfig: func() any {
			return &struct {
				ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
			}{}
		},
	}))
	registered, err := registry.Get("TEST", 1)
	require.NoError(t, err)

	errors := registered.ValidateConfig(map[string]any{"consent_management": nil})

	assertConfigError(
		t,
		errors,
		"/consent_management",
		"'consent_management' must be an object",
	)
}

func TestValidateConfigRejectsNullConsentSourceEntries(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "TEST",
		Version:     1,
		SourceTypes: []string{"web"},
		ConnectionModes: map[string][]string{
			"web": {"cloud"},
		},
		NewConfig: func() any {
			return &struct {
				ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
			}{}
		},
	}))
	registered, err := registry.Get("TEST", 1)
	require.NoError(t, err)

	errors := registered.ValidateConfig(map[string]any{
		"consent_management": map[string]any{"web": nil},
	})

	assertConfigError(
		t,
		errors,
		"/consent_management/web",
		"'web' consent entries must be an array",
	)
}

func TestValidateConfigRejectsNullConsentProvider(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "TEST",
		Version:     1,
		SourceTypes: []string{"web"},
		ConnectionModes: map[string][]string{
			"web": {"cloud"},
		},
		NewConfig: func() any {
			return &struct {
				ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
			}{}
		},
	}))
	registered, err := registry.Get("TEST", 1)
	require.NoError(t, err)

	errors := registered.ValidateConfig(map[string]any{
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{"provider": nil},
			},
		},
	})

	assertConfigError(
		t,
		errors,
		"/consent_management/web/0/provider",
		"'provider' must be a string",
	)
}

func TestValidateConfigRejectsNullConsentEntry(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "TEST",
		Version:     1,
		SourceTypes: []string{"web"},
		ConnectionModes: map[string][]string{
			"web": {"cloud"},
		},
		NewConfig: func() any {
			return &struct {
				ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
			}{}
		},
	}))
	registered, err := registry.Get("TEST", 1)
	require.NoError(t, err)

	errors := registered.ValidateConfig(map[string]any{
		"consent_management": map[string]any{
			"web": []any{nil},
		},
	})

	assertConfigError(
		t,
		errors,
		"/consent_management/web/0",
		"consent entry must be an object",
	)
}

func TestValidateConfigRejectsNullConsents(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "TEST",
		Version:     1,
		SourceTypes: []string{"web"},
		ConnectionModes: map[string][]string{
			"web": {"cloud"},
		},
		NewConfig: func() any {
			return &struct {
				ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
			}{}
		},
	}))
	registered, err := registry.Get("TEST", 1)
	require.NoError(t, err)

	errors := registered.ValidateConfig(map[string]any{
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{"consents": nil},
			},
		},
	})

	assertConfigError(
		t,
		errors,
		"/consent_management/web/0/consents",
		"'consents' must be an array",
	)
}

func TestValidateConfigRejectsNullConsentValue(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:        "TEST",
		Version:     1,
		SourceTypes: []string{"web"},
		ConnectionModes: map[string][]string{
			"web": {"cloud"},
		},
		NewConfig: func() any {
			return &struct {
				ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
			}{}
		},
	}))
	registered, err := registry.Get("TEST", 1)
	require.NoError(t, err)

	errors := registered.ValidateConfig(map[string]any{
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{"consents": []any{nil}},
			},
		},
	})

	assertConfigError(
		t,
		errors,
		"/consent_management/web/0/consents/0",
		"'consent' must be a string",
	)
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
