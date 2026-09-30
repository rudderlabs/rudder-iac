package destination

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/webhook"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testutil"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRuleDocExamplesLoadWithRealRegistry loads destination doc examples through
// project validation so authored examples use public CLI type names the registry
// accepts, not upstream API type names.
func TestRuleDocExamplesLoadWithRealRegistry(t *testing.T) {
	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(webhook.NewDefinition()))
	p := NewProvider(nil, registry)

	var examples int
	for _, entry := range p.RuleDocEntries() {
		for _, behaviour := range entry.MatchBehavior {
			for _, example := range behaviour.Valid {
				examples++
				t.Run(example.ExampleID, func(t *testing.T) {
					diagnostics, err := loadRuleDocExample(t, example.Files)
					assert.NoError(t, err)
					assert.Empty(t, diagnostics, "unexpected diagnostics:\n%s", describeRuleDocDiagnostics(diagnostics))
				})
			}
			for _, example := range behaviour.Invalid {
				examples++
				t.Run(example.ExampleID, func(t *testing.T) {
					diagnostics, err := loadRuleDocExample(t, example.Files)
					assertRuleDocDiagnostics(t, entry.RuleID, example, diagnostics, err)
				})
			}
		}
	}

	require.NotZero(t, examples, "no destination rule doc example ran")
}

func loadRuleDocExample(t *testing.T, files map[string]string) (validation.Diagnostics, error) {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(webhook.NewDefinition()))

	cp, err := provider.NewCompositeProvider(map[string]provider.Provider{
		"destination":     NewProvider(nil, registry),
		"transformations": transformations.NewProviderWithStore(&testutil.MockTransformationStore{}),
	})
	require.NoError(t, err)

	recorder := &ruleDocDiagnosticsRecorder{}
	err = project.New(cp, project.WithLoader(ruleDocExampleFiles(files)), project.WithRenderer(recorder)).Load("example")
	return recorder.diagnostics, err
}

func assertRuleDocDiagnostics(t *testing.T, ruleID string, example docs.InvalidExample, diagnostics validation.Diagnostics, loadErr error) {
	t.Helper()

	wantErr := slices.ContainsFunc(example.ExpectedDiagnostics, func(d docs.ExpectedDiagnostic) bool {
		return d.Severity == rules.Error.String()
	})
	assert.Equal(t, wantErr, loadErr != nil, "load error: %v", loadErr)
	assert.Len(t, diagnostics, len(example.ExpectedDiagnostics), "diagnostics:\n%s\nload error: %v", describeRuleDocDiagnostics(diagnostics), loadErr)
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
		}), "expected %+v, got:\n%s\n(load error: %v)", expected, describeRuleDocDiagnostics(diagnostics), loadErr)
	}

	assert.True(t, slices.ContainsFunc(diagnostics, func(d validation.Diagnostic) bool {
		return d.RuleID == ruleID
	}), "no diagnostic from %s, got:\n%s", ruleID, describeRuleDocDiagnostics(diagnostics))
}

func describeRuleDocDiagnostics(diagnostics validation.Diagnostics) string {
	described := make([]string, 0, len(diagnostics))
	for _, d := range diagnostics {
		described = append(described, fmt.Sprintf("%s:%d:%d %s[%s] %s", d.File, d.Position.Line, d.Position.Column, d.Severity, d.RuleID, d.Message))
	}
	return strings.Join(described, "\n")
}

type ruleDocExampleFiles map[string]string

func (f ruleDocExampleFiles) Load(string) (map[string]*specs.RawSpec, error) {
	raw := make(map[string]*specs.RawSpec, len(f))
	for name, content := range f {
		raw[name] = &specs.RawSpec{Data: []byte(content)}
	}
	return raw, nil
}

type ruleDocDiagnosticsRecorder struct {
	diagnostics validation.Diagnostics
}

func (r *ruleDocDiagnosticsRecorder) Render(diagnostics validation.Diagnostics) error {
	r.diagnostics = append(r.diagnostics, diagnostics...)
	return nil
}
