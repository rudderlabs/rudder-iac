package handler_test

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/handlers/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type orderedImportWriters struct {
	*writer.HandlerImpl
	remotes []*model.RemoteWriter
}

func (h orderedImportWriters) LoadImportableResources(context.Context) ([]*model.RemoteWriter, error) {
	return h.remotes, nil
}

func TestImportIDsDoNotDependOnAPIListOrder(t *testing.T) {
	first := &model.RemoteWriter{RemoteWriter: &backend.RemoteWriter{ID: "remote-1", Name: "Duplicate Name"}}
	second := &model.RemoteWriter{RemoteWriter: &backend.RemoteWriter{ID: "remote-2", Name: "Duplicate Name"}}
	load := func(remotes ...*model.RemoteWriter) map[string]string {
		impl := writer.NewHandler(backend.NewBackend()).Impl.(*writer.HandlerImpl)
		h := handler.NewHandler(orderedImportWriters{HandlerImpl: impl, remotes: remotes})
		collection, err := h.LoadImportable(context.Background(), namer.NewExternalIdNamer(namer.NewKebabCase()))
		require.NoError(t, err)
		ids := map[string]string{}
		for remoteID, remote := range collection.GetAll(writer.HandlerMetadata.ResourceType) {
			ids[remoteID] = remote.ExternalID
		}
		return ids
	}

	assert.Equal(t, load(first, second), load(second, first))
}
