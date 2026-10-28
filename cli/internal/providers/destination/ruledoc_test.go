package destination_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs/docstest"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

// TestSecretReferenceRuleDocs loads the secret reference fragments' examples
// as projects, so a fragment cannot drift from what validate reports. The
// other destination fragments predate this check and write `type: WEBHOOK`,
// which validate rejects, so they are not loaded yet.
func TestSecretReferenceRuleDocs(t *testing.T) {
	t.Parallel()

	registry := destination.SecretRuleRegistry(t)
	p := destination.NewProvider(nil, registry)

	syntactic, semantic := p.SyntacticRules(), p.SemanticRules()
	doc, verrs := docs.Generate(syntactic, semantic, p.RuleDocEntries(), "test", "2026-10-21T00:00:00Z")
	assert.Empty(t, verrs)
	require.Len(t, doc.Rules, len(syntactic)+len(semantic))

	var examples int
	for _, entry := range p.RuleDocEntries() {
		if !slices.Contains([]string{destination.SecretReferenceRuleID, destination.SecretReferenceUniqueRuleID}, entry.RuleID) {
			continue
		}
		for _, behaviour := range entry.MatchBehavior {
			examples += len(behaviour.Valid) + len(behaviour.Invalid)
			for _, example := range behaviour.Valid {
				t.Run(example.ExampleID, func(t *testing.T) {
					t.Parallel()

					diagnostics, err := docstest.LoadDiagnostics(destinationProvider(t, registry), example.Files, filledVarFile)
					assert.NoError(t, err)
					assert.Empty(t, diagnostics, "unexpected diagnostics:\n%s", docstest.Describe(diagnostics))
				})
			}
			for _, example := range behaviour.Invalid {
				t.Run(example.ExampleID, func(t *testing.T) {
					t.Parallel()

					diagnostics, err := docstest.LoadDiagnostics(destinationProvider(t, registry), example.Files, filledVarFile)
					docstest.AssertExpectedDiagnostics(t, entry.RuleID, example, diagnostics, err)
					// The shared check lets an expected diagnostic come from
					// another rule, which some rETL examples need. These
					// examples document one rule each, so every diagnostic
					// must come from it.
					for _, d := range diagnostics {
						assert.Equal(t, entry.RuleID, d.RuleID, "diagnostics:\n%s", docstest.Describe(diagnostics))
					}
				})
			}
		}
	}
	require.NotZero(t, examples, "no rule doc example ran")
}

// filledVarFile stands in for the var file validate is given: every variable
// resolves, to a value each secret key's pattern accepts.
var filledVarFile = varsubst.NewSubstitutor(anyVariable{})

type anyVariable struct{}

func (anyVariable) Resolve(string) (string, bool) {
	return "fake-variable-value", true
}
