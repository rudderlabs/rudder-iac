package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitCloneIsApplyClean is the end-to-end promise of `init`: a workspace
// that is already under CLI management can be written out into an empty
// directory, and applying what comes out changes nothing.
//
// The apply cycle is what this covers, so it needs a real workspace whose
// contents it is allowed to replace — the same DISPOSABLE-workspace requirement
// TestProjectApply carries. It starts from the data-catalog project fixture so
// the clone has managed resources to reproduce, and asserts the dry run of the
// clone reports no changes: any identifier init failed to preserve would show up
// here as a create plus a delete.
//
// Opt-in until it has a CI-proven run against a dedicated workspace.
func TestInitCloneIsApplyClean(t *testing.T) {
	if os.Getenv("RUN_INIT_E2E") != "1" {
		t.Skip("set RUN_INIT_E2E=1 with a disposable live stack to run the init clone e2e")
	}

	allowManagedResidue(t)
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDER_API_TRACKING_NAME", "API Tracking")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	// Start from a known managed state.
	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	createDir := filepath.Join("testdata", "project", "create")
	out, err = executor.Execute(cliBinPath, "apply", "-l", createDir, "--var-file", varFilePath, "--confirm=false")
	require.NoError(t, err, "seeding apply failed: %s", out)

	// Clone the workspace into an empty directory.
	cloneDir := t.TempDir()
	out, err = executor.Execute(cliBinPath, "init", "-l", cloneDir)
	require.NoError(t, err, "init failed: %s", out)

	// Init writes the project at the root — no imported/ wrapper to unpack.
	_, err = os.Stat(filepath.Join(cloneDir, importer.ImportedDir))
	assert.True(t, os.IsNotExist(err), "init must not create an %s directory", importer.ImportedDir)
	assert.NotEmpty(t, collectSpecFiles(t, cloneDir), "init must write spec files")

	// The clone reproduces the workspace: applying it is a no-op. Any resource
	// whose identifier was not preserved would appear as a create here.
	verifyNoChangesToApply(t, executor, cloneDir)
}

// TestInitRefusesNonEmptyProject needs no live workspace: the guard runs before
// anything is fetched.
func TestInitRefusesNonEmptyProject(t *testing.T) {
	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "events.yaml"),
		[]byte("version: rudder/v1\nkind: events\n"), 0o600))

	out, err := executor.Execute(cliBinPath, "init", "-l", dir)
	require.Error(t, err, "init must refuse a directory that already holds specs, output: %s", out)
	assert.Contains(t, string(out), "already contains spec files")
	assert.Contains(t, string(out), "import workspace",
		"the error must point at the command that does handle an existing project")
}

func collectSpecFiles(t *testing.T, dir string) []string {
	t.Helper()

	var found []string
	require.NoError(t, filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, importer.SecretsVarFileName) {
			found = append(found, path)
		}
		return nil
	}))
	return found
}
