package converter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func TestConstructorsPopulateLocalKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		prop     converter.ConfigProperty
		localKey string
	}{
		{"Simple", converter.Simple("apiKey", "api_key"), "api_key"},
		{"Conditional", converter.Conditional("apiKey", "api_key", converter.Equals("k", "v")), "api_key"},
		{"ArrayWithStrings", converter.ArrayWithStrings("root", "nested", "items"), "items"},
		{"ArrayWithObjects", converter.ArrayWithObjects("root", "objects", map[string]any{"a": "b"}), "objects"},
		{"Discriminator", converter.Discriminator("apiKey", converter.DiscriminatorValues{"k": "v"}), ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.localKey, tc.prop.LocalKey)
			assert.Empty(t, tc.prop.SourceTypes)
		})
	}
}

func TestGatedSetsSourceTypesAndPreservesBehavior(t *testing.T) {
	t.Parallel()

	prop := converter.Gated(converter.Simple("apiKey", "api_key"), "web", "android")

	assert.Equal(t, "api_key", prop.LocalKey)
	assert.Equal(t, []string{"web", "android"}, prop.SourceTypes)

	api, err := prop.FromLocalFunc(`{}`, `{ "api_key": "123" }`)
	require.NoError(t, err)
	assert.JSONEq(t, `{ "apiKey": "123" }`, api)

	local, err := prop.ToLocalFunc(`{}`, `{ "apiKey": "123" }`)
	require.NoError(t, err)
	assert.JSONEq(t, `{ "api_key": "123" }`, local)
}
