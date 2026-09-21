package tests

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
)

const (
	retlConnectionExternalID  = "orders-to-http"
	retlDestinationExternalID = "e2e-retl-http"
)

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
// Ungated for the reason given on TestRETLSourcesApply. Against production it
// fails until the config-backend release carrying DEX-892 (externalId on
// connection create) ships; the PR that adds it is held until then.
func TestRETLConnectionsApply(t *testing.T) {
	allowManagedResidue(t)

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
	var connectionID string
	t.Run("apply update", func(t *testing.T) {
		applyRETLProject(t, executor, filepath.Join(projectDir, "update"), credentials)
		connectionID = assertRETLConnection(t, 60, "traits.emailAddress")
	})

	// The convergence assertion this suite exists for. Before #891, Create wrote
	// a connection the read path then skipped, so every later apply re-planned
	// the same create. The id is what tells convergence apart from a quiet
	// delete-and-recreate.
	t.Run("re-apply does not re-create the connection", func(t *testing.T) {
		applyRETLProject(t, executor, filepath.Join(projectDir, "update"), credentials)
		assert.Equal(t, connectionID, assertRETLConnection(t, 60, "traits.emailAddress"), "the connection was re-created")
	})
}

// assertRETLConnection reads the one managed connection back through the API,
// checks every field the fixture sets and that it joins the managed source and
// destination — not merely some pair — and returns its server id.
func assertRETLConnection(t *testing.T, everyMinutes int, emailTarget string) string {
	t.Helper()

	matches := lo.Filter(managedRETLConnections(t), func(c retlClient.RETLConnection, _ int) bool {
		return c.ExternalID == retlConnectionExternalID
	})
	require.Len(t, matches, 1, "managed RETL connections claiming %q", retlConnectionExternalID)
	actual := matches[0]

	assert.True(t, actual.Enabled, "connection should be enabled")
	assert.Equal(t, managedRETLSource(t, retlClient.ModelSourceType, retlModelExternalID).ID, actual.SourceID, "connection is not on the managed source")
	assert.Equal(t, managedDestinationID(t, retlDestinationExternalID), actual.DestinationID, "connection is not on the managed destination")
	assert.Equal(t, retlClient.SyncBehaviourUpsert, actual.SyncBehaviour)
	assert.Equal(t, &retlClient.Event{Type: retlClient.EventTypeIdentify}, actual.Event)
	require.NotNil(t, actual.Schedule.EveryMinutes, "basic schedule came back without everyMinutes")
	assert.Equal(t, everyMinutes, *actual.Schedule.EveryMinutes, "schedule did not follow the spec")
	assert.Equal(t, []retlClient.Mapping{{From: "id", To: "user_id"}}, actual.Identifiers)
	assert.Equal(t, []retlClient.Mapping{{From: "email", To: emailTarget}}, actual.Mappings)
	return actual.ID
}

// managedRETLConnections lists every managed connection in the workspace,
// walking all pages.
func managedRETLConnections(t *testing.T) []retlClient.RETLConnection {
	t.Helper()

	store := retlClient.NewRudderRETLStore(newAccountsAPIClient(t))
	var all []retlClient.RETLConnection
	for page := 1; ; page++ {
		result, err := store.ListConnections(context.Background(), &retlClient.ListRETLConnectionsRequest{
			HasExternalID: lo.ToPtr(true), Page: page, PageSize: 100,
		})
		require.NoError(t, err, "listing managed RETL connections")
		all = append(all, result.Data...)
		if result.Paging.Next == "" {
			return all
		}
	}
}

// managedDestinationID returns the server id of the one destination claiming
// the given externalId.
func managedDestinationID(t *testing.T, externalID string) string {
	t.Helper()

	destinations, err := retlClient.NewRudderRETLStore(newAccountsAPIClient(t)).GetDestinations(context.Background())
	require.NoError(t, err, "listing destinations")

	var ids []string
	for _, d := range destinations {
		if d.ExternalID == externalID {
			ids = append(ids, d.ID)
		}
	}
	require.Len(t, ids, 1, "destinations claiming %q", externalID)
	return ids[0]
}
