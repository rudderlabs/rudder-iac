package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// findRule locates a rule by ID within the provided slice.
func findRule(t *testing.T, rules []vrules.Rule, id string) vrules.Rule {
	t.Helper()
	for _, r := range rules {
		if r.ID() == id {
			return r
		}
	}
	t.Fatalf("rule %q not found in rule set", id)
	return nil
}

// projectRule is a local alias so tests can type-assert without importing the
// vrules package symbol directly.
type projectRule interface {
	ValidateProject(allSpecs map[string]*vrules.ValidationContext) map[string][]vrules.ValidationResult
}

// validManifestSpec builds a well-formed spec.workspaces payload.
func validManifestSpec(workspaceID, urn, remoteID string) map[string]any {
	return map[string]any{
		"workspaces": []any{
			map[string]any{
				"workspace_id": workspaceID,
				"resources": []any{
					map[string]any{
						"urn":       urn,
						"remote_id": remoteID,
					},
				},
			},
		},
	}
}

// manifestCtx returns a ValidationContext for an import-manifest spec.
func manifestCtx(filePath string, spec map[string]any) *vrules.ValidationContext {
	return &vrules.ValidationContext{
		FilePath: filePath,
		FileName: filePath,
		Kind:     specs.KindImportManifest,
		Version:  specs.SpecVersionV1,
		Spec:     spec,
		Metadata: map[string]any{},
	}
}

// --- specShapeRule tests ---

func TestSpecShape_Valid(t *testing.T) {
	t.Parallel()

	rule := findRule(t, Syntactic(), "import-manifest/spec-shape")
	ctx := manifestCtx("manifest.yaml", validManifestSpec("ws-1", "source:my-src", "r-1"))

	results := rule.Validate(ctx)

	assert.Empty(t, results)
}

func TestSpecShape_MissingWorkspaces(t *testing.T) {
	t.Parallel()

	rule := findRule(t, Syntactic(), "import-manifest/spec-shape")
	ctx := manifestCtx("manifest.yaml", map[string]any{})

	results := rule.Validate(ctx)

	require.Len(t, results, 1)
	assert.Equal(t, "/spec/workspaces", results[0].Reference)
}

func TestSpecShape_MissingWorkspaceID(t *testing.T) {
	t.Parallel()

	rule := findRule(t, Syntactic(), "import-manifest/spec-shape")
	spec := map[string]any{
		"workspaces": []any{
			map[string]any{
				// workspace_id intentionally omitted
				"resources": []any{
					map[string]any{"urn": "source:my-src", "remote_id": "r-1"},
				},
			},
		},
	}
	ctx := manifestCtx("manifest.yaml", spec)

	results := rule.Validate(ctx)

	require.Len(t, results, 1)
	assert.Equal(t, "/spec/workspaces/0/workspace_id", results[0].Reference)
}

func TestSpecShape_MissingURN(t *testing.T) {
	t.Parallel()

	rule := findRule(t, Syntactic(), "import-manifest/spec-shape")
	spec := map[string]any{
		"workspaces": []any{
			map[string]any{
				"workspace_id": "ws-1",
				"resources": []any{
					map[string]any{
						// urn intentionally omitted
						"remote_id": "r-1",
					},
				},
			},
		},
	}
	ctx := manifestCtx("manifest.yaml", spec)

	results := rule.Validate(ctx)

	require.Len(t, results, 1)
	assert.Equal(t, "/spec/workspaces/0/resources/0/urn", results[0].Reference)
}

func TestSpecShape_MissingRemoteID(t *testing.T) {
	t.Parallel()

	rule := findRule(t, Syntactic(), "import-manifest/spec-shape")
	spec := map[string]any{
		"workspaces": []any{
			map[string]any{
				"workspace_id": "ws-1",
				"resources": []any{
					map[string]any{
						"urn": "source:my-src",
						// remote_id intentionally omitted
					},
				},
			},
		},
	}
	ctx := manifestCtx("manifest.yaml", spec)

	results := rule.Validate(ctx)

	require.Len(t, results, 1)
	assert.Equal(t, "/spec/workspaces/0/resources/0/remote_id", results[0].Reference)
}

