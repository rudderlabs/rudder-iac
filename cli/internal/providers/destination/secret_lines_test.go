package destination

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// referencesRule returns one result for each reference, none of them marked
// Sensitive, as a rule that leaves masking to maskSecretLines does.
type referencesRule []string

func (referencesRule) ID() string                       { return "destination/references" }
func (referencesRule) Severity() vrules.Severity        { return vrules.Error }
func (referencesRule) Description() string              { return "returns a result at each reference" }
func (referencesRule) AppliesTo() []vrules.MatchPattern { return nil }
func (referencesRule) Examples() vrules.Examples        { return vrules.Examples{} }
func (r referencesRule) Validate(*vrules.ValidationContext) []vrules.ValidationResult {
	return r.results()
}

func (r referencesRule) results() []vrules.ValidationResult {
	results := make([]vrules.ValidationResult, 0, len(r))
	for _, reference := range r {
		results = append(results, vrules.ValidationResult{Reference: reference, Message: reference})
	}
	return results
}

type referencesSourceRule struct{ referencesRule }

func (r referencesSourceRule) ValidateSources(map[string]*vrules.ValidationContext) map[string][]vrules.ValidationResult {
	return map[string][]vrules.ValidationResult{"dest.yaml": r.results()}
}

type referencesMultiSpecRule struct{ referencesRule }

func (r referencesMultiSpecRule) ValidateSpecs(map[string]*vrules.ValidationContext) map[string][]vrules.ValidationResult {
	return map[string][]vrules.ValidationResult{"dest.yaml": r.results()}
}

// sensitiveByReference maps each result's reference to its Sensitive flag.
func sensitiveByReference(results []vrules.ValidationResult) map[string]bool {
	sensitive := make(map[string]bool, len(results))
	for _, result := range results {
		sensitive[result.Reference] = result.Sensitive
	}
	return sensitive
}

// After substitution every value is real, so any line at, under or leading
// to a secret key can show a secret, whichever rule reports it.
func TestMaskSecretLinesAfterSubstitution(t *testing.T) {
	t.Parallel()

	rule := maskSecretLines(secretRuleRegistry(t), referencesRule{
		"/spec/config/api_key",
		"/spec/config/api_secret",
		"/spec/config/residency_server",
		"/spec/config/missing",
		"/spec/config",
		"/spec/display_name",
	})[0]

	results := rule.Validate(&vrules.ValidationContext{
		Kind:    DestinationSpecKind,
		Version: "rudder/v1",
		Spec: map[string]any{
			"id":                 "shop-amplitude",
			"type":               "am",
			"definition_version": 1,
			"display_name":       "Shop Amplitude",
			"config": map[string]any{
				"api_key":          "fake-substituted-key",
				"api_secret":       "fake-substituted-secret",
				"residency_server": "standard",
			},
		},
	})

	assert.Equal(t, map[string]bool{
		"/spec/config/api_key":          true,
		"/spec/config/api_secret":       true,
		"/spec/config/residency_server": false,
		// A missing key prints the config line, which holds no value.
		"/spec/config/missing": false,
		"/spec/config":         false,
		"/spec/display_name":   false,
	}, sensitiveByReference(results))
}

// Before substitution a secret key may still hold a literal. One holding
// exactly one reference with no default shows only that reference, so its
// line is printed as written.
func TestMaskSecretLinesAsWritten(t *testing.T) {
	t.Parallel()

	source := []byte(`version: rudder/v1
kind: destination
metadata:
  name: shop-webhook
spec:
  id: shop-webhook
  type: webhook
  definition_version: 1
  config:
    webhook_url: https://example.com/hook
    headers:
      - to: '{{ .SHOP_WEBHOOK_HEADERS_0_TO }}'
        from: X-Api-Key
      - to: '{{ .TOKEN | default "fake-default-token" }}'
        from: X-Token
      - to: fake-literal-header
        from: X-Other
`)
	contexts := map[string]*vrules.ValidationContext{
		"dest.yaml": vrules.NewSourceContext("dest.yaml", &specs.RawSpec{Data: source}),
	}
	rule := maskSecretLines(secretRuleRegistry(t), referencesSourceRule{referencesRule{
		"/spec/config/headers/0/to",
		"/spec/config/headers/1/to",
		"/spec/config/headers/2/to",
		"/spec/config/headers/2/from",
		"/spec/config/webhook_url",
	}})[0]

	results := rule.(vrules.SourceRule).ValidateSources(contexts)

	assert.Equal(t, map[string]bool{
		"/spec/config/headers/0/to": false,
		"/spec/config/headers/1/to": true,
		"/spec/config/headers/2/to": true,
		// headers.from is not a secret key of webhook.
		"/spec/config/headers/2/from": false,
		"/spec/config/webhook_url":    false,
	}, sensitiveByReference(results["dest.yaml"]))
}

// The engine finds source and multi-spec rules by type assertion, so a
// wrapped rule keeps the interface it had, and its results are masked on
// each path.
func TestMaskSecretLinesKeepsRuleInterfaces(t *testing.T) {
	t.Parallel()

	const pointer = "/spec/config/api_key"
	rules := maskSecretLines(secretRuleRegistry(t),
		referencesRule{pointer},
		referencesSourceRule{referencesRule{pointer}},
		referencesMultiSpecRule{referencesRule{pointer}},
	)
	var (
		ctx = &vrules.ValidationContext{Spec: map[string]any{
			"type":               "am",
			"definition_version": 1,
			"config":             map[string]any{"api_key": "fake-amplitude-key"},
		}}
		contexts = map[string]*vrules.ValidationContext{"dest.yaml": ctx}
		want     = []vrules.ValidationResult{{Reference: pointer, Message: pointer, Sensitive: true}}
	)

	_, isSource := rules[0].(vrules.SourceRule)
	_, isMultiSpec := rules[0].(vrules.MultiSpecRule)
	assert.False(t, isSource || isMultiSpec, "a plain rule stays plain")
	assert.Equal(t, want, rules[0].Validate(ctx))

	source, ok := rules[1].(vrules.SourceRule)
	require.True(t, ok)
	assert.Equal(t, map[string][]vrules.ValidationResult{"dest.yaml": want}, source.ValidateSources(contexts))

	multi, ok := rules[2].(vrules.MultiSpecRule)
	require.True(t, ok)
	assert.Equal(t, map[string][]vrules.ValidationResult{"dest.yaml": want}, multi.ValidateSpecs(contexts))

	for _, rule := range rules {
		assert.Equal(t, "destination/references", rule.ID())
	}
}

// Every rule the provider registers goes through maskSecretLines, so a new
// rule cannot print a secret by leaving Sensitive unset.
func TestProviderMasksEveryRule(t *testing.T) {
	t.Parallel()

	p := NewProvider(nil, secretRuleRegistry(t))
	for _, rule := range slices.Concat(p.SyntacticRules(), p.SemanticRules()) {
		switch rule.(type) {
		case *secretLinesRule, *secretLinesSourceRule, *secretLinesMultiSpecRule:
		default:
			t.Errorf("rule %s is not wrapped by maskSecretLines", rule.ID())
		}
	}
}
