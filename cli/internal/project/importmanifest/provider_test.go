package importmanifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

func TestProvider_SupportedKinds(t *testing.T) {
	t.Parallel()

	p := New()
	assert.Equal(t, []string{specs.KindImportManifest}, p.SupportedKinds())
}

func TestProvider_ImportManifest_NilWhenEmpty(t *testing.T) {
	t.Parallel()

	p := New()
	assert.Nil(t, p.ImportManifest())
}

func TestProvider_LoadSpec_Appends(t *testing.T) {
	t.Parallel()

	p := New()

	spec1 := &specs.Spec{
		Version: specs.SpecVersionV1,
		Kind:    specs.KindImportManifest,
		Spec: map[string]any{
			"workspaces": []any{
				map[string]any{
					"workspace_id": "ws-1",
					"resources": []any{
						map[string]any{"urn": "source:my-src", "remote_id": "r-1"},
					},
				},
			},
		},
	}

	spec2 := &specs.Spec{
		Version: specs.SpecVersionV1,
		Kind:    specs.KindImportManifest,
		Spec: map[string]any{
			"workspaces": []any{
				map[string]any{
					"workspace_id": "ws-2",
					"resources": []any{
						map[string]any{"urn": "destination:my-dst", "remote_id": "r-2"},
					},
				},
			},
		},
	}

	require.NoError(t, p.LoadSpec("manifest1.yaml", spec1))
	require.NoError(t, p.LoadSpec("manifest2.yaml", spec2))

	got := p.ImportManifest()
	require.NotNil(t, got)
	assert.Equal(t, &specs.WorkspacesImportMetadata{
		Workspaces: []specs.WorkspaceImportMetadata{
			{
				WorkspaceID: "ws-1",
				Resources: []specs.ImportIds{
					{URN: "source:my-src", RemoteID: "r-1"},
				},
			},
			{
				WorkspaceID: "ws-2",
				Resources: []specs.ImportIds{
					{URN: "destination:my-dst", RemoteID: "r-2"},
				},
			},
		},
	}, got)
}

func TestProvider_LoadLegacySpec_Errors(t *testing.T) {
	t.Parallel()

	p := New()
	err := p.LoadLegacySpec("manifest.yaml", &specs.Spec{
		Version: "rudder/0.1",
		Kind:    specs.KindImportManifest,
	})
	assert.Error(t, err)
}

func TestProvider_ResourceGraph_Empty(t *testing.T) {
	t.Parallel()

	p := New()
	graph, err := p.ResourceGraph()
	require.NoError(t, err)
	require.NotNil(t, graph)
	assert.Empty(t, graph.Resources())
}
