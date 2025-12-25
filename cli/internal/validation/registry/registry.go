package registry

import (
	"fmt"
	"log"
	"sync"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
)

// RuleRegistry manages validation rules indexed by resource kind
type RuleRegistry struct {
	mu    sync.RWMutex
	rules map[string][]validation.Rule
}

// NewRegistry creates a new rule registry
func NewRegistry() *RuleRegistry {
	return &RuleRegistry{
		rules: make(map[string][]validation.Rule),
	}
}

// Register adds a rule to the registry for the kinds it applies to
func (r *RuleRegistry) Register(rule validation.Rule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate rule ID in all kinds
	for _, rules := range r.rules {
		for _, existingRule := range rules {
			if existingRule.ID() == rule.ID() {
				return fmt.Errorf("rule with ID %s is already registered", rule.ID())
			}
		}
	}

	kinds := rule.AppliesTo()
	if len(kinds) == 0 {
		return fmt.Errorf("rule %s must apply to at least one kind", rule.ID())
	}

	for _, kind := range kinds {
		r.rules[kind] = append(r.rules[kind], rule)
	}

	log.Printf("[registry] Registered rule %s for kinds %v", rule.ID(), kinds)

	return nil
}

// RulesForKind returns all rules registered for a specific kind
func (r *RuleRegistry) RulesForKind(kind string) []validation.Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rules := r.rules[kind]
	if rules == nil {
		return []validation.Rule{}
	}

	// Return a copy to avoid external modifications
	cp := make([]validation.Rule, len(rules))
	copy(cp, rules)
	return cp
}

// AllRules returns all unique rules registered in the registry
func (r *RuleRegistry) AllRules() []validation.Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	uniqueRules := make(map[string]validation.Rule)
	for _, rules := range r.rules {
		for _, rule := range rules {
			uniqueRules[rule.ID()] = rule
		}
	}

	all := make([]validation.Rule, 0, len(uniqueRules))
	for _, rule := range uniqueRules {
		all = append(all, rule)
	}
	return all
}
