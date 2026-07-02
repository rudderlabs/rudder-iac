package importmanifest

import (
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSpec(t *testing.T) {
	t.Run("nil when no entries", func(t *testing.T) {
		assert.Nil(t, BuildSpec(nil))
		assert.Nil(t, BuildSpec([]ImportEntry{}))
	})

	t.Run("groups entries by workspace, first-seen order", func(t *testing.T) {
		entries := []ImportEntry{
			{WorkspaceID: "ws-a", URN: "event:login", RemoteID: "r1"},
			{WorkspaceID: "ws-b", URN: "event:logout", RemoteID: "r2"},
			{WorkspaceID: "ws-a", URN: "property:email", RemoteID: "r3"},
		}
		spec := BuildSpec(entries)
		require.NotNil(t, spec)
		assert.Equal(t, KindImportManifest, spec.Kind)
		assert.Equal(t, specs.SpecVersionV1, spec.Version)
		assert.Equal(t, "import-manifest", spec.Metadata["name"])

		// Round-trips through the existing parser to the grouped shape.
		got, err := parseWorkspaces(spec)
		require.NoError(t, err)
		assert.Equal(t, []specs.WorkspaceImportMetadata{
			{
				WorkspaceID: "ws-a",
				Resources: []specs.ImportIds{
					{URN: "event:login", RemoteID: "r1"},
					{URN: "property:email", RemoteID: "r3"},
				},
			},
			{
				WorkspaceID: "ws-b",
				Resources:   []specs.ImportIds{{URN: "event:logout", RemoteID: "r2"}},
			},
		}, got)
	})
}

func TestBuildNode(t *testing.T) {
	t.Run("nil when no entries", func(t *testing.T) {
		node, err := BuildNode(nil)
		require.NoError(t, err)
		assert.Nil(t, node)
	})

	t.Run("carries the header comment and round-trips through the formatter", func(t *testing.T) {
		entries := []ImportEntry{
			{WorkspaceID: "ws-a", URN: "event:login", RemoteID: "r1"},
		}
		node, err := BuildNode(entries)
		require.NoError(t, err)
		require.NotNil(t, node)
		assert.Equal(t, ManifestHeaderComment, node.HeadComment)

		// The YAML formatter must preserve the head comment as leading '#' lines.
		out, err := formatter.YAMLFormatter{}.Format(node)
		require.NoError(t, err)
		yamlStr := string(out)
		assert.True(t, strings.HasPrefix(yamlStr, "#"), "manifest YAML should start with a comment, got:\n%s", yamlStr)
		assert.Contains(t, yamlStr, "Do not edit unless required")
		// String values are double-quoted by the formatter.
		assert.Contains(t, yamlStr, `kind: "import-manifest"`)
		assert.Contains(t, yamlStr, "workspace_id")
	})
}
