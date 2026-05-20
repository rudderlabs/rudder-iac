package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportManifestSemanticRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewImportManifestSemanticRule()

	assert.Equal(t, "project/import-manifest-semantic", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, []rules.MatchPattern{rules.MatchKind(specs.KindImportManifest)}, rule.AppliesTo())
}

func TestImportManifestSemanticRule_Validate(t *testing.T) {
	t.Parallel()

	t.Run("no graph available returns nil", func(t *testing.T) {
		t.Parallel()

		rule := NewImportManifestSemanticRule()
		results := rule.Validate(&rules.ValidationContext{
			Kind: specs.KindImportManifest,
			Spec: map[string]any{
				"workspaces": []any{
					map[string]any{
						"workspace_id": "ws-1",
						"resources": []any{
							map[string]any{"urn": "source:orphan", "remote_id": "r1"},
						},
					},
				},
			},
			Graph: nil,
		})

		assert.Nil(t, results)
	})

	t.Run("all URNs exist in graph", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("my-source", "source", nil, nil))

		rule := NewImportManifestSemanticRule()
		results := rule.Validate(&rules.ValidationContext{
			FilePath: "manifest.yaml",
			FileName: "manifest.yaml",
			Kind:     specs.KindImportManifest,
			Spec: map[string]any{
				"workspaces": []any{
					map[string]any{
						"workspace_id": "ws-1",
						"resources": []any{
							map[string]any{"urn": "source:my-source", "remote_id": "r1"},
						},
					},
				},
			},
			Graph: graph,
		})

		assert.Empty(t, results)
	})

	t.Run("orphaned URN not in graph", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		rule := NewImportManifestSemanticRule()
		results := rule.Validate(&rules.ValidationContext{
			FilePath: "manifest.yaml",
			FileName: "manifest.yaml",
			Kind:     specs.KindImportManifest,
			Spec: map[string]any{
				"workspaces": []any{
					map[string]any{
						"workspace_id": "ws-1",
						"resources": []any{
							map[string]any{"urn": "source:orphan", "remote_id": "r1"},
						},
					},
				},
			},
			Graph: graph,
		})

		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "source:orphan")
		assert.Contains(t, results[0].Message, "does not match any resource")
		assert.Equal(t, "/spec/workspaces/0/resources/0/urn", results[0].Reference)
	})
}
