package destination

import (
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// maskSecretLines wraps each rule so that every result it returns is marked
// Sensitive when its line in a destination spec can show a secret value. The
// provider wraps every rule it registers this way, so a rule does not decide
// on its own whether its results are sensitive, and a new rule cannot print a
// secret by leaving the flag out. A rule may still set Sensitive on a result
// the check here passes over.
//
// The wrapper keeps the optional interface the rule implements, since the
// engine finds source rules and multi-spec rules by type assertion.
func maskSecretLines(registry *definitions.Registry, rules ...vrules.Rule) []vrules.Rule {
	masked := make([]vrules.Rule, 0, len(rules))
	for _, rule := range rules {
		base := secretLinesRule{Rule: rule, registry: registry}
		switch r := rule.(type) {
		case vrules.SourceRule:
			masked = append(masked, &secretLinesSourceRule{secretLinesRule: base, source: r})
		case vrules.MultiSpecRule:
			masked = append(masked, &secretLinesMultiSpecRule{secretLinesRule: base, multi: r})
		default:
			masked = append(masked, &base)
		}
	}
	return masked
}

type secretLinesRule struct {
	vrules.Rule
	registry *definitions.Registry
}

func (r *secretLinesRule) Validate(ctx *vrules.ValidationContext) []vrules.ValidationResult {
	return r.mark(ctx, r.Rule.Validate(ctx))
}

// mark sets Sensitive on each result whose line can show a secret value.
func (r *secretLinesRule) mark(ctx *vrules.ValidationContext, results []vrules.ValidationResult) []vrules.ValidationResult {
	for i := range results {
		if !results[i].Sensitive {
			results[i].Sensitive = r.lineShowsSecret(ctx, results[i].Reference)
		}
	}
	return results
}

// markFiles is mark for results keyed by file, each with its context.
func (r *secretLinesRule) markFiles(contexts map[string]*vrules.ValidationContext, results map[string][]vrules.ValidationResult) map[string][]vrules.ValidationResult {
	for path, fileResults := range results {
		r.mark(contexts[path], fileResults)
	}
	return results
}

// lineShowsSecret reports whether the line validate prints for a result at
// reference in the destination spec ctx holds can show a secret value. Only a
// line in config can, and it does when showsSecret says so. Read as written,
// before substitution, a secret key holding exactly one reference with no
// default shows that reference and nothing else, so its line is left as it
// is.
func (r *secretLinesRule) lineShowsSecret(ctx *vrules.ValidationContext, reference string) bool {
	configPath, ok := strings.CutPrefix(reference, "/spec/config/")
	if !ok || ctx == nil {
		return false
	}

	spec := readSourceSpec(ctx.Spec)
	secretKeys, _ := specSecretKeys(r.registry, spec)
	if !showsSecret(secretKeys, spec.config, configPath) {
		return false
	}
	for _, slot := range readSecretSlots(r.registry, ctx) {
		if slot.reference == reference && slot.referencesVariable() {
			return false
		}
	}
	return true
}

type secretLinesSourceRule struct {
	secretLinesRule
	source vrules.SourceRule
}

func (r *secretLinesSourceRule) ValidateSources(specs map[string]*vrules.ValidationContext) map[string][]vrules.ValidationResult {
	return r.markFiles(specs, r.source.ValidateSources(specs))
}

type secretLinesMultiSpecRule struct {
	secretLinesRule
	multi vrules.MultiSpecRule
}

func (r *secretLinesMultiSpecRule) ValidateSpecs(specs map[string]*vrules.ValidationContext) map[string][]vrules.ValidationResult {
	return r.markFiles(specs, r.multi.ValidateSpecs(specs))
}
