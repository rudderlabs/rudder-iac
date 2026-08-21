package tests

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	essource "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const concurrencyForTest = 1

// varFilePath supplies values for the {{ .VAR }} placeholders in the create/update
// specs. It lives outside create/ and update/ (and uses the .vars.yaml suffix the
// loader skips), so it is never parsed as a resource spec.
var varFilePath = filepath.Join("testdata", "project", "substitution.vars.yaml")

func TestProjectApply(t *testing.T) {
	t.Setenv("RUDDERSTACK_X_TRANSFORMATIONS", "true")
	// The project also carries an event stream source, a destination, and the
	// connection linking them (both kinds are flag-gated). The create project
	// connects the endpoints; the update project drops the connection spec,
	// which must disconnect them without touching either endpoint.
	t.Setenv("RUDDERSTACK_X_DESTINATION_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_X_CONNECTION_SUPPORT", "true")

	// The api_tracking event keeps its name and description as {{ .VAR }}
	// placeholders resolved at apply time. The feature is gated, so both
	// experimental switches must be on for substitution to run at all.
	//   - API_TRACKING_DESCRIPTION comes from the var file only (no env var set).
	//   - API_TRACKING_NAME is in both the var file and the env var below; the env
	//     var wins, resolving to "API Tracking" (the var file value is ignored).
	// Both resolve to the values already in the snapshots, so a precedence
	// regression — env losing to the file — would fail the snapshot comparison.
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_ENABLE_VAR_SUBSTITUTION", "true")
	t.Setenv("RUDDER_API_TRACKING_NAME", "API Tracking")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	projectDir := filepath.Join("testdata", "project")

	t.Run("rudder specs", func(t *testing.T) {
		applyAndVerify(t, executor, projectDir)
	})

	t.Run("rudder/v1 specs after migration", func(t *testing.T) {
		migratedDir := copyAndMigrateProject(t, executor, projectDir)
		// to make sure migration is applied correctly, we need to verify no
		// changes are reported if we re-apply the same project, therefore we dedicatedly
		// test this scenario below
		verifyNoChangesToApply(t, executor, filepath.Join(migratedDir, "update"))
		// then we apply this project again from scratch and verify no
		// changes are reported in snapshot tests meaning after migration of the directory
		// the upstream resources are created same
		applyAndVerify(t, executor, migratedDir)
	})
}

func applyAndVerify(t *testing.T, executor *CmdExecutor, projectDir string) {
	t.Helper()

	output, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "Failed to destroy resources: %v, output: %s", err, string(output))

	var (
		createDir = filepath.Join(projectDir, "create")
		updateDir = filepath.Join(projectDir, "update")
	)

	t.Run("should create entities in catalog from project", func(t *testing.T) {
		output, err := executor.Execute(cliBinPath, "apply", "-l", createDir, "--var-file", varFilePath, "--confirm=false")
		require.NoError(t, err, "Initial apply command failed with output: %s", string(output))
		verifyState(t, "create")
		verifyConnectionsState(t, "create")
	})

	t.Run("should update entities in catalog from project", func(t *testing.T) {
		time.Sleep(5 * time.Second)

		output, err := executor.Execute(cliBinPath, "apply", "-l", updateDir, "--var-file", varFilePath, "--confirm=false")
		require.NoError(t, err, "Update apply command failed with output: %s", string(output))
		verifyState(t, "update")
		// The update project has no connection spec: the endpoints must be
		// disconnected. That they are otherwise untouched is covered by the
		// no-diff check that follows — both endpoints are part of the update
		// project, so touching them would surface as a diff.
		verifyConnectionsState(t, "update")
	})

	t.Run("applying on already applied project should not create any diff", func(t *testing.T) {
		// If we reapply the update directory, we should
		// not see any changes meaning double apply without any changes
		// should report no changes to apply.
		verifyNoChangesToApply(t, executor, updateDir)
	})
}

// Volatile upstream fields excluded from the connection-scenario snapshot
// comparisons. Like the other e2e ignore lists these are ignored by value, not
// by presence: the API must still return each key or the comparison fails.
var (
	connSourceSnapshotIgnore      = []string{"id", "workspaceId"}
	connDestinationSnapshotIgnore = []string{"id", "workspaceId", "version", "createdAt", "updatedAt"}
	connectionSnapshotIgnore      = []string{"id", "sourceId", "destinationId", "createdAt", "updatedAt"}
)

