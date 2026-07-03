package importmanifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func manifestSpec(workspaceID string, ids ...specs.ImportIds) *specs.Spec {
	resources := make([]any, 0, len(ids))
	for _, id := range ids {
		resources = append(resources, map[string]any{
			"urn":       id.URN,
			"remote_id": id.RemoteID,
		})
	}
	return &specs.Spec{
		Kind:    KindImportManifest,
		Version: specs.SpecVersionV1,
		Spec: map[string]any{
			"workspaces": []any{
				map[string]any{
					"workspace_id": workspaceID,
					"resources":    resources,
				},
			},
		},
	}
}

func TestProvider_TypeSurfaces(t *testing.T) {
	t.Parallel()

	p := New()
	assert.Equal(t, []string{KindImportManifest}, p.SupportedKinds())
	assert.Nil(t, p.SupportedTypes())
	assert.Equal(t, []rules.MatchPattern{
		rules.MatchKindVersion(KindImportManifest, specs.SpecVersionV1),
	}, p.SupportedMatchPatterns())
}

func TestProvider_LoadSpec_RejectsMalformed(t *testing.T) {
	t.Parallel()

	p := New()
	err := p.LoadSpec("bad.yaml", &specs.Spec{
		Spec: map[string]any{"workspaces": []any{}, "unexpected": "value"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing import-manifest bad.yaml")
}

func TestProvider_LoadLegacySpec_Unsupported(t *testing.T) {
	t.Parallel()

	err := New().LoadLegacySpec("legacy.yaml", &specs.Spec{Version: specs.SpecVersionV0_1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support legacy version")
}

func TestProvider_ParseSpec_ExtractsURNs(t *testing.T) {
	t.Parallel()

	spec := manifestSpec("ws-1",
		specs.ImportIds{URN: "source:src-1", RemoteID: "remote-1"},
		specs.ImportIds{URN: "destination:dst-1", RemoteID: "remote-2"},
	)

	parsed, err := New().ParseSpec("a.yaml", spec)
	require.NoError(t, err)
	assert.Equal(t, &specs.ParsedSpec{
		URNs: []specs.URNEntry{
			{URN: "source:src-1", JSONPointerPath: "/spec/workspaces/0/resources/0/urn"},
			{URN: "destination:dst-1", JSONPointerPath: "/spec/workspaces/0/resources/1/urn"},
		},
	}, parsed)
}

func TestProvider_ParseSpec_MultiWorkspace(t *testing.T) {
	t.Parallel()

	spec := &specs.Spec{
		Kind:    KindImportManifest,
		Version: specs.SpecVersionV1,
		Spec: map[string]any{
			"workspaces": []any{
				map[string]any{
					"workspace_id": "ws-1",
					"resources":    []any{map[string]any{"urn": "source:src-1", "remote_id": "r1"}},
				},
				map[string]any{
					"workspace_id": "ws-2",
					"resources":    []any{map[string]any{"urn": "source:src-2", "remote_id": "r2"}},
				},
			},
		},
	}

	parsed, err := New().ParseSpec("a.yaml", spec)
	require.NoError(t, err)
	assert.Equal(t, &specs.ParsedSpec{
		URNs: []specs.URNEntry{
			{URN: "source:src-1", JSONPointerPath: "/spec/workspaces/0/resources/0/urn"},
			{URN: "source:src-2", JSONPointerPath: "/spec/workspaces/1/resources/0/urn"},
		},
	}, parsed)
}

func TestProvider_ResourceGraph_Empty(t *testing.T) {
	t.Parallel()

	graph, err := New().ResourceGraph()
	require.NoError(t, err)
	require.NotNil(t, graph)
	assert.Empty(t, graph.Resources())
}

func TestProvider_RuleSets(t *testing.T) {
	t.Parallel()

	p := New()

	syntactic := p.SyntacticRules()
	ids := make([]string, 0, len(syntactic))
	for _, r := range syntactic {
		ids = append(ids, r.ID())
	}
	assert.Equal(t, []string{
		"import-manifest/spec-syntax-valid",
		"import-manifest/duplicate-urn",
	}, ids)

	semantic := p.SemanticRules()
	semanticIDs := make([]string, 0, len(semantic))
	for _, r := range semantic {
		semanticIDs = append(semanticIDs, r.ID())
	}
	assert.Equal(t, []string{"import-manifest/orphaned-urn"}, semanticIDs)
}

func TestProvider_ImportManifest(t *testing.T) {
	t.Parallel()

	wsA := specs.WorkspaceImportMetadata{
		WorkspaceID: "ws-a",
		Resources:   []specs.ImportIds{{URN: "event:login", RemoteID: "rem-a"}},
	}
	wsB := specs.WorkspaceImportMetadata{
		WorkspaceID: "ws-b",
		Resources:   []specs.ImportIds{{URN: "event:login", RemoteID: "rem-b"}},
	}

	t.Run("empty when no entries loaded", func(t *testing.T) {
		p := New()
		assert.Empty(t, p.ImportManifest())
	})

	t.Run("returns all loaded workspace entries unscoped", func(t *testing.T) {
		p := &Provider{entries: []specs.WorkspaceImportMetadata{wsA, wsB}}
		assert.Equal(t,
			[]specs.WorkspaceImportMetadata{wsA, wsB},
			p.ImportManifest(),
		)
	})
}
