package importmanifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

func TestParseWorkspaces(t *testing.T) {
	t.Parallel()

	t.Run("decodes workspaces", func(t *testing.T) {
		t.Parallel()

		s := &specs.Spec{
			Kind:    KindImportManifest,
			Version: specs.SpecVersionV1,
			Spec: map[string]any{
				"workspaces": []any{
					map[string]any{
						"workspace_id": "ws-1",
						"resources": []any{
							map[string]any{"urn": "source:src-1", "remote_id": "remote-1"},
							map[string]any{"urn": "destination:dst-1", "remote_id": "remote-2"},
						},
					},
				},
			},
		}

		got, err := parseWorkspaces(s)
		require.NoError(t, err)
		assert.Equal(t, []specs.WorkspaceImportMetadata{
			{
				WorkspaceID: "ws-1",
				Resources: []specs.ImportIds{
					{URN: "source:src-1", RemoteID: "remote-1"},
					{URN: "destination:dst-1", RemoteID: "remote-2"},
				},
			},
		}, got)
	})

	t.Run("empty workspaces", func(t *testing.T) {
		t.Parallel()

		s := &specs.Spec{Spec: map[string]any{"workspaces": []any{}}}
		got, err := parseWorkspaces(s)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("rejects unknown top-level field", func(t *testing.T) {
		t.Parallel()

		s := &specs.Spec{
			Spec: map[string]any{
				"workspaces": []any{},
				"unexpected": "value",
			},
		}
		_, err := parseWorkspaces(s)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decoding spec")
	})

	t.Run("rejects unknown nested field", func(t *testing.T) {
		t.Parallel()

		s := &specs.Spec{
			Spec: map[string]any{
				"workspaces": []any{
					map[string]any{
						"workspace_id": "ws-1",
						"resources": []any{
							map[string]any{
								"urn":       "source:src-1",
								"remote_id": "remote-1",
								"bogus":     "value",
							},
						},
					},
				},
			},
		}
		_, err := parseWorkspaces(s)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decoding spec")
	})
}