// verifyConnectionsState snapshot-compares the event stream source, the
// destination, and the connection linking them against
// testdata/expected/upstream/connections/<dir>, mirroring verifyAccountUpstream.
// The update dir deliberately has no connection snapshot: dropping the
// connection spec must disconnect the endpoints, so no managed connection may
// remain while both endpoint snapshots still match.
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
	var destination *client.Destination
	for i := range destinations {
		if destinations[i].ExternalID == "e2e-conn-s3" {
			destination = &destinations[i]
			break
		}
	}
	require.NotNil(t, destination, "managed destination missing upstream")
	assert.NoError(t, helpers.CompareStates(
		toJSONMap(t, destination),
		readJSONFile(t, filepath.Join(expectedDir, "destination_e2e-conn-s3.json")),
		connDestinationSnapshotIgnore,
	), "upstream destination snapshot mismatch for %s", dir)

	conns := listConnections(t, ctx, apiClient, client.WithConnectionsHasExternalID(true))

	// The snapshot files are the expectation: a connection snapshot present
	// means exactly that managed connection must exist; absent means none may.
	connSnapshot := filepath.Join(expectedDir, "connection_e2e-android-to-s3.json")
	if _, statErr := os.Stat(connSnapshot); statErr == nil {
		require.Len(t, conns, 1, "expected exactly one managed connection upstream")
		// The snapshot ignores the server-assigned endpoint ids, so the wiring
		// is asserted directly: the connection must link exactly these endpoints.
		assert.Equal(t, source.ID, conns[0].SourceID, "connection must link the managed source")
		assert.Equal(t, destination.ID, conns[0].DestinationID, "connection must link the managed destination")
		assert.NoError(t, helpers.CompareStates(
			toJSONMap(t, conns[0]),
			readJSONFile(t, connSnapshot),
			connectionSnapshotIgnore,
		), "upstream connection snapshot mismatch for %s", dir)
	} else {
		require.ErrorIs(t, statErr, os.ErrNotExist,
			"unexpected error probing connection snapshot %s", connSnapshot)
		assert.Empty(t, conns, "no connection snapshot for %q: no managed connection may remain", dir)
		// Disconnected must hold for the endpoints themselves, not just for
		// externalId-carrying rows: no connection at all may link this pair.
		for _, conn := range listConnections(t, ctx, apiClient) {
			assert.False(t, conn.SourceID == source.ID && conn.DestinationID == destination.ID,
				"source and destination must be disconnected, but connection %s links them", conn.ID)
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

func verifyNoChangesToApply(t *testing.T, executor *CmdExecutor, path string) {
	t.Helper()

	// we only verify no diff after migration for the update directory, as the last apply was run on it.
	// The var file is passed so the {{ .VAR }} placeholders resolve to the same values that were
	// applied; otherwise the file-only variable would be undefined and the dry run would error.
	output, err := executor.Execute(
		cliBinPath,
		"apply",
		"-l",
		path,
		"--var-file",
		varFilePath,
		"--dry-run",
		"--confirm=false",
	)
	require.NoError(t, err, "Dry run failed for update: %s", string(output))
	assert.Contains(t, string(output), "No changes to apply", "Expected no diff after migration, but got: %s", string(output))
}

func copyAndMigrateProject(t *testing.T, executor *CmdExecutor, projectDir string) string {
	t.Helper()

	tempDir := t.TempDir()
	for _, dir := range []string{"create", "update"} {
		src := filepath.Join(projectDir, dir)
		dst := filepath.Join(tempDir, dir)

		out, err := exec.Command("cp", "-r", src, dst).CombinedOutput()
		require.NoError(t, err, "Failed to copy %s to %s: %s", src, dst, string(out))

		// migrate now substitutes {{ .VAR }} placeholders too (experimental flag is
		// on for this test), so it needs the var file to resolve the file-only variable.
		output, err := executor.Execute(cliBinPath, "migrate", "-l", dst, "--var-file", varFilePath, "--confirm=false")
		require.NoError(t, err, "Migration failed for %s: %s", dir, string(output))
	}

	return tempDir
}

func verifyState(t *testing.T, dir string) {
	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)

	require.NoError(t, err)
	dataCatalog, err := catalog.NewRudderDataCatalog(
		apiClient,
		catalog.WithConcurrency(concurrencyForTest),
		catalog.WithEventUpdateBatchSize(1),
	)
	require.NoError(t, err)
	reader := helpers.NewAPIClientAdapter(dataCatalog)

	expectedStateDir := filepath.Join("testdata", "expected", "upstream", dir)
	fileManager, err := helpers.NewSnapshotFileManager(expectedStateDir)
	require.NoError(t, err)

	upstreamTester := helpers.NewUpstreamSnapshotTester(
		dataCatalog,
		reader,
		fileManager,
		[]string{
			"id",
			"createdAt",
			"updatedAt",
			"createdBy",
			"updatedBy",
			"workspaceId",
			"categoryId",
			"version",
			"definitionId",
			"itemDefinitionId",
			"properties[0].id",
			"properties[1].id",
			"events[0].properties[0].id",
			"events[0].properties[1].id",
			"events[0].properties[2].id",
			"events[0].properties[3].id",
			"events[0].properties[4].id",
			"events[0].id",
			"events[0].createdAt",
			"events[0].updatedAt",
			"events[0].workspaceId",
			"events[0].createdBy",
			"events[0].updatedBy",
			"events[0].categoryId",
			"events[0].variants[0].discriminator",
			"events[0].variants[0].cases[0].properties[0].id",
			"events[0].variants[0].cases[0].properties[1].id",
			"events[1].properties[0].id",
			"events[1].properties[1].id",
			"events[1].properties[1].properties[0].id",
			"events[1].properties[1].properties[0].properties[0].id",
			"events[1].properties[1].properties[0].properties[0].properties[0].id",
			"events[1].properties[1].properties[0].properties[1].id",
			"events[1].properties[1].properties[1].id",
			"events[1].properties[2].id",
			"events[1].properties[2].properties[0].id",
			"events[1].properties[2].properties[0].properties[0].id",
			"events[1].properties[2].properties[0].properties[0].properties[0].id",
			"events[1].properties[2].properties[0].properties[1].id",
			"events[1].properties[2].properties[1].id",
			"events[1].properties[2].properties[1].properties[0].id",
			"events[1].properties[2].properties[1].properties[1].id",
			"events[1].properties[2].properties[1].properties[1].properties[0].id",
			"events[1].properties[3].id",
			"events[2].properties[0].id",
			"events[2].properties[1].id",
			"events[2].properties[1].properties[0].id",
			"events[2].properties[1].properties[0].properties[0].id",
			"events[2].properties[1].properties[0].properties[1].id",
			"events[2].properties[1].properties[0].properties[0].properties[0].id",
			"events[2].properties[1].properties[0].properties[0].properties[1].id",
			"events[2].properties[1].properties[1].id",
			"events[2].properties[2].id",
			"events[2].properties[2].properties[0].id",
			"events[2].properties[2].properties[1].id",
			"events[2].properties[2].properties[0].properties[0].id",
			"events[2].properties[2].properties[0].properties[1].id",
			"events[2].properties[2].properties[0].properties[0].properties[0].id",
			"events[2].properties[2].properties[0].properties[0].properties[1].id",
			"events[2].properties[3].id",
			"events[1].properties[2].id",
			"events[1].properties[3].id",
			"events[1].variants[0].discriminator",
			"events[1].variants[0].cases[0].properties[0].id",
			"events[1].variants[0].cases[1].properties[0].id",
			"events[1].variants[0].default[0].id",
			"events[1].variants[0].default[1].id",
			"events[1].id",
			"events[1].createdAt",
			"events[1].updatedAt",
			"events[1].workspaceId",
			"events[1].createdBy",
			"events[1].updatedBy",
			"events[1].categoryId",
			"events[2].properties[0].id",
			"events[2].properties[1].id",
			"events[2].id",
			"events[2].createdAt",
			"events[2].updatedAt",
			"events[2].workspaceId",
			"events[2].createdBy",
			"events[2].updatedBy",
			"events[2].categoryId",
		},
	)
	err = upstreamTester.SnapshotTest(context.Background())
	assert.NoError(t, err, "Upstream state verification failed")
}
