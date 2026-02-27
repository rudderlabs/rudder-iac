package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRule is a simple Rule implementation for testing
type mockRule struct {
	id                string
	severity          Severity
	description       string
	appliesTo         []string
	appliesToVersions []string
}

func (m *mockRule) ID() string               { return m.id }
func (m *mockRule) Severity() Severity       { return m.severity }
func (m *mockRule) Description() string      { return m.description }
func (m *mockRule) AppliesToKinds() []string { return m.appliesTo }
func (m *mockRule) AppliesToVersions() []string {
	if len(m.appliesToVersions) == 0 {
		return []string{"*"}
	}
	return m.appliesToVersions
}
func (m *mockRule) Examples() Examples                                 { return Examples{} }
func (m *mockRule) Validate(ctx *ValidationContext) []ValidationResult { return nil }

func TestRegistry_RegisterRule(t *testing.T) {
	t.Run("syntactic rules", func(t *testing.T) {
		registry := NewRegistry()

		rule := &mockRule{
			id:        "test-syntactic-rule",
			appliesTo: []string{"properties"},
		}

		err := registry.RegisterSyntactic(rule)
		require.NoError(t, err)

		// Verify rule is retrievable for its kind
		rules := registry.SyntacticRulesForKind("properties")
		require.Len(t, rules, 1)
		assert.Equal(t, "test-syntactic-rule", rules[0].ID())
	})

	t.Run("semantic rules", func(t *testing.T) {
		registry := NewRegistry()

		rule := &mockRule{
			id:        "test-semantic-rule",
			appliesTo: []string{"events"},
		}

		err := registry.RegisterSemantic(rule)
		require.NoError(t, err)

		// Verify rule is retrievable for its kind
		rules := registry.SemanticRulesForKind("events")
		require.Len(t, rules, 1)
		assert.Equal(t, "test-semantic-rule", rules[0].ID())
	})
}

func TestRegistry_RegisterRule_DuplicateError(t *testing.T) {
	t.Run("duplicate syntactic rule", func(t *testing.T) {
		registry := NewRegistry()

		rule1 := &mockRule{id: "duplicate-rule", appliesTo: []string{"properties"}}
		rule2 := &mockRule{id: "duplicate-rule", appliesTo: []string{"events"}}

		err := registry.RegisterSyntactic(rule1)
		require.NoError(t, err)

		err = registry.RegisterSyntactic(rule2)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDuplicateRule)
	})

	t.Run("duplicate semantic rule", func(t *testing.T) {
		registry := NewRegistry()

		rule1 := &mockRule{id: "duplicate-rule", appliesTo: []string{"properties"}}
		rule2 := &mockRule{id: "duplicate-rule", appliesTo: []string{"events"}}

		err := registry.RegisterSemantic(rule1)
		require.NoError(t, err)

		err = registry.RegisterSemantic(rule2)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDuplicateRule)
	})

	t.Run("duplicate across syntactic and semantic", func(t *testing.T) {
		registry := NewRegistry()

		syntacticRule := &mockRule{id: "cross-phase-rule", appliesTo: []string{"properties"}}
		semanticRule := &mockRule{id: "cross-phase-rule", appliesTo: []string{"events"}}

		err := registry.RegisterSyntactic(syntacticRule)
		require.NoError(t, err)

		err = registry.RegisterSemantic(semanticRule)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDuplicateRule)
		assert.Contains(t, err.Error(), "already registered as syntactic")
	})

	t.Run("duplicate with wildcard rule", func(t *testing.T) {
		registry := NewRegistry()

		rule1 := &mockRule{id: "wildcard-dup", appliesTo: []string{"*"}}
		rule2 := &mockRule{id: "wildcard-dup", appliesTo: []string{"properties"}}

		err := registry.RegisterSyntactic(rule1)
		require.NoError(t, err)

		err = registry.RegisterSyntactic(rule2)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDuplicateRule)
	})
}

