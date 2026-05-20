package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportManifestSyntaxRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewImportManifestSyntaxRule()

	assert.Equal(t, "project/import-manifest-syntax", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, []rules.MatchPattern{rules.MatchKind(specs.KindImportManifest)}, rule.AppliesTo())
}

func TestImportManifestSyntaxRule_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid manifest passes", func(t *testing.T) {
		t.Parallel()

		rule := NewImportManifestSyntaxRule()
		results := rule.Validate(&rules.ValidationContext{
			FilePath: "manifest.yaml",
			FileName: "manifest.yaml",
			Kind:     specs.KindImportManifest,
			Version:  specs.SpecVersionV1,
			Spec: map[string]any{
				"workspaces": []any{
					map[string]any{
						"workspace_id": "ws-123",
						"resources": []any{
							map[string]any{
								"urn":       "source:my-source",
								"remote_id": "remote-456",
							},
						},
					},
				},
			},
		})

		assert.Empty(t, results)
	})

	t.Run("missing workspaces field", func(t *testing.T) {
		t.Parallel()

		rule := NewImportManifestSyntaxRule()
		results := rule.Validate(&rules.ValidationContext{
			FilePath: "manifest.yaml",
			FileName: "manifest.yaml",
			Kind:     specs.KindImportManifest,
			Version:  specs.SpecVersionV1,
			Spec:     map[string]any{},
		})

		require.Len(t, results, 1)
		assert.Equal(t, "manifest must contain 'workspaces' field", results[0].Message)
		assert.Equal(t, "/spec/workspaces", results[0].Reference)
	})

	t.Run("empty workspaces list", func(t *testing.T) {
		t.Parallel()

		rule := NewImportManifestSyntaxRule()
		results := rule.Validate(&rules.ValidationContext{
			FilePath: "manifest.yaml",
			FileName: "manifest.yaml",
			Kind:     specs.KindImportManifest,
			Version:  specs.SpecVersionV1,
			Spec: map[string]any{
				"workspaces": []any{},
			},
		})

		require.Len(t, results, 1)
		assert.Equal(t, "manifest must contain at least one workspace", results[0].Message)
	})

	t.Run("missing workspace_id", func(t *testing.T) {
		t.Parallel()

		rule := NewImportManifestSyntaxRule()
		results := rule.Validate(&rules.ValidationContext{
			FilePath: "manifest.yaml",
			FileName: "manifest.yaml",
			Kind:     specs.KindImportManifest,
			Version:  specs.SpecVersionV1,
			Spec: map[string]any{
				"workspaces": []any{
					map[string]any{
						"resources": []any{
							map[string]any{
								"urn":       "source:my-source",
								"remote_id": "remote-456",
							},
						},
					},
				},
			},
		})

		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "missing required field 'workspace_id'")
		assert.Equal(t, "/spec/workspaces/0/workspace_id", results[0].Reference)
	})

	t.Run("invalid import ids - missing urn and remote_id", func(t *testing.T) {
		t.Parallel()

		rule := NewImportManifestSyntaxRule()
		results := rule.Validate(&rules.ValidationContext{
			FilePath: "manifest.yaml",
			FileName: "manifest.yaml",
			Kind:     specs.KindImportManifest,
			Version:  specs.SpecVersionV1,
			Spec: map[string]any{
				"workspaces": []any{
					map[string]any{
						"workspace_id": "ws-123",
						"resources": []any{
							map[string]any{},
						},
					},
				},
			},
		})

		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "invalid import resource")
		assert.Equal(t, "/spec/workspaces/0/resources/0", results[0].Reference)
	})

	t.Run("duplicate URNs within manifest", func(t *testing.T) {
		t.Parallel()

		rule := NewImportManifestSyntaxRule()
		results := rule.Validate(&rules.ValidationContext{
			FilePath: "manifest.yaml",
			FileName: "manifest.yaml",
			Kind:     specs.KindImportManifest,
			Version:  specs.SpecVersionV1,
			Spec: map[string]any{
				"workspaces": []any{
					map[string]any{
						"workspace_id": "ws-123",
						"resources": []any{
							map[string]any{
								"urn":       "source:my-source",
								"remote_id": "remote-1",
							},
							map[string]any{
								"urn":       "source:my-source",
								"remote_id": "remote-2",
							},
						},
					},
				},
			},
		})

		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "duplicate URN 'source:my-source'")
		assert.Equal(t, "/spec/workspaces/0/resources/1/urn", results[0].Reference)
	})

	t.Run("unknown fields rejected", func(t *testing.T) {
		t.Parallel()

		rule := NewImportManifestSyntaxRule()
		results := rule.Validate(&rules.ValidationContext{
			FilePath: "manifest.yaml",
			FileName: "manifest.yaml",
			Kind:     specs.KindImportManifest,
			Version:  specs.SpecVersionV1,
			Spec: map[string]any{
				"workspaces": []any{
					map[string]any{
						"workspace_id":  "ws-123",
						"unknown_field": "bad",
					},
				},
			},
		})

		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "manifest contains unknown or invalid fields")
		assert.Equal(t, "/spec", results[0].Reference)
	})
}
