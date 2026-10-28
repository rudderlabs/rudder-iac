// Package docstest loads the examples in rule doc fragments as projects, so a
// provider's tests can check that a fragment still matches what validate
// reports.
package docstest

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
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

// Load runs files, keyed by path, through project.Load, the path validate
// takes, with p and substitutor. r renders the diagnostics the load reports.
func Load(p provider.Provider, files map[string]string, substitutor varsubst.Substitutor, r renderer.Renderer) error {
	return project.New(p,
		project.WithLoader(memoryFiles(files)),
		project.WithRenderer(r),
		project.WithSubstitutor(substitutor),
	).Load("example")
}

// Recorder is a renderer that keeps every diagnostic it is given.
type Recorder struct {
	Diagnostics validation.Diagnostics
}

func (r *Recorder) Render(diagnostics validation.Diagnostics) error {
	r.Diagnostics = append(r.Diagnostics, diagnostics...)
	return nil
}

// LoadDiagnostics is Load with a Recorder, and returns what it recorded.
func LoadDiagnostics(p provider.Provider, files map[string]string, substitutor varsubst.Substitutor) (validation.Diagnostics, error) {
	recorder := &Recorder{}
	err := Load(p, files, substitutor, recorder)
	return recorder.Diagnostics, err
}

// AssertExpectedDiagnostics requires the example to produce exactly its
// expected diagnostics, at least one of them from the rule it documents, and
// the load to fail only when one of them is an error. Diagnostics carry
// positions rather than JSON pointers, so each reference is resolved against
// its file. The lookup is exact: the engine's fallback to the nearest
// ancestor would let a pointer to a missing key match its parent.
func AssertExpectedDiagnostics(t *testing.T, ruleID string, example docs.InvalidExample, diagnostics validation.Diagnostics, loadErr error) {
	t.Helper()

	// Warnings alone must not fail the load, or validate would exit non-zero on them.
	wantErr := slices.ContainsFunc(example.ExpectedDiagnostics, func(d docs.ExpectedDiagnostic) bool {
		return d.Severity == rules.Error.String()
	})
	assert.Equal(t, wantErr, loadErr != nil, "load error: %v", loadErr)
	assert.Len(t, diagnostics, len(example.ExpectedDiagnostics), "diagnostics:\n%s\nload error: %v", Describe(diagnostics), loadErr)
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
		}), "expected %+v, got:\n%s\n(load error: %v)", expected, Describe(diagnostics), loadErr)
	}

	assert.True(t, slices.ContainsFunc(diagnostics, func(d validation.Diagnostic) bool {
		return d.RuleID == ruleID
	}), "no diagnostic from %s, got:\n%s", ruleID, Describe(diagnostics))
}

// Describe renders diagnostics for failure output. Printing the structs
// directly buries the fields that matter under LineText and Examples.
func Describe(diagnostics validation.Diagnostics) string {
	described := make([]string, 0, len(diagnostics))
	for _, d := range diagnostics {
		described = append(described, fmt.Sprintf("%s:%d:%d %s[%s] %s", d.File, d.Position.Line, d.Position.Column, d.Severity, d.RuleID, d.Message))
	}
	return strings.Join(described, "\n")
}

// memoryFiles is a project loader that serves files from memory.
type memoryFiles map[string]string

func (f memoryFiles) Load(string) (map[string]*specs.RawSpec, error) {
	raw := make(map[string]*specs.RawSpec, len(f))
	for name, content := range f {
		raw[name] = &specs.RawSpec{Data: []byte(content)}
	}
	return raw, nil
}
