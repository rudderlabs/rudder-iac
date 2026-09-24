package source_test

import (
	"context"
	"testing"

	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadImportable_IncludeManaged(t *testing.T) {
	t.Parallel()

	remoteSources := []sourceClient.EventStreamSource{
		{ID: "remote-managed", ExternalID: "prod-web", Name: "Prod Web", Type: "javascript", WorkspaceID: "ws-1"},
		{ID: "remote-unmanaged", Name: "Staging Web", Type: "javascript", WorkspaceID: "ws-1"},
	}

	newHandler := func() *source.Handler {
		client := source.NewMockSourceClient()
		client.SetGetSourcesFunc(func(context.Context) ([]sourceClient.EventStreamSource, error) {
			return remoteSources, nil
		})
		return source.NewHandler(client, "event-stream")
	}

	t.Run("by default managed sources are skipped", func(t *testing.T) {
		got, err := newHandler().LoadImportable(context.Background(), namer.NewExternalIdNamer(namer.NewKebabCase()))
		require.NoError(t, err)

		all := got.GetAll(source.ResourceType)
		require.Len(t, all, 1)
		assert.Equal(t, "staging-web", all["remote-unmanaged"].ExternalID)
	})

	t.Run("with IncludeManaged both come back, managed keeping its externalId", func(t *testing.T) {
		got, err := newHandler().LoadImportable(context.Background(), namer.NewExternalIdNamer(namer.NewKebabCase()),
			resources.ImportableFilter{IncludeManaged: true})
		require.NoError(t, err)

		all := got.GetAll(source.ResourceType)
		require.Len(t, all, 2)
		assert.Equal(t, "prod-web", all["remote-managed"].ExternalID)
		assert.Equal(t, "#event-stream-source:prod-web", all["remote-managed"].Reference)
		assert.Equal(t, "staging-web", all["remote-unmanaged"].ExternalID)
	})
}
