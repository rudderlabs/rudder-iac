package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportManifestProjectRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewImportManifestProjectRule()

	assert.Equal(t, "project/import-manifest-cross-file", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, []rules.MatchPattern{rules.MatchAll()}, rule.AppliesTo())
	assert.Nil(t, rule.Validate(nil))
}

func TestImportManifestProjectRule_ValidateProject(t *testing.T) {
	t.Parallel()

	t.Run("no manifests produces no violations", func(t *testing.T) {
		t.Parallel()

		rule := NewImportManifestProjectRule()
		pr := rule.(rules.ProjectRule)

		results := pr.ValidateProject(map[string]*rules.ValidationContext{
			"props.yaml": {
				Kind: "properties",
				Spec: map[string]any{
					"properties": []any{map[string]any{"id": "email"}},
				},
			},
		})

		assert.Empty(t, results)
	})

	t.Run("single manifest with no cross-file issues", func(t *testing.T) {
		t.Parallel()

		rule := NewImportManifestProjectRule()
		pr := rule.(rules.ProjectRule)

		results := pr.ValidateProject(map[string]*rules.ValidationContext{
			"manifest.yaml": {
				Kind: specs.KindImportManifest,
				Spec: map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "ws-1",
							"resources": []any{
								map[string]any{"urn": "source:a", "remote_id": "r1"},
							},
						},
					},
				},
			},
			"props.yaml": {
				Kind:     "properties",
				Metadata: map[string]any{},
				Spec:     map[string]any{},
			},
		})

		assert.Empty(t, results)
	})

	t.Run("duplicate URN across two manifest files", func(t *testing.T) {
		t.Parallel()

		rule := NewImportManifestProjectRule()
		pr := rule.(rules.ProjectRule)

		results := pr.ValidateProject(map[string]*rules.ValidationContext{
			"manifest-a.yaml": {
				Kind: specs.KindImportManifest,
				Spec: map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "ws-1",
							"resources": []any{
								map[string]any{"urn": "source:dup", "remote_id": "r1"},
							},
						},
					},
				},
			},
			"manifest-b.yaml": {
				Kind:     specs.KindImportManifest,
				FileName: "manifest-b.yaml",
				Spec: map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "ws-2",
							"resources": []any{
								map[string]any{"urn": "source:dup", "remote_id": "r2"},
							},
						},
					},
				},
			},
		})

		// One of the two files should have a violation
		totalViolations := 0
		for _, fileResults := range results {
			totalViolations += len(fileResults)
		}
		require.Equal(t, 1, totalViolations)

		// Find the violation and verify message
		for _, fileResults := range results {
			for _, r := range fileResults {
				assert.Contains(t, r.Message, "source:dup")
				assert.Contains(t, r.Message, "defined in both")
			}
		}
	})

	t.Run("URN in both manifest and inline metadata", func(t *testing.T) {
		t.Parallel()

		rule := NewImportManifestProjectRule()
		pr := rule.(rules.ProjectRule)

		results := pr.ValidateProject(map[string]*rules.ValidationContext{
			"manifest.yaml": {
				Kind: specs.KindImportManifest,
				Spec: map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "ws-1",
							"resources": []any{
								map[string]any{"urn": "source:shared", "remote_id": "r1"},
							},
						},
					},
				},
			},
			"source.yaml": {
				Kind: "event-stream-source",
				Metadata: map[string]any{
					"import": map[string]any{
						"workspaces": []any{
							map[string]any{
								"workspace_id": "ws-1",
								"resources": []any{
									map[string]any{"urn": "source:shared", "remote_id": "r2"},
								},
							},
						},
					},
				},
				Spec: map[string]any{},
			},
		})

		require.Contains(t, results, "manifest.yaml")
		require.Len(t, results["manifest.yaml"], 1)
		assert.Contains(t, results["manifest.yaml"][0].Message, "source:shared")
		assert.Contains(t, results["manifest.yaml"][0].Message, "inline metadata")
	})
}
