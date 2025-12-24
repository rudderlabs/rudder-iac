package registry

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRule struct {
	id     string
	kinds  []string
	called bool
}

func (m *mockRule) ID() string { return m.id }
func (m *mockRule) Validate(ctx *validation.ValidationContext, graph *resources.Graph) []validation.ValidationError {
	m.called = true
	return nil
}
func (m *mockRule) Severity() validation.Severity { return validation.SeverityError }
func (m *mockRule) Description() string           { return "mock description" }
func (m *mockRule) Examples() [][]byte            { return nil }
func (m *mockRule) AppliesTo() []string           { return m.kinds }

func TestRuleRegistry(t *testing.T) {
	t.Run("register and retrieve rules", func(t *testing.T) {
		reg := NewRegistry()
		rule1 := &mockRule{id: "rule1", kinds: []string{"kind1"}}
		rule2 := &mockRule{id: "rule2", kinds: []string{"kind2"}}
		rule3 := &mockRule{id: "rule3", kinds: []string{"kind1", "kind2"}}

		require.NoError(t, reg.Register(rule1))
		require.NoError(t, reg.Register(rule2))
		require.NoError(t, reg.Register(rule3))

		kind1Rules := reg.RulesForKind("kind1")
		assert.Len(t, kind1Rules, 2)
		assert.Contains(t, kind1Rules, rule1)
		assert.Contains(t, kind1Rules, rule3)

		kind2Rules := reg.RulesForKind("kind2")
		assert.Len(t, kind2Rules, 2)
		assert.Contains(t, kind2Rules, rule2)
		assert.Contains(t, kind2Rules, rule3)

		allRules := reg.AllRules()
		assert.Len(t, allRules, 3)
	})

	t.Run("duplicate rule ID", func(t *testing.T) {
		reg := NewRegistry()
		rule1 := &mockRule{id: "rule1", kinds: []string{"kind1"}}
		rule2 := &mockRule{id: "rule1", kinds: []string{"kind2"}}

		require.NoError(t, reg.Register(rule1))
		assert.Error(t, reg.Register(rule2))
	})

	t.Run("empty kinds", func(t *testing.T) {
		reg := NewRegistry()
		rule := &mockRule{id: "rule1", kinds: []string{}}
		assert.Error(t, reg.Register(rule))
	})

	t.Run("non-existent kind", func(t *testing.T) {
		reg := NewRegistry()
		rules := reg.RulesForKind("unknown")
		assert.Empty(t, rules)
	})
}

