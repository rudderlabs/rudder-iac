package docs

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func (c *DocumentedRules) Validate(registeredRuleIDs []string) []error {
	var errs []error
	for i := range c.Rules {
		structErrs := validateRuleStruct(&c.Rules[i])

		errs = append(errs, structErrs...)
		if len(structErrs) > 0 {
			// Skip per-rule checks when structure is
			// invalid — guard clause per spec.
			continue
		}

		errs = append(errs, validateAppliesToCoverage(&c.Rules[i])...)
		errs = append(errs, validateUniqueExampleIDs(&c.Rules[i])...)
	}
	errs = append(errs, c.validateRegisteredCompleteness(registeredRuleIDs)...)
	return errs
}

func validateUniqueExampleIDs(rule *DocumentedRule) []error {
	seen := make(map[string]struct{})
	var errs []error
	for _, mb := range rule.MatchBehavior {
		for _, ex := range mb.Valid {
			if _, ok := seen[ex.ExampleID]; ok {
				errs = append(errs, fmt.Errorf("rule %s: duplicate example_id %q", rule.RuleID, ex.ExampleID))
			}
			seen[ex.ExampleID] = struct{}{}
		}
		for _, ex := range mb.Invalid {
			if _, ok := seen[ex.ExampleID]; ok {
				errs = append(errs, fmt.Errorf("rule %s: duplicate example_id %q", rule.RuleID, ex.ExampleID))
			}
			seen[ex.ExampleID] = struct{}{}
		}
	}
	return errs
}

// validateAppliesToCoverage enforces a bidirectional, wildcard-aware
// equivalence between the rule's real AppliesTo() (the "code" side, serialized
// by the generator) and the union of authored match_behavior.applies_to (the
// "docs" side). Both directions guard against docs↔code drift (DEX-406):
//
//   - code ⊆ docs: every (kind, version) the rule actually matches must be
//     documented by some authored match_behavior — otherwise a rule that grows
//     a new kind/version goes undocumented.
//   - docs ⊆ code: every authored (kind, version) must be matched by the rule —
//     otherwise a stale/over-declared fragment (referencing a kind/version the
//     rule no longer matches) passes silently.
//
// Coverage uses pattern-subset semantics rather than string equality so "*"
// wildcards (e.g. MatchAll gatekeeper rules) are handled consistently in both
// directions.
func validateAppliesToCoverage(rule *DocumentedRule) []error {
	var authored []MatchPatternDoc
	for _, mb := range rule.MatchBehavior {
		authored = append(authored, mb.AppliesTo...)
	}

	var errs []error

	// code ⊆ docs: each rule pattern must be contained in some authored pattern.
	for _, c := range rule.AppliesTo {
		if !coveredBy(c, authored) {
			errs = append(errs, fmt.Errorf("rule %s: applies_to entry {kind: %s, version: %s} has no matching match_behavior coverage", rule.RuleID, c.Kind, c.Version))
		}
	}

	// docs ⊆ code: each authored pattern must be contained in some rule pattern.
	for _, a := range authored {
		if !coveredBy(a, rule.AppliesTo) {
			errs = append(errs, fmt.Errorf("rule %s: match_behavior applies_to entry {kind: %s, version: %s} is not covered by the rule's AppliesTo() (stale or over-declared)", rule.RuleID, a.Kind, a.Version))
		}
	}

	return errs
}

// coveredBy reports whether pattern p is a subset of at least one pattern in
// set — i.e. every concrete (kind, version) that p would match is also matched
// by some pattern in set.
func coveredBy(p MatchPatternDoc, set []MatchPatternDoc) bool {
	for _, q := range set {
		if patternSubset(p, q) {
			return true
		}
	}
	return false
}

// patternSubset reports whether every concrete (kind, version) matched by a is
// also matched by b. A "*" on b widens b's coverage; a "*" on a widens what a
// must have covered, so a wildcard in a is only a subset of a correspondingly
// wildcarded b.
func patternSubset(a, b MatchPatternDoc) bool {
	kindOK := b.Kind == "*" || (a.Kind != "*" && a.Kind == b.Kind)
	versionOK := b.Version == "*" || (a.Version != "*" && a.Version == b.Version)
	return kindOK && versionOK
}

func validateRuleStruct(rule *DocumentedRule) []error {
	validationErrs, err := rules.ValidateStruct(rule, "")
	if err != nil {
		return []error{fmt.Errorf("rule %s: structural validation failed: %w", rule.RuleID, err)}
	}

	var errs []error
	for _, fe := range validationErrs {
		errs = append(errs, fmt.Errorf("rule %q: field %s failed validation %q", rule.RuleID, fe.Field(), fe.Tag()))
	}
	return errs
}

func (c *DocumentedRules) validateRegisteredCompleteness(registeredRuleIDs []string) []error {
	catalogSet := make(map[string]struct{}, len(c.Rules))
	for _, r := range c.Rules {
		catalogSet[r.RuleID] = struct{}{}
	}

	registeredSet := make(map[string]struct{}, len(registeredRuleIDs))
	for _, id := range registeredRuleIDs {
		registeredSet[id] = struct{}{}
	}

	var errs []error
	for id := range registeredSet {
		if _, ok := catalogSet[id]; !ok {
			errs = append(errs, fmt.Errorf("rule %s is registered but has no doc entry", id))
		}
	}
	for id := range catalogSet {
		if _, ok := registeredSet[id]; !ok {
			errs = append(errs, fmt.Errorf("rule %s has a doc entry but is not registered (orphan)", id))
		}
	}
	return errs
}
