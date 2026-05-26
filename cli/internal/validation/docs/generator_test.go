package docs

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRule struct {
	id          string
	severity    rules.Severity
	description string
	appliesTo   []rules.MatchPattern
}

func (f *fakeRule) ID() string                       { return f.id }
func (f *fakeRule) Severity() rules.Severity         { return f.severity }
func (f *fakeRule) Description() string              { return f.description }
func (f *fakeRule) AppliesTo() []rules.MatchPattern  { return f.appliesTo }
func (f *fakeRule) Examples() rules.Examples         { return rules.Examples{} }
func (f *fakeRule) Validate(_ *rules.ValidationContext) []rules.ValidationResult {
	return nil
}

type fakeResolver struct {
	byID map[string]*RuleDocEntry
}

func (f *fakeResolver) ResolveFor(r rules.Rule) (*RuleDocEntry, error) {
	return f.byID[r.ID()], nil
}

func TestGenerator_Generate_EnrichesResolvedRule(t *testing.T) {
	syntactic := &fakeRule{
		id:          "rule-syn",
		severity:    rules.Error,
		description: "syntactic rule",
		appliesTo:   []rules.MatchPattern{rules.MatchKindVersion("source", "v1")},
	}
	semantic := &fakeRule{
		id:          "rule-sem",
		severity:    rules.Warning,
		description: "semantic rule",
		appliesTo:   []rules.MatchPattern{rules.MatchAll()},
	}

	reg := rules.NewRegistry()
	reg.RegisterSyntactic(syntactic)
	reg.RegisterSemantic(semantic)

	resolver := &fakeResolver{byID: map[string]*RuleDocEntry{
		"rule-syn": {
			RuleID: "rule-syn",
			MatchBehavior: []MatchBehaviorEntry{
				{AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v1"}}},
			},
		},
		"rule-sem": {
			RuleID: "rule-sem",
			MatchBehavior: []MatchBehaviorEntry{
				{AppliesTo: []MatchPatternDoc{{Kind: "*", Version: "*"}}},
			},
		},
	}}

	gen := NewGenerator(reg, resolver, GeneratorOptions{CLIVersion: "test-1.0", SchemaVersion: 1})

	doc, err := gen.Generate()
	require.NoError(t, err)

	require.Len(t, doc.Rules, 2)
	// Order: syntactic then semantic, registration order preserved.
	assert.Equal(t, "rule-syn", doc.Rules[0].RuleID)
	assert.Equal(t, "syntactic", doc.Rules[0].Phase)
	assert.Equal(t, "error", doc.Rules[0].Severity)
	assert.Equal(t, "syntactic rule", doc.Rules[0].Description)
	assert.Equal(t, []MatchPatternDoc{{Kind: "source", Version: "v1"}}, doc.Rules[0].AppliesTo)

	assert.Equal(t, "rule-sem", doc.Rules[1].RuleID)
	assert.Equal(t, "semantic", doc.Rules[1].Phase)
	assert.Equal(t, "warning", doc.Rules[1].Severity)
}

func TestGenerator_Generate_SkipsRulesWithoutDocs(t *testing.T) {
	reg := rules.NewRegistry()
	reg.RegisterSyntactic(&fakeRule{id: "rule-undocumented"})
	resolver := &fakeResolver{byID: map[string]*RuleDocEntry{}}

	gen := NewGenerator(reg, resolver, GeneratorOptions{CLIVersion: "x", SchemaVersion: 1})

	doc, err := gen.Generate()
	require.NoError(t, err)
	assert.Empty(t, doc.Rules)
}

func TestGenerator_Generate_PassesStructuralValidate(t *testing.T) {
	rule := &fakeRule{
		id:          "rule-syn",
		severity:    rules.Error,
		description: "syntactic rule",
		appliesTo:   []rules.MatchPattern{rules.MatchKindVersion("source", "v1")},
	}
	reg := rules.NewRegistry()
	reg.RegisterSyntactic(rule)
	resolver := &fakeResolver{byID: map[string]*RuleDocEntry{
		"rule-syn": {
			RuleID: "rule-syn",
			MatchBehavior: []MatchBehaviorEntry{
				{AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v1"}}},
			},
		},
	}}

	gen := NewGenerator(reg, resolver, GeneratorOptions{CLIVersion: "x", SchemaVersion: 1})
	doc, err := gen.Generate()
	require.NoError(t, err)

	errs := doc.Validate(nil)
	assert.Empty(t, errs)
}
