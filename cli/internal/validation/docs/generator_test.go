package docs

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRule is a minimal rules.Rule implementation for generator fixtures.
type fakeRule struct {
	id          string
	severity    rules.Severity
	description string
	appliesTo   []rules.MatchPattern
}

func (r fakeRule) ID() string                                                   { return r.id }
func (r fakeRule) Severity() rules.Severity                                     { return r.severity }
func (r fakeRule) Description() string                                          { return r.description }
func (r fakeRule) AppliesTo() []rules.MatchPattern                              { return r.appliesTo }
func (r fakeRule) Examples() rules.Examples                                     { return rules.Examples{} }
func (r fakeRule) Validate(_ *rules.ValidationContext) []rules.ValidationResult { return nil }

func sampleMatchBehavior(kind, version string) []MatchBehaviorEntry {
	return []MatchBehaviorEntry{
		{
			AppliesTo: []MatchPatternDoc{{Kind: kind, Version: version}},
		},
	}
}

func TestGenerate_EnrichmentAndJoin(t *testing.T) {
	var (
		syntactic = []rules.Rule{
			fakeRule{
				id:          "datacatalog/categories/spec-syntax-valid",
				severity:    rules.Error,
				description: "categories spec must be syntactically valid",
				appliesTo:   []rules.MatchPattern{{Kind: "categories", Version: "rudder/v1"}},
			},
		}
		semantic = []rules.Rule{
			fakeRule{
				id:          "project/duplicate-urn",
				severity:    rules.Warning,
				description: "URNs must be unique across the project",
				appliesTo:   []rules.MatchPattern{{Kind: "*", Version: "*"}},
			},
		}
		entries = []RuleDocEntry{
			{
				RuleID:        "datacatalog/categories/spec-syntax-valid",
				MatchBehavior: sampleMatchBehavior("categories", "rudder/v1"),
			},
			{
				RuleID:        "project/duplicate-urn",
				MatchBehavior: sampleMatchBehavior("*", "*"),
			},
		}
	)

	doc, errs := Generate(syntactic, semantic, entries, "1.2.3", "2026-06-01T00:00:00Z")

	require.Empty(t, errs)
	require.Equal(t, 1, doc.SchemaVersion)
	require.Equal(t, "1.2.3", doc.ToolMetadata.CLIVersion)

	// Sorted by RuleID: datacatalog/... comes before project/...
	require.Len(t, doc.Rules, 2)
	assert.Equal(t, DocumentedRule{
		RuleID:        "datacatalog/categories/spec-syntax-valid",
		Phase:         "syntactic",
		Severity:      "error",
		Description:   "categories spec must be syntactically valid",
		AppliesTo:     []MatchPatternDoc{{Kind: "categories", Version: "rudder/v1"}},
		MatchBehavior: sampleMatchBehavior("categories", "rudder/v1"),
	}, doc.Rules[0])
	assert.Equal(t, DocumentedRule{
		RuleID:        "project/duplicate-urn",
		Phase:         "semantic",
		Severity:      "warning",
		Description:   "URNs must be unique across the project",
		AppliesTo:     []MatchPatternDoc{{Kind: "*", Version: "*"}},
		MatchBehavior: sampleMatchBehavior("*", "*"),
	}, doc.Rules[1])
}

func TestGenerate_DeterministicSortByRuleID(t *testing.T) {
	var (
		syntactic = []rules.Rule{
			fakeRule{
				id:          "zeta/last",
				severity:    rules.Info,
				description: "z",
				appliesTo:   []rules.MatchPattern{{Kind: "k", Version: "v"}},
			},
			fakeRule{
				id:          "alpha/first",
				severity:    rules.Info,
				description: "a",
				appliesTo:   []rules.MatchPattern{{Kind: "k", Version: "v"}},
			},
		}
		entries = []RuleDocEntry{
			{RuleID: "zeta/last", MatchBehavior: sampleMatchBehavior("k", "v")},
			{RuleID: "alpha/first", MatchBehavior: sampleMatchBehavior("k", "v")},
		}
	)

	doc, errs := Generate(syntactic, nil, entries, "1.0.0", "now")

	require.Empty(t, errs)
	require.Len(t, doc.Rules, 2)
	assert.Equal(t, "alpha/first", doc.Rules[0].RuleID)
	assert.Equal(t, "zeta/last", doc.Rules[1].RuleID)
}

func TestGenerate_MissingAuthoredEntryYieldsErrorsButReturnsDoc(t *testing.T) {
	var (
		syntactic = []rules.Rule{
			fakeRule{
				id:          "datacatalog/undocumented",
				severity:    rules.Error,
				description: "no authored entry exists for this rule",
				appliesTo:   []rules.MatchPattern{{Kind: "categories", Version: "rudder/v1"}},
			},
		}
	)

	doc, errs := Generate(syntactic, nil, nil, "1.0.0", "now")

	// MatchBehavior is required (min=1), so a missing authored entry surfaces
	// as a validation error — expected during the skeleton phase.
	require.NotEmpty(t, errs)

	// The doc is still returned so the caller can inspect it.
	require.Len(t, doc.Rules, 1)
	resolved := doc.Rules[0]
	assert.Equal(t, "datacatalog/undocumented", resolved.RuleID)
	assert.Equal(t, "syntactic", resolved.Phase)
	assert.Nil(t, resolved.MatchBehavior)
}
