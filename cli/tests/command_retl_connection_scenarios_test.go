package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
)

// The scenario fixture's ids are its own, so it never contends with another
// suite for a destination: config-backend allows one rETL connection per
// JSON-mapper destination.
const (
	scenarioAccountExternalID           = "e2e-scn-pg"
	scenarioModelExternalID             = "e2e-scn-model"
	scenarioTableExternalID             = "e2e-scn-table"
	scenarioTrackDestinationExternalID  = "e2e-scn-http-track"
	scenarioColumnDestinationExternalID = "e2e-scn-http-column"
	scenarioTrackConnectionExternalID   = "e2e-scn-orders-track"
	scenarioColumnConnectionExternalID  = "e2e-scn-customers-full"
)

// TestRETLConnectionScenarios covers the connection settings the create/update
// suite leaves at one value: cron and manual schedules, track events named by a
// constant and by a column, constants, a cursor column, sync settings and the
// full sync behaviour. Each is read back as the whole connection the API
// returns, and each apply is followed by a dry run, because a setting the read
// path normalizes differently from the spec shows up as a re-planned update
// rather than as a wrong value.
//
// The HTTP destination runs the JSON mapper flow, so the sync behaviours on
// offer are upsert and full (the backend refuses mirror there), and the
// postgres source definition accepts sync settings.
func TestRETLConnectionScenarios(t *testing.T) {
	if os.Getenv("RUN_RETL_E2E") != "1" {
		t.Skip("set RUN_RETL_E2E=1; this suite applies to a live workspace")
	}
	allowManagedResidue(t)

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	var (
		projectDir  = filepath.Join("testdata", "retl_connection_scenarios")
		credentials = filepath.Join(projectDir, "credentials.vars.yaml")
		step        = func(name string) string { return filepath.Join(projectDir, name) }
	)

	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		assert.NoError(t, err, "cleanup destroy failed: %s", out)
	})

	var trackID, columnID string

	t.Run("apply create", func(t *testing.T) {
		assertRETLConnectionChangesPlanned(t, executor, step("create"), credentials,
			scenarioTrackConnectionExternalID, scenarioColumnConnectionExternalID)
		applyRETLProject(t, executor, step("create"), credentials)

		var track, column retlClient.RETLConnection
		trackID, track = managedRETLConnectionByExternalID(t, scenarioTrackConnectionExternalID)
		columnID, column = managedRETLConnectionByExternalID(t, scenarioColumnConnectionExternalID)

		assert.Equal(t, retlClient.RETLConnection{
			SourceID:      managedRETLSource(t, retlClient.ModelSourceType, scenarioModelExternalID).ID,
			DestinationID: managedDestinationID(t, scenarioTrackDestinationExternalID),
			Enabled:       true,
			ExternalID:    scenarioTrackConnectionExternalID,
			Schedule:      retlClient.Schedule{Type: retlClient.ScheduleTypeBasic, EveryMinutes: lo.ToPtr(30)},
			SyncSettings:  syncSettings(false, 0, 0, false),
			SyncBehaviour: retlClient.SyncBehaviourUpsert,
			Identifiers:   []retlClient.Mapping{{From: "id", To: "user_id"}},
			Mappings:      []retlClient.Mapping{{From: "email", To: "properties.email"}, {From: "plan", To: "properties.plan"}},
			Event:         &retlClient.Event{Type: retlClient.EventTypeTrack, Name: "Order Synced"},
			Constants:     []retlClient.Constant{{Key: "properties.origin", Value: "rudder-cli-e2e"}},
			CursorColumn:  "updated_at",
		}, track)

		// No sync_settings in the spec: the backend stores its defaults.
		assert.Equal(t, retlClient.RETLConnection{
			SourceID:      managedRETLSource(t, retlClient.TableSourceType, scenarioTableExternalID).ID,
			DestinationID: managedDestinationID(t, scenarioColumnDestinationExternalID),
			Enabled:       true,
			ExternalID:    scenarioColumnConnectionExternalID,
			Schedule:      retlClient.Schedule{Type: retlClient.ScheduleTypeCron, CronExpression: lo.ToPtr("0 */6 * * *")},
			SyncSettings:  syncSettings(true, 30, 5, true),
			SyncBehaviour: retlClient.SyncBehaviourFull,
			Identifiers:   []retlClient.Mapping{{From: "id", To: "anonymous_id"}},
			Mappings:      []retlClient.Mapping{{From: "email", To: "properties.email"}},
			Event:         &retlClient.Event{Type: retlClient.EventTypeTrack, NameColumn: "event_name"},
		}, column)
	})

	t.Run("created connections converge", func(t *testing.T) {
		assertNoRETLConnectionChanges(t, executor, step("create"), credentials)
	})

	t.Run("apply update", func(t *testing.T) {
		applyRETLProject(t, executor, step("update"), credentials)

		id, track := managedRETLConnectionByExternalID(t, scenarioTrackConnectionExternalID)
		assert.Equal(t, trackID, id, "a schedule, enabled, constants and sync settings change must update the connection in place")
		// A partial sync_settings block: the CLI fills every field it leaves out
		// with the backend default before sending, so the stored non-default
		// values (false/0/0) are replaced rather than merged into. everyMinutes
		// must be gone with the basic schedule that carried it.
		assert.Equal(t, retlClient.RETLConnection{
			SourceID:      managedRETLSource(t, retlClient.ModelSourceType, scenarioModelExternalID).ID,
			DestinationID: managedDestinationID(t, scenarioTrackDestinationExternalID),
			Enabled:       false,
			ExternalID:    scenarioTrackConnectionExternalID,
			Schedule:      retlClient.Schedule{Type: retlClient.ScheduleTypeCron, CronExpression: lo.ToPtr("30 2 * * *")},
			SyncSettings:  syncSettings(true, 30, 5, true),
			SyncBehaviour: retlClient.SyncBehaviourUpsert,
			Identifiers:   []retlClient.Mapping{{From: "id", To: "user_id"}},
			Mappings:      []retlClient.Mapping{{From: "email", To: "properties.email"}, {From: "plan", To: "properties.plan"}},
			Event:         &retlClient.Event{Type: retlClient.EventTypeTrack, Name: "Order Synced"},
			Constants: []retlClient.Constant{
				{Key: "properties.origin", Value: "rudder-cli-e2e-v2"},
				{Key: "properties.team", Value: "dex"},
			},
			CursorColumn: "updated_at",
		}, track)

		id, column := managedRETLConnectionByExternalID(t, scenarioColumnConnectionExternalID)
		assert.Equal(t, columnID, id, "a schedule and sync settings change must update the connection in place")
		// A full non-default block replaces the defaults create stored, and the
		// cronExpression must be gone with the cron schedule that carried it.
		assert.Equal(t, retlClient.RETLConnection{
			SourceID:      managedRETLSource(t, retlClient.TableSourceType, scenarioTableExternalID).ID,
			DestinationID: managedDestinationID(t, scenarioColumnDestinationExternalID),
			Enabled:       true,
			ExternalID:    scenarioColumnConnectionExternalID,
			Schedule:      retlClient.Schedule{Type: retlClient.ScheduleTypeManual},
			SyncSettings:  syncSettings(false, 14, 3, false),
			SyncBehaviour: retlClient.SyncBehaviourFull,
			Identifiers:   []retlClient.Mapping{{From: "id", To: "anonymous_id"}},
			Mappings:      []retlClient.Mapping{{From: "email", To: "properties.email"}},
			Event:         &retlClient.Event{Type: retlClient.EventTypeTrack, NameColumn: "event_name"},
		}, column)
	})

	t.Run("updated connections converge", func(t *testing.T) {
		assertNoRETLConnectionChanges(t, executor, step("update"), credentials)
	})

	// destroy removes connections and the endpoints they still join in one run.
	// The assertions below check the end state; a wrong delete order shows up as
	// the require.NoError failing, because the backend refuses to delete an
	// endpoint a connection still points at.
	t.Run("destroy removes live connections and their endpoints", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		require.NoError(t, err, "destroy failed: %s", out)

		connections := managedRETLConnectionExternalIDs(t)
		assert.NotContains(t, connections, scenarioTrackConnectionExternalID)
		assert.NotContains(t, connections, scenarioColumnConnectionExternalID)

		destinations := managedDestinationExternalIDs(t)
		assert.NotContains(t, destinations, scenarioTrackDestinationExternalID)
		assert.NotContains(t, destinations, scenarioColumnDestinationExternalID)

		sources := managedRETLSourceExternalIDs(t)
		assert.NotContains(t, sources, scenarioModelExternalID)
		assert.NotContains(t, sources, scenarioTableExternalID)

		assert.NotContains(t, managedAccountExternalIDs(t), scenarioAccountExternalID)
	})
}