func TestSpecShape_DuplicateURNWithinWorkspace(t *testing.T) {
	t.Parallel()

	rule := findRule(t, Syntactic(), "import-manifest/spec-shape")
	spec := map[string]any{
		"workspaces": []any{
			map[string]any{
				"workspace_id": "ws-1",
				"resources": []any{
					map[string]any{"urn": "source:my-src", "remote_id": "r-1"},
					map[string]any{"urn": "source:my-src", "remote_id": "r-2"},
				},
			},
		},
	}
	ctx := manifestCtx("manifest.yaml", spec)

	results := rule.Validate(ctx)

	require.Len(t, results, 1)
	assert.Equal(t, "/spec/workspaces/0/resources/1/urn", results[0].Reference)
	assert.Contains(t, results[0].Message, "source:my-src")
}

// --- urnUniqueRule tests ---

func TestURNUnique_NoDuplicates(t *testing.T) {
	t.Parallel()

	rule := findRule(t, Syntactic(), "import-manifest/urn-unique")
	pr, ok := rule.(projectRule)
	require.True(t, ok, "urnUniqueRule must implement ProjectRule")

	allSpecs := map[string]*vrules.ValidationContext{
		"file-a.yaml": manifestCtx("file-a.yaml", validManifestSpec("ws-1", "source:src-a", "r-1")),
		"file-b.yaml": manifestCtx("file-b.yaml", validManifestSpec("ws-1", "source:src-b", "r-2")),
	}

	results := pr.ValidateProject(allSpecs)

	assert.Empty(t, results)
}

func TestURNUnique_CrossFileDuplicate(t *testing.T) {
	t.Parallel()

	rule := findRule(t, Syntactic(), "import-manifest/urn-unique")
	pr, ok := rule.(projectRule)
	require.True(t, ok, "urnUniqueRule must implement ProjectRule")

	sharedURN := "source:shared-src"
	allSpecs := map[string]*vrules.ValidationContext{
		"file-a.yaml": manifestCtx("file-a.yaml", validManifestSpec("ws-1", sharedURN, "r-1")),
		"file-b.yaml": manifestCtx("file-b.yaml", validManifestSpec("ws-2", sharedURN, "r-2")),
	}

	results := pr.ValidateProject(allSpecs)

	require.Len(t, results["file-a.yaml"], 1)
	assert.Contains(t, results["file-a.yaml"][0].Message, sharedURN)
	require.Len(t, results["file-b.yaml"], 1)
	assert.Contains(t, results["file-b.yaml"][0].Message, sharedURN)
}

// --- inlineClashRule tests ---

func TestInlineClash_NoClash(t *testing.T) {
	t.Parallel()

	rule := findRule(t, Syntactic(), "import-manifest/inline-clash")
	pr, ok := rule.(projectRule)
	require.True(t, ok, "inlineClashRule must implement ProjectRule")

	allSpecs := map[string]*vrules.ValidationContext{
		"manifest.yaml": manifestCtx("manifest.yaml", validManifestSpec("ws-1", "source:manifest-src", "r-1")),
		"source.yaml": {
			FilePath: "source.yaml",
			FileName: "source.yaml",
			Kind:     "source",
			Version:  specs.SpecVersionV1,
			Spec:     map[string]any{"name": "my-source"},
			Metadata: map[string]any{
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "ws-1",
							"resources": []any{
								map[string]any{"urn": "source:inline-src", "remote_id": "r-2"},
							},
						},
					},
				},
			},
		},
	}

	results := pr.ValidateProject(allSpecs)

	assert.Empty(t, results)
}

func TestInlineClash_Clash(t *testing.T) {
	t.Parallel()

	rule := findRule(t, Syntactic(), "import-manifest/inline-clash")
	pr, ok := rule.(projectRule)
	require.True(t, ok, "inlineClashRule must implement ProjectRule")

	clashingURN := "source:my-src"
	allSpecs := map[string]*vrules.ValidationContext{
		"manifest.yaml": manifestCtx("manifest.yaml", validManifestSpec("ws-1", clashingURN, "r-1")),
		"source.yaml": {
			FilePath: "source.yaml",
			FileName: "source.yaml",
			Kind:     "source",
			Version:  specs.SpecVersionV1,
			Spec:     map[string]any{"name": "my-source"},
			Metadata: map[string]any{
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "ws-1",
							"resources": []any{
								map[string]any{"urn": clashingURN, "remote_id": "r-1"},
							},
						},
					},
				},
			},
		},
	}

	results := pr.ValidateProject(allSpecs)

	require.Len(t, results["source.yaml"], 1)
	assert.Equal(t, "/metadata/import/workspaces/0/resources/0/urn", results["source.yaml"][0].Reference)
	assert.Contains(t, results["source.yaml"][0].Message, clashingURN)
}
