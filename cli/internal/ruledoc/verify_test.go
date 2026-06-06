package ruledoc

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerify_SyntacticInvalidExampleMatches exercises the full path: build a real
// registry (gatekeeper rules only), craft an invalid example that triggers
// "project/spec-syntax-valid", and confirm verify() returns no misses.
func TestVerify_SyntacticInvalidExampleMatches(t *testing.T) {
	cp := &testutils.MockProvider{}
	reg, err := project.BuildRegistry(cp)
	require.NoError(t, err)

	// A spec that parses fine as YAML but is missing the `kind` field, which
	// triggers the "'kind' is required" diagnostic from spec-syntax-valid.
	missingKindYAML := `version: rudder/v1
metadata:
  name: test-spec
spec:
  properties: []
`
	doc := docs.DocumentedRules{
		Rules: []docs.DocumentedRule{
			{
				RuleID: "project/spec-syntax-valid",
				Phase:  "syntactic",
				MatchBehavior: []docs.MatchBehaviorEntry{
					{
						AppliesTo: []docs.MatchPatternDoc{{Kind: "*", Version: "*"}},
						Invalid: []docs.InvalidExample{
							{
								ExampleID:   "missing-kind",
								Title:       "spec missing kind field",
								Files:       map[string]string{"spec.yaml": missingKindYAML},
								ExpectedDiagnostics: []docs.ExpectedDiagnostic{
									{
										File:            "spec.yaml",
										Reference:       "/kind",
										Severity:        "error",
										MessageContains: "'kind' is required",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Gatekeeper rules are syntactic, so no provider factory is needed.
	errs := verify(reg, doc, docs.ModeSubset, nil)
	assert.Empty(t, errs, "expected no verification misses for a matching invalid example")
}

// TestVerify_SemanticWithoutFactoryErrors proves the nil-factory guard: a
// semantic-phase example encountered with no provider factory yields a clear
// error rather than a panic. This is the contract syntactic-only callers rely
// on when they pass nil.
func TestVerify_SemanticWithoutFactoryErrors(t *testing.T) {
	cp := &testutils.MockProvider{}
	reg, err := project.BuildRegistry(cp)
	require.NoError(t, err)

	doc := docs.DocumentedRules{
		Rules: []docs.DocumentedRule{
			{
				RuleID: "datacatalog/properties/some-semantic-rule",
				Phase:  "semantic",
				MatchBehavior: []docs.MatchBehaviorEntry{
					{
						AppliesTo: []docs.MatchPatternDoc{{Kind: "properties", Version: "rudder/v1"}},
						Invalid: []docs.InvalidExample{
							{
								ExampleID:   "semantic-invalid",
								Title:       "semantic invalid example",
								Files:       map[string]string{"spec.yaml": "version: rudder/v1\nkind: properties\n"},
								ExpectedDiagnostics: []docs.ExpectedDiagnostic{
									{File: "spec.yaml", Reference: "/spec/properties", Severity: "error", MessageContains: "some message"},
								},
							},
						},
					},
				},
			},
		},
	}

	errs := verify(reg, doc, docs.ModeSubset, nil)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "provider factory")
}

// newDataCatalogProvider builds a fresh, credential-free datacatalog provider.
// Each call yields an isolated provider, exactly what the semantic verifier
// needs so resources from one example don't leak into the next.
func newDataCatalogProvider() (provider.Provider, error) {
	return datacatalog.New(nil), nil
}

// TestVerify_SemanticInvalidExampleMatches drives the full semantic path
// (parse -> syntax -> graph -> ValidateSemantic) against a real datacatalog
// provider built per example by the injected factory. The example declares two
// categories with the same name, which the category name-uniqueness rule flags;
// the verifier must find the expected diagnostic with no misses.
func TestVerify_SemanticInvalidExampleMatches(t *testing.T) {
	reg, err := project.BuildRegistry(datacatalog.New(nil))
	require.NoError(t, err)

	duplicateCategories := `version: rudder/v1
kind: categories
metadata:
  name: my-categories
spec:
  categories:
    - id: user_actions
      name: User Actions
    - id: user_actions_2
      name: User Actions
`
	doc := docs.DocumentedRules{
		Rules: []docs.DocumentedRule{
			{
				RuleID: "datacatalog/categories/semantic-valid",
				Phase:  "semantic",
				MatchBehavior: []docs.MatchBehaviorEntry{
					{
						AppliesTo: []docs.MatchPatternDoc{{Kind: "categories", Version: "rudder/v1"}},
						Invalid: []docs.InvalidExample{
							{
								ExampleID: "categories-duplicate-name",
								Title:     "two categories sharing a name",
								Files:     map[string]string{"spec.yaml": duplicateCategories},
								ExpectedDiagnostics: []docs.ExpectedDiagnostic{
									{
										File:            "spec.yaml",
										Reference:       "/categories/0/name",
										Severity:        "error",
										MessageContains: "duplicate name 'User Actions' within kind 'categories'",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	errs := verify(reg, doc, docs.ModeSubset, newDataCatalogProvider)
	assert.Empty(t, errs, "expected the duplicate-name semantic example to match cleanly")
}

// TestVerify_SemanticValidExampleProducesNoDiagnostics confirms a clean
// semantic example (unique category names) yields no error/warning diagnostics
// through the real engine.
func TestVerify_SemanticValidExampleProducesNoDiagnostics(t *testing.T) {
	reg, err := project.BuildRegistry(datacatalog.New(nil))
	require.NoError(t, err)

	uniqueCategories := `version: rudder/v1
kind: categories
metadata:
  name: my-categories
spec:
  categories:
    - id: user_actions
      name: User Actions
    - id: system_events
      name: System Events
`
	doc := docs.DocumentedRules{
		Rules: []docs.DocumentedRule{
			{
				RuleID: "datacatalog/categories/semantic-valid",
				Phase:  "semantic",
				MatchBehavior: []docs.MatchBehaviorEntry{
					{
						AppliesTo: []docs.MatchPatternDoc{{Kind: "categories", Version: "rudder/v1"}},
						Valid: []docs.ValidExample{
							{
								ExampleID: "categories-unique-names",
								Title:     "unique category names",
								Files:     map[string]string{"spec.yaml": uniqueCategories},
							},
						},
					},
				},
			},
		},
	}

	errs := verify(reg, doc, docs.ModeSubset, newDataCatalogProvider)
	assert.Empty(t, errs, "a clean semantic example must produce no diagnostics")
}
