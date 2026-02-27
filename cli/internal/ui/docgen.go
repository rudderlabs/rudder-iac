package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// RuleDocGenerator generates markdown documentation for validation rules.
type RuleDocGenerator struct {
	registry rules.Registry
}

// NewRuleDocGenerator creates a new documentation generator.
func NewRuleDocGenerator(reg rules.Registry) *RuleDocGenerator {
	return &RuleDocGenerator{registry: reg}
}

// Generate produces the complete markdown documentation for all registered rules.
func (g *RuleDocGenerator) Generate() string {
	var sb strings.Builder

	sb.WriteString(g.renderHeader())

	allRules := g.registry.AllRules()
	if len(allRules) == 0 {
		sb.WriteString("\n*No validation rules registered.*\n")
		return sb.String()
	}

	// Group rules by their first "AppliesTo" kind for organization
	rulesByKind := g.groupRulesByKind(allRules)

	// Get sorted kinds
	kinds := make([]string, 0, len(rulesByKind))
	for kind := range rulesByKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)

	// Render rules grouped by kind
	for _, kind := range kinds {
		kindRules := rulesByKind[kind]
		sb.WriteString(g.renderKindSection(kind, kindRules))
	}

	return sb.String()
}

// groupRulesByKind organizes rules by their primary kind.
// Rules with "*" (wildcard) are grouped under "Global".
func (g *RuleDocGenerator) groupRulesByKind(allRules []rules.Rule) map[string][]rules.Rule {
	rulesByKind := make(map[string][]rules.Rule)

	for _, rule := range allRules {
		appliesTo := rule.AppliesTo()
		if len(appliesTo) == 0 {
			continue
		}

		// Use the first kind as the primary grouping
		primaryKind := appliesTo[0]
		if primaryKind == "*" {
			primaryKind = "global"
		}

		rulesByKind[primaryKind] = append(rulesByKind[primaryKind], rule)
	}

	return rulesByKind
}

// renderHeader produces the document title and introduction.
func (g *RuleDocGenerator) renderHeader() string {
	return `# Validation Rules

Documentation for all validation rules in the rudder-cli.

`
}

// renderKindSection produces a section for a specific resource kind.
func (g *RuleDocGenerator) renderKindSection(kind string, kindRules []rules.Rule) string {
	var sb strings.Builder

	// Section header (capitalize first letter)
	sectionTitle := strings.ToUpper(kind[:1]) + kind[1:]
	sb.WriteString(fmt.Sprintf("## %s\n\n", sectionTitle))

	// Sort rules by ID for consistent output
	sort.Slice(kindRules, func(i, j int) bool {
		return kindRules[i].ID() < kindRules[j].ID()
	})

	for _, rule := range kindRules {
		sb.WriteString(g.renderRule(rule))
	}

	return sb.String()
}

// renderRule produces documentation for a single rule.
func (g *RuleDocGenerator) renderRule(rule rules.Rule) string {
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

	// Examples
	examples := rule.Examples()
	if examples.HasExamples() {
		sb.WriteString(g.renderExamples(examples))
	}

	// Separator
	sb.WriteString("---\n\n")

	return sb.String()
}

// renderExamples renders the valid and invalid examples for a rule.
func (g *RuleDocGenerator) renderExamples(examples rules.Examples) string {
	var sb strings.Builder

	if len(examples.Valid) > 0 {
		sb.WriteString("**Valid Examples:**\n\n")
		for _, example := range examples.Valid {
			sb.WriteString("```yaml\n")
			sb.WriteString(strings.TrimSpace(example))
			sb.WriteString("\n```\n\n")
		}
	}

	if len(examples.Invalid) > 0 {
		sb.WriteString("**Invalid Examples:**\n\n")
		for _, example := range examples.Invalid {
			sb.WriteString("```yaml\n")
			sb.WriteString(strings.TrimSpace(example))
			sb.WriteString("\n```\n\n")
		}
	}

	return sb.String()
}

// severityBadge returns a shields.io badge for the severity level.
func (g *RuleDocGenerator) severityBadge(severity rules.Severity) string {
	var color string
	switch severity {
	case rules.Error:
		color = "red"
	case rules.Warning:
		color = "yellow"
	case rules.Info:
		color = "blue"
	default:
		color = "lightgrey"
	}

	severityStr := severity.String()
	return fmt.Sprintf("![Severity: %s](https://img.shields.io/badge/severity-%s-%s)",
		strings.ToUpper(severityStr[:1])+severityStr[1:],
		severityStr,
		color,
	)
}
