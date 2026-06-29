package rules

import (
	"fmt"
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

// manifestParseSpecExtracting returns a ParseSpecFunc that emits the manifest's
// urn entries (mirrors the real manifest provider's ParseSpec).
func manifestParseSpecExtracting() ParseSpecFunc {
	return func(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
		var urns []specs.URNEntry
		ws, _ := s.Spec["workspaces"].([]any)
		for wi, w := range ws {
			wm, _ := w.(map[string]any)
			res, _ := wm["resources"].([]any)
			for ri, r := range res {
				rm, _ := r.(map[string]any)
				if urn, ok := rm["urn"].(string); ok && urn != "" {
					urns = append(urns, specs.URNEntry{
						URN:             urn,
						JSONPointerPath: fmt.Sprintf("/spec/workspaces/%d/resources/%d/urn", wi, ri),
					})
				}
			}
		}
		return &specs.ParsedSpec{URNs: urns}, nil
	}
}

// resourceParseSpecLegacy reports the legacy resource type so inline local_id
// entries resolve to URNs.
func resourceParseSpecLegacy() ParseSpecFunc {
	return func(_ string, _ *specs.Spec) (*specs.ParsedSpec, error) {
		return &specs.ParsedSpec{LegacyResourceType: conflictResourceType}, nil
	}
}

func manifestConflictCtx(urns ...string) *rules.ValidationContext {
	res := make([]any, 0, len(urns))
	for _, u := range urns {
		res = append(res, map[string]any{"urn": u, "remote_id": "r"})
	}
	return &rules.ValidationContext{
		FilePath: "import-manifest.yaml",
		Kind:     manifestspec.KindImportManifest,
		Version:  specs.SpecVersionV1,
		Spec: map[string]any{
			"workspaces": []any{map[string]any{"workspace_id": "ws-1", "resources": res}},
		},
	}
}

// resourceCtxInlineURN builds a resource spec carrying an inline metadata.import urn.
func resourceCtxInlineURN(path, urn string) *rules.ValidationContext {
	return &rules.ValidationContext{
		FilePath: path,
		Kind:     conflictResourceType,
		Version:  specs.SpecVersionV1,
		Metadata: map[string]any{
			"name": "src",
			"import": map[string]any{
				"workspaces": []any{
					map[string]any{
						"workspace_id": "ws-9",
						"resources":    []any{map[string]any{"urn": urn, "remote_id": "r"}},
					},
				},
			},
		},
		Spec: map[string]any{"id": "src"},
	}
}

func newRule() rules.MultiSpecRule {
	return NewManifestInlineConflictRule(
		manifestParseSpecExtracting(), resourceParseSpecLegacy(), conflictPatterns,
	).(rules.MultiSpecRule)
}

func TestManifestInlineConflictRule_Metadata(t *testing.T) {
	t.Parallel()
	r := NewManifestInlineConflictRule(manifestParseSpecExtracting(), resourceParseSpecLegacy(), conflictPatterns)
	assert.Equal(t, "project/manifest-inline-conflict", r.ID())
	assert.Equal(t, rules.Error, r.Severity())
	assert.Equal(t, conflictPatterns, r.AppliesTo())
	assert.Nil(t, r.Validate(manifestConflictCtx("a:b")))
}

func TestManifestInlineConflictRule_ValidateSpecs(t *testing.T) {
	t.Parallel()

	t.Run("urn in both manifest and inline errors at both", func(t *testing.T) {
		t.Parallel()
		results := newRule().ValidateSpecs(map[string]*rules.ValidationContext{
			"import-manifest.yaml": manifestConflictCtx("event-stream-source:shared"),
			"src.yaml":             resourceCtxInlineURN("src.yaml", "event-stream-source:shared"),
		})
		msg := "URN 'event-stream-source:shared' is defined in both an import-manifest and inline metadata; remove the inline metadata entry"
		assert.Equal(t, []rules.ValidationResult{
			{Reference: "/spec/workspaces/0/resources/0/urn", Message: msg},
		}, results["import-manifest.yaml"])
		assert.Equal(t, []rules.ValidationResult{
			{Reference: "/metadata/import/workspaces/0/resources/0/urn", Message: msg},
		}, results["src.yaml"])
	})

	t.Run("manifest-only urn is clean", func(t *testing.T) {
		t.Parallel()
		results := newRule().ValidateSpecs(map[string]*rules.ValidationContext{
			"import-manifest.yaml": manifestConflictCtx("event-stream-source:only-manifest"),
			"src.yaml":             resourceCtxInlineURN("src.yaml", "event-stream-source:only-inline"),
		})
		assert.Empty(t, results)
	})

	t.Run("inline local_id resolving to a manifest urn conflicts", func(t *testing.T) {
		t.Parallel()
		// inline entry uses local_id "shared"; the resource legacy type is
		// event-stream-source, so it resolves to event-stream-source:shared,
		// which the manifest also declares.
		inline := &rules.ValidationContext{
			FilePath: "src.yaml",
			Kind:     conflictResourceType,
			Version:  specs.SpecVersionV1,
			Metadata: map[string]any{
				"name": "src",
				"import": map[string]any{
					"workspaces": []any{map[string]any{
						"workspace_id": "ws-9",
						"resources":    []any{map[string]any{"local_id": "shared", "remote_id": "r"}},
					}},
				},
			},
			Spec: map[string]any{"id": "src"},
		}
		results := newRule().ValidateSpecs(map[string]*rules.ValidationContext{
			"import-manifest.yaml": manifestConflictCtx("event-stream-source:shared"),
			"src.yaml":             inline,
		})
		require.Len(t, results, 2)
		assert.Equal(t, "/metadata/import/workspaces/0/resources/0/urn", results["src.yaml"][0].Reference)
	})

	t.Run("conflict holds even when workspace_ids differ (workspace-agnostic)", func(t *testing.T) {
		t.Parallel()
		// manifest workspace is ws-1, inline workspace is ws-9 — still a conflict.
		results := newRule().ValidateSpecs(map[string]*rules.ValidationContext{
			"import-manifest.yaml": manifestConflictCtx("event-stream-source:shared"),
			"src.yaml":             resourceCtxInlineURN("src.yaml", "event-stream-source:shared"),
		})
		require.Len(t, results, 2)
	})
}
