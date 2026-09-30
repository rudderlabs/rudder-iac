package destination

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDestinationSchemaUsesYAMLFieldNamesAndRequiredConstraints(t *testing.T) {
	generated := NewProvider(nil, ruleTestRegistry(t)).SpecSchemas()[DestinationSpecKind]
	require.NotEmpty(t, generated.OneOf)
	spec, ok := generated.OneOf[0].Properties.Get("spec")
	require.True(t, ok)

	assert.ElementsMatch(t, []string{"id", "display_name", "type", "definition_version"}, spec.Required)
	for _, name := range []string{"id", "display_name", "type", "enabled", "definition_version", "transformation", "config"} {
		_, ok := spec.Properties.Get(name)
		assert.True(t, ok, name)
	}
}

func TestProviderRules(t *testing.T) {
	p := NewProvider(nil, ruleTestRegistry(t))

	assert.ElementsMatch(t, p.SupportedKinds(), schema.Kinds(p.SpecSchemas()))

	syntactic := p.SyntacticRules()
	require.Len(t, syntactic, 1)
	assert.Equal(t, SpecSyntaxValidRuleID, syntactic[0].ID())

	semantic := p.SemanticRules()
	require.Len(t, semantic, 1)
	assert.Equal(t, SemanticValidRuleID, semantic[0].ID())

	ruleDocEntries := p.RuleDocEntries()
	require.True(t, len(ruleDocEntries) >= 1)
}
