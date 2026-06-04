package docs_test

import (
	"testing"

	projectdocs "github.com/rudderlabs/rudder-iac/cli/internal/project/docs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/project/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// projectGatekeeperRules returns the project-level rules whose doc fragments
// live in this package, paired with the phase the docs generator records.
func projectGatekeeperRules() []rules.Rule {
	return []rules.Rule{
		prules.NewMetadataSyntaxValidRule(nil),
		prules.NewDuplicateURNRule(nil),
		prules.NewSpecSyntaxValidRule(nil),
		prules.NewResourceKindVersionValidRule(nil),
	}
}

// TestProjectFragmentsHaveEntries asserts every embedded fragment parses and
// carries match behavior.
func TestProjectFragmentsHaveEntries(t *testing.T) {
	entries, err := docs.LoadRuleDocEntries(projectdocs.FragmentsFS, ".")
	require.NoError(t, err)

	byID := make(map[string]docs.RuleDocEntry, len(entries))
	for _, e := range entries {
		byID[e.RuleID] = e
	}

	for _, rule := range projectGatekeeperRules() {
		entry, ok := byID[rule.ID()]
		require.True(t, ok, "expected a doc entry for %s", rule.ID())
		assert.NotEmpty(t, entry.MatchBehavior, "rule %s has no match_behavior", rule.ID())
	}
}

// TestProjectFragmentsPassGeneration runs the embedded fragments through the
// real docs generator together with the live rules and asserts that all
// DocumentedRules.Validate invariants pass — proving the wildcard applies_to
// in each fragment exactly covers what the rule reports from AppliesTo().
func TestProjectFragmentsPassGeneration(t *testing.T) {
	entries, err := docs.LoadRuleDocEntries(projectdocs.FragmentsFS, ".")
	require.NoError(t, err)

	syntactic := projectGatekeeperRules()

	doc, verrs := docs.Generate(syntactic, nil, entries, "test", "2026-06-03T00:00:00Z")
	assert.Empty(t, verrs, "expected no validation errors, got: %v", verrs)

	require.Len(t, doc.Rules, len(syntactic))
	for _, r := range doc.Rules {
		require.Len(t, r.AppliesTo, 1)
		assert.Equal(t, docs.MatchPatternDoc{Kind: "*", Version: "*"}, r.AppliesTo[0])
	}
}
