package docs

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRegistry struct {
	syntactic []rules.Rule
	semantic  []rules.Rule
}

func (f fakeRegistry) RegisterSyntactic(r rules.Rule)                      {}
func (f fakeRegistry) RegisterSemantic(r rules.Rule)                       {}
func (f fakeRegistry) SyntacticRulesFor(kind, version string) []rules.Rule { return f.syntactic }
func (f fakeRegistry) SemanticRulesFor(kind, version string) []rules.Rule  { return f.semantic }
func (f fakeRegistry) AllSyntacticRules() []rules.Rule                     { return f.syntactic }
func (f fakeRegistry) AllSemanticRules() []rules.Rule                      { return f.semantic }

type docRuleStub struct {
	id          string
	description string
	severity    rules.Severity
	appliesTo   []rules.MatchPattern
	entries     []MatchBehaviorEntry
}

func (d docRuleStub) ID() string                                              { return d.id }
func (d docRuleStub) Severity() rules.Severity                                { return d.severity }
func (d docRuleStub) Description() string                                     { return d.description }
func (d docRuleStub) AppliesTo() []rules.MatchPattern                         { return d.appliesTo }
func (d docRuleStub) Examples() rules.Examples                                { return rules.Examples{} }
func (d docRuleStub) Validate(*rules.ValidationContext) []rules.ValidationResult { return nil }
func (d docRuleStub) DocExamples() []MatchBehaviorEntry                       { return d.entries }

func TestGenerator_SkipsRulesWithoutDocs(t *testing.T) {
	g := NewGenerator(ExamplesResolver{}, "test-version")
	reg := fakeRegistry{syntactic: []rules.Rule{stubRule{}}}
	doc, err := g.Generate(reg)
	require.NoError(t, err)
	assert.Empty(t, doc.Rules)
	assert.Equal(t, "test-version", doc.ToolMetadata.CLIVersion)
}

func TestGenerator_EnrichesResolvedRuleFromRuleInterface(t *testing.T) {
	rule := docRuleStub{
		id:          "test/rule",
		description: "test rule",
		severity:    rules.Error,
		appliesTo:   []rules.MatchPattern{{Kind: "source", Version: "v1"}},
		entries: []MatchBehaviorEntry{
			{AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v1"}}},
		},
	}
	g := NewGenerator(ExamplesResolver{}, "v0.0.0")
	reg := fakeRegistry{syntactic: []rules.Rule{rule}}

	doc, err := g.Generate(reg)
	require.NoError(t, err)
	require.Len(t, doc.Rules, 1)
	assert.Equal(t, ResolvedRule{
		RuleID:      "test/rule",
		Phase:       "syntactic",
		Severity:    "error",
		Description: "test rule",
		AppliesTo:   []MatchPatternDoc{{Kind: "source", Version: "v1"}},
		MatchBehavior: []MatchBehaviorEntry{
			{AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v1"}}},
		},
	}, doc.Rules[0])
}

func TestGenerator_SemanticRulesGetSemanticPhase(t *testing.T) {
	rule := docRuleStub{
		id:          "sem/rule",
		description: "x",
		severity:    rules.Warning,
		appliesTo:   []rules.MatchPattern{rules.MatchAll()},
		entries: []MatchBehaviorEntry{
			{AppliesTo: []MatchPatternDoc{{Kind: "*", Version: "*"}}},
		},
	}
	g := NewGenerator(ExamplesResolver{}, "v")
	reg := fakeRegistry{semantic: []rules.Rule{rule}}
	doc, err := g.Generate(reg)
	require.NoError(t, err)
	require.Len(t, doc.Rules, 1)
	assert.Equal(t, "semantic", doc.Rules[0].Phase)
}
