package docs

import (
	"fmt"
	"io/fs"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// Resolver returns the authored doc data for a given rule.
// A nil return (with nil error) means no docs are authored for the rule.
// An error indicates an authoring or load problem, not missing docs.
type Resolver interface {
	ResolveFor(r rules.Rule) (*RuleDocEntry, error)
}

// YAMLResolver loads all YAML fragments from a directory at construction
// and indexes them by rule_id. ResolveFor is a map lookup.
type YAMLResolver struct {
	byRuleID map[string]RuleDocEntry
}

// NewYAMLResolver reads every .yaml/.yml file under dir on fsys and builds
// the rule_id index. Returns an error if any fragment fails to parse or
// two fragments declare the same rule_id.
func NewYAMLResolver(fsys fs.FS, dir string) (*YAMLResolver, error) {
	entries, err := LoadRuleDocEntries(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("loading rule doc fragments: %w", err)
	}

	byID := make(map[string]RuleDocEntry, len(entries))
	for _, e := range entries {
		if _, exists := byID[e.RuleID]; exists {
			return nil, fmt.Errorf("duplicate rule_id %q across fragments", e.RuleID)
		}
		byID[e.RuleID] = e
	}

	return &YAMLResolver{byRuleID: byID}, nil
}

func (r *YAMLResolver) ResolveFor(rule rules.Rule) (*RuleDocEntry, error) {
	entry, ok := r.byRuleID[rule.ID()]
	if !ok {
		return nil, nil
	}
	out := entry
	return &out, nil
}
