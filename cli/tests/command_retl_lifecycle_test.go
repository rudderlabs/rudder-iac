package tests

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
)

// The lifecycle fixture's ids. They are its own, not TestRETLConnectionsApply's,
// so the two suites never contend for a destination: config-backend allows one
// rETL connection per JSON-mapper destination.
const (
	lifecyclePGAccountExternalID   = "e2e-lifecycle-pg"
	lifecycleSFAccountExternalID   = "e2e-lifecycle-sf"
	lifecycleModelExternalID       = "e2e-lifecycle-model"
	lifecycleTableExternalID       = "e2e-lifecycle-table"
	lifecycleDestinationExternalID = "e2e-lifecycle-http"
	lifecycleConnectionExternalID  = "e2e-lifecycle-connection"
)

// TestRETLLifecycle covers what the create/update/re-apply suites do not: the
// paths where a change cannot be sent as a plain update, and the paths that
// remove things.
//
//   - A connection moved to another source is a replacement (delete, then
//     create), because the PUT body cannot carry endpoints.
//   - A change to a field the API cannot update — a connection's
//     sync_behaviour, a table source's source_definition — is refused with the
//     remedy, and leaves the remote row as it was. Sending it would report
//     success and change nothing, so every later plan would show the same diff.
//   - Removing a connection and the source it reads from in one apply has to
//     delete the connection first: the backend refuses to delete a source a
//     connection still uses.
//   - destroy leaves none of the fixture's resources behind.
//
// The steps share one project and run in order; each fixture directory is the
// whole project at that step.
func TestRETLLifecycle(t *testing.T) {
	allowManagedResidue(t)

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	var (
		projectDir  = filepath.Join("testdata", "retl_lifecycle")
		credentials = filepath.Join(projectDir, "credentials.vars.yaml")
		step        = func(name string) string { return filepath.Join(projectDir, name) }
	)

	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		assert.NoError(t, err, "cleanup destroy failed: %s", out)
	})

	var modelID, tableID, connectionID string

	t.Run("apply create", func(t *testing.T) {
		applyRETLProject(t, executor, step("create"), credentials)

		modelID = managedRETLSource(t, retlClient.ModelSourceType, lifecycleModelExternalID).ID
		tableID = assertLifecycleTable(t, lifecyclePGAccountExternalID, "postgres")
		connectionID = assertLifecycleConnection(t, modelID)
	})

	t.Run("moving the connection to another source replaces it", func(t *testing.T) {
		applyRETLProject(t, executor, step("replace"), credentials)

		replacedID := assertLifecycleConnection(t, tableID)
		assert.NotEqual(t, connectionID, replacedID, "an endpoint change must create a new connection, not update the old one")
		connectionID = replacedID

		assert.Equal(t, modelID, managedRETLSource(t, retlClient.ModelSourceType, lifecycleModelExternalID).ID,
			"the source the connection left must be untouched")
	})

	t.Run("a sync_behaviour change is refused and changes nothing", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "apply", "-l", step("immutable_connection"),
			"--var-file", credentials, "--confirm=false")
		require.Error(t, err, "apply should refuse an immutable connection change, got: %s", out)
		assert.Contains(t, string(out), `sync_behaviour is immutable ("upsert" -> "mirror"); delete and recreate the connection to apply it`)

		assert.Equal(t, connectionID, assertLifecycleConnection(t, tableID), "the refused connection must not be replaced")
	})

	t.Run("a table source_definition change is refused and changes nothing", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "apply", "-l", step("immutable_source_definition"),
			"--var-file", credentials, "--confirm=false")
		require.Error(t, err, "apply should refuse a source_definition change, got: %s", out)
		assert.Contains(t, string(out), `source_definition cannot be changed from "postgres" to "snowflake"`)

		assert.Equal(t, tableID, assertLifecycleTable(t, lifecyclePGAccountExternalID, "postgres"),
			"the refused table source must keep its id, account and definition")
		assert.Equal(t, connectionID, assertLifecycleConnection(t, tableID), "the connection on the refused source must be untouched")
	})

	// The prune fixture drops the connection, the table it reads from and the
	// Snowflake account the refused step created, and keeps everything else.
	t.Run("removing a connection and its source deletes both", func(t *testing.T) {
		applyRETLProject(t, executor, step("prune"), credentials)

		assert.NotContains(t, managedRETLConnectionExternalIDs(t), lifecycleConnectionExternalID)
		assert.NotContains(t, managedRETLSourceExternalIDs(t, retlClient.TableSourceType), lifecycleTableExternalID)
		assert.NotContains(t, managedAccountExternalIDs(t), lifecycleSFAccountExternalID)

		assert.Equal(t, modelID, managedRETLSource(t, retlClient.ModelSourceType, lifecycleModelExternalID).ID,
			"a source still in the project must survive the prune")
		managedDestinationID(t, lifecycleDestinationExternalID)
	})

	t.Run("destroy removes every fixture resource", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		require.NoError(t, err, "destroy failed: %s", out)

		assert.NotContains(t, managedRETLSourceExternalIDs(t, retlClient.ModelSourceType), lifecycleModelExternalID)
		assert.NotContains(t, managedAccountExternalIDs(t), lifecyclePGAccountExternalID)
		assert.NotContains(t, managedDestinationExternalIDs(t), lifecycleDestinationExternalID)
	})
}

