package definitions_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/stretchr/testify/assert"
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
