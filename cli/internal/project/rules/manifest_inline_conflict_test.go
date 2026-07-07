package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest/manifestspec"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const conflictResourceType = "event-stream-source"

var conflictPatterns = []rules.MatchPattern{
	rules.MatchKindVersion(manifestspec.KindImportManifest, specs.SpecVersionV1),
	rules.MatchKindVersion(conflictResourceType, specs.SpecVersionV1),
}

// resourceParseSpecLegacy reports the legacy resource type so inline local_id
// entries resolve to URNs.
func resourceParseSpecLegacy() ParseSpecFunc {
	return func(_ string, _ *specs.Spec) (*specs.ParsedSpec, error) {
		return &specs.ParsedSpec{LegacyResourceType: conflictResourceType}, nil
	}
}

// manifestConflictCtx builds a manifest declaring urn under workspaceID with the
// given remote_id.
func manifestConflictCtx(workspaceID, urn, remoteID string) *rules.ValidationContext {
	return &rules.ValidationContext{
		FilePath: "import-manifest.yaml",
		Kind:     manifestspec.KindImportManifest,
		Version:  specs.SpecVersionV1,
		Spec: map[string]any{
			"workspaces": []any{map[string]any{
				"workspace_id": workspaceID,
				"resources":    []any{map[string]any{"urn": urn, "remote_id": remoteID}},
			}},
		},
	}
}

// resourceCtxInlineURN builds a resource spec carrying an inline metadata.import
// urn under workspaceID with the given remote_id.
func resourceCtxInlineURN(path, workspaceID, urn, remoteID string) *rules.ValidationContext {
	return &rules.ValidationContext{
		FilePath: path,
		Kind:     conflictResourceType,
		Version:  specs.SpecVersionV1,
		Metadata: map[string]any{
			"name": "src",
			"import": map[string]any{
				"workspaces": []any{
					map[string]any{
						"workspace_id": workspaceID,
						"resources":    []any{map[string]any{"urn": urn, "remote_id": remoteID}},
					},
				},
			},
		},
		Spec: map[string]any{"id": "src"},
	}
}

func newRule() rules.MultiSpecRule {
	return NewManifestInlineConflictRule(
		resourceParseSpecLegacy(), conflictPatterns,
	).(rules.MultiSpecRule)
}

func TestManifestInlineConflictRule_Metadata(t *testing.T) {
	t.Parallel()
	r := NewManifestInlineConflictRule(resourceParseSpecLegacy(), conflictPatterns)
	assert.Equal(t, "project/manifest-inline-conflict", r.ID())
	assert.Equal(t, rules.Error, r.Severity())
	assert.Equal(t, conflictPatterns, r.AppliesTo())
	assert.Nil(t, r.Validate(manifestConflictCtx("ws-1", "a:b", "r")))
}

func TestManifestInlineConflictRule_ValidateSpecs(t *testing.T) {
	t.Parallel()

	t.Run("same (ws, urn) with differing remote_ids errors at both", func(t *testing.T) {
		t.Parallel()
		results := newRule().ValidateSpecs(map[string]*rules.ValidationContext{
			"import-manifest.yaml": manifestConflictCtx("ws-1", "event-stream-source:shared", "remote-a"),
			"src.yaml":             resourceCtxInlineURN("src.yaml", "ws-1", "event-stream-source:shared", "remote-b"),
		})
		msg := "URN 'event-stream-source:shared' in workspace 'ws-1' is defined in both an import-manifest and inline metadata with differing remote_ids; remove the inline metadata entry"
		assert.Equal(t, []rules.ValidationResult{
			{Reference: "/spec/workspaces/0/resources/0/urn", Message: msg},
		}, results["import-manifest.yaml"])
		assert.Equal(t, []rules.ValidationResult{
			{Reference: "/metadata/import/workspaces/0/resources/0/urn", Message: msg},
		}, results["src.yaml"])
	})

	t.Run("same (ws, urn) with matching remote_id is clean", func(t *testing.T) {
		t.Parallel()
		results := newRule().ValidateSpecs(map[string]*rules.ValidationContext{
			"import-manifest.yaml": manifestConflictCtx("ws-1", "event-stream-source:shared", "remote-a"),
			"src.yaml":             resourceCtxInlineURN("src.yaml", "ws-1", "event-stream-source:shared", "remote-a"),
		})
		assert.Empty(t, results)
	})

	t.Run("manifest-only urn is clean", func(t *testing.T) {
		t.Parallel()
		results := newRule().ValidateSpecs(map[string]*rules.ValidationContext{
			"import-manifest.yaml": manifestConflictCtx("ws-1", "event-stream-source:only-manifest", "remote-a"),
			"src.yaml":             resourceCtxInlineURN("src.yaml", "ws-1", "event-stream-source:only-inline", "remote-b"),
		})
		assert.Empty(t, results)
	})

	t.Run("inline local_id resolving to a manifest urn conflicts when remote_ids differ", func(t *testing.T) {
		t.Parallel()
		// inline entry uses local_id "shared"; the resource legacy type is
		// event-stream-source, so it resolves to event-stream-source:shared,
		// which the manifest also declares under the same workspace.
		inline := &rules.ValidationContext{
			FilePath: "src.yaml",
			Kind:     conflictResourceType,
			Version:  specs.SpecVersionV1,
			Metadata: map[string]any{
				"name": "src",
				"import": map[string]any{
					"workspaces": []any{map[string]any{
						"workspace_id": "ws-1",
						"resources":    []any{map[string]any{"local_id": "shared", "remote_id": "remote-b"}},
					}},
				},
			},
			Spec: map[string]any{"id": "src"},
		}
		results := newRule().ValidateSpecs(map[string]*rules.ValidationContext{
			"import-manifest.yaml": manifestConflictCtx("ws-1", "event-stream-source:shared", "remote-a"),
			"src.yaml":             inline,
		})
		require.Len(t, results, 2)
		assert.Equal(t, "/metadata/import/workspaces/0/resources/0/urn", results["src.yaml"][0].Reference)
	})

	t.Run("same urn under different workspaces is clean (workspace-scoped)", func(t *testing.T) {
		t.Parallel()
		// manifest workspace is ws-1, inline workspace is ws-9 — different
		// (workspace_id, urn) keys, so no conflict even though remote_ids differ.
		results := newRule().ValidateSpecs(map[string]*rules.ValidationContext{
			"import-manifest.yaml": manifestConflictCtx("ws-1", "event-stream-source:shared", "remote-a"),
			"src.yaml":             resourceCtxInlineURN("src.yaml", "ws-9", "event-stream-source:shared", "remote-b"),
		})
		assert.Empty(t, results)
	})
}
