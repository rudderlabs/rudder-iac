package docs

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// docValidator is local to the docs package — instantiating here is what
// dissolves the docs → rules import dep that PR #471 introduced. The leaf
// rule packages can now import docs for MatchBehaviorEntry without an
// import cycle.
var docValidator = func() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.ToLower(fld.Name)
		if t, ok := fld.Tag.Lookup("yaml"); ok {
			name = strings.SplitN(t, ",", 2)[0]
		}
		return name
	})
	return v
}()

func (c *RulesDoc) Validate(expectedRuleIDs []string) []error {
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
	// TODO(spike DEX-370): re-enable after pilot phase. Disabled because the
	// spike only documents 3 of ~13 registered rules.
	// errs = append(errs, c.validateRegisteredCompleteness(expectedRuleIDs)...)
	_ = expectedRuleIDs
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
	if err := docValidator.Struct(rule); err != nil {
		var verrs validator.ValidationErrors
		if !errors.As(err, &verrs) {
			return []error{fmt.Errorf("rule %s: structural validation failed: %w", rule.RuleID, err)}
		}
		out := make([]error, 0, len(verrs))
		for _, fe := range verrs {
			out = append(out, fmt.Errorf("rule %q: field %s failed validation %q", rule.RuleID, fe.Field(), fe.Tag()))
		}
		return out
	}
	return nil
}

func (c *RulesDoc) validateRegisteredCompleteness(expectedRuleIDs []string) []error {
	catalogSet := make(map[string]struct{}, len(c.Rules))
	for _, r := range c.Rules {
		catalogSet[r.RuleID] = struct{}{}
	}

	expectedSet := make(map[string]struct{}, len(expectedRuleIDs))
	for _, id := range expectedRuleIDs {
		expectedSet[id] = struct{}{}
	}

	var errs []error
	for id := range expectedSet {
		if _, ok := catalogSet[id]; !ok {
			errs = append(errs, fmt.Errorf("rule %s is registered but has no doc entry", id))
		}
	}
	for id := range catalogSet {
		if _, ok := expectedSet[id]; !ok {
			errs = append(errs, fmt.Errorf("rule %s has a doc entry but is not registered (orphan)", id))
		}
	}
	return errs
}
