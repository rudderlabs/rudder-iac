package docs

import "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"

// Resolver is the only seam between Path A and Path B spike strategies.
// It returns the authored documentation data for a given rule, or
// (nil, nil) when the rule has no docs authored. Errors are reserved
// for authoring/load problems.
type Resolver interface {
	ResolveFor(r rules.Rule) (*ResolvedRule, error)
}

// ExamplesResolver resolves docs by type-asserting rules to the
// Documented interface. Rules without it produce no entry.
type ExamplesResolver struct{}

func (ExamplesResolver) ResolveFor(r rules.Rule) (*ResolvedRule, error) {
	d, ok := r.(Documented)
	if !ok {
		return nil, nil
	}
	entries := d.DocExamples()
	if len(entries) == 0 {
		return nil, nil
	}
	return &ResolvedRule{MatchBehavior: entries}, nil
}
