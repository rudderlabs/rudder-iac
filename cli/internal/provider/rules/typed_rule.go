package rules

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var _ rules.Rule = &typedRule{}

type typedRule struct {
	id          string
	severity    rules.Severity
	description string
	examples    rules.Examples
	variants    []Variant
}

// NewTypedRule creates a validation rule composed of one or more Variants.
// Each variant handles a subset of (Kind, Version) patterns and may
// deserialize into a different spec type. During validation the first
// variant whose patterns match the context is invoked.
func NewTypedRule(
	id string,
	severity rules.Severity,
	description string,
	examples rules.Examples,
	variants ...Variant,
) rules.Rule {
	return &typedRule{
		id:          id,
		severity:    severity,
		description: description,
		examples:    examples,
		variants:    variants,
	}
}

func (r *typedRule) ID() string               { return r.id }
func (r *typedRule) Severity() rules.Severity  { return r.severity }
func (r *typedRule) Description() string       { return r.description }
func (r *typedRule) Examples() rules.Examples   { return r.examples }

func (r *typedRule) AppliesTo() []rules.MatchPattern {
	var patterns []rules.MatchPattern
	for _, v := range r.variants {
		patterns = append(patterns, v.patterns...)
	}
	return patterns
}

// Validate dispatches to the first variant whose patterns match the
// context's (Kind, Version). Returns nil when no variant matches.
func (r *typedRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	for _, v := range r.variants {
		for _, p := range v.patterns {
			if p.Matches(ctx.Kind, ctx.Version) {
				return v.validate(ctx)
			}
		}
	}
	return nil
}
