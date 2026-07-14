package destination

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderRules(t *testing.T) {
	t.Parallel()

	p := NewProvider(nil, ruleTestRegistry(t))

	syntactic := p.SyntacticRules()
	require.Len(t, syntactic, 1)
	assert.Equal(t, SpecSyntaxValidRuleID, syntactic[0].ID())

	semantic := p.SemanticRules()
	require.Len(t, semantic, 1)
	assert.Equal(t, SemanticValidRuleID, semantic[0].ID())
}

func TestProviderRuleDocEntries(t *testing.T) {
	t.Parallel()

	p := NewProvider(nil, ruleTestRegistry(t))

	entries := p.RuleDocEntries()
	require.Len(t, entries, 2)

	ids := []string{entries[0].RuleID, entries[1].RuleID}
	assert.ElementsMatch(t, []string{SpecSyntaxValidRuleID, SemanticValidRuleID}, ids)
}
