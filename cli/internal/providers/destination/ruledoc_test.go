package destination_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
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

					diagnostics, err := loadFiles(t, registry, example.Files, filledVarFile)
					assert.NoError(t, err)
					assert.Empty(t, diagnostics, "unexpected diagnostics:\n%s", describe(diagnostics))
				})
			}
			for _, example := range behaviour.Invalid {
				t.Run(example.ExampleID, func(t *testing.T) {
					t.Parallel()

					diagnostics, err := loadFiles(t, registry, example.Files, filledVarFile)
					assert.NoError(t, err, "warnings alone must not fail the load")
					assertExpectedDiagnostics(t, entry.RuleID, example, diagnostics)
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

// loadFiles runs files through project.Load, the path validate takes.
func loadFiles(t *testing.T, registry *definitions.Registry, files map[string]string, substitutor varsubst.Substitutor) (validation.Diagnostics, error) {
	t.Helper()

	cp, err := provider.NewCompositeProvider(map[string]provider.Provider{
		"destination": destination.NewProvider(nil, registry),
	})
	require.NoError(t, err)

	recorder := &diagnosticsRecorder{}
	err = project.New(cp,
		project.WithLoader(memoryFiles(files)),
		project.WithRenderer(recorder),
		project.WithSubstitutor(substitutor),
	).Load("example")
	return recorder.diagnostics, err
}

// assertExpectedDiagnostics requires exactly the example's expected
// diagnostics. Diagnostics carry positions rather than JSON pointers, so each
// reference is resolved against its file.
func assertExpectedDiagnostics(t *testing.T, ruleID string, example docs.InvalidExample, diagnostics validation.Diagnostics) {
	t.Helper()

	assert.Len(t, diagnostics, len(example.ExpectedDiagnostics), "diagnostics:\n%s", describe(diagnostics))
	for _, expected := range example.ExpectedDiagnostics {
		indexer, err := pathindex.NewPathIndexer([]byte(example.Files[expected.File]))
		require.NoError(t, err)
		position, err := indexer.PositionLookup(expected.Reference)
		require.NoError(t, err)

		assert.True(t, slices.ContainsFunc(diagnostics, func(d validation.Diagnostic) bool {
			return d.RuleID == ruleID &&
				d.File == expected.File &&
				d.Position.Line == position.Line &&
				d.Position.Column == position.Column &&
				d.Severity.String() == expected.Severity &&
				strings.Contains(d.Message, expected.MessageContains)
		}), "expected %+v, got:\n%s", expected, describe(diagnostics))
	}
}

func describe(diagnostics validation.Diagnostics) string {
	described := make([]string, 0, len(diagnostics))
	for _, d := range diagnostics {
		described = append(described, fmt.Sprintf("%s:%d:%d %s[%s] %s", d.File, d.Position.Line, d.Position.Column, d.Severity, d.RuleID, d.Message))
	}
	return strings.Join(described, "\n")
}

type memoryFiles map[string]string

func (f memoryFiles) Load(string) (map[string]*specs.RawSpec, error) {
	raw := make(map[string]*specs.RawSpec, len(f))
	for name, content := range f {
		raw[name] = &specs.RawSpec{Data: []byte(content)}
	}
	return raw, nil
}

type diagnosticsRecorder struct {
	diagnostics validation.Diagnostics
}

func (r *diagnosticsRecorder) Render(diagnostics validation.Diagnostics) error {
	r.diagnostics = append(r.diagnostics, diagnostics...)
	return nil
}
