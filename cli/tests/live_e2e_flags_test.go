package tests

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	essource "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/stretchr/testify/require"
)

// enableLiveWorkspaceCleanupFlags enables every gated resource family that can
// leave managed destinations behind in the shared live E2E workspace, so a
// workspace-wide destroy can see and remove them. It deliberately excludes
// RUDDERSTACK_X_CONNECTION_SUPPORT: cleanupLiveWorkspaceEventStreamConnections
// removes every connection via direct API calls before destroy runs, so
// destroy never needs to see connections as a resource kind itself — verified
// live by running destroy with the flag unset immediately after that cleanup.
func enableLiveWorkspaceCleanupFlags(t *testing.T) {
	t.Helper()

	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_DESTINATION_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_X_UNVERIFIED_DESTINATIONS", "true")
}

// cleanupLiveWorkspaceEventStreamConnections deletes event-stream connections
// before workspace-wide destroy. The shared CI workspace can contain old
// unmanaged links with no externalId; the CLI destroy plan cannot see those
// rows, but the backend still refuses to delete their sources/destinations.
func cleanupLiveWorkspaceEventStreamConnections(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	apiClient := newLiveE2EAPIClient(t)

	sources, err := essource.NewRudderSourceStore(apiClient).GetSources(ctx)
	require.NoError(t, err, "listing event stream sources before cleanup")
	sourceIDs := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		sourceIDs[source.ID] = struct{}{}
	}
	if len(sourceIDs) == 0 {
		return
	}

	for _, conn := range listConnections(t, ctx, apiClient) {
		if _, ok := sourceIDs[conn.SourceID]; !ok {
			continue
		}
		require.NoError(t, apiClient.Connections.Delete(ctx, conn.ID),
			"deleting event stream connection %s before cleanup", conn.ID)
	}
}

// newLiveE2EAPIClient builds an API client from the same config the CLI binary
// uses, so test-side cleanup and assertions act on the same workspace.
func newLiveE2EAPIClient(t *testing.T) *client.Client {
	t.Helper()

	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)
	require.NoError(t, err)
	return apiClient
}
