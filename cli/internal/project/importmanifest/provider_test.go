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

func TestProvider_ParseSpec_Empty(t *testing.T) {
	t.Parallel()

	parsed, err := New().ParseSpec("a.yaml", manifestSpec("ws-1"))
	require.NoError(t, err)
	assert.Equal(t, &specs.ParsedSpec{}, parsed)
}

func TestProvider_ResourceGraph_Empty(t *testing.T) {
	t.Parallel()

	graph, err := New().ResourceGraph()
	require.NoError(t, err)
	require.NotNil(t, graph)
	assert.Empty(t, graph.Resources())
}

func TestProvider_RuleSets_NilForNow(t *testing.T) {
	t.Parallel()

	p := New()
	assert.Nil(t, p.SyntacticRules())
	assert.Nil(t, p.SemanticRules())
}
