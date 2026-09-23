package destination

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderRules(t *testing.T) {
	p := NewProvider(nil, ruleTestRegistry(t))

	syntactic := p.SyntacticRules()
	require.Len(t, syntactic, 1)
	assert.Equal(t, SpecSyntaxValidRuleID, syntactic[0].ID())

	semantic := p.SemanticRules()
	require.Len(t, semantic, 1)
	assert.Equal(t, SemanticValidRuleID, semantic[0].ID())

	ruleDocEntries := p.RuleDocEntries()
	require.True(t, len(ruleDocEntries) >= 1)
}
