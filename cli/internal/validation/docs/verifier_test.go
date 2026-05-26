package docs

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeEngine returns whatever diagnostics it was configured with, ignoring inputs.
type fakeEngine struct {
	syntaxDiags validation.Diagnostics
	syntaxErr   error
}

func (f fakeEngine) ValidateSyntax(_ context.Context, _ map[string]*specs.RawSpec) (validation.Diagnostics, error) {
	return f.syntaxDiags, f.syntaxErr
}

func (f fakeEngine) ValidateSemantic(_ context.Context, _ map[string]*specs.RawSpec, _ *resources.Graph) (validation.Diagnostics, error) {
	return nil, nil
}

func TestVerifier_SubsetMatchSucceedsWhenExpectedDiagnosticIsProduced(t *testing.T) {
	doc := &RulesDoc{Rules: []ResolvedRule{{
		RuleID: "r1",
		MatchBehavior: []MatchBehaviorEntry{{
			Invalid: []InvalidExample{{
				ExampleID: "ex1",
				Files:     map[string]string{"main.yaml": "kind: foo\nversion: v1\nspec: {}\n"},
				ExpectedDiagnostics: []ExpectedDiagnostic{
					{File: "main.yaml", Reference: "/name", Severity: "error", MessageContains: "required"},
				},
			}},
		}},
	}}}

	engine := fakeEngine{syntaxDiags: validation.Diagnostics{{
		RuleID:   "r1",
		Severity: rules.Error,
		Message:  "field 'name' is required",
		File:     "main.yaml",
		Position: pathindex.Position{Line: 1, Column: 1},
	}}}

	v := NewVerifier(func() validation.ValidationEngine { return engine })
	require.NoError(t, v.Verify(doc))
}

func TestVerifier_FailsWhenExpectedDiagnosticIsMissing(t *testing.T) {
	doc := &RulesDoc{Rules: []ResolvedRule{{
		RuleID: "r1",
		MatchBehavior: []MatchBehaviorEntry{{
			Invalid: []InvalidExample{{
				ExampleID: "ex1",
				Files:     map[string]string{"main.yaml": "kind: foo\n"},
				ExpectedDiagnostics: []ExpectedDiagnostic{
					{File: "main.yaml", Reference: "/name", Severity: "error", MessageContains: "required"},
				},
			}},
		}},
	}}}

	engine := fakeEngine{syntaxDiags: validation.Diagnostics{}}
	v := NewVerifier(func() validation.ValidationEngine { return engine })

	err := v.Verify(doc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ex1")
	assert.Contains(t, err.Error(), "/name")
}