// assertLifecycleTable checks the managed table source against the fixture and
// returns its server id.
func assertLifecycleTable(t *testing.T, accountExternalID, sourceDefinition string) string {
	t.Helper()

	actual := managedRETLSource(t, retlClient.TableSourceType, lifecycleTableExternalID)
	id := actual.ID
	actual.ID = ""
	assert.Equal(t, retlClient.RETLSource{
		Name:                 "E2E Lifecycle Table",
		Config:               retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "analytics", Table: "users"},
		IsEnabled:            true,
		SourceType:           retlClient.TableSourceType,
		SourceDefinitionName: sourceDefinition,
		AccountID:            managedAccountID(t, accountExternalID),
		ExternalID:           lifecycleTableExternalID,
	}, actual)
	return id
}

// assertLifecycleConnection checks the fixture's one connection, which every
// step keeps on the same destination with the same config, and returns its
// server id.
func assertLifecycleConnection(t *testing.T, sourceID string) string {
	t.Helper()

	matches := lo.Filter(managedRETLConnections(t), func(c retlClient.RETLConnection, _ int) bool {
		return c.ExternalID == lifecycleConnectionExternalID
	})
	require.Len(t, matches, 1, "managed RETL connections claiming %q", lifecycleConnectionExternalID)
	actual := matches[0]

	assert.True(t, actual.Enabled, "connection should be enabled")
	assert.Equal(t, sourceID, actual.SourceID, "connection is not on the expected source")
	assert.Equal(t, managedDestinationID(t, lifecycleDestinationExternalID), actual.DestinationID, "connection is not on the managed destination")
	assert.Equal(t, retlClient.SyncBehaviourUpsert, actual.SyncBehaviour)
	assert.Equal(t, &retlClient.Event{Type: retlClient.EventTypeIdentify}, actual.Event)
	require.NotNil(t, actual.Schedule.EveryMinutes, "basic schedule came back without everyMinutes")
	assert.Equal(t, 30, *actual.Schedule.EveryMinutes)
	assert.Equal(t, []retlClient.Mapping{{From: "id", To: "user_id"}}, actual.Identifiers)
	assert.Equal(t, []retlClient.Mapping{{From: "email", To: "traits.email"}}, actual.Mappings)
	return actual.ID
}

// The *ExternalIDs helpers list what is managed upstream, so an absence check
// fails on a leaked row instead of passing because a lookup required exactly
// one match.

func managedRETLConnectionExternalIDs(t *testing.T) []string {
	t.Helper()
	return lo.Map(managedRETLConnections(t), func(c retlClient.RETLConnection, _ int) string { return c.ExternalID })
}

func managedRETLSourceExternalIDs(t *testing.T, sourceType retlClient.SourceType) []string {
	t.Helper()

	sources, err := retlClient.NewRudderRETLStore(newAccountsAPIClient(t)).ListRetlSources(
		context.Background(),
		retlClient.WithSourceType(string(sourceType)),
		retlClient.WithHasExternalId(lo.ToPtr(true)),
	)
	require.NoError(t, err, "listing managed RETL sources")
	return lo.Map(sources.Data, func(s retlClient.RETLSource, _ int) string { return s.ExternalID })
}

func managedAccountExternalIDs(t *testing.T) []string {
	t.Helper()

	accounts, err := newAccountsAPIClient(t).Accounts.ListAll(context.Background(), client.WithHasExternalID(true))
	require.NoError(t, err, "listing managed accounts")
	return lo.Map(accounts, func(a client.Account, _ int) string { return a.ExternalID })
}

func managedDestinationExternalIDs(t *testing.T) []string {
	t.Helper()

	destinations, err := retlClient.NewRudderRETLStore(newAccountsAPIClient(t)).GetDestinations(context.Background())
	require.NoError(t, err, "listing destinations")
	return lo.Map(destinations, func(d client.Destination, _ int) string { return d.ExternalID })
}
