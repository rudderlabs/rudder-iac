package docs

import (
	"testing"
	"testing/fstest"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalRuleStub satisfies rules.Rule for resolver tests without depending
// on any real rule registration.
type minimalRuleStub struct {
	id string
}

func (s *minimalRuleStub) ID() string                       { return s.id }
func (s *minimalRuleStub) Severity() rules.Severity         { return rules.Error }
func (s *minimalRuleStub) Description() string              { return "" }
func (s *minimalRuleStub) AppliesTo() []rules.MatchPattern  { return nil }
func (s *minimalRuleStub) Examples() rules.Examples         { return rules.Examples{} }
func (s *minimalRuleStub) Validate(_ *rules.ValidationContext) []rules.ValidationResult {
	return nil
}

func TestYAMLResolver_ResolveFor_ReturnsAuthoredEntry(t *testing.T) {
	fsys := fstest.MapFS{
		"frags/rule-a.docs.yaml": &fstest.MapFile{Data: []byte("rule_id: rule-a\nmatch_behavior:\n  - applies_to:\n      - {kind: source, version: v1}\n")},
	}
	r, err := NewYAMLResolver(fsys, "frags")
	require.NoError(t, err)

	got, err := r.ResolveFor(&minimalRuleStub{id: "rule-a"})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "rule-a", got.RuleID)
	require.Len(t, got.MatchBehavior, 1)
	assert.Equal(t, MatchPatternDoc{Kind: "source", Version: "v1"}, got.MatchBehavior[0].AppliesTo[0])
}

func TestYAMLResolver_ResolveFor_ReturnsNilWhenNoDocs(t *testing.T) {
	fsys := fstest.MapFS{
		"frags/rule-a.docs.yaml": &fstest.MapFile{Data: []byte("rule_id: rule-a\nmatch_behavior: []\n")},
	}
	r, err := NewYAMLResolver(fsys, "frags")
	require.NoError(t, err)

	got, err := r.ResolveFor(&minimalRuleStub{id: "rule-not-authored"})
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestYAMLResolver_ResolveFor_ErrorsOnDuplicateRuleID(t *testing.T) {
	fsys := fstest.MapFS{
		"frags/a.docs.yaml": &fstest.MapFile{Data: []byte("rule_id: dup\n")},
		"frags/b.docs.yaml": &fstest.MapFile{Data: []byte("rule_id: dup\n")},
	}
	_, err := NewYAMLResolver(fsys, "frags")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate rule_id")
}

func TestNewYAMLResolver_PropagatesLoadError(t *testing.T) {
	fsys := fstest.MapFS{}
	_, err := NewYAMLResolver(fsys, "no-such-dir")
	require.Error(t, err)
}
