package retl_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	bingads "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/bingads_offline_conversions"
	customerioaudience "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/customerio_audience"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/googleads"
	httpdest "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/http"
	eventstream "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderRuleDocs runs the provider's authored fragments through the real
// docs generator together with its live rules, asserting every rule resolves
// and passes the DocumentedRules validation invariants. The connection rule
// examples are then loaded as projects, so a fragment cannot drift from what
// validate reports.
//
// Both experimental kinds are on, as in the CI catalog, because the embedded
// fragments cover their rules too; without them those fragments would be
// orphans, which is exactly what the gen-rule-docs workflow sets the flags for.
func TestProviderRuleDocs(t *testing.T) {
	// http and Google Ads are verified, so a default CLI registers them and
	// the examples using them run as written. Bing and Customer.io Audience are
	// unverified and registered only behind a flag, but they are the only
	// destinations that support object mapping and the only ones running their
	// own rETL flow, so those examples cannot avoid them; their fragments name
	// the flag, and it is registered unconditionally here.
	registry := definitions.NewRegistry()
	for _, definition := range []*definitions.DestinationDefinition{
		httpdest.NewDefinition(),
		googleads.NewDefinition(),
		bingads.NewDefinition(),
		customerioaudience.NewDefinition(),
	} {
		require.NoError(t, registry.Register(definition))
	}
	p := retl.New(
		newDefaultMockClient(),
		retl.WithTableSupport(),
		retl.WithConnectionSupport(registry),
	)

	syntactic := p.SyntacticRules()
	semantic := p.SemanticRules()

	doc, verrs := docs.Generate(syntactic, semantic, p.RuleDocEntries(), "test", "2026-06-03T00:00:00Z")
	assert.Empty(t, verrs, "expected no validation errors, got: %v", verrs)
	require.Len(t, doc.Rules, len(syntactic)+len(semantic))

	var examples int
	for _, entry := range p.RuleDocEntries() {
		// Only the connection fragments run as projects. The SQL model fragments
		// predate this check (references without /spec, duplicates against
		// models their files do not carry); DEX-915 fixes them and drops the
		// filter.
		if !strings.HasPrefix(entry.RuleID, "retl/connection/") {
			continue
		}
		for _, behaviour := range entry.MatchBehavior {
			examples += len(behaviour.Valid) + len(behaviour.Invalid)
			for _, example := range behaviour.Valid {
				t.Run(example.ExampleID, func(t *testing.T) {
					diagnostics, err := loadExample(t, registry, example.Files)
					assert.NoError(t, err)
					assert.Empty(t, diagnostics, "unexpected diagnostics:\n%s", describe(diagnostics))
				})
			}
			for _, example := range behaviour.Invalid {
				t.Run(example.ExampleID, func(t *testing.T) {
					diagnostics, err := loadExample(t, registry, example.Files)
					assertExpectedDiagnostics(t, entry.RuleID, example, diagnostics, err)
				})
			}
		}
	}

	// The filter is a prefix match on rule IDs: rename the connection rules or
	// move their fragments and every subtest above disappears silently, leaving
	// the docs.Generate assertions to report a pass on their own.
	require.NotZero(t, examples, "no rule doc example ran")
}

// loadExample runs the example files through project.Load, the path validate
// takes, with every provider that owns a kind the examples use.
func loadExample(t *testing.T, registry *definitions.Registry, files map[string]string) (validation.Diagnostics, error) {
	t.Helper()

	cp, err := provider.NewCompositeProvider(map[string]provider.Provider{
		"retl":        retl.New(newDefaultMockClient(), retl.WithConnectionSupport(registry)),
		"eventstream": eventstream.New(source.NewMockSourceClient(), eventstream.WithDestinationRegistry(registry)),
		"destination": destination.NewProvider(nil, registry),
	})
	require.NoError(t, err)

	recorder := &diagnosticsRecorder{}
	err = project.New(cp, project.WithLoader(exampleFiles(files)), project.WithRenderer(recorder)).Load("example")
	return recorder.diagnostics, err
}

// assertExpectedDiagnostics requires the example to produce exactly its
// expected diagnostics, at least one of them from the rule it documents.
// Diagnostics carry positions rather than JSON pointers, so each reference is
// resolved against its file. The lookup is exact: the engine's fallback to the
// nearest ancestor would let a pointer to a missing key match its parent.
func assertExpectedDiagnostics(t *testing.T, ruleID string, example docs.InvalidExample, diagnostics validation.Diagnostics, loadErr error) {
	t.Helper()

	// Warnings alone must not fail the load, or validate would exit non-zero on them.
	wantErr := slices.ContainsFunc(example.ExpectedDiagnostics, func(d docs.ExpectedDiagnostic) bool {
		return d.Severity == rules.Error.String()
	})
	assert.Equal(t, wantErr, loadErr != nil, "load error: %v", loadErr)
	assert.Len(t, diagnostics, len(example.ExpectedDiagnostics), "diagnostics:\n%s\nload error: %v", describe(diagnostics), loadErr)
	for _, expected := range example.ExpectedDiagnostics {
		indexer, err := pathindex.NewPathIndexer([]byte(example.Files[expected.File]))
		require.NoError(t, err)
		position, err := indexer.PositionLookup(expected.Reference)
		require.NoError(t, err)

		assert.True(t, slices.ContainsFunc(diagnostics, func(d validation.Diagnostic) bool {
			return d.File == expected.File &&
				d.Position.Line == position.Line &&
				d.Position.Column == position.Column &&
				d.Severity.String() == expected.Severity &&
				strings.Contains(d.Message, expected.MessageContains)
		}), "expected %+v, got:\n%s\n(load error: %v)", expected, describe(diagnostics), loadErr)
	}

	assert.True(t, slices.ContainsFunc(diagnostics, func(d validation.Diagnostic) bool {
		return d.RuleID == ruleID
	}), "no diagnostic from %s, got:\n%s", ruleID, describe(diagnostics))
}

// describe renders diagnostics for failure output. Printing the structs
// directly buries the four fields that matter under LineText and Examples.
func describe(diagnostics validation.Diagnostics) string {
	described := make([]string, 0, len(diagnostics))
	for _, d := range diagnostics {
		described = append(described, fmt.Sprintf("%s:%d:%d %s[%s] %s", d.File, d.Position.Line, d.Position.Column, d.Severity, d.RuleID, d.Message))
	}
	return strings.Join(described, "\n")
}

type exampleFiles map[string]string

func (f exampleFiles) Load(string) (map[string]*specs.RawSpec, error) {
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
