package export_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler/export"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testRemote struct {
	id          string
	workspaceID string
}

func (r testRemote) Metadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:          r.id,
		WorkspaceID: r.workspaceID,
	}
}

type testSingleSpecHandler struct {
	meta handler.HandlerMetadata
}

func (h *testSingleSpecHandler) Metadata() handler.HandlerMetadata {
	return h.meta
}

func (h *testSingleSpecHandler) MapRemoteToSpec(
	_ map[string]*testRemote,
	_ resolver.ReferenceResolver,
) (*export.SpecExportData[map[string]any], error) {
	return &export.SpecExportData[map[string]any]{
		RelativePath: "test.yaml",
		Data:         &map[string]any{},
	}, nil
}

func TestSingleSpecExportStrategy_FormatForExport_WritesURN(t *testing.T) {
	t.Parallel()

	strategy := &export.SingleSpecExportStrategy[map[string]any, testRemote]{
		Handler: &testSingleSpecHandler{
			meta: handler.HandlerMetadata{
				SpecKind:         "test-kind",
				ResourceType:     "my-resource-type",
				SpecMetadataName: "test",
			},
		},
	}

	remotes := map[string]*testRemote{
		"local-id-1": {id: "remote-id-1", workspaceID: "ws-123"},
	}

	entities, err := strategy.FormatForExport(remotes, nil, nil)
	require.NoError(t, err)
	require.Len(t, entities, 1)

	spec, ok := entities[0].Content.(*specs.Spec)
	require.True(t, ok)

	importSection := spec.Metadata["import"].(map[string]any)
	workspaces := importSection["workspaces"].([]any)
	resource := workspaces[0].(map[string]any)["resources"].([]any)[0].(map[string]any)

	assert.Equal(t, "my-resource-type:local-id-1", resource["urn"], "import metadata must use urn, not local_id")
	assert.Empty(t, resource["local_id"], "local_id must not be set in exported metadata")
}
