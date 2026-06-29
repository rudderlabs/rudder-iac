package rules

import (
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest/manifestspec"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

// graphWith builds a resource graph containing the given URNs. Each urn is
// "<type>:<id>"; NewResource recomputes URN(id, type), so we split to feed it the
// bare id and type and land on the same urn.
func graphWith(urns ...string) *resources.Graph {
	g := resources.NewGraph()
	for _, urn := range urns {
		typ, id, _ := strings.Cut(urn, ":")
		g.AddResource(resources.NewResource(id, typ, map[string]any{}, nil))
	}
	return g
}

// orphanCtx builds a manifest semantic context: one workspace block per (id, urns).
func orphanCtx(graph *resources.Graph, workspaceID string, blocks ...workspaceBlock) *vrules.ValidationContext {
	ws := make([]any, 0, len(blocks))
	for _, b := range blocks {
		res := make([]any, 0, len(b.urns))
		for _, u := range b.urns {
			res = append(res, map[string]any{"urn": u, "remote_id": "r"})
		}
		ws = append(ws, map[string]any{"workspace_id": b.id, "resources": res})
	}
	return &vrules.ValidationContext{
		FilePath:    "import-manifest.yaml",
		Kind:        manifestspec.KindImportManifest,
		Version:     specs.SpecVersionV1,
		Spec:        map[string]any{"workspaces": ws},
		Graph:       graph,
		WorkspaceID: workspaceID,
	}
}

type workspaceBlock struct {
	id   string
	urns []string
}

func TestOrphanedURNRule_Metadata(t *testing.T) {
	t.Parallel()
	r := NewOrphanedURNRule()
	assert.Equal(t, "import-manifest/orphaned-urn", r.ID())
	assert.Equal(t, vrules.Error, r.Severity())
	assert.Equal(t,
		[]vrules.MatchPattern{vrules.MatchKindVersion(manifestspec.KindImportManifest, specs.SpecVersionV1)},
		r.AppliesTo(),
	)
}

func TestOrphanedURNRule_Validate(t *testing.T) {
	t.Parallel()

	t.Run("urn present in graph is clean", func(t *testing.T) {
		t.Parallel()
		ctx := orphanCtx(graphWith("event-stream-source:my-src"), "ws-1",
			workspaceBlock{"ws-1", []string{"event-stream-source:my-src"}})
		assert.Empty(t, NewOrphanedURNRule().Validate(ctx))
	})

	t.Run("urn absent from graph is an orphan error", func(t *testing.T) {
		t.Parallel()
		ctx := orphanCtx(graphWith(), "ws-1",
			workspaceBlock{"ws-1", []string{"event-stream-source:missing"}})
		assert.Equal(t, []vrules.ValidationResult{{
			Reference: "/spec/workspaces/0/resources/0/urn",
			Message:   "manifest URN 'event-stream-source:missing' does not match any resource in the project",
		}}, NewOrphanedURNRule().Validate(ctx))
	})

	t.Run("orphan under a non-active workspace is ignored", func(t *testing.T) {
		t.Parallel()
		// Applying to ws-1; ws-2 references a urn absent from the graph, but ws-2
		// is not the active workspace, so it must not be flagged.
		ctx := orphanCtx(graphWith("event-stream-source:in-dev"), "ws-1",
			workspaceBlock{"ws-1", []string{"event-stream-source:in-dev"}},
			workspaceBlock{"ws-2", []string{"event-stream-source:prod-only"}},
		)
		assert.Empty(t, NewOrphanedURNRule().Validate(ctx))
	})

	t.Run("empty workspaceID checks all workspaces", func(t *testing.T) {
		t.Parallel()
		ctx := orphanCtx(graphWith(), "",
			workspaceBlock{"ws-1", []string{"event-stream-source:a"}},
			workspaceBlock{"ws-2", []string{"event-stream-source:b"}},
		)
		results := NewOrphanedURNRule().Validate(ctx)
		assert.Len(t, results, 2)
	})

	t.Run("no graph returns nil", func(t *testing.T) {
		t.Parallel()
		ctx := orphanCtx(nil, "ws-1", workspaceBlock{"ws-1", []string{"event-stream-source:x"}})
		assert.Nil(t, NewOrphanedURNRule().Validate(ctx))
	})
}
