package destination

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderRules(t *testing.T) {
	p := NewProvider(nil, ruleTestRegistry(t))

	var syntactic []string
	for _, rule := range p.SyntacticRules() {
		syntactic = append(syntactic, rule.ID())
	}
	assert.Equal(t, []string{SpecSyntaxValidRuleID, SecretReferenceRuleID, SecretReferenceUniqueRuleID}, syntactic)

	semantic := p.SemanticRules()
	require.Len(t, semantic, 1)
	assert.Equal(t, SemanticValidRuleID, semantic[0].ID())

	ruleDocEntries := p.RuleDocEntries()
	require.True(t, len(ruleDocEntries) >= 1)
}
