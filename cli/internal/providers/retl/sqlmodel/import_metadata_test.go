package sqlmodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// Import metadata used to live in a package-level map, so two providers — two
// workspaces in one process, or two parallel tests — shared and raced on it.
func TestImportMetadataIsPerHandler(t *testing.T) {
	t.Parallel()

	first, second := NewHandler(nil, "retl"), NewHandler(nil, "retl")
	urn := resources.URN("orders", ResourceType)
	require.NoError(t, first.LoadImportMetadata(&specs.WorkspacesImportMetadata{
		Workspaces: []specs.WorkspaceImportMetadata{{
			WorkspaceID: "ws-1",
			Resources:   []specs.ImportIds{{URN: urn, RemoteID: "src-1"}},
		}},
	}))

	assert.Equal(t, map[string]*ImportResourceInfo{urn: {WorkspaceId: "ws-1", RemoteId: "src-1"}}, first.importMetadata)
	assert.Empty(t, second.importMetadata)
}
