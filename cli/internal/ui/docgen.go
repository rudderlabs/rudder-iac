package ui

import (
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/registry"
)

// RuleDocGenerator generates markdown documentation for validation rules
type RuleDocGenerator struct {
	registry *registry.RuleRegistry
}

// NewRuleDocGenerator creates a new documentation generator
func NewRuleDocGenerator(reg *registry.RuleRegistry) *RuleDocGenerator {
	return &RuleDocGenerator{registry: reg}
}

// Generate produces the complete markdown documentation
func (g *RuleDocGenerator) Generate() string {
	var sb strings.Builder

	// Header
	sb.WriteString(g.renderHeader())

	kinds := g.registry.AllKinds()
	if len(kinds) == 0 {
		sb.WriteString("\n*No validation rules registered.*\n")
		return sb.String()
	}

	// Track which rules we've already rendered (for rules that apply to multiple kinds)
	rendered := make(map[string]bool)

	// Render rules grouped by kind
	for _, kind := range kinds {
		rules := g.registry.RulesForKind(kind)
		if len(rules) == 0 {
			continue
		}

		// Filter out already-rendered rules
		var unrenderedRules []validation.Rule
		for _, rule := range rules {
			if !rendered[rule.ID()] {
				unrenderedRules = append(unrenderedRules, rule)
			}
		}

		if len(unrenderedRules) == 0 {
			continue
		}

		sb.WriteString(g.renderKindSection(kind, unrenderedRules))

		// Mark these rules as rendered
		for _, rule := range unrenderedRules {
			rendered[rule.ID()] = true
		}
	}

	return sb.String()
}

// renderHeader produces the document title and introduction
func (g *RuleDocGenerator) renderHeader() string {
	return `# Validation Rules

Documentation for all validation rules in the rudder-cli.

`
}

// renderKindSection produces a section for a specific resource kind
func (g *RuleDocGenerator) renderKindSection(kind string, rules []validation.Rule) string {
	var sb strings.Builder

	// Section header (capitalize first letter)
	sectionTitle := strings.ToUpper(kind[:1]) + kind[1:]
	sb.WriteString(fmt.Sprintf("## %s\n\n", sectionTitle))

	// Render each rule
	for _, rule := range rules {
		sb.WriteString(g.renderRule(rule))
	}

	return sb.String()
}

// renderRule produces documentation for a single rule
func (g *RuleDocGenerator) renderRule(rule validation.Rule) string {
	var sb strings.Builder

	// Rule ID as heading
	sb.WriteString(fmt.Sprintf("### %s\n\n", rule.ID()))

	// Severity badge
	sb.WriteString(g.severityBadge(rule.Severity()))
	sb.WriteString("\n\n")

	// Description
	sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", rule.Description()))

	// Applies to
	kinds := rule.AppliesTo()
	kindsList := make([]string, len(kinds))
	for i, k := range kinds {
		kindsList[i] = fmt.Sprintf("`%s`", k)
	}
	sb.WriteString(fmt.Sprintf("**Applies to:** %s\n\n", strings.Join(kindsList, ", ")))

	// Examples (if any)
	examples := rule.Examples()
	if len(examples) > 0 {
		for i, example := range examples {
			if i == 0 {
				sb.WriteString("**Example:**\n")
			} else {
				sb.WriteString(fmt.Sprintf("**Example %d:**\n", i+1))
			}
			sb.WriteString("```yaml\n")
			sb.WriteString(strings.TrimSpace(string(example)))
			sb.WriteString("\n```\n\n")
		}
	}

	// Separator
	sb.WriteString("---\n\n")

	return sb.String()
}

// severityBadge returns a shield.io badge URL for the severity
func (g *RuleDocGenerator) severityBadge(severity validation.Severity) string {
	var color string
	switch severity {
	case validation.SeverityError:
		color = "red"
	case validation.SeverityWarning:
		color = "yellow"
	case validation.SeverityInfo:
		color = "blue"
	default:
		color = "lightgrey"
	}

	return fmt.Sprintf("![Severity: %s](https://img.shields.io/badge/severity-%s-%s)",
		strings.Title(string(severity)),
		severity,
		color,
	)
}
