package ruledoc

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
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

	errs := verify(reg, doc, docs.ModeSubset)
	assert.Empty(t, errs, "expected no verification misses for a matching invalid example")
}

// TestVerify_SkipsSemanticPhase ensures that semantic-phase rules are silently
// skipped (counted and logged) without triggering any errors or panics.
func TestVerify_SkipsSemanticPhase(t *testing.T) {
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

	errs := verify(reg, doc, docs.ModeSubset)
	assert.Empty(t, errs, "semantic-phase examples must be skipped without error")
}
