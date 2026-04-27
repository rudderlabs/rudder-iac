package export_test

import (
	"testing"

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

func TestMultiSpecExportStrategy_FormatForExport_ReturnsImportEntry(t *testing.T) {
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

	entities, entries, err := strategy.FormatForExport(remotes, nil, nil)
	require.NoError(t, err)
	require.Len(t, entities, 1)

	require.Len(t, entries, 1)
	assert.Equal(t, "ws-123", entries[0].WorkspaceID)
	assert.Equal(t, "my-resource-type:local-id-1", entries[0].URN, "import entry must use urn, not local_id")
	assert.Equal(t, "remote-id-1", entries[0].RemoteID)
}
