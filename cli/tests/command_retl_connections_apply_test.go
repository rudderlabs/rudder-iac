package tests

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const retlConnectionExternalID = "orders-to-http"

// TestRETLConnectionsApply drives a retl-connections entry end to end: the
// project carries the account, the SQL model source and the destination the
// connection joins, so one apply has to order all four and substitute two
// server-assigned ids into the connection.
//
// The destination is http rather than webhook on purpose. Only three
// destination definitions declare warehouse in SupportedSourceTypes —
// http, bingads_offline_conversions and customerio_audience — and the
// connection read path skips any row whose destination is not one of them,
// which would leave this suite creating a connection it could never read back.
//
// Ungated, for the reason written into TestRETLSourcesApply: the two gated
// suites in this package have never run because their repository variables were
// never defined.
func TestRETLConnectionsApply(t *testing.T) {
	allowUnverifiedDestinationResidue(t)
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_DESTINATION_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_X_RETL_CONNECTION_SUPPORT", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	projectDir := filepath.Join("testdata", "retl_connections")
	credentials := filepath.Join(projectDir, "credentials.vars.yaml")

	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		assert.NoError(t, err, "cleanup destroy failed: %s", out)
	})

	t.Run("apply create", func(t *testing.T) {
		applyRETLProject(t, executor, filepath.Join(projectDir, "create"), credentials)
		assertRETLConnection(t, 30, "traits.email")
	})

	// A mutable change: schedule and mapping move in one PUT. An endpoint change
	// would be a replacement instead, which is a different path and belongs in
	// its own case once this one is established upstream.
	t.Run("apply update", func(t *testing.T) {
		applyRETLProject(t, executor, filepath.Join(projectDir, "update"), credentials)
		assertRETLConnection(t, 60, "traits.emailAddress")
	})

	// The convergence assertion this suite exists for. Before #891, Create wrote
	// a connection the read path then skipped, so state stayed empty and every
	// later apply re-planned the same create until the backend refused it as a
	// duplicate. A second apply that changes nothing but the account secret is
	// what proves the row round-trips.
	t.Run("re-apply does not re-create the connection", func(t *testing.T) {
		applyRETLProject(t, executor, filepath.Join(projectDir, "update"), credentials)
		assertRETLConnection(t, 60, "traits.emailAddress")
	})
}

// assertRETLConnection reads the managed connection back through the API and
// checks the fields the fixture sets, plus that it still points at the managed
// source and destination.
func assertRETLConnection(t *testing.T, everyMinutes int, emailTarget string) {
	t.Helper()

	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)
	require.NoError(t, err)

	store := retlClient.NewRudderRETLStore(apiClient)
	page, err := store.ListConnections(context.Background(), &retlClient.ListRETLConnectionsRequest{Page: 1})
	require.NoError(t, err, "listing RETL connections")

	var actual *retlClient.RETLConnection
	for i, conn := range page.Data {
		if conn.ExternalID == retlConnectionExternalID {
			actual = &page.Data[i]
			break
		}
	}
	require.NotNil(t, actual, "managed RETL connection %q missing upstream", retlConnectionExternalID)

	assert.True(t, actual.Enabled, "connection should be enabled")
	assert.NotEmpty(t, actual.SourceID, "connection lost its source")
	assert.NotEmpty(t, actual.DestinationID, "connection lost its destination")
	require.NotNil(t, actual.Schedule.EveryMinutes, "basic schedule came back without everyMinutes")
	assert.Equal(t, everyMinutes, *actual.Schedule.EveryMinutes, "schedule did not follow the spec")
	assert.Equal(t, []retlClient.Mapping{{From: "id", To: "user_id"}}, actual.Identifiers)
	assert.Equal(t, []retlClient.Mapping{{From: "email", To: emailTarget}}, actual.Mappings)
}
