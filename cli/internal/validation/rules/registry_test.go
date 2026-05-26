package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockRule is a simple Rule implementation for testing
type mockRule struct {
	id          string
	severity    Severity
	description string
	appliesTo   []MatchPattern
}

func (m *mockRule) ID() string               { return m.id }
func (m *mockRule) Severity() Severity       { return m.severity }
func (m *mockRule) Description() string      { return m.description }
func (m *mockRule) AppliesTo() []MatchPattern { return m.appliesTo }
func (m *mockRule) Examples() Examples       { return Examples{} }
func (m *mockRule) Validate(ctx *ValidationContext) []ValidationResult {
	return nil
}

func TestRegistry_SyntacticRulesFor(t *testing.T) {
	t.Run("kind-specific rule matches", func(t *testing.T) {
		registry := NewRegistry()

		rule := &mockRule{
			id:        "prop-rule",
			appliesTo: []MatchPattern{MatchKind("properties")},
		}
		registry.RegisterSyntactic(rule)

		rules := registry.SyntacticRulesFor("properties", "rudder/v1")
		assert.Len(t, rules, 1)
		assert.Equal(t, "prop-rule", rules[0].ID())
	})

	t.Run("wildcard kind rule matches any kind", func(t *testing.T) {
		registry := NewRegistry()

		rule := &mockRule{
			id:        "wildcard-rule",
			appliesTo: []MatchPattern{MatchAll()},
		}
		registry.RegisterSyntactic(rule)

		rules := registry.SyntacticRulesFor("properties", "rudder/v1")
		assert.Len(t, rules, 1)

		rules = registry.SyntacticRulesFor("events", "rudder/v1")
		assert.Len(t, rules, 1)
	})

	t.Run("kind-specific plus wildcard rules combined", func(t *testing.T) {
		registry := NewRegistry()

		registry.RegisterSyntactic(&mockRule{
			id:        "prop-rule",
			appliesTo: []MatchPattern{MatchKind("properties")},
		})
		registry.RegisterSyntactic(&mockRule{
			id:        "wildcard-rule",
			appliesTo: []MatchPattern{MatchAll()},
		})

		rules := registry.SyntacticRulesFor("properties", "rudder/v1")
		assert.Len(t, rules, 2)
		assert.Contains(t, ruleIDs(rules), "prop-rule")
		assert.Contains(t, ruleIDs(rules), "wildcard-rule")
	})

	t.Run("unknown kind only gets wildcard", func(t *testing.T) {
		registry := NewRegistry()

		registry.RegisterSyntactic(&mockRule{
			id:        "prop-rule",
			appliesTo: []MatchPattern{MatchKind("properties")},
		})
		registry.RegisterSyntactic(&mockRule{
			id:        "wildcard-rule",
			appliesTo: []MatchPattern{MatchAll()},
		})

		rules := registry.SyntacticRulesFor("unknown", "rudder/v1")
		assert.Len(t, rules, 1)
		assert.Equal(t, "wildcard-rule", rules[0].ID())
	})

	t.Run("no matching rules returns empty", func(t *testing.T) {
		registry := NewRegistry()

		registry.RegisterSyntactic(&mockRule{
			id:        "prop-rule",
			appliesTo: []MatchPattern{MatchKind("properties")},
		})

		rules := registry.SyntacticRulesFor("events", "rudder/v1")
		assert.Empty(t, rules)
	})
}

func TestRegistry_SemanticRulesFor(t *testing.T) {
	registry := NewRegistry()

	registry.RegisterSemantic(&mockRule{
		id:        "ref-rule",
		appliesTo: []MatchPattern{MatchKind("properties"), MatchKind("events")},
	})
	registry.RegisterSemantic(&mockRule{
		id:        "dep-rule",
		appliesTo: []MatchPattern{MatchAll()},
	})

	t.Run("properties gets specific rule plus wildcard", func(t *testing.T) {
		rules := registry.SemanticRulesFor("properties", "rudder/v1")
		assert.Len(t, rules, 2)
		assert.Contains(t, ruleIDs(rules), "ref-rule")
		assert.Contains(t, ruleIDs(rules), "dep-rule")
	})

	t.Run("events gets specific rule plus wildcard", func(t *testing.T) {
		rules := registry.SemanticRulesFor("events", "rudder/v1")
		assert.Len(t, rules, 2)
		assert.Contains(t, ruleIDs(rules), "ref-rule")
		assert.Contains(t, ruleIDs(rules), "dep-rule")
	})

	t.Run("unknown kind only gets wildcard", func(t *testing.T) {
		rules := registry.SemanticRulesFor("custom-types", "rudder/v1")
		assert.Len(t, rules, 1)
		assert.Contains(t, ruleIDs(rules), "dep-rule")
	})
}

