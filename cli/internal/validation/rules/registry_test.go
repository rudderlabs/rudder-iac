package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registryTestSupportedPatterns lists (kind, version) pairs for tests that register
// rules with concrete AppliesTo patterns.
var registryTestSupportedPatterns = []MatchPattern{
	MatchKindVersion("properties", "rudder/v1"),
	MatchKindVersion("properties", "rudder/v2"),
	MatchKindVersion("events", "rudder/v1"),
	MatchKindVersion("events", "rudder/v2"),
	MatchKindVersion("tp", "rudder/v1"),
	MatchKindVersion("foo-a", "rudder/v2"),
}

// mockRule is a simple Rule implementation for testing
type mockRule struct {
	id          string
	severity    Severity
	description string
	appliesTo   []MatchPattern
}

func (m *mockRule) ID() string                { return m.id }
func (m *mockRule) Severity() Severity        { return m.severity }
func (m *mockRule) Description() string       { return m.description }
func (m *mockRule) AppliesTo() []MatchPattern { return m.appliesTo }
func (m *mockRule) Examples() Examples        { return Examples{} }
func (m *mockRule) Validate(ctx *ValidationContext) []ValidationResult {
	return nil
}

func TestRegistry_SyntacticRulesFor(t *testing.T) {
	t.Run("kind-specific rule matches", func(t *testing.T) {
		registry := NewRegistry(registryTestSupportedPatterns)

		rule := &mockRule{
			id:        "prop-rule",
			appliesTo: []MatchPattern{MatchKind("properties")},
		}
		require.NoError(t, registry.RegisterSyntactic(rule))

		rules := registry.SyntacticRulesFor("properties", "rudder/v1")
		assert.Len(t, rules, 1)
		assert.Equal(t, "prop-rule", rules[0].ID())
	})

	t.Run("wildcard kind rule matches any kind", func(t *testing.T) {
		registry := NewRegistry(nil)

		rule := &mockRule{
			id:        "wildcard-rule",
			appliesTo: []MatchPattern{MatchAll()},
		}
		require.NoError(t, registry.RegisterSyntactic(rule))

		rules := registry.SyntacticRulesFor("properties", "rudder/v1")
		assert.Len(t, rules, 1)

		rules = registry.SyntacticRulesFor("events", "rudder/v1")
		assert.Len(t, rules, 1)
	})

	t.Run("kind-specific plus wildcard rules combined", func(t *testing.T) {
		registry := NewRegistry(registryTestSupportedPatterns)

		require.NoError(t, registry.RegisterSyntactic(&mockRule{
			id:        "prop-rule",
			appliesTo: []MatchPattern{MatchKind("properties")},
		}))
		require.NoError(t, registry.RegisterSyntactic(&mockRule{
			id:        "wildcard-rule",
			appliesTo: []MatchPattern{MatchAll()},
		}))

		rules := registry.SyntacticRulesFor("properties", "rudder/v1")
		assert.Len(t, rules, 2)
		assert.Contains(t, ruleIDs(rules), "prop-rule")
		assert.Contains(t, ruleIDs(rules), "wildcard-rule")
	})

	t.Run("unknown kind only gets wildcard", func(t *testing.T) {
		registry := NewRegistry(registryTestSupportedPatterns)

		require.NoError(t, registry.RegisterSyntactic(&mockRule{
			id:        "prop-rule",
			appliesTo: []MatchPattern{MatchKind("properties")},
		}))
		require.NoError(t, registry.RegisterSyntactic(&mockRule{
			id:        "wildcard-rule",
			appliesTo: []MatchPattern{MatchAll()},
		}))

		rules := registry.SyntacticRulesFor("unknown", "rudder/v1")
		assert.Len(t, rules, 1)
		assert.Equal(t, "wildcard-rule", rules[0].ID())
	})

	t.Run("no matching rules returns empty", func(t *testing.T) {
		registry := NewRegistry(registryTestSupportedPatterns)

		require.NoError(t, registry.RegisterSyntactic(&mockRule{
			id:        "prop-rule",
			appliesTo: []MatchPattern{MatchKind("properties")},
		}))

		rules := registry.SyntacticRulesFor("events", "rudder/v1")
		assert.Empty(t, rules)
	})
}

