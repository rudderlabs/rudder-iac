package tests

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	essource "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Volatile upstream fields excluded from the connection-scenario snapshot
// comparisons. Like the other e2e ignore lists, a key a snapshot records must
// still come back — ignoring it drops the value comparison, not the presence
// check — while an ignored key the snapshot does not record may be extra in the
// response. That is what keeps the conditional versionInfo advisory harmless.
var (
	connSourceSnapshotIgnore      = []string{"id", "workspaceId"}
	connDestinationSnapshotIgnore = []string{"id", "workspaceId", "version", "versionInfo", "createdAt", "updatedAt"}
	connectionSnapshotIgnore      = []string{"id", "sourceId", "destinationId", "createdAt", "updatedAt"}
)

// connScenarioEndpoints pairs each managed destination with the connection
// linking the android source to it. Both are here because a destination
// declares its per-source settings in one of two blocks and the connect-time
// check accepts either: s3 satisfies it through connection_mode, while
// firebase declares no connection_mode at all (schema.json has no such
// property), so its use_native_sdk entry is the only thing that can.
var connScenarioEndpoints = []struct {
	destination string
	connection  string
}{
	{destination: "e2e-conn-s3", connection: "e2e-android-to-s3"},
	{destination: "e2e-conn-firebase", connection: "e2e-android-to-firebase"},
}

// TestConnectionsApply drives the event stream source → destination → connection
// trio end-to-end against a live stack: apply create wires the endpoints
// together, apply update drops the connection spec and must disconnect them
// without touching either endpoint, and a final re-apply proves nothing but the
// write-only secret churns.
//
// These specs live in their own project rather than in testdata/project/ because
// the destination carries a write-only access key. TestProjectApply asserts that
// re-applying its project reports "No changes to apply", and a spec whose secret
// can never be read back can never satisfy that. The destinations e2e hit the
// same wall and solved it the same way, so this suite follows it: snapshot-compare
// the non-secret upstream fields instead of asserting a clean no-op.
//
// Gated behind RUN_CONNECTION_E2E, mirroring TestDestinationsApply: it needs a
// live stack whose destination snapshot matches the committed fixture, and the
// destroy below would otherwise wipe the workspace before the (failing) apply on
// a stack without that support. The skip must come before the destroy.
func TestConnectionsApply(t *testing.T) {
	if os.Getenv("RUN_CONNECTION_E2E") != "1" {
		t.Skip("set RUN_CONNECTION_E2E=1 with a live connection-enabled stack")
	}

	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_DESTINATION_SUPPORT", "true")
	// firebase is registered behind UnverifiedDestinations, and it is the
	// use_native_sdk half of connScenarioEndpoints — s3, the only definition
	// registered without the flag, models connection_mode alone.
	t.Setenv("RUDDERSTACK_X_UNVERIFIED_DESTINATIONS", "true")
	t.Setenv("RUDDERSTACK_X_CONNECTION_SUPPORT", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	projectDir := filepath.Join("testdata", "connections")
	varFile := filepath.Join(projectDir, "connections.vars.yaml")

	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)
	assertNoRawSecrets(t, out)

	// Registered after t.Setenv so the flags still hold when this runs, otherwise
	// the managed destination is an unrecognised kind and cleanup cannot remove
	// it. assert (not require) avoids FailNow mid-cleanup.
	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		assert.NoError(t, err, "cleanup destroy failed: %s", out)
		assertNoRawSecrets(t, out)
	})

	apply := func(t *testing.T, dir string) {
		t.Helper()
		out, err := executor.Execute(cliBinPath, "apply", "-l",
			filepath.Join(projectDir, dir), "--var-file", varFile, "--confirm=false")
		require.NoError(t, err, "%s apply failed: %s", dir, out)
		assertNoRawSecrets(t, out)
	}

	t.Run("apply create connects the endpoints", func(t *testing.T) {
		apply(t, "create")
		verifyConnectionsState(t, "create")
	})

	// The update project has no connection spec: the endpoints must be
	// disconnected. That they are otherwise untouched is covered by their
	// snapshots, which are compared again below.
	t.Run("apply update disconnects the endpoints", func(t *testing.T) {
		apply(t, "update")
		verifyConnectionsState(t, "update")
	})

	// Re-apply cannot be a full no-op here: the destination's access keys are
	// write-only, so they map to always-unknown secrets that re-apply every run
	// (see secret.String.Diff). A dry-run would therefore always report a diff.
	// Snapshot the non-secret upstream fields instead to prove nothing else
	// churns, matching TestDestinationsApply's re-apply subtest.
	t.Run("re-apply churns only the write-only secret", func(t *testing.T) {
		apply(t, "update")
		verifyConnectionsState(t, "update")
	})
}

