package handler_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/handlers/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func remoteWriter(remoteID, externalID, name string, matched *resources.Resource) *resources.RemoteResource {
	return &resources.RemoteResource{
		ID:         remoteID,
		ExternalID: externalID,
		Reference:  "#/writer/writers/" + externalID,
		Data:       &model.RemoteWriter{RemoteWriter: &backend.RemoteWriter{ID: remoteID, Name: name}},
		Matched:    matched,
	}
}

func TestBaseHandler_FormatForExport_PartitionsMatchedResources(t *testing.T) {
	t.Parallel()

	h := writer.NewHandler(backend.NewBackend())
	local := resources.NewResource("tolkien", "example-writer", resources.ResourceData{}, []string{})

	collection := resources.NewRemoteResources()
	collection.Set("example-writer", map[string]*resources.RemoteResource{
		"remote-1": remoteWriter("remote-1", "tolkien", "J.R.R. Tolkien", local),
		"remote-2": remoteWriter("remote-2", "pratchett", "Terry Pratchett", nil),
	})

	entities, entries, err := h.FormatForExport(collection, nil, nil)
	require.NoError(t, err)

	// Spec content only for the unmatched resource.
	require.Len(t, entities, 1)
	assert.Equal(t, "writers/pratchett.yaml", entities[0].RelativePath)

	// Manifest entries for both: the unmatched one from the export strategy,
	// the matched one emitted by BaseHandler with the adopted local identity.
	assert.ElementsMatch(t, []importmanifest.ImportEntry{
		{WorkspaceID: backend.WorkspaceID, URN: "example-writer:pratchett", RemoteID: "remote-2"},
		{WorkspaceID: backend.WorkspaceID, URN: "example-writer:tolkien", RemoteID: "remote-1"},
	}, entries)
}

func TestBaseHandler_FormatForExport_AllMatchedEmitsEntriesOnly(t *testing.T) {
	t.Parallel()

	h := writer.NewHandler(backend.NewBackend())
	local := resources.NewResource("tolkien", "example-writer", resources.ResourceData{}, []string{})

	collection := resources.NewRemoteResources()
	collection.Set("example-writer", map[string]*resources.RemoteResource{
		"remote-1": remoteWriter("remote-1", "tolkien", "J.R.R. Tolkien", local),
	})

	entities, entries, err := h.FormatForExport(collection, nil, nil)
	require.NoError(t, err)

	assert.Empty(t, entities)
	assert.Equal(t, []importmanifest.ImportEntry{
		{WorkspaceID: backend.WorkspaceID, URN: "example-writer:tolkien", RemoteID: "remote-1"},
	}, entries)
}
