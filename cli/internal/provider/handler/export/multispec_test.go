package export_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler/export"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testMultiSpecHandler struct {
	meta handler.HandlerMetadata
}

func (h *testMultiSpecHandler) Metadata() handler.HandlerMetadata {
	return h.meta
}

func (h *testMultiSpecHandler) MapRemoteToSpec(
	localID string,
	_ *testRemote,
) (*export.SpecExportData[map[string]any], error) {
	return &export.SpecExportData[map[string]any]{
		RelativePath: localID + ".yaml",
		Data:         &map[string]any{},
	}, nil
}

func TestMultiSpecExportStrategy_FormatForExport_WritesURN(t *testing.T) {
	t.Parallel()

	strategy := &export.MultiSpecExportStrategy[map[string]any, testRemote]{
		Handler: &testMultiSpecHandler{
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
