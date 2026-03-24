package rules

import "fmt"

// Registry manages validation rules and provides lookup by (kind, version) and phase.
// Rules are stored in flat lists per phase (syntactic/semantic) and matched
// against specs using their AppliesTo() patterns at query time.
type Registry interface {
	// RegisterSyntactic adds a rule that runs before resource graph construction.
	RegisterSyntactic(rule Rule) error

	// RegisterSemantic adds a rule that runs after resource graph construction.
	RegisterSemantic(rule Rule) error

	// SyntacticRulesFor returns syntactic rules whose AppliesTo() patterns
	// match the given (kind, version).
	SyntacticRulesFor(kind, version string) []Rule

	// SemanticRulesFor returns semantic rules whose AppliesTo() patterns
	// match the given (kind, version).
	SemanticRulesFor(kind, version string) []Rule

	// AllSyntacticRules returns every registered syntactic rule regardless of pattern.
	// Used to discover ProjectRule implementations which are not limited to MatchAll().
	AllSyntacticRules() []Rule
}

// defaultRegistry stores rules in flat slices per phase.
// Matching is done at query time by checking each rule's AppliesTo() patterns.
//
// Registration validates AppliesTo() against supportedPatterns: non-wildcard kind/version
// must appear among provider-declared pairs. Fully concrete patterns must match an exact
// supported pair. Wildcard-only rules (e.g. MatchKind) are not proven to only hit supported
// combos at runtime — see project/resource-kind-version-valid and gatekeeper rules.
type defaultRegistry struct {
	syntactic         []Rule
	semantic          []Rule
	supportedPatterns []MatchPattern
	uniqueKinds       map[string]struct{}
	uniqueVersions    map[string]struct{}
}

// NewRegistry creates a new empty rule registry.
// supportedPatterns is the aggregated set of (kind, version) pairs from all providers,
// used to validate rule AppliesTo() patterns at registration time.
func NewRegistry(supportedPatterns []MatchPattern) Registry {
	uniqueKinds := make(map[string]struct{})
	uniqueVersions := make(map[string]struct{})
	for _, p := range supportedPatterns {
		if p.Kind != "*" {
			uniqueKinds[p.Kind] = struct{}{}
		}
		if p.Version != "*" {
			uniqueVersions[p.Version] = struct{}{}
		}
	}
	return &defaultRegistry{
		supportedPatterns: supportedPatterns,
		uniqueKinds:       uniqueKinds,
		uniqueVersions:    uniqueVersions,
	}
}

func (r *defaultRegistry) RegisterSyntactic(rule Rule) error {
	if err := r.validateAppliesTo(rule); err != nil {
		return err
	}
	r.syntactic = append(r.syntactic, rule)
	return nil
}

func (r *defaultRegistry) RegisterSemantic(rule Rule) error {
	if err := r.validateAppliesTo(rule); err != nil {
		return err
	}
	r.semantic = append(r.semantic, rule)
	return nil
}

func (r *defaultRegistry) validateAppliesTo(rule Rule) error {
	patterns := rule.AppliesTo()
	if len(patterns) == 0 {
		return nil
	}

	if len(r.supportedPatterns) == 0 {
		for _, p := range patterns {
			if p.Kind != "*" || p.Version != "*" {
				return fmt.Errorf(
					"register rule %q: AppliesTo pattern {kind:%q version:%q} requires non-empty supported match patterns",
					rule.ID(), p.Kind, p.Version,
				)
			}
		}
		return nil
	}

	for _, p := range patterns {
		if p.Kind != "*" {
			if _, ok := r.uniqueKinds[p.Kind]; !ok {
				return fmt.Errorf("register rule %q: kind %q is not in supported match patterns", rule.ID(), p.Kind)
			}
		}
		if p.Version != "*" {
			if _, ok := r.uniqueVersions[p.Version]; !ok {
				return fmt.Errorf("register rule %q: version %q is not in supported match patterns", rule.ID(), p.Version)
			}
		}
		if p.Kind != "*" && p.Version != "*" {
			if !r.hasExactSupportedPair(p.Kind, p.Version) {
				return fmt.Errorf(
					"register rule %q: (kind,version) (%q,%q) is not a supported match pattern",
					rule.ID(), p.Kind, p.Version,
				)
			}
		}
	}
	return nil
}

func (r *defaultRegistry) hasExactSupportedPair(kind, version string) bool {
	for _, sp := range r.supportedPatterns {
		if sp.Kind == kind && sp.Version == version {
			return true
		}
	}
	return false
}

func (r *defaultRegistry) SyntacticRulesFor(kind, version string) []Rule {
	return matchRules(r.syntactic, kind, version)
}

func (r *defaultRegistry) SemanticRulesFor(kind, version string) []Rule {
	return matchRules(r.semantic, kind, version)
}

func (r *defaultRegistry) AllSyntacticRules() []Rule {
	return r.syntactic
}

// matchRules returns rules that have at least one AppliesTo() pattern
// matching the given (kind, version).
func matchRules(rules []Rule, kind, version string) []Rule {
	var matched []Rule
	for _, rule := range rules {
		for _, p := range rule.AppliesTo() {
			if p.Matches(kind, version) {
				matched = append(matched, rule)
				break
			}
		}
	}
	return matched
}
