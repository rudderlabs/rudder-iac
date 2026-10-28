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

// mark sets Sensitive on each result whose line can show a secret value. It
// reads the spec's secret keys and slots once for all of its results.
func (r *secretLinesRule) mark(ctx *vrules.ValidationContext, results []vrules.ValidationResult) []vrules.ValidationResult {
	var lines *secretLines
	for i := range results {
		if results[i].Sensitive {
			continue
		}
		if lines == nil {
			lines = readSecretLines(r.registry, ctx)
		}
		results[i].Sensitive = lines.showSecret(results[i].Reference)
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

// secretLines is what deciding whether a line of one destination spec can show
// a secret needs, read once per spec.
type secretLines struct {
	config     map[string]any
	secretKeys []string
	// asWritten holds the secret keys, by reference, that hold exactly one
	// variable reference with no default.
	asWritten map[string]bool
}

// readSecretLines reads the secret keys and slots of the destination spec ctx
// holds. A nil ctx gives lines that show no secret.
func readSecretLines(registry *definitions.Registry, ctx *vrules.ValidationContext) *secretLines {
	if ctx == nil {
		return &secretLines{}
	}
	spec := readSourceSpec(ctx.Spec)
	secretKeys, _ := specSecretKeys(registry, spec)
	lines := &secretLines{config: spec.config, secretKeys: secretKeys, asWritten: map[string]bool{}}
	for _, slot := range readSecretSlots(registry, ctx) {
		if slot.referencesVariable() {
			lines.asWritten[slot.reference] = true
		}
	}
	return lines
}

// showSecret reports whether the line validate prints for a result at
// reference can show a secret value. Only a line in config can, and it does
// when showsSecret says so. Read as written, before substitution, a secret key
// holding exactly one reference with no default shows that reference and
// nothing else, so its line is left as it is.
func (l *secretLines) showSecret(reference string) bool {
	configPath, ok := strings.CutPrefix(reference, "/spec/config/")
	if !ok || !showsSecret(l.secretKeys, l.config, configPath) {
		return false
	}
	return !l.asWritten[reference]
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
