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
// Wildcard rules (AppliesTo: ["*"]) are stored under the "*" key.
type defaultRegistry struct {
	// syntacticRulesByKind maps kind to syntactic rules
	// Wildcard rules are stored under the "*" key
	syntacticRulesByKind map[string][]Rule

	// semanticRulesByKind maps kind to semantic rules
	// Wildcard rules are stored under the "*" key
	semanticRulesByKind map[string][]Rule
}

// NewRegistry creates a new empty rule registry.
func NewRegistry() Registry {
	return &defaultRegistry{
		syntacticRulesByKind: make(map[string][]Rule),
		semanticRulesByKind:  make(map[string][]Rule),
	}
}

// RegisterSyntactic registers a syntactic rule.
// Returns ErrDuplicateRule if a rule with the same ID already exists.
func (r *defaultRegistry) RegisterSyntactic(rule Rule) error {
	ruleID := rule.ID()

	// Check for duplicates across both syntactic and semantic rules
	if r.isDuplicateRule(ruleID, r.syntacticRulesByKind) {
		return fmt.Errorf("%w: %s", ErrDuplicateRule, ruleID)
	}
	if r.isDuplicateRule(ruleID, r.semanticRulesByKind) {
		return fmt.Errorf("%w: %s (already registered as semantic)", ErrDuplicateRule, ruleID)
	}

	kinds := rule.AppliesTo()

	if lo.Contains(kinds, "*") {
		r.syntacticRulesByKind["*"] = append(r.syntacticRulesByKind["*"], rule)
	} else {
		for _, kind := range lo.Uniq(kinds) {
			r.syntacticRulesByKind[kind] = append(r.syntacticRulesByKind[kind], rule)
		}
	}

	return nil
}

// RegisterSemantic registers a semantic rule.
// Returns ErrDuplicateRule if a rule with the same ID already exists.
func (r *defaultRegistry) RegisterSemantic(rule Rule) error {
	ruleID := rule.ID()

	// Check for duplicates across both semantic and syntactic rules
	if r.isDuplicateRule(ruleID, r.semanticRulesByKind) {
		return fmt.Errorf("%w: %s", ErrDuplicateRule, ruleID)
	}
	if r.isDuplicateRule(ruleID, r.syntacticRulesByKind) {
		return fmt.Errorf("%w: %s (already registered as syntactic)", ErrDuplicateRule, ruleID)
	}

	kinds := rule.AppliesTo()

	if lo.Contains(kinds, "*") {
		r.semanticRulesByKind["*"] = append(r.semanticRulesByKind["*"], rule)
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
	// Combine kind-specific rules with wildcard rules (stored under "*")
	rules := make([]Rule, 0)
	rules = append(rules, r.syntacticRulesByKind[kind]...)
	rules = append(rules, r.syntacticRulesByKind["*"]...)

	return rules
}

// SemanticRulesForKind returns all semantic rules applicable to the given kind.
// This includes both kind-specific rules and wildcard rules.
func (r *defaultRegistry) SemanticRulesForKind(kind string) []Rule {
	// Combine kind-specific rules with wildcard rules (stored under "*")
	rules := make([]Rule, 0)
	rules = append(rules, r.semanticRulesByKind[kind]...)
	rules = append(rules, r.semanticRulesByKind["*"]...)

	return rules
}

// isDuplicateRule checks if a rule with the given ID already exists in the provided rule map.
// It iterates through all kinds in the map to find any rule with a matching ID.
func (r *defaultRegistry) isDuplicateRule(ruleID string, rulesByKind map[string][]Rule) bool {
	for _, rules := range rulesByKind {
		for _, rule := range rules {
			if rule.ID() == ruleID {
				return true
			}
		}
	}
	return false
}
