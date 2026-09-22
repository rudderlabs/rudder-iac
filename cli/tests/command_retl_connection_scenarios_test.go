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
		applyRETLProject(t, executor, step("create"), credentials)

		var track, column retlClient.RETLConnection
		trackID, track = managedRETLConnectionByExternalID(t, scenarioTrackConnectionExternalID)
		columnID, column = managedRETLConnectionByExternalID(t, scenarioColumnConnectionExternalID)

		assert.Equal(t, retlClient.RETLConnection{
			SourceID:      managedRETLSource(t, retlClient.ModelSourceType, scenarioModelExternalID).ID,
			DestinationID: managedDestinationID(t, scenarioTrackDestinationExternalID),
			Enabled:       true,
			ExternalID:    scenarioTrackConnectionExternalID,
			Schedule:      retlClient.Schedule{Type: retlClient.ScheduleTypeCron, CronExpression: lo.ToPtr("0 */6 * * *")},
			SyncSettings:  syncSettings(false, 7, 2, false),
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
			Schedule:      retlClient.Schedule{Type: retlClient.ScheduleTypeManual},
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
		assert.Equal(t, trackID, id, "a schedule, constants and sync settings change must update the connection in place")
		// Dropping sync_settings from the spec resets them: the omitted block
		// reads as the defaults, not as "keep what is stored".
		assert.Equal(t, retlClient.RETLConnection{
			SourceID:      managedRETLSource(t, retlClient.ModelSourceType, scenarioModelExternalID).ID,
			DestinationID: managedDestinationID(t, scenarioTrackDestinationExternalID),
			Enabled:       true,
			ExternalID:    scenarioTrackConnectionExternalID,
			Schedule:      retlClient.Schedule{Type: retlClient.ScheduleTypeManual},
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
		// A partial block: only failed_keys is set, the sync logs keep their defaults.
		assert.Equal(t, retlClient.RETLConnection{
			SourceID:      managedRETLSource(t, retlClient.TableSourceType, scenarioTableExternalID).ID,
			DestinationID: managedDestinationID(t, scenarioColumnDestinationExternalID),
			Enabled:       true,
			ExternalID:    scenarioColumnConnectionExternalID,
			Schedule:      retlClient.Schedule{Type: retlClient.ScheduleTypeCron, CronExpression: lo.ToPtr("30 2 * * *")},
			SyncSettings:  syncSettings(true, 30, 5, false),
			SyncBehaviour: retlClient.SyncBehaviourFull,
			Identifiers:   []retlClient.Mapping{{From: "id", To: "anonymous_id"}},
			Mappings:      []retlClient.Mapping{{From: "email", To: "properties.email"}},
			Event:         &retlClient.Event{Type: retlClient.EventTypeTrack, NameColumn: "event_name"},
		}, column)
	})

	t.Run("updated connections converge", func(t *testing.T) {
		assertNoRETLConnectionChanges(t, executor, step("update"), credentials)
	})

	// Unlike a prune, destroy removes connections and the endpoints they still
	// join in one run, so it has to order the connection deletes first.
	t.Run("destroy removes live connections and their endpoints", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		require.NoError(t, err, "destroy failed: %s", out)

		externalIDs := lo.Map(managedRETLConnections(t), func(c retlClient.RETLConnection, _ int) string { return c.ExternalID })
		assert.NotContains(t, externalIDs, scenarioTrackConnectionExternalID)
		assert.NotContains(t, externalIDs, scenarioColumnConnectionExternalID)

		destinations, err := retlClient.NewRudderRETLStore(newAccountsAPIClient(t)).GetDestinations(context.Background())
		require.NoError(t, err, "listing destinations")
		destinationIDs := lo.Map(destinations, func(d client.Destination, _ int) string { return d.ExternalID })
		assert.NotContains(t, destinationIDs, scenarioTrackDestinationExternalID)
		assert.NotContains(t, destinationIDs, scenarioColumnDestinationExternalID)

		sources, err := retlClient.NewRudderRETLStore(newAccountsAPIClient(t)).ListRetlSources(
			context.Background(), retlClient.WithHasExternalId(lo.ToPtr(true)))
		require.NoError(t, err, "listing managed RETL sources")
		sourceIDs := lo.Map(sources.Data, func(s retlClient.RETLSource, _ int) string { return s.ExternalID })
		assert.NotContains(t, sourceIDs, scenarioModelExternalID)
		assert.NotContains(t, sourceIDs, scenarioTableExternalID)

		accounts, err := newAccountsAPIClient(t).Accounts.ListAll(context.Background(), client.WithHasExternalID(true))
		require.NoError(t, err, "listing managed accounts")
		assert.NotContains(t, lo.Map(accounts, func(a client.Account, _ int) string { return a.ExternalID }), scenarioAccountExternalID)
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

// assertNoRETLConnectionChanges dry-runs the project and checks that no
// connection is planned. The whole plan cannot be required empty: the account
// secret cannot be read back, so the account re-plans on every apply.
func assertNoRETLConnectionChanges(t *testing.T, executor *CmdExecutor, dir, credentials string) {
	t.Helper()

	out, err := executor.Execute(cliBinPath, "apply", "-l", dir, "--var-file", credentials, "--dry-run", "--confirm=false")
	require.NoError(t, err, "dry run failed: %s", out)
	assert.NotContains(t, string(out), "retl-connection:", "a re-apply must not plan any connection change")
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
