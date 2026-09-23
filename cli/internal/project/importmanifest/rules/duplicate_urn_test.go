package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest/manifestspec"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestDuplicateURNRule_AppliesTo(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]vrules.MatchPattern{vrules.MatchKindVersion(manifestspec.KindImportManifest, specs.SpecVersionV1)},
		NewManifestDuplicateURNRule().AppliesTo(),
	)
}

// manifestCtx builds a single-workspace manifest context for one file.
func manifestCtx(workspaceID string, urns ...string) *vrules.ValidationContext {
	resources := make([]any, 0, len(urns))
	for _, u := range urns {
		resources = append(resources, map[string]any{"urn": u, "remote_id": "r"})
	}
	return &vrules.ValidationContext{
		Kind:    manifestspec.KindImportManifest,
		Version: specs.SpecVersionV1,
		Spec: map[string]any{
			"workspaces": []any{
				map[string]any{"workspace_id": workspaceID, "resources": resources},
			},
		},
	}
}

func TestManifestDuplicateURNRule_ValidateSpecs(t *testing.T) {
	t.Parallel()

	asMultiSpec := func() vrules.MultiSpecRule {
		return NewManifestDuplicateURNRule().(vrules.MultiSpecRule)
	}

	t.Run("distinct urns across files are clean", func(t *testing.T) {
		t.Parallel()
		results := asMultiSpec().ValidateSpecs(map[string]*vrules.ValidationContext{
			"a.yaml": manifestCtx("ws-1", "source:src-1"),
			"b.yaml": manifestCtx("ws-1", "source:src-2"),
		})
		assert.Empty(t, results)
	})

	t.Run("same (workspace, urn) across two files errors at both", func(t *testing.T) {
		t.Parallel()
		results := asMultiSpec().ValidateSpecs(map[string]*vrules.ValidationContext{
			"a.yaml": manifestCtx("ws-1", "source:src-1"),
			"b.yaml": manifestCtx("ws-1", "source:src-1"),
		})
		require.Len(t, results, 2)
		assert.Equal(t, []vrules.ValidationResult{{
			Reference: "/spec/workspaces/0/resources/0/urn",
			Message:   "duplicate URN 'source:src-1' in workspace 'ws-1'",
		}}, results["a.yaml"])
		assert.Equal(t, []vrules.ValidationResult{{
			Reference: "/spec/workspaces/0/resources/0/urn",
			Message:   "duplicate URN 'source:src-1' in workspace 'ws-1'",
		}}, results["b.yaml"])
	})

	t.Run("same (workspace, urn) within one file errors at both occurrences", func(t *testing.T) {
		t.Parallel()
		results := asMultiSpec().ValidateSpecs(map[string]*vrules.ValidationContext{
			"a.yaml": manifestCtx("ws-1", "source:src-1", "source:src-1"),
		})
		assert.Equal(t, []vrules.ValidationResult{
			{Reference: "/spec/workspaces/0/resources/0/urn", Message: "duplicate URN 'source:src-1' in workspace 'ws-1'"},
			{Reference: "/spec/workspaces/0/resources/1/urn", Message: "duplicate URN 'source:src-1' in workspace 'ws-1'"},
		}, results["a.yaml"])
	})

	t.Run("same urn under different workspaces is clean", func(t *testing.T) {
		t.Parallel()
		results := asMultiSpec().ValidateSpecs(map[string]*vrules.ValidationContext{
			"a.yaml": manifestCtx("ws-1", "source:src-1"),
			"b.yaml": manifestCtx("ws-2", "source:src-1"),
		})
		assert.Empty(t, results)
	})

	t.Run("missing workspace_id is not grouped into a duplicate", func(t *testing.T) {
		t.Parallel()
		// Workspace blocks that omit workspace_id must not collide under the
		// ("", urn) key — the missing id is the spec-syntax-valid rule's concern.
		results := asMultiSpec().ValidateSpecs(map[string]*vrules.ValidationContext{
			"a.yaml": manifestCtx("", "source:src-1"),
			"b.yaml": manifestCtx("", "source:src-1"),
		})
		assert.Empty(t, results)
	})

	t.Run("Validate is a no-op", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, NewManifestDuplicateURNRule().Validate(manifestCtx("ws-1", "source:src-1")))
	})
}
