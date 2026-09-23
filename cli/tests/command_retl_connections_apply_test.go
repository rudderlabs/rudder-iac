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

// The fixture's two connections: one on the SQL model source, one on the table
// source. The table-backed one is expressible only because retl-source-table
// is a registered connection source kind (DEX-865).
const (
	retlModelConnectionExternalID = "orders-to-http"
	retlTableConnectionExternalID = "customers-to-http"
	// One destination per connection: a JSON-mapper destination (anything outside
	// DESTINATION_SPECIFIC_REGISTRY) accepts a single rETL connection, and
	// config-backend refuses a second with "Destination does not support multiple
	// connections". Sharing one only passed when concurrent creates raced past
	// that check.
	retlModelDestinationExternalID = "e2e-retl-http"
	retlTableDestinationExternalID = "e2e-retl-http-customers"
)

type retlConnectionWant struct {
	externalID       string
	sourceType       retlClient.SourceType
	sourceExternalID string
	destExternalID   string
	everyMinutes     int
	emailTarget      string
}

// TestRETLConnectionsApply drives a retl-connections entry end to end: the
// project carries the account, both source kinds and the destinations the
// connections join, so one apply has to order them all and substitute
// server-assigned ids into each connection.
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

	model := retlConnectionWant{retlModelConnectionExternalID, retlClient.ModelSourceType, retlModelExternalID, retlModelDestinationExternalID, 30, "traits.email"}
	table := retlConnectionWant{retlTableConnectionExternalID, retlClient.TableSourceType, retlTableExternalID, retlTableDestinationExternalID, 45, "traits.email"}

	t.Run("apply create", func(t *testing.T) {
		applyRETLProject(t, executor, filepath.Join(projectDir, "create"), credentials)
		assertRETLConnection(t, model)
		assertRETLConnection(t, table)
	})

	// A mutable change: the model connection's schedule and mapping move in one
	// PUT, the table connection's mapping moves and its schedule holds at 45. An
	// endpoint change would be a replacement instead, which is a different path
	// and belongs in its own case once this one is established upstream.
	model.everyMinutes, model.emailTarget = 60, "traits.emailAddress"
	table.emailTarget = "traits.emailAddress"
	var modelID, tableID string
	t.Run("apply update", func(t *testing.T) {
		applyRETLProject(t, executor, filepath.Join(projectDir, "update"), credentials)
		modelID = assertRETLConnection(t, model)
		tableID = assertRETLConnection(t, table)
	})

	// The convergence assertion this suite exists for. Before #891, Create wrote
	// a connection the read path then skipped, so every later apply re-planned
	// the same create. The id is what tells convergence apart from a quiet
	// delete-and-recreate.
	t.Run("re-apply does not re-create the connection", func(t *testing.T) {
		applyRETLProject(t, executor, filepath.Join(projectDir, "update"), credentials)
		assert.Equal(t, modelID, assertRETLConnection(t, model), "the model connection was re-created")
		assert.Equal(t, tableID, assertRETLConnection(t, table), "the table connection was re-created")
	})
}

// assertRETLConnection reads one managed connection back through the API,
// checks every field the fixture sets and that it joins the managed source and
// destination — not merely some pair — and returns its server id.
func assertRETLConnection(t *testing.T, want retlConnectionWant) string {
	t.Helper()

	matches := lo.Filter(managedRETLConnections(t), func(c retlClient.RETLConnection, _ int) bool {
		return c.ExternalID == want.externalID
	})
	require.Len(t, matches, 1, "managed RETL connections claiming %q", want.externalID)
	actual := matches[0]

	assert.True(t, actual.Enabled, "connection should be enabled")
	assert.Equal(t, managedRETLSource(t, want.sourceType, want.sourceExternalID).ID, actual.SourceID, "connection is not on the managed source")
	assert.Equal(t, managedDestinationID(t, want.destExternalID), actual.DestinationID, "connection is not on the managed destination")
	assert.Equal(t, retlClient.SyncBehaviourUpsert, actual.SyncBehaviour)
	assert.Equal(t, &retlClient.Event{Type: retlClient.EventTypeIdentify}, actual.Event)
	require.NotNil(t, actual.Schedule.EveryMinutes, "basic schedule came back without everyMinutes")
	assert.Equal(t, want.everyMinutes, *actual.Schedule.EveryMinutes, "schedule did not follow the spec")
	assert.Equal(t, []retlClient.Mapping{{From: "id", To: "user_id"}}, actual.Identifiers)
	assert.Equal(t, []retlClient.Mapping{{From: "email", To: want.emailTarget}}, actual.Mappings)
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
