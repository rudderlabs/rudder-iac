package definitions_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDynamicConfigValue(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "ui env reference", value: "env.API_SECRET", want: true},
		{name: "ui template fallback", value: "{{ config.url || https://example.com }}", want: true},
		{name: "iac variable substitution", value: "{{ .API_SECRET }}", want: true},
		{name: "iac variable with default", value: "{{ .API_SECRET | fallback }}", want: true},
		{name: "plain literal", value: "secret-value", want: false},
		{name: "invalid template without dot", value: "{{ API_SECRET }}", want: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, c.want, definitions.IsDynamicConfigValue(c.value))
		})
	}
}

func TestValidateConfigAllowsDynamicValues(t *testing.T) {
	t.Parallel()

	registered := registerTestDefinition(t)

	t.Run("secret via env reference", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"api_secret":      "env.API_SECRET",
			"types_of_client": "gtag",
			"measurement_id":  "G-123",
		})
		assert.Empty(t, errors)
	})

	t.Run("secret via ui template fallback", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"api_secret":      "{{ config.apiSecret || fallback }}",
			"types_of_client": "gtag",
			"measurement_id":  "G-123",
		})
		assert.Empty(t, errors)
	})

	t.Run("secret via iac variable substitution", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"api_secret":      "{{ .API_SECRET }}",
			"types_of_client": "gtag",
			"measurement_id":  "G-123",
		})
		assert.Empty(t, errors)
	})

	t.Run("connection mode via env reference", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"api_secret":      "secret",
			"types_of_client": "gtag",
			"measurement_id":  "G-123",
			"connection_mode": map[string]any{
				"web": "env.WEB_CONNECTION_MODE",
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("client type via env reference", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"api_secret":      "secret",
			"types_of_client": "env.CLIENT_TYPE",
			"measurement_id":  "G-123",
		})
		assert.Empty(t, errors)
	})
}

func registerDynamicPatternDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(definitions.DynamicPatternTestDefinition()))

	registered, err := registry.Get("DYNAMICPATTERN", 1)
	require.NoError(t, err)
	return registered
}

func TestDynamicOrPattern(t *testing.T) {
	t.Parallel()

	registered := registerDynamicPatternDefinition(t)

	valid := func(overrides map[string]any) map[string]any {
		config := map[string]any{"account_id": "123456"}
		for k, v := range overrides {
			config[k] = v
		}
		return config
	}

	t.Run("accepts literal matching the pattern", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(valid(map[string]any{"sign_up_source_id": "7890"})))
	})

	t.Run("rejects literal violating the pattern", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(valid(map[string]any{"sign_up_source_id": "abc123"}))
		require.Len(t, errors, 1)
		assert.Equal(t, "/sign_up_source_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must contain only digits")
	})

	// The `{{ ... || ... }}` branch is the only dynamic form schema.json declares.
	t.Run("accepts ui template values", func(t *testing.T) {
		t.Parallel()

		for _, value := range []string{
			"{{ config.signUpSourceId || 123 }}",
			"{{config.signUpSourceId||123}}",
			"{{ config.signUpSourceId || }}",
		} {
			assert.Empty(t, registered.ValidateConfig(valid(map[string]any{"sign_up_source_id": value})),
				"value %q must be accepted", value)
		}
	})

	t.Run("ui template satisfies a required field", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{"account_id": "{{ config.accountId || 1 }}"}))
	})

	// env.VAR is deprecated and resolves only in rudder-server, behind an
	// enterprise handler and a flag that can be off.
	t.Run("rejects env references", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(valid(map[string]any{"sign_up_source_id": "env.SIGN_UP_SOURCE_ID"}))
		require.Len(t, errors, 1)
		assert.Equal(t, "/sign_up_source_id", errors[0].Path)
	})

	// {{ .VAR }} is CLI var substitution, resolved before validation runs, and
	// lacking `||` upstream would reject the literal too.
	t.Run("rejects unresolved iac variables and templates without ||", func(t *testing.T) {
		t.Parallel()

		for _, value := range []string{"{{ .SIGN_UP_SOURCE_ID }}", "{{ .SIGN_UP_SOURCE_ID | 1 }}", "{{ NO_SEPARATOR }}"} {
			errors := registered.ValidateConfig(valid(map[string]any{"sign_up_source_id": value}))
			require.Len(t, errors, 1, "value %q must be rejected", value)
			assert.Equal(t, "/sign_up_source_id", errors[0].Path)
		}
	})

	t.Run("absent optional pointer is valid", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(valid(nil)))
	})

	t.Run("error message names the pattern rule and the template escape", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{"account_id": "not-digits"})
		require.Len(t, errors, 1)
		assert.Contains(t, errors[0].Message, "must contain only digits")
		assert.Contains(t, errors[0].Message, "{{ path || fallback }}")
		assert.NotContains(t, errors[0].Message, "env.")
	})
}
