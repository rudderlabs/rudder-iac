package destination

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
)

func TestProviderRules(t *testing.T) {
	prevExp := viper.Get("experimental")
	prevFlag := viper.Get("flags.destinationSupport")
	viper.Set("experimental", true)
	viper.Set("flags.destinationSupport", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.destinationSupport", prevFlag)
	})

	p := NewProvider(nil, ruleTestRegistry(t))

	syntactic := p.SyntacticRules()
	require.Len(t, syntactic, 1)
	assert.Equal(t, SpecSyntaxValidRuleID, syntactic[0].ID())

	semantic := p.SemanticRules()
	require.Len(t, semantic, 1)
	assert.Equal(t, SemanticValidRuleID, semantic[0].ID())
}

func TestProviderRulesWhenDestinationSupportDisabled(t *testing.T) {
	prevExp := viper.Get("experimental")
	prevFlag := viper.Get("flags.destinationSupport")
	viper.Set("experimental", true)
	viper.Set("flags.destinationSupport", false)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.destinationSupport", prevFlag)
	})

	p := NewProvider(nil, ruleTestRegistry(t))
	assert.Nil(t, p.SyntacticRules())
	assert.Nil(t, p.SemanticRules())
	assert.Nil(t, p.RuleDocEntries())
}

func TestProviderRuleDocEntries(t *testing.T) {
	prevExp := viper.Get("experimental")
	prevFlag := viper.Get("flags.destinationSupport")
	viper.Set("experimental", true)
	viper.Set("flags.destinationSupport", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.destinationSupport", prevFlag)
	})

	p := NewProvider(nil, ruleTestRegistry(t))

	entries := p.RuleDocEntries()
	require.Len(t, entries, 2)

	ids := []string{entries[0].RuleID, entries[1].RuleID}
	assert.ElementsMatch(t, []string{SpecSyntaxValidRuleID, SemanticValidRuleID}, ids)
}

func TestSupportedMatchPatternsWhenDestinationSupportEnabled(t *testing.T) {
	prevExp := viper.Get("experimental")
	prevFlag := viper.Get("flags.destinationSupport")
	viper.Set("experimental", true)
	viper.Set("flags.destinationSupport", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.destinationSupport", prevFlag)
	})

	p := NewProvider(nil, ruleTestRegistry(t))
	assert.Equal(t, prules.V1VersionPatterns(DestinationSpecKind), p.SupportedMatchPatterns())
}

func TestSupportedMatchPatternsWhenDestinationSupportDisabled(t *testing.T) {
	prevExp := viper.Get("experimental")
	prevFlag := viper.Get("flags.destinationSupport")
	viper.Set("experimental", true)
	viper.Set("flags.destinationSupport", false)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.destinationSupport", prevFlag)
	})

	p := NewProvider(nil, ruleTestRegistry(t))
	assert.Nil(t, p.SupportedMatchPatterns())
}
