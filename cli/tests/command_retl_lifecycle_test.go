package tests

import (
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		tableID = assertLifecycleTable(t)
		connectionID = assertLifecycleConnection(t, modelID)
	})

	t.Run("moving the connection to another source replaces it", func(t *testing.T) {
		applyRETLProject(t, executor, step("replace"), credentials)

		// The id is deliberately not required to change. A PUT cannot move
		// endpoints, so reading the connection back on the table source is what
		// proves this was a delete-and-create, and the backend may revive the
		// soft-deleted row under its old id.
		connectionID = assertLifecycleConnection(t, tableID)

		assert.Equal(t, modelID, managedRETLSource(t, retlClient.ModelSourceType, lifecycleModelExternalID).ID,
			"the source the connection left must be untouched")
	})

	// A replacement whose config round-trips differently would re-plan a
	// delete-and-create on every apply, churning a production connection
	// forever. Nothing else in the suite would notice.
	t.Run("the replaced connection converges", func(t *testing.T) {
		assertNoRETLConnectionChanges(t, executor, step("replace"), credentials)
	})

	t.Run("a sync_behaviour change is refused and changes nothing", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "apply", "-l", step("immutable_connection"),
			"--var-file", credentials, "--confirm=false")
		require.Error(t, err, "apply should refuse an immutable connection change, got: %s", out)
		assert.Contains(t, string(out), `sync_behaviour is immutable ("upsert" -> "full"); delete and recreate the connection to apply it`)

		assert.Equal(t, connectionID, assertLifecycleConnection(t, tableID), "the refused connection must not be replaced")
	})

	// "Refused" is about the table source, not about the whole apply: the
	// handler errors inside Update, by which point the graph has already created
	// the Snowflake account the new definition needs. That account is what the
	// prune step then removes.
	t.Run("a table source_definition change is refused and leaves the source alone", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "apply", "-l", step("immutable_source_definition"),
			"--var-file", credentials, "--confirm=false")
		require.Error(t, err, "apply should refuse a source_definition change, got: %s", out)
		assert.Contains(t, string(out), `source_definition cannot be changed from "postgres" to "snowflake"`)

		assert.Equal(t, tableID, assertLifecycleTable(t), "the refused table source must keep its id, account and definition")
		assert.Equal(t, connectionID, assertLifecycleConnection(t, tableID), "the connection on the refused source must be untouched")
		assert.Contains(t, managedAccountExternalIDs(t), lifecycleSFAccountExternalID,
			"the account the refused step created is what the prune step removes")
	})

	// The prune fixture drops the connection, the table it reads from and the
	// Snowflake account the refused step created, and keeps everything else.
	t.Run("removing a connection and its source deletes both", func(t *testing.T) {
		// Every other apply here takes the concurrent path (the syncer defaults
		// to 30); this one pins the sequential executor, where a wrong delete
		// order has nothing to hide behind.
		t.Setenv("RUDDERSTACK_CLI_CONCURRENCY_SYNCER", "1")
		applyRETLProject(t, executor, step("prune"), credentials)

		assert.NotContains(t, managedRETLConnectionExternalIDs(t), lifecycleConnectionExternalID)
		assert.NotContains(t, managedRETLSourceExternalIDs(t), lifecycleTableExternalID)
		assert.NotContains(t, managedAccountExternalIDs(t), lifecycleSFAccountExternalID)

		assert.Equal(t, modelID, managedRETLSource(t, retlClient.ModelSourceType, lifecycleModelExternalID).ID,
			"a source still in the project must survive the prune")
		assert.Contains(t, managedDestinationExternalIDs(t), lifecycleDestinationExternalID,
			"a destination still in the project must survive the prune")
	})

	t.Run("destroy removes every fixture resource", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		require.NoError(t, err, "destroy failed: %s", out)

		sources := managedRETLSourceExternalIDs(t)
		assert.NotContains(t, sources, lifecycleModelExternalID)
		assert.NotContains(t, sources, lifecycleTableExternalID)
		assert.NotContains(t, managedRETLConnectionExternalIDs(t), lifecycleConnectionExternalID)
		assert.NotContains(t, managedAccountExternalIDs(t), lifecyclePGAccountExternalID)
		assert.NotContains(t, managedDestinationExternalIDs(t), lifecycleDestinationExternalID)
	})
}

// assertLifecycleTable checks the managed table source against the fixture —
// the postgres account and definition every step keeps — and returns its id.
func assertLifecycleTable(t *testing.T) string {
	t.Helper()

	actual := managedRETLSource(t, retlClient.TableSourceType, lifecycleTableExternalID)
	id := actual.ID
	actual.ID = ""
	assert.Equal(t, retlClient.RETLSource{
		Name:                 "E2E Lifecycle Table",
		Config:               retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "analytics", Table: "users"},
		IsEnabled:            true,
		SourceType:           retlClient.TableSourceType,
		SourceDefinitionName: "postgres",
		AccountID:            managedAccountID(t, lifecyclePGAccountExternalID),
		ExternalID:           lifecycleTableExternalID,
	}, actual)
	return id
}

// assertLifecycleConnection checks the fixture's one connection, which every
// step keeps on the same destination with the same config, and returns its
// server id.
//
// The whole struct is compared rather than the fields the fixture sets. A
// replacement rebuilds the connection from scratch, so a create that dropped
// the schedule type or leaked a cursorColumn or a syncSettings block is exactly
// the failure this suite exists to catch, and no list of named fields catches
// what it did not think to name.
func assertLifecycleConnection(t *testing.T, sourceID string) string {
	t.Helper()

	id, actual := managedRETLConnectionByExternalID(t, lifecycleConnectionExternalID)
	assert.Equal(t, retlClient.RETLConnection{
		SourceID:      sourceID,
		DestinationID: managedDestinationID(t, lifecycleDestinationExternalID),
		Enabled:       true,
		ExternalID:    lifecycleConnectionExternalID,
		Schedule:      retlClient.Schedule{Type: retlClient.ScheduleTypeBasic, EveryMinutes: lo.ToPtr(30)},
		SyncSettings:  syncSettings(true, 30, 5, true),
		SyncBehaviour: retlClient.SyncBehaviourUpsert,
		Identifiers:   []retlClient.Mapping{{From: "id", To: "user_id"}},
		Mappings:      []retlClient.Mapping{{From: "email", To: "traits.email"}},
		Event:         &retlClient.Event{Type: retlClient.EventTypeIdentify},
	}, actual)
	return id
}
