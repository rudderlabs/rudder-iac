package docs

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// GeneratorOptions configures the Generator at construction.
type GeneratorOptions struct {
	CLIVersion    string
	SchemaVersion int
}

// Generator builds a RulesDoc from a registry + resolver. The resolver
// supplies authored data; the generator enriches each entry with metadata
// pulled from the rule itself.
type Generator struct {
	registry rules.Registry
	resolver Resolver
	opts     GeneratorOptions
}

func NewGenerator(reg rules.Registry, resolver Resolver, opts GeneratorOptions) *Generator {
	return &Generator{registry: reg, resolver: resolver, opts: opts}
}

// Generate walks both syntactic and semantic rule lists, resolves each one,
// and assembles a *RulesDoc. Rules without authored docs are skipped silently —
// the spike has only 3 pilots, full coverage is enforced later via
// validateRegisteredCompleteness (currently gated, see rules_doc.go).
func (g *Generator) Generate() (*RulesDoc, error) {
	resolved := make([]ResolvedRule, 0)

	syn, err := g.resolvePhase(g.registry.AllSyntacticRules(), "syntactic")
	if err != nil {
		return nil, err
	}
	resolved = append(resolved, syn...)

	sem, err := g.resolvePhase(g.registry.AllSemanticRules(), "semantic")
	if err != nil {
		return nil, err
	}
	resolved = append(resolved, sem...)

	return &RulesDoc{
		SchemaVersion: g.opts.SchemaVersion,
		ToolMetadata: ToolMetadata{
			CLIVersion: g.opts.CLIVersion,
		},
		Rules: resolved,
	}, nil
}

func (g *Generator) resolvePhase(ruleList []rules.Rule, phase string) ([]ResolvedRule, error) {
	out := make([]ResolvedRule, 0, len(ruleList))
	for _, r := range ruleList {
		entry, err := g.resolver.ResolveFor(r)
		if err != nil {
			return nil, fmt.Errorf("resolving docs for rule %s: %w", r.ID(), err)
		}
		if entry == nil {
			continue
		}
		out = append(out, ResolvedRule{
			RuleID:        r.ID(),
			Phase:         phase,
			Severity:      r.Severity().String(),
			Description:   r.Description(),
			AppliesTo:     matchPatternsToDocs(r.AppliesTo()),
			MatchBehavior: entry.MatchBehavior,
		})
	}
	return out, nil
}

// matchPatternsToDocs converts the rule's runtime MatchPattern slice into
// the YAML-tagged MatchPatternDoc used in the catalog.
func matchPatternsToDocs(in []rules.MatchPattern) []MatchPatternDoc {
	out := make([]MatchPatternDoc, len(in))
	for i, p := range in {
		out[i] = MatchPatternDoc{Kind: p.Kind, Version: p.Version}
	}
	return out
}
