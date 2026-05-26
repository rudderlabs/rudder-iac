package docs

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRule struct{}

func (stubRule) ID() string                                                 { return "" }
func (stubRule) Severity() rules.Severity                                   { return rules.Error }
func (stubRule) Description() string                                        { return "" }
func (stubRule) AppliesTo() []rules.MatchPattern                            { return nil }
func (stubRule) Examples() rules.Examples                                   { return rules.Examples{} }
func (stubRule) Validate(*rules.ValidationContext) []rules.ValidationResult { return nil }

type stubDocumentedRule struct {
	stubRule
	entries []MatchBehaviorEntry
}

func (d stubDocumentedRule) DocExamples() []MatchBehaviorEntry { return d.entries }

func TestExamplesResolver_ReturnsNilForRulesWithoutDocumented(t *testing.T) {
	got, err := ExamplesResolver{}.ResolveFor(stubRule{})
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestExamplesResolver_ReturnsNilForEmptyDocExamples(t *testing.T) {
	got, err := ExamplesResolver{}.ResolveFor(stubDocumentedRule{entries: nil})
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestExamplesResolver_ReturnsPopulatedResolvedRule(t *testing.T) {
	entries := []MatchBehaviorEntry{
		{AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v1"}}},
	}
	got, err := ExamplesResolver{}.ResolveFor(stubDocumentedRule{entries: entries})
	require.NoError(t, err)
	assert.Equal(t, &ResolvedRule{MatchBehavior: entries}, got)
}
