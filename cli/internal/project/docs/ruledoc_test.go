package docs_test

import (
	"testing"

	projectdocs "github.com/rudderlabs/rudder-iac/cli/internal/project/docs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/project/rules"
	providerrules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	dgHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	essource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gatekeeperScopedPatterns mirrors the resource (kind, version) pairs declared in
// the metadata-syntax-valid and duplicate-urn fragments. The rules are constructed
// with these so their AppliesTo() matches the authored fragments exactly (the
// generator enforces bidirectional coverage). Production scopes them to the live
// provider patterns via BuildRegistry; this fragment-plumbing test reproduces the
// same set the fragments document using the shared pattern helpers (the idiom every
// provider's SupportedMatchPatterns uses) rather than hand-written literals.
func gatekeeperScopedPatterns() []rules.MatchPattern {
	var p []rules.MatchPattern
	// Kinds carrying both legacy and v1 specs.
	for _, kind := range []string{
		localcatalog.KindCategories,
		localcatalog.KindCustomTypes,
		essource.ResourceKind,
		localcatalog.KindEvents,
		localcatalog.KindProperties,
		sqlmodel.ResourceKind,
	} {
		p = append(p, providerrules.LegacyVersionPatterns(kind)...)
		p = append(p, providerrules.V1VersionPatterns(kind)...)
	}
	// Tracking plans: legacy kind "tp", v1 kind "tracking-plan".
	p = append(p, providerrules.LegacyVersionPatterns(localcatalog.KindTrackingPlans)...)
	p = append(p, providerrules.V1VersionPatterns(localcatalog.KindTrackingPlansV1)...)
	// v1-only kinds.
	p = append(p, providerrules.V1VersionPatterns(dgHandler.HandlerMetadata.SpecKind)...)
	p = append(p, providerrules.V1VersionPatterns(ttypes.TransformationSpecKind)...)
	p = append(p, providerrules.V1VersionPatterns(ttypes.LibrarySpecKind)...)
	return p
}

// projectGatekeeperRules returns the project-level rules whose doc fragments
// live in this package, paired with the phase the docs generator records.
func projectGatekeeperRules() []rules.Rule {
	return []rules.Rule{
		prules.NewMetadataSyntaxValidRule(nil, gatekeeperScopedPatterns()),
		prules.NewDuplicateURNRule(nil, gatekeeperScopedPatterns()),
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
// real docs generator together with the live rules and asserts that the
// DocumentedRules.Validate invariants pass — proving each fragment's applies_to
// exactly covers what the rule reports from AppliesTo().
func TestProjectFragmentsPassGeneration(t *testing.T) {
	entries, err := docs.LoadRuleDocEntries(projectdocs.FragmentsFS, ".")
	require.NoError(t, err)

	syntactic := projectGatekeeperRules()

	doc, verrs := docs.Generate(syntactic, nil, entries, "test", "2026-06-03T00:00:00Z")
	assert.Empty(t, verrs, "expected no validation errors, got: %v", verrs)

	require.Len(t, doc.Rules, len(syntactic))
	for _, r := range doc.Rules {
		assert.NotEmpty(t, r.AppliesTo, "rule %s has no applies_to", r.RuleID)
	}
}