// verifyConnectionsState snapshot-compares the event stream source, every
// destination in connScenarioEndpoints, and the connections linking them
// against testdata/expected/upstream/connections/<dir>, mirroring
// verifyAccountUpstream. The update dir deliberately has no connection
// snapshots: dropping the connection specs must disconnect the endpoints, so
// no managed connection may remain while every endpoint snapshot still matches.
func verifyConnectionsState(t *testing.T, dir string) {
	t.Helper()

	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)
	require.NoError(t, err)

	ctx := context.Background()
	expectedDir := filepath.Join("testdata", "expected", "upstream", "connections", dir)

	sources, err := essource.NewRudderSourceStore(apiClient).GetSources(ctx)
	require.NoError(t, err, "listing event stream sources")
	var source *essource.EventStreamSource
	for i := range sources {
		if sources[i].ExternalID == "e2e-conn-android" {
			source = &sources[i]
			break
		}
	}
	require.NotNil(t, source, "managed event stream source missing upstream")
	assert.NoError(t, helpers.CompareStates(
		toJSONMap(t, source),
		readJSONFile(t, filepath.Join(expectedDir, "event-stream-source_e2e-conn-android.json")),
		connSourceSnapshotIgnore,
	), "upstream event stream source snapshot mismatch for %s", dir)

	destinations, err := apiClient.Destinations.GetAll(ctx)
	require.NoError(t, err, "listing destinations")
	destinationIDs := make(map[string]string, len(connScenarioEndpoints))
	for _, endpoint := range connScenarioEndpoints {
		var destination *client.Destination
		for i := range destinations {
			if destinations[i].ExternalID == endpoint.destination {
				destination = &destinations[i]
				break
			}
		}
		require.NotNil(t, destination, "managed destination %s missing upstream", endpoint.destination)
		destinationIDs[endpoint.destination] = destination.ID
		assert.NoError(t, helpers.CompareStates(
			toJSONMap(t, destination),
			readJSONFile(t, filepath.Join(expectedDir, "destination_"+endpoint.destination+".json")),
			connDestinationSnapshotIgnore,
		), "upstream destination snapshot mismatch for %s/%s", dir, endpoint.destination)
	}

	conns := listConnections(t, ctx, apiClient, client.WithConnectionsHasExternalID(true))
	connsByExternalID := make(map[string]client.Connection, len(conns))
	for _, conn := range conns {
		connsByExternalID[conn.ExternalID] = conn
	}

	// The snapshot files are the expectation: a connection snapshot present
	// means exactly that managed connection must exist; absent means none may.
	// Counting the expected ones keeps "no extras upstream" an assertion too.
	var expected int
	for _, endpoint := range connScenarioEndpoints {
		connSnapshot := filepath.Join(expectedDir, "connection_"+endpoint.connection+".json")
		if _, statErr := os.Stat(connSnapshot); statErr != nil {
			require.ErrorIs(t, statErr, os.ErrNotExist,
				"unexpected error probing connection snapshot %s", connSnapshot)
			assert.NotContains(t, connsByExternalID, endpoint.connection,
				"no snapshot for %q in %q: that managed connection may not remain", endpoint.connection, dir)
			continue
		}

		expected++
		conn, found := connsByExternalID[endpoint.connection]
		if !assert.True(t, found, "managed connection %s missing upstream", endpoint.connection) {
			continue
		}
		// The snapshot ignores the server-assigned endpoint ids, so the wiring
		// is asserted directly: the connection must link exactly these endpoints.
		assert.Equal(t, source.ID, conn.SourceID, "connection %s must link the managed source", endpoint.connection)
		assert.Equal(t, destinationIDs[endpoint.destination], conn.DestinationID,
			"connection %s must link destination %s", endpoint.connection, endpoint.destination)
		assert.NoError(t, helpers.CompareStates(
			toJSONMap(t, conn),
			readJSONFile(t, connSnapshot),
			connectionSnapshotIgnore,
		), "upstream connection snapshot mismatch for %s/%s", dir, endpoint.connection)
	}
	require.Len(t, conns, expected, "unexpected managed connections upstream")

	if expected == 0 {
		// Disconnected must hold for the endpoints themselves, not just for
		// externalId-carrying rows: no connection at all may link these pairs.
		for _, conn := range listConnections(t, ctx, apiClient) {
			if conn.SourceID != source.ID {
				continue
			}
			for destExternalID, destID := range destinationIDs {
				assert.NotEqual(t, destID, conn.DestinationID,
					"source and destination %s must be disconnected, but connection %s links them",
					destExternalID, conn.ID)
			}
		}
	}
}

// listConnections pages through the connections list with the given options.
func listConnections(t *testing.T, ctx context.Context, apiClient *client.Client, opts ...client.ListConnectionsOption) []client.Connection {
	t.Helper()
	var conns []client.Connection
	page, err := apiClient.Connections.List(ctx, opts...)
	require.NoError(t, err, "listing connections")
	for page != nil {
		conns = append(conns, page.Connections...)
		page, err = apiClient.Connections.Next(ctx, page.Paging)
		require.NoError(t, err, "paging connections")
	}
	return conns
}

// toJSONMap round-trips a value through JSON so the actual values are maps with
// the API's field names, matching the snapshot files.
func toJSONMap(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}
