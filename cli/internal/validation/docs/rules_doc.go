package docs

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

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
	// TODO(spike DEX-371): re-enable after pilot phase. Disabled because
	// only 3 of ~13 registered rules currently have docs — see spec §3 Layer 3.
	// errs = append(errs, c.validateRegisteredCompleteness(expectedRuleIDs)...)
	_ = expectedRuleIDs // suppress unused-parameter lint until restoration
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

// docsTagNameFunc reports field names using the YAML or JSON tag so validation
// error messages reference user-facing keys instead of Go struct field names.
var docsTagNameFunc = func(fld reflect.StructField) string {
	if t, ok := fld.Tag.Lookup("yaml"); ok {
		return strings.SplitN(t, ",", 2)[0]
	}
	if t, ok := fld.Tag.Lookup("json"); ok {
		return strings.SplitN(t, ",", 2)[0]
	}
	return strings.ToLower(fld.Name)
}

func validateRuleStruct(rule *ResolvedRule) []error {
	v := validator.New()
	v.RegisterTagNameFunc(docsTagNameFunc)

	err := v.Struct(rule)
	if err == nil {
		return nil
	}

	var invalid *validator.InvalidValidationError
	if errors.As(err, &invalid) {
		return []error{fmt.Errorf("rule %s: structural validation failed: %w", rule.RuleID, err)}
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		return []error{fmt.Errorf("rule %s: structural validation failed: %w", rule.RuleID, err)}
	}

	var errs []error
	for _, fe := range validationErrs {
		errs = append(errs, fmt.Errorf("rule %q: field %s failed validation %q", rule.RuleID, fe.Field(), fe.Tag()))
	}
	return errs
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
