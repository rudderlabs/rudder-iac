package docs

import (
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// Generate joins the live validation rule registry (passed as flat syntactic
// and semantic rule slices) with authored YAML doc entries and produces a
// validated DocumentedRules.
//
// It returns BOTH the built doc and the validation errors so the caller can
// inspect the doc even when validation fails (e.g. during the skeleton phase
// when some rules lack authored entries).
func Generate(syntactic, semantic []rules.Rule, entries []RuleDocEntry, cliVersion, generatedAt string) (DocumentedRules, []error) {
	entryByID := make(map[string]RuleDocEntry, len(entries))
	for _, e := range entries {
		entryByID[e.RuleID] = e
	}

	resolved := make([]DocumentedRule, 0, len(syntactic)+len(semantic))
	registeredRuleIDs := make([]string, 0, len(syntactic)+len(semantic))

	for _, r := range syntactic {
		resolved = append(resolved, resolveRule(r, "syntactic", entryByID))
		registeredRuleIDs = append(registeredRuleIDs, r.ID())
	}

	for _, r := range semantic {
		resolved = append(resolved, resolveRule(r, "semantic", entryByID))
		registeredRuleIDs = append(registeredRuleIDs, r.ID())
	}

	sort.Slice(resolved, func(i, j int) bool {
		return resolved[i].RuleID < resolved[j].RuleID
	})

	doc := DocumentedRules{
		SchemaVersion: 1,
		ToolMetadata: ToolMetadata{
			CLIVersion:  cliVersion,
			GeneratedAt: generatedAt,
		},
		Rules: resolved,
	}

	return doc, doc.Validate(registeredRuleIDs)
}

func resolveRule(r rules.Rule, phase string, entryByID map[string]RuleDocEntry) DocumentedRule {
	ruleID := r.ID()

	appliesTo := make([]MatchPatternDoc, 0, len(r.AppliesTo()))
	for _, p := range r.AppliesTo() {
		appliesTo = append(appliesTo, MatchPatternDoc{Kind: p.Kind, Version: p.Version})
	}

	// MatchBehavior stays nil when no authored entry exists — that's expected
	// during the skeleton phase and surfaces as a validation error downstream.
	var matchBehavior []MatchBehaviorEntry
	if entry, ok := entryByID[ruleID]; ok {
		matchBehavior = entry.MatchBehavior
	}

	return DocumentedRule{
		RuleID:        ruleID,
		Provider:      providerFromRuleID(ruleID),
		Phase:         phase,
		Severity:      r.Severity().String(),
		Description:   r.Description(),
		AppliesTo:     appliesTo,
		MatchBehavior: matchBehavior,
	}
}

// providerFromRuleID extracts the first "/"-separated segment of a rule ID,
// which by convention names the owning provider (e.g.
// "datacatalog/categories/spec-syntax-valid" -> "datacatalog").
func providerFromRuleID(ruleID string) string {
	if idx := strings.Index(ruleID, "/"); idx >= 0 {
		return ruleID[:idx]
	}
	return ruleID
}
