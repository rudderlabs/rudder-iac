package accounts

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountSchemaUsesYAMLFieldNamesAndRequiredConstraints(t *testing.T) {
	generated := NewProvider(nil).SpecSchemas()[AccountSpecKind]
	spec, ok := generated.Properties.Get("spec")
	require.True(t, ok)

	assert.ElementsMatch(t, []string{"id", "name", "account_definition_name", "config"}, spec.Required)
	for _, name := range []string{"id", "name", "account_definition_name", "config"} {
		_, ok := spec.Properties.Get(name)
		assert.True(t, ok, name)
	}
}

func TestProviderSpecSchemasCoverSupportedKinds(t *testing.T) {
	t.Parallel()

	p := NewProvider(nil)

	assert.ElementsMatch(t, p.SupportedKinds(), schema.Kinds(p.SpecSchemas()))
}
