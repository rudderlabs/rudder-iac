package docs

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const schemaVersion = 1

// Generator walks a registry and produces a structurally valid RulesDoc.
// The verifier is a separate step (see verifier.go) — Generator does NOT
// execute examples against the validation engine.
type Generator struct {
	resolver   Resolver
	cliVersion string
}

func NewGenerator(resolver Resolver, cliVersion string) *Generator {
	return &Generator{resolver: resolver, cliVersion: cliVersion}
}

// Generate walks the registry, resolves authored docs per rule, enriches
// each ResolvedRule with metadata from the rule itself, runs structural
// validation, and returns the populated RulesDoc.
func (g *Generator) Generate(reg rules.Registry) (*RulesDoc, error) {
	doc := &RulesDoc{
		SchemaVersion: schemaVersion,
		ToolMetadata:  ToolMetadata{CLIVersion: g.cliVersion},
	}

	if err := g.appendResolved(doc, reg.AllSyntacticRules(), "syntactic"); err != nil {
		return nil, err
	}
	if err := g.appendResolved(doc, reg.AllSemanticRules(), "semantic"); err != nil {
		return nil, err
	}

	if errs := doc.Validate(nil); len(errs) > 0 {
		return nil, fmt.Errorf("structural validation failed: %v", errs)
	}
	return doc, nil
}

func (g *Generator) appendResolved(doc *RulesDoc, ruleSet []rules.Rule, phase string) error {
	for _, r := range ruleSet {
		resolved, err := g.resolver.ResolveFor(r)
		if err != nil {
			return fmt.Errorf("resolving docs for rule %s: %w", r.ID(), err)
		}
		if resolved == nil {
			continue
		}
		resolved.RuleID = r.ID()
		resolved.Phase = phase
		resolved.Severity = r.Severity().String()
		resolved.Description = r.Description()
		resolved.AppliesTo = patternsToDocs(r.AppliesTo())
		doc.Rules = append(doc.Rules, *resolved)
	}
	return nil
}

func patternsToDocs(ps []rules.MatchPattern) []MatchPatternDoc {
	out := make([]MatchPatternDoc, len(ps))
	for i, p := range ps {
		out[i] = MatchPatternDoc{Kind: p.Kind, Version: p.Version}
	}
	return out
}
