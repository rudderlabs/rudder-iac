package source_test

import (
	"testing"

	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatForExportSkipsMatched(t *testing.T) {
	t.Parallel()

	h := source.NewHandler(source.NewMockSourceClient(), "event-stream/sources")

	matched := &resources.RemoteResource{
		ID:          "src1",
		ExternalID:  "web-source",
		Data:        &sourceClient.EventStreamSource{ID: "src1", Name: "Web Source", WorkspaceID: "ws1"},
		MatchedWith: resources.NewResource("web-source", source.ResourceType, resources.ResourceData{}, []string{}),
	}
	unmatched := &resources.RemoteResource{
		ID:         "src2",
		ExternalID: "mobile-source",
		Data:       &sourceClient.EventStreamSource{ID: "src2", Name: "Mobile Source", WorkspaceID: "ws1"},
	}

	collection := resources.NewRemoteResources()
	collection.Set(source.ResourceType, map[string]*resources.RemoteResource{"src1": matched, "src2": unmatched})

	entities, entries, err := h.FormatForExport(collection, nil, &mockResolver{})
	require.NoError(t, err)

	// One spec file per unmatched source; the matched source writes nothing.
	require.Len(t, entities, 1)
	assert.Contains(t, entities[0].RelativePath, "mobile-source.yaml")
	assert.NotContains(t, entities[0].RelativePath, "web-source")

	// Manifest entries for both, the matched one under its adopted local URN.
	assert.ElementsMatch(t, []importmanifest.ImportEntry{
		{WorkspaceID: "ws1", URN: "event-stream-source:web-source", RemoteID: "src1"},
		{WorkspaceID: "ws1", URN: "event-stream-source:mobile-source", RemoteID: "src2"},
	}, entries)
}