func TestRegistry_SemanticRulesFor(t *testing.T) {
	registry := NewRegistry(registryTestSupportedPatterns)

	require.NoError(t, registry.RegisterSemantic(&mockRule{
		id:        "ref-rule",
		appliesTo: []MatchPattern{MatchKind("properties"), MatchKind("events")},
	}))
	require.NoError(t, registry.RegisterSemantic(&mockRule{
		id:        "dep-rule",
		appliesTo: []MatchPattern{MatchAll()},
	}))

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
		registry := NewRegistry(registryTestSupportedPatterns)

		require.NoError(t, registry.RegisterSyntactic(&mockRule{
			id:        "v1-only",
			appliesTo: []MatchPattern{MatchKindVersion("properties", "rudder/v1")},
		}))

		rules := registry.SyntacticRulesFor("properties", "rudder/v1")
		assert.Len(t, rules, 1)

		rules = registry.SyntacticRulesFor("properties", "rudder/v2")
		assert.Empty(t, rules)
	})

	t.Run("wildcard version matches any version", func(t *testing.T) {
		registry := NewRegistry(registryTestSupportedPatterns)

		require.NoError(t, registry.RegisterSyntactic(&mockRule{
			id:        "all-versions",
			appliesTo: []MatchPattern{MatchKind("properties")},
		}))

		rules := registry.SyntacticRulesFor("properties", "rudder/v1")
		assert.Len(t, rules, 1)

		rules = registry.SyntacticRulesFor("properties", "rudder/v2")
		assert.Len(t, rules, 1)
	})

	t.Run("mixed patterns with selective version matching", func(t *testing.T) {
		registry := NewRegistry(registryTestSupportedPatterns)

		// Rule applies to all kinds for v1, but only foo-a for v2
		require.NoError(t, registry.RegisterSyntactic(&mockRule{
			id: "mixed-rule",
			appliesTo: []MatchPattern{
				{Kind: "*", Version: "rudder/v1"},
				{Kind: "foo-a", Version: "rudder/v2"},
			},
		}))

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
		registry := NewRegistry(registryTestSupportedPatterns)

		require.NoError(t, registry.RegisterSyntactic(&mockRule{
			id:        "shared-id",
			appliesTo: []MatchPattern{MatchKindVersion("properties", "rudder/v1")},
		}))
		require.NoError(t, registry.RegisterSyntactic(&mockRule{
			id:        "shared-id",
			appliesTo: []MatchPattern{MatchKindVersion("events", "rudder/v2")},
		}))

		rules := registry.SyntacticRulesFor("properties", "rudder/v1")
		assert.Len(t, rules, 1)

		rules = registry.SyntacticRulesFor("events", "rudder/v2")
		assert.Len(t, rules, 1)

		rules = registry.SyntacticRulesFor("properties", "rudder/v2")
		assert.Empty(t, rules)
	})
}

func TestRegistry_MultipleKindsPerRule(t *testing.T) {
	registry := NewRegistry(registryTestSupportedPatterns)

	require.NoError(t, registry.RegisterSyntactic(&mockRule{
		id:        "multi-kind-rule",
		appliesTo: []MatchPattern{MatchKind("properties"), MatchKind("events"), MatchKind("tp")},
	}))

	t.Run("rule appears for all specified kinds", func(t *testing.T) {
		assert.Contains(t, ruleIDs(registry.SyntacticRulesFor("properties", "rudder/v1")), "multi-kind-rule")
		assert.Contains(t, ruleIDs(registry.SyntacticRulesFor("events", "rudder/v1")), "multi-kind-rule")
		assert.Contains(t, ruleIDs(registry.SyntacticRulesFor("tp", "rudder/v1")), "multi-kind-rule")
	})

	t.Run("rule does not appear for other kinds", func(t *testing.T) {
		assert.NotContains(t, ruleIDs(registry.SyntacticRulesFor("custom-types", "rudder/v1")), "multi-kind-rule")
	})
}

