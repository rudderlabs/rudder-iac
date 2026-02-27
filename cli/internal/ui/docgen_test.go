package ui

import (
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRule struct {
	id          string
	severity    rules.Severity
	description string
	appliesTo   []string
	examples    rules.Examples
}

func (m *mockRule) ID() string                                          { return m.id }
func (m *mockRule) Severity() rules.Severity                            { return m.severity }
func (m *mockRule) Description() string                                 { return m.description }
func (m *mockRule) AppliesTo() []string                                 { return m.appliesTo }
func (m *mockRule) Examples() rules.Examples                            { return m.examples }
func (m *mockRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult { return nil }

func TestRuleDocGenerator_Generate_EmptyRegistry(t *testing.T) {
	registry := rules.NewRegistry()
	generator := NewRuleDocGenerator(registry)

	result := generator.Generate()

	assert.Contains(t, result, "# Validation Rules")
	assert.Contains(t, result, "*No validation rules registered.*")
}

func TestRuleDocGenerator_Generate_WithRules(t *testing.T) {
	registry := rules.NewRegistry()

	rule1 := &mockRule{
		id:          "test/rule-one",
		severity:    rules.Error,
		description: "This is rule one",
		appliesTo:   []string{"properties"},
		examples: rules.Examples{
			Valid:   []string{"valid: example"},
			Invalid: []string{"invalid: example"},
		},
	}

	rule2 := &mockRule{
		id:          "test/rule-two",
		severity:    rules.Warning,
		description: "This is rule two",
		appliesTo:   []string{"events"},
		examples:    rules.Examples{},
	}

	require.NoError(t, registry.RegisterSyntactic(rule1))
	require.NoError(t, registry.RegisterSemantic(rule2))

	generator := NewRuleDocGenerator(registry)
	result := generator.Generate()

	// Check header
	assert.Contains(t, result, "# Validation Rules")

	// Check rule one
	assert.Contains(t, result, "### test/rule-one")
	assert.Contains(t, result, "**Description:** This is rule one")
	assert.Contains(t, result, "`properties`")
	assert.Contains(t, result, "severity-error-red")
	assert.Contains(t, result, "**Valid Examples:**")
	assert.Contains(t, result, "valid: example")
	assert.Contains(t, result, "**Invalid Examples:**")
	assert.Contains(t, result, "invalid: example")

	// Check rule two
	assert.Contains(t, result, "### test/rule-two")
	assert.Contains(t, result, "**Description:** This is rule two")
	assert.Contains(t, result, "`events`")
	assert.Contains(t, result, "severity-warning-yellow")
}

func TestRuleDocGenerator_Generate_WildcardRule(t *testing.T) {
	registry := rules.NewRegistry()

	rule := &mockRule{
		id:          "global/wildcard-rule",
		severity:    rules.Info,
		description: "Applies to all kinds",
		appliesTo:   []string{"*"},
		examples:    rules.Examples{},
	}

	require.NoError(t, registry.RegisterSyntactic(rule))

	generator := NewRuleDocGenerator(registry)
	result := generator.Generate()

	// Wildcard rules should be grouped under "Global"
	assert.Contains(t, result, "## Global")
	assert.Contains(t, result, "### global/wildcard-rule")
	assert.Contains(t, result, "severity-info-blue")
}

func TestRuleDocGenerator_severityBadge(t *testing.T) {
	generator := &RuleDocGenerator{}

	tests := []struct {
		severity rules.Severity
		expected string
	}{
		{rules.Error, "severity-error-red"},
		{rules.Warning, "severity-warning-yellow"},
		{rules.Info, "severity-info-blue"},
	}

	for _, tt := range tests {
		badge := generator.severityBadge(tt.severity)
		assert.Contains(t, badge, tt.expected)
		assert.True(t, strings.HasPrefix(badge, "![Severity:"))
	}
}
