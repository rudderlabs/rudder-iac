package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConnectionsApply drives the event stream connections provider end-to-end
// against a live stack. A project with a source, a destination, and a
// connection applies in one shot — the connection depends on both endpoints,
// so a successful apply proves the create order. Re-applying the same project
// without the connection spec must disconnect the endpoints without touching
// them: the connection row disappears while both endpoints survive with the
// same remote ids.
//
// Gated behind RUN_CONNECTION_E2E because it needs a live stack with
// externalId support on the connections API (DEX-648 backend work) and a
// destination-enabled workspace.
func TestConnectionsApply(t *testing.T) {
	if os.Getenv("RUN_CONNECTION_E2E") != "1" {
		t.Skip("set RUN_CONNECTION_E2E=1 with a live stack that supports connection externalIds")
	}

	t.Setenv("RUDDERSTACK_X_CONNECTION_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_X_DESTINATION_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_ENABLE_VAR_SUBSTITUTION", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	projectDir := filepath.Join("testdata", "connections")
	varFile := filepath.Join(projectDir, "connections.vars.yaml")

	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	// Registered after t.Setenv so the flags still hold when this cleanup
	// runs; a leftover destination breaks other e2e tests' destroy when
	// DestinationSupport is off. assert (not require) avoids FailNow
	// mid-cleanup.
	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		assert.NoError(t, err, "cleanup destroy failed: %s", out)
	})

	apply := func(t *testing.T, dir string) {
		t.Helper()
		out, err := executor.Execute(cliBinPath, "apply", "-l",
			filepath.Join(projectDir, dir), "--var-file", varFile, "--confirm=false")
		require.NoError(t, err, "%s apply failed: %s", dir, out)
	}

	apiClient := newConnectionsAPIClient(t)

	var sourceID, destinationID string

	t.Run("apply connects source and destination", func(t *testing.T) {
		apply(t, "connect")

		conns := listManagedConnections(t, apiClient)
		require.Len(t, conns, 1, "expected exactly one managed connection")
		conn := conns[0]
		assert.Equal(t, "e2e-android-to-s3", conn.ExternalID)
		assert.True(t, conn.IsEnabled)
		require.NotEmpty(t, conn.SourceID)
		require.NotEmpty(t, conn.DestinationID)
		sourceID, destinationID = conn.SourceID, conn.DestinationID
	})

	t.Run("removing the connection spec disconnects without touching endpoints", func(t *testing.T) {
		require.NotEmpty(t, sourceID, "connect subtest must have run first")

		apply(t, "disconnect")

		conns := listManagedConnections(t, apiClient)
		assert.Empty(t, conns, "connection must be gone after its spec is removed")

		source, err := apiClient.Sources.Get(context.Background(), sourceID)
		require.NoError(t, err, "source must still exist after disconnect")
		assert.Equal(t, sourceID, source.ID)

		destination, err := apiClient.Destinations.Get(context.Background(), destinationID)
		require.NoError(t, err, "destination must still exist after disconnect")
		assert.Equal(t, destinationID, destination.ID)
	})
}

// listManagedConnections pages through the generic connections list, keeping
// only rows that carry an externalId — the CLI-managed ones.
func listManagedConnections(t *testing.T, apiClient *client.Client) []client.Connection {
	t.Helper()
	var conns []client.Connection
	page, err := apiClient.Connections.List(context.Background(), client.WithConnectionsHasExternalID(true))
	require.NoError(t, err, "listing connections")
	for page != nil {
		conns = append(conns, page.Connections...)
		page, err = apiClient.Connections.Next(context.Background(), page.Paging)
		require.NoError(t, err, "paging connections")
	}
	return conns
}

func newConnectionsAPIClient(t *testing.T) *client.Client {
	t.Helper()
	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
	)
	require.NoError(t, err)
	return apiClient
}