func TestRegistry_SyntacticRulesForKind(t *testing.T) {
	registry := NewRegistry()

	propertiesRule := &mockRule{id: "prop-rule", appliesTo: []string{"properties"}}
	eventsRule := &mockRule{id: "event-rule", appliesTo: []string{"events"}}
	wildcardRule := &mockRule{id: "wildcard-rule", appliesTo: []string{"*"}}

	require.NoError(t, registry.RegisterSyntactic(propertiesRule))
	require.NoError(t, registry.RegisterSyntactic(eventsRule))
	require.NoError(t, registry.RegisterSyntactic(wildcardRule))

	t.Run("properties kind gets its rule plus wildcard", func(t *testing.T) {
		propRules := registry.SyntacticRulesForKind("properties")
		require.Len(t, propRules, 2)
		assert.Contains(t, ruleIDs(propRules), "prop-rule")
		assert.Contains(t, ruleIDs(propRules), "wildcard-rule")
	})

	t.Run("events kind gets its rule plus wildcard", func(t *testing.T) {
		eventRules := registry.SyntacticRulesForKind("events")
		require.Len(t, eventRules, 2)
		assert.Contains(t, ruleIDs(eventRules), "event-rule")
		assert.Contains(t, ruleIDs(eventRules), "wildcard-rule")
	})

	t.Run("unknown kind only gets wildcard", func(t *testing.T) {
		unknownRules := registry.SyntacticRulesForKind("unknown")
		require.Len(t, unknownRules, 1)
		assert.Contains(t, ruleIDs(unknownRules), "wildcard-rule")
	})
}

func TestRegistry_SemanticRulesForKind(t *testing.T) {
	registry := NewRegistry()

	refRule := &mockRule{id: "ref-rule", appliesTo: []string{"properties", "events"}}
	depRule := &mockRule{id: "dep-rule", appliesTo: []string{"*"}}

	require.NoError(t, registry.RegisterSemantic(refRule))
	require.NoError(t, registry.RegisterSemantic(depRule))

	t.Run("properties gets specific rule plus wildcard", func(t *testing.T) {
		propRules := registry.SemanticRulesForKind("properties")
		require.Len(t, propRules, 2)
		assert.Contains(t, ruleIDs(propRules), "ref-rule")
		assert.Contains(t, ruleIDs(propRules), "dep-rule")
	})

	t.Run("events gets specific rule plus wildcard", func(t *testing.T) {
		eventRules := registry.SemanticRulesForKind("events")
		require.Len(t, eventRules, 2)
		assert.Contains(t, ruleIDs(eventRules), "ref-rule")
		assert.Contains(t, ruleIDs(eventRules), "dep-rule")
	})

	t.Run("unknown kind only gets wildcard", func(t *testing.T) {
		unknownRules := registry.SemanticRulesForKind("custom-types")
		require.Len(t, unknownRules, 1)
		assert.Contains(t, ruleIDs(unknownRules), "dep-rule")
	})
}

func TestRegistry_MultipleKindsPerRule(t *testing.T) {
	registry := NewRegistry()

	multiKindRule := &mockRule{
		id:        "multi-kind-rule",
		appliesTo: []string{"properties", "events", "tp"},
	}

	require.NoError(t, registry.RegisterSyntactic(multiKindRule))

	t.Run("rule appears for all specified kinds", func(t *testing.T) {
		propRules := registry.SyntacticRulesForKind("properties")
		assert.Contains(t, ruleIDs(propRules), "multi-kind-rule")

		eventRules := registry.SyntacticRulesForKind("events")
		assert.Contains(t, ruleIDs(eventRules), "multi-kind-rule")

		tpRules := registry.SyntacticRulesForKind("tp")
		assert.Contains(t, ruleIDs(tpRules), "multi-kind-rule")
	})

	t.Run("rule does not appear for other kinds", func(t *testing.T) {
		customTypeRules := registry.SyntacticRulesForKind("custom-types")
		assert.NotContains(t, ruleIDs(customTypeRules), "multi-kind-rule")
	})
}

func TestRegistry_NoRuleForKind(t *testing.T) {
	registry := NewRegistry()

	// Register rule for specific kind
	rule := &mockRule{id: "specific-rule", appliesTo: []string{"properties"}}
	require.NoError(t, registry.RegisterSyntactic(rule))

	t.Run("kind with no rules returns empty slice", func(t *testing.T) {
		rules := registry.SyntacticRulesForKind("unknown-kind")
		assert.Empty(t, rules)
	})

	t.Run("semantic rules for unknown kind returns empty slice", func(t *testing.T) {
		rules := registry.SemanticRulesForKind("unknown-kind")
		assert.Empty(t, rules)
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