func TestRegistry_VersionFiltering(t *testing.T) {
	t.Run("version-specific rule only matches that version", func(t *testing.T) {
		registry := NewRegistry()

		registry.RegisterSyntactic(&mockRule{
			id:        "v1-only",
			appliesTo: []MatchPattern{MatchKindVersion("properties", "rudder/v1")},
		})

		rules := registry.SyntacticRulesFor("properties", "rudder/v1")
		assert.Len(t, rules, 1)

		rules = registry.SyntacticRulesFor("properties", "rudder/v2")
		assert.Empty(t, rules)
	})

	t.Run("wildcard version matches any version", func(t *testing.T) {
		registry := NewRegistry()

		registry.RegisterSyntactic(&mockRule{
			id:        "all-versions",
			appliesTo: []MatchPattern{MatchKind("properties")},
		})

		rules := registry.SyntacticRulesFor("properties", "rudder/v1")
		assert.Len(t, rules, 1)

		rules = registry.SyntacticRulesFor("properties", "rudder/v2")
		assert.Len(t, rules, 1)
	})

	t.Run("mixed patterns with selective version matching", func(t *testing.T) {
		registry := NewRegistry()

		// Rule applies to all kinds for v1, but only foo-a for v2
		registry.RegisterSyntactic(&mockRule{
			id: "mixed-rule",
			appliesTo: []MatchPattern{
				{Kind: "*", Version: "rudder/v1"},
				{Kind: "foo-a", Version: "rudder/v2"},
			},
		})

		// events + v1 → matches via wildcard kind pattern
		rules := registry.SyntacticRulesFor("events", "rudder/v1")
		assert.Len(t, rules, 1)

		// foo-a + v2 → matches via specific pattern
		rules = registry.SyntacticRulesFor("foo-a", "rudder/v2")
		assert.Len(t, rules, 1)

		// events + v2 → no match (wildcard is v1-only, specific is foo-a-only)
		rules = registry.SyntacticRulesFor("events", "rudder/v2")
		assert.Empty(t, rules)
	})

	t.Run("multiple rules with same ID different patterns", func(t *testing.T) {
		registry := NewRegistry()

		registry.RegisterSyntactic(&mockRule{
			id:        "shared-id",
			appliesTo: []MatchPattern{MatchKindVersion("properties", "rudder/v1")},
		})
		registry.RegisterSyntactic(&mockRule{
			id:        "shared-id",
			appliesTo: []MatchPattern{MatchKindVersion("events", "rudder/v2")},
		})

		rules := registry.SyntacticRulesFor("properties", "rudder/v1")
		assert.Len(t, rules, 1)

		rules = registry.SyntacticRulesFor("events", "rudder/v2")
		assert.Len(t, rules, 1)

		rules = registry.SyntacticRulesFor("properties", "rudder/v2")
		assert.Empty(t, rules)
	})
}

func TestRegistry_MultipleKindsPerRule(t *testing.T) {
	registry := NewRegistry()

	registry.RegisterSyntactic(&mockRule{
		id:        "multi-kind-rule",
		appliesTo: []MatchPattern{MatchKind("properties"), MatchKind("events"), MatchKind("tp")},
	})

	t.Run("rule appears for all specified kinds", func(t *testing.T) {
		assert.Contains(t, ruleIDs(registry.SyntacticRulesFor("properties", "rudder/v1")), "multi-kind-rule")
		assert.Contains(t, ruleIDs(registry.SyntacticRulesFor("events", "rudder/v1")), "multi-kind-rule")
		assert.Contains(t, ruleIDs(registry.SyntacticRulesFor("tp", "rudder/v1")), "multi-kind-rule")
	})

	t.Run("rule does not appear for other kinds", func(t *testing.T) {
		assert.NotContains(t, ruleIDs(registry.SyntacticRulesFor("custom-types", "rudder/v1")), "multi-kind-rule")
	})
}

// ruleIDs is a helper function to extract rule IDs from a slice of rules
func ruleIDs(rules []Rule) []string {
	ids := make([]string, len(rules))
	for i, rule := range rules {
		ids[i] = rule.ID()
	}
	return ids
}

func TestRegistry_AllSyntacticRules_ReturnsRegisteredCopy(t *testing.T) {
	r := NewRegistry()
	r1 := &mockRule{id: "rule-1"}
	r2 := &mockRule{id: "rule-2"}
	r.RegisterSyntactic(r1)
	r.RegisterSyntactic(r2)

	got := r.AllSyntacticRules()

	assert.Equal(t, []Rule{r1, r2}, got)
}

func TestRegistry_AllSemanticRules_ReturnsRegisteredCopy(t *testing.T) {
	r := NewRegistry()
	r1 := &mockRule{id: "rule-1"}
	r2 := &mockRule{id: "rule-2"}
	r.RegisterSemantic(r1)
	r.RegisterSemantic(r2)

	got := r.AllSemanticRules()

	assert.Equal(t, []Rule{r1, r2}, got)
}

func TestRegistry_AllSyntacticRules_DefensiveCopy(t *testing.T) {
	r := NewRegistry()
	r.RegisterSyntactic(&mockRule{id: "rule-1"})

	got := r.AllSyntacticRules()
	got[0] = &mockRule{id: "mutated"}

	// mutating the returned slice must not affect future calls
	assert.Equal(t, "rule-1", r.AllSyntacticRules()[0].ID())
}
