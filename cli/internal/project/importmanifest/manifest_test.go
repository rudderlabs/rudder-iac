package importmanifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

func TestParseWorkspaces_HappyPath(t *testing.T) {
	t.Parallel()

	s := &specs.Spec{
		Version: specs.SpecVersionV1,
		Kind:    specs.KindImportManifest,
		Metadata: map[string]any{
			"name": "import-manifest",
		},
		Spec: map[string]any{
			"workspaces": []any{
				map[string]any{
					"workspace_id": "ws-aaa",
					"resources": []any{
						map[string]any{"urn": "source:my-source", "remote_id": "rs-001"},
						map[string]any{"urn": "destination:my-dest", "remote_id": "rd-002"},
					},
				},
				map[string]any{
					"workspace_id": "ws-bbb",
					"resources": []any{
						map[string]any{"urn": "tracking-plan:tp-1", "remote_id": "rtp-003"},
					},
				},
			},
		},
	}

	got, err := parseWorkspaces(s)
	require.NoError(t, err)
	require.Len(t, got, 2)

	assert.Equal(t, []specs.WorkspaceImportMetadata{
		{
			WorkspaceID: "ws-aaa",
			Resources: []specs.ImportIds{
				{URN: "source:my-source", RemoteID: "rs-001"},
				{URN: "destination:my-dest", RemoteID: "rd-002"},
			},
		},
		{
			WorkspaceID: "ws-bbb",
			Resources: []specs.ImportIds{
				{URN: "tracking-plan:tp-1", RemoteID: "rtp-003"},
			},
		},
	}, got)
}

func TestParseWorkspaces_RejectsUnknownFields(t *testing.T) {
	t.Parallel()

	s := &specs.Spec{
		Version: specs.SpecVersionV1,
		Kind:    specs.KindImportManifest,
		Metadata: map[string]any{
			"name": "import-manifest",
		},
		Spec: map[string]any{
			"workspaces": []any{
				map[string]any{
					"workspace_id": "ws-aaa",
					"resources":    []any{},
					"unknown_key":  "this-should-fail",
				},
			},
		},
	}

	got, err := parseWorkspaces(s)
	require.Error(t, err)
	assert.ErrorContains(t, err, "decoding manifest spec")
	assert.Nil(t, got)
}

func TestBuildSpec_DeterministicOutput(t *testing.T) {
	t.Parallel()

	// Two different orderings of the same entries.
	entriesA := []ImportEntry{
		{WorkspaceID: "ws-bbb", URN: "source:src-2", RemoteID: "rs-200"},
		{WorkspaceID: "ws-aaa", URN: "destination:dst-b", RemoteID: "rd-102"},
		{WorkspaceID: "ws-bbb", URN: "destination:dst-1", RemoteID: "rd-101"},
		{WorkspaceID: "ws-aaa", URN: "source:src-a", RemoteID: "rs-100"},
	}
	entriesB := []ImportEntry{
		{WorkspaceID: "ws-aaa", URN: "source:src-a", RemoteID: "rs-100"},
		{WorkspaceID: "ws-bbb", URN: "destination:dst-1", RemoteID: "rd-101"},
		{WorkspaceID: "ws-aaa", URN: "destination:dst-b", RemoteID: "rd-102"},
		{WorkspaceID: "ws-bbb", URN: "source:src-2", RemoteID: "rs-200"},
	}

	specA := BuildSpec(entriesA)
	specB := BuildSpec(entriesB)

	require.Equal(t, specA, specB)

	// Assert workspace order is deterministic (ws-aaa before ws-bbb).
	workspacesA, ok := specA.Spec["workspaces"].([]specs.WorkspaceImportMetadata)
	require.True(t, ok)
	require.Len(t, workspacesA, 2)
	assert.Equal(t, "ws-aaa", workspacesA[0].WorkspaceID)
	assert.Equal(t, "ws-bbb", workspacesA[1].WorkspaceID)

	// Assert URNs within each workspace are sorted.
	assert.Equal(t, "destination:dst-b", workspacesA[0].Resources[0].URN)
	assert.Equal(t, "source:src-a", workspacesA[0].Resources[1].URN)
	assert.Equal(t, "destination:dst-1", workspacesA[1].Resources[0].URN)
	assert.Equal(t, "source:src-2", workspacesA[1].Resources[1].URN)
}

func TestBuildSpec_GroupsByWorkspace(t *testing.T) {
	t.Parallel()

	entries := []ImportEntry{
		{WorkspaceID: "ws-1", URN: "source:s1", RemoteID: "r1"},
		{WorkspaceID: "ws-2", URN: "source:s2", RemoteID: "r2"},
		{WorkspaceID: "ws-1", URN: "destination:d1", RemoteID: "r3"},
	}

	result := BuildSpec(entries)
	require.NotNil(t, result)

	workspaces, ok := result.Spec["workspaces"].([]specs.WorkspaceImportMetadata)
	require.True(t, ok)
	require.Len(t, workspaces, 2)

	assert.Equal(t, "ws-1", workspaces[0].WorkspaceID)
	assert.Len(t, workspaces[0].Resources, 2)

	assert.Equal(t, "ws-2", workspaces[1].WorkspaceID)
	assert.Len(t, workspaces[1].Resources, 1)
}

func TestBuildSpec_SpecShape(t *testing.T) {
	t.Parallel()

	entries := []ImportEntry{
		{WorkspaceID: "ws-x", URN: "source:s1", RemoteID: "r1"},
	}

	result := BuildSpec(entries)
	require.NotNil(t, result)

	assert.Equal(t, specs.SpecVersionV1, result.Version)
	assert.Equal(t, specs.KindImportManifest, result.Kind)
	assert.Equal(t, map[string]any{"name": "import-manifest"}, result.Metadata)
	assert.Contains(t, result.Spec, "workspaces")
}
