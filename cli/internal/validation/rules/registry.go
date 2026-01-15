package rules

import (
	"fmt"

	"github.com/samber/lo"
)

var (
	// ErrDuplicateRule is returned when attempting to register a rule with an ID that already exists
	ErrDuplicateRule = fmt.Errorf("rule with id already registered")

	ErrDuplicateKinds = fmt.Errorf("duplicate kinds found in a rule")

	// ErrNoRuleForKind is returned when no rules are found for a given kind
	ErrNoRuleForKind = fmt.Errorf("no rule found for kind")
)

// Registry manages validation rules and provides lookup by kind and validation phase.
// The registry maintains separate collections for syntactic rules (pre-graph)
// and semantic rules (post-graph), enabling efficient rule execution during validation.
//
// Rules are indexed by kind for O(1) lookup, and wildcard rules (AppliesTo: ["*"])
// are automatically included for all kinds.
type Registry interface {
	// RegisterSyntactic registers a rule that runs before resource graph construction.
	// Syntactic rules validate spec structure, required fields, format, etc.
	// Returns an error if a rule with the same ID is already registered.
	RegisterSyntactic(rule Rule) error

	// RegisterSemantic registers a rule that runs after resource graph construction.
	// Semantic rules validate cross-resource references, dependencies, business logic, etc.
	// Returns an error if a rule with the same ID is already registered.
	RegisterSemantic(rule Rule) error

	// SyntacticRulesForKind returns all syntactic rules applicable to the given kind.
	// This includes both kind-specific rules and rules with wildcard "*".
	SyntacticRulesForKind(kind string) []Rule

	// SemanticRulesForKind returns all semantic rules applicable to the given kind.
	// This includes both kind-specific rules and rules with wildcard "*".
	SemanticRulesForKind(kind string) []Rule
}

// defaultRegistry is the concrete implementation of Registry.
// It maintains separate indices for syntactic and semantic rules,
// and pre-computes rule lookups by kind for efficient access.
type defaultRegistry struct {
	// syntacticRules maps rule ID to syntactic rules
	syntacticRules map[string]Rule

	// semanticRules maps rule ID to semantic rules
	semanticRules map[string]Rule

	// syntacticRulesByKind caches rules by kind for fast lookup
	syntacticRulesByKind map[string][]Rule

	// semanticRulesByKind caches rules by kind for fast lookup
	semanticRulesByKind map[string][]Rule

	// wildcardSyntacticRules stores rules that apply to all kinds
	wildcardSyntacticRules []Rule

	// wildcardSemanticRules stores rules that apply to all kinds
	wildcardSemanticRules []Rule
}

// NewRegistry creates a new empty rule registry.
func NewRegistry() Registry {
	return &defaultRegistry{
		syntacticRules:         make(map[string]Rule),
		semanticRules:          make(map[string]Rule),
		syntacticRulesByKind:   make(map[string][]Rule),
		semanticRulesByKind:    make(map[string][]Rule),
		wildcardSyntacticRules: make([]Rule, 0),
		wildcardSemanticRules:  make([]Rule, 0),
	}
}

// RegisterSyntactic registers a syntactic rule.
// Returns ErrDuplicateRule if a rule with the same ID already exists.
func (r *defaultRegistry) RegisterSyntactic(rule Rule) error {
	ruleID := rule.ID()

	// Check for duplicates across both phases
	if _, exists := r.syntacticRules[ruleID]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateRule, ruleID)
	}
	if _, exists := r.semanticRules[ruleID]; exists {
		return fmt.Errorf("%w: %s (already registered as semantic)", ErrDuplicateRule, ruleID)
	}

	r.syntacticRules[ruleID] = rule

	kinds := rule.AppliesTo()

	if lo.Contains(kinds, "*") {
		r.wildcardSyntacticRules = append(r.wildcardSyntacticRules, rule)
	} else {
		for _, kind := range lo.Uniq(kinds) {
			// Filter out duplicate kinds if any
			r.syntacticRulesByKind[kind] = append(r.syntacticRulesByKind[kind], rule)
		}
	}

	return nil
}

// RegisterSemantic registers a semantic rule.
// Returns ErrDuplicateRule if a rule with the same ID already exists.
func (r *defaultRegistry) RegisterSemantic(rule Rule) error {
	ruleID := rule.ID()

	// Check for duplicates across both phases
	if _, exists := r.semanticRules[ruleID]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateRule, ruleID)
	}
	if _, exists := r.syntacticRules[ruleID]; exists {
		return fmt.Errorf("%w: %s (already registered as syntactic)", ErrDuplicateRule, ruleID)
	}

	r.semanticRules[ruleID] = rule

	kinds := rule.AppliesTo()

	if lo.Contains(kinds, "*") {
		r.wildcardSemanticRules = append(r.wildcardSemanticRules, rule)
	} else {
		for _, kind := range lo.Uniq(kinds) {
			r.semanticRulesByKind[kind] = append(r.semanticRulesByKind[kind], rule)
		}
	}

	return nil
}

// SyntacticRulesForKind returns all syntactic rules applicable to the given kind.
// This includes both kind-specific rules and wildcard rules.
func (r *defaultRegistry) SyntacticRulesForKind(kind string) []Rule {
	// Combine kind-specific rules with wildcard rules
	rules := make([]Rule, 0)
	rules = append(rules, r.syntacticRulesByKind[kind]...)
	rules = append(rules, r.wildcardSyntacticRules...)

	return rules
}

// SemanticRulesForKind returns all semantic rules applicable to the given kind.
// This includes both kind-specific rules and wildcard rules.
func (r *defaultRegistry) SemanticRulesForKind(kind string) []Rule {
	// Combine kind-specific rules with wildcard rules
	rules := make([]Rule, 0)
	rules = append(rules, r.semanticRulesByKind[kind]...)
	rules = append(rules, r.wildcardSemanticRules...)

	return rules
}
