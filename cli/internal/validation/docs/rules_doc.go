package docs

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func (c *RulesDoc) Validate(registeredRuleIDs []string) []error {
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

func validateUniqueExampleIDs(rule *ResolvedRule) []error {
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

func validateAppliesToCoverage(rule *ResolvedRule) []error {
	covered := make(map[string]struct{})
	for _, mb := range rule.MatchBehavior {
		for _, p := range mb.AppliesTo {
			covered[p.Kind+":"+p.Version] = struct{}{}
		}
	}

	var errs []error
	for _, p := range rule.AppliesTo {
		key := p.Kind + ":" + p.Version
		if _, ok := covered[key]; !ok {
			errs = append(errs, fmt.Errorf("rule %s: applies_to entry {kind: %s, version: %s} has no matching match_behavior coverage", rule.RuleID, p.Kind, p.Version))
		}
	}
	return errs
}

func validateRuleStruct(rule *ResolvedRule) []error {
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

func (c *RulesDoc) validateRegisteredCompleteness(registeredRuleIDs []string) []error {
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