// managedRETLConnectionByExternalID reads back the one managed connection
// claiming externalID and returns its server id together with the connection,
// the server-assigned and time-varying fields cleared so callers can compare
// the whole struct.
func managedRETLConnectionByExternalID(t *testing.T, externalID string) (string, retlClient.RETLConnection) {
	t.Helper()

	matches := lo.Filter(managedRETLConnections(t), func(c retlClient.RETLConnection, _ int) bool {
		return c.ExternalID == externalID
	})
	require.Len(t, matches, 1, "managed RETL connections claiming %q", externalID)

	connection := matches[0]
	id := connection.ID
	require.NotEmpty(t, id, "managed RETL connection %q has no id", externalID)
	connection.ID, connection.CreatedAt, connection.UpdatedAt = "", nil, nil
	return id, connection
}

// connectionURNPrefix is what the plan reporter prints for a connection. It is
// the production constant rather than a literal, so a rename cannot leave every
// convergence assertion below passing unconditionally.
const connectionURNPrefix = connection.ResourceType + ":"

// assertRETLConnectionChangesPlanned is the positive half of the convergence
// checks: it proves the plan really does print this prefix, so the
// NotContains below is a statement about the plan rather than about a string
// nothing emits.
func assertRETLConnectionChangesPlanned(t *testing.T, executor *CmdExecutor, dir, credentials string, localIDs ...string) {
	t.Helper()

	out, err := executor.Execute(cliBinPath, "apply", "-l", dir, "--var-file", credentials, "--dry-run", "--confirm=false")
	require.NoError(t, err, "dry run failed: %s", out)
	for _, localID := range localIDs {
		assert.Contains(t, string(out), connectionURNPrefix+localID, "the plan must name the connection it will create")
	}
}

// assertNoRETLConnectionChanges dry-runs the project and checks that no
// connection is planned. The whole plan cannot be required empty: the account
// secret cannot be read back, so the account re-plans on every apply.
func assertNoRETLConnectionChanges(t *testing.T, executor *CmdExecutor, dir, credentials string) {
	t.Helper()

	out, err := executor.Execute(cliBinPath, "apply", "-l", dir, "--var-file", credentials, "--dry-run", "--confirm=false")
	require.NoError(t, err, "dry run failed: %s", out)
	assert.NotContains(t, string(out), connectionURNPrefix, "a re-apply must not plan any connection change")
}

func syncSettings(logsEnabled bool, retentionDays, snapshots int, retryFailedKeys bool) *retlClient.SyncSettings {
	return &retlClient.SyncSettings{
		SyncLogsConfig: &retlClient.SyncLogsConfig{
			Enabled:            lo.ToPtr(logsEnabled),
			LogRetentionInDays: lo.ToPtr(retentionDays),
			SnapshotsToRetain:  lo.ToPtr(snapshots),
		},
		FailedKeysConfig: &retlClient.FailedKeysConfig{EnableFailedKeysRetry: lo.ToPtr(retryFailedKeys)},
	}
}
