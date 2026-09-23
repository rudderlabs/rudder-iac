package project

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildRegistry_AggregatesManifestPatterns proves the two-provider
// BuildRegistry unions the manifest provider's SupportedMatchPatterns into the
// active set when importMerge is enabled, so the import-manifest kind is treated
// as known by the gatekeeper rules (SpecSyntaxValidRule) instead of being
// rejected as an unknown kind.
func TestBuildRegistry_AggregatesManifestPatterns(t *testing.T) {
	resourceProvider := &testutils.MockProvider{
		MatchPatterns: []rules.MatchPattern{
			rules.MatchKindVersion("properties", specs.SpecVersionV1),
		},
	}
	manifestProvider := importmanifest.New()

	reg, err := BuildRegistry(resourceProvider, manifestProvider, true)
	require.NoError(t, err)

	// The manifest kind/version must resolve to at least the gatekeeper rules,
	// proving its pattern is in the active set (an unknown kind would still match
	// MatchAll gatekeepers, so we assert the spec-syntax-valid rule treats it as
	// a known kind by validating a well-formed manifest spec with no errors).
	manifestRules := reg.SyntacticRulesFor(importmanifest.KindImportManifest, specs.SpecVersionV1)
	require.NotEmpty(t, manifestRules)

	var specSyntax rules.Rule
	for _, r := range manifestRules {
		if r.ID() == "project/spec-syntax-valid" {
			specSyntax = r
			break
		}
	}
	require.NotNil(t, specSyntax, "spec-syntax-valid must apply to the manifest kind")

	results := specSyntax.Validate(&rules.ValidationContext{
		Kind:     importmanifest.KindImportManifest,
		Version:  specs.SpecVersionV1,
		Metadata: map[string]any{"name": "import-manifest"},
		Spec:     map[string]any{"workspaces": []any{}},
	})
	assert.Empty(t, results, "manifest kind must be a known kind/version (no 'unknown kind' error)")
}

// TestBuildRegistry_RejectsManifestKindWhenFlagOff proves that with importMerge
// disabled (the default), the import-manifest kind is excluded from the active
// set and rejected as an unknown kind by spec-syntax-valid.
func TestBuildRegistry_RejectsManifestKindWhenFlagOff(t *testing.T) {
	resourceProvider := &testutils.MockProvider{
		MatchPatterns: []rules.MatchPattern{
			rules.MatchKindVersion("properties", specs.SpecVersionV1),
		},
	}
	manifestProvider := importmanifest.New()

	reg, err := BuildRegistry(resourceProvider, manifestProvider, false)
	require.NoError(t, err)

	var specSyntax rules.Rule
	for _, r := range reg.AllSyntacticRules() {
		if r.ID() == "project/spec-syntax-valid" {
			specSyntax = r
			break
		}
	}
	require.NotNil(t, specSyntax)

	results := specSyntax.Validate(&rules.ValidationContext{
		Kind:     importmanifest.KindImportManifest,
		Version:  specs.SpecVersionV1,
		Metadata: map[string]any{"name": "import-manifest"},
		Spec:     map[string]any{"workspaces": []any{}},
	})
	require.NotEmpty(t, results, "manifest kind must be rejected when importMerge is off")
	assert.Contains(t, results[0].Message, "'kind' must be one of")
}
