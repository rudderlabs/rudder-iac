package docs

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeEngine satisfies the narrow verifierEngine interface declared in
// verifier.go (only ValidateSyntax). It returns whatever diagnostics it was
// constructed with — used to exercise the verifier's subset-match logic
// without standing up the real engine.
type fakeEngine struct {
	syntax validation.Diagnostics
}

func (f *fakeEngine) ValidateSyntax(_ context.Context, _ map[string]*specs.RawSpec) (validation.Diagnostics, error) {
	return f.syntax, nil
}

func TestVerifier_Verify_SubsetMatchSucceeds(t *testing.T) {
	example := InvalidExample{
		ExampleID: "ex-1",
		Title:     "missing name",
		Files:     map[string]string{"a.yaml": "version: rudder/v1\nkind: categories\nmetadata:\n  name: x\nspec: {}\n"},
		ExpectedDiagnostics: []ExpectedDiagnostic{
			{File: "a.yaml", Reference: "/name", Severity: "error", MessageContains: "required"},
		},
	}

	produced := validation.Diagnostics{
		{
			RuleID:   "rule-a",
			Severity: rules.Error,
			Message:  "field 'name' is required",
			File:     "a.yaml",
			Position: pathindex.Position{Line: 1},
		},
	}

	v := newVerifierForTest(&fakeEngine{syntax: produced})
	err := v.verifyExample(context.Background(), example, "rule-a")
	require.NoError(t, err)
}

func TestVerifier_Verify_FailsWhenExpectedDiagnosticMissing(t *testing.T) {
	example := InvalidExample{
		ExampleID: "ex-1",
		Title:     "missing name",
		Files:     map[string]string{"a.yaml": "version: rudder/v1\nkind: categories\nmetadata:\n  name: x\nspec: {}\n"},
		ExpectedDiagnostics: []ExpectedDiagnostic{
			{File: "a.yaml", Reference: "/name", Severity: "error", MessageContains: "required"},
		},
	}

	v := newVerifierForTest(&fakeEngine{syntax: nil}) // engine produces nothing

	err := v.verifyExample(context.Background(), example, "rule-a")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ex-1")
	assert.Contains(t, err.Error(), "expected diagnostic")
}
