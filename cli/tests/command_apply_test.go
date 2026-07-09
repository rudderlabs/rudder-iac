package tests

import (
	"context"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
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
		assertWorkspaceBanner(t, output)
		verifyState(t, "create")
	})

	t.Run("should update entities in catalog from project", func(t *testing.T) {
		time.Sleep(5 * time.Second)

		output, err := executor.Execute(cliBinPath, "apply", "-l", updateDir, "--var-file", varFilePath, "--confirm=false")
		require.NoError(t, err, "Update apply command failed with output: %s", string(output))
		assertWorkspaceBanner(t, output)
		verifyState(t, "update")
	})

	t.Run("applying on already applied project should not create any diff", func(t *testing.T) {
		// If we reapply the update directory, we should
		// not see any changes meaning double apply without any changes
		// should report no changes to apply.
		verifyNoChangesToApply(t, executor, updateDir)
	})
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
	assertWorkspaceBanner(t, output)
	assert.Contains(t, string(output), "No changes to apply", "Expected no diff after migration, but got: %s", string(output))
}

func assertWorkspaceBanner(t *testing.T, output []byte) {
	t.Helper()

	assert.Regexp(
		t,
		regexp.MustCompile(`(?m)^Workspace: .+ \(.+\)$`),
		string(output),
		"Expected apply output to include workspace name and id banner, got: %s",
		string(output),
	)
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