func TestRegistry_AllSyntacticRules(t *testing.T) {
	t.Run("returns all syntactic rules regardless of pattern", func(t *testing.T) {
		registry := NewRegistry(registryTestSupportedPatterns)

		require.NoError(t, registry.RegisterSyntactic(&mockRule{id: "wildcard-rule", appliesTo: []MatchPattern{MatchAll()}}))
		require.NoError(t, registry.RegisterSyntactic(&mockRule{id: "kind-rule", appliesTo: []MatchPattern{MatchKind("properties")}}))
		require.NoError(t, registry.RegisterSyntactic(&mockRule{id: "exact-rule", appliesTo: []MatchPattern{MatchKindVersion("events", "rudder/v1")}}))

		all := registry.AllSyntacticRules()
		assert.Len(t, all, 3)
		assert.Contains(t, ruleIDs(all), "wildcard-rule")
		assert.Contains(t, ruleIDs(all), "kind-rule")
		assert.Contains(t, ruleIDs(all), "exact-rule")
	})
}

func TestRegistry_RegisterAppliesToValidation(t *testing.T) {
	phases := []struct {
		name     string
		register func(Registry, Rule) error
	}{
		{
			name: "RegisterSyntactic",
			register: func(r Registry, rule Rule) error {
				return r.RegisterSyntactic(rule)
			},
		},
		{
			name: "RegisterSemantic",
			register: func(r Registry, rule Rule) error {
				return r.RegisterSemantic(rule)
			},
		},
	}

	for _, phase := range phases {
		t.Run(phase.name, func(t *testing.T) {
			t.Run("empty supported patterns allows MatchAll only", func(t *testing.T) {
				r := NewRegistry(nil)
				require.NoError(t, phase.register(r, &mockRule{id: "a", appliesTo: []MatchPattern{MatchAll()}}))
			})

			t.Run("empty supported patterns rejects concrete kind", func(t *testing.T) {
				r := NewRegistry(nil)
				err := phase.register(r, &mockRule{id: "bad", appliesTo: []MatchPattern{MatchKind("properties")}})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "bad")
				assert.Contains(t, err.Error(), "requires non-empty supported match patterns")
			})

			t.Run("empty AppliesTo allowed", func(t *testing.T) {
				r := NewRegistry(nil)
				require.NoError(t, phase.register(r, &mockRule{id: "no-patterns", appliesTo: nil}))
			})

			t.Run("unknown kind rejected", func(t *testing.T) {
				r := NewRegistry(registryTestSupportedPatterns)
				err := phase.register(r, &mockRule{id: "x", appliesTo: []MatchPattern{MatchKind("unknown-kind")}})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unknown-kind")
			})

			t.Run("unknown version rejected", func(t *testing.T) {
				r := NewRegistry(registryTestSupportedPatterns)
				err := phase.register(r, &mockRule{
					id:        "x",
					appliesTo: []MatchPattern{{Kind: "*", Version: "rudder/v99"}},
				})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "rudder/v99")
			})

			t.Run("exact pair required when both concrete", func(t *testing.T) {
				// Marginals satisfied (properties kind, rudder/v2 version from events) but pair must exist.
				marginalOnly := []MatchPattern{
					MatchKindVersion("properties", "rudder/v1"),
					MatchKindVersion("events", "rudder/v2"),
				}
				r := NewRegistry(marginalOnly)
				err := phase.register(r, &mockRule{
					id:        "x",
					appliesTo: []MatchPattern{MatchKindVersion("properties", "rudder/v2")},
				})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "not a supported match pattern")
			})
		})
	}
}

// ruleIDs is a helper function to extract rule IDs from a slice of rules
func ruleIDs(rules []Rule) []string {
	ids := make([]string, len(rules))
	for i, rule := range rules {
		ids[i] = rule.ID()
	}
	return ids
}
