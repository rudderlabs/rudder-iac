package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initWizardE2EEnv gates the live arm. The wizard's apply reconciles the whole
// workspace, so the arm that actually applies needs a workspace whose contents
// it is allowed to replace, the same DISPOSABLE requirement TestProjectApply
// carries.
const initWizardE2EEnv = "RUN_INIT_WIZARD_E2E"

// TestInitWizardZeroSignalUnattended drives the binary with stdin closed and no
// flags beyond the two answers that have no default. Two things are being
// checked at once, and both are failures that would otherwise ship quietly:
//
//   - the run reaches a real plan rather than stalling on a prompt nobody can
//     answer, and
//   - when that plan would delete resources the wizard never asked about, the
//     unattended run refuses with a non-zero exit instead of reconciling the
//     workspace down to three specs.
//
// Against a workspace that holds nothing else, the refusal does not trigger and
// the run applies; that arm is TestInitWizardAppliesAndReportsWriteKey below.
// Either way the specs are on disk, so this also covers the generated project
// being readable by validate.
func TestInitWizardZeroSignalUnattended(t *testing.T) {
	dir := t.TempDir()

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	out, err := executor.Execute(cliBinPath, "init",
		"-l", dir,
		"--source-type", "node",
		"--webhook-url", "https://webhooks.example.com/rudder",
		"--yes",
	)

	// The ids are derived from the directory name, so the layout is asserted by
	// shape: one spec under sources/, one under destinations/, and connections.yaml.
	for _, pattern := range []string{"sources/*.yaml", "destinations/*.yaml", "connections.yaml"} {
		matches, globErr := filepath.Glob(filepath.Join(dir, pattern))
		require.NoError(t, globErr)
		assert.Len(t, matches, 1, "init must write %s even when the apply is refused, output: %s", pattern, out)
	}

	if err != nil {
		assert.Contains(t, string(out), "refusing to remove",
			"the only acceptable failure here is the deletion guard, output: %s", out)
		return
	}

	assert.NotContains(t, string(out), "No changes to apply",
		"an unattended run must not report success having applied nothing, output: %s", out)

	validateOut, err := executor.Execute(cliBinPath, "validate", "-l", dir)
	require.NoError(t, err, "validate rejected the generated project: %s", validateOut)
}

// TestInitWizardRefusesAnUnknownFlow needs no workspace: the flow check runs
// before anything is fetched or written.
func TestInitWizardRefusesAnUnknownFlow(t *testing.T) {
	dir := t.TempDir()

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	out, err := executor.Execute(cliBinPath, "init", "-l", dir, "--flow", "reverse-etl", "--yes")
	require.Error(t, err, "an unknown flow must fail, output: %s", out)
	assert.Contains(t, string(out), "unknown flow")

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries, "a rejected flow must not write anything")
}

// TestInitWizardNeedsASourceTypeWithoutPackageJSON: --yes must fail rather than
// prompt when there is no signal and no flag.
func TestInitWizardNeedsASourceTypeWithoutPackageJSON(t *testing.T) {
	dir := t.TempDir()

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	out, err := executor.Execute(cliBinPath, "init", "-l", dir,
		"--webhook-url", "https://webhooks.example.com/rudder", "--yes")
	require.Error(t, err, "output: %s", out)
	assert.Contains(t, string(out), "--source-type")
}

// TestInitWizardAppliesAndReportsWriteKey is the end-to-end promise: the wizard
// creates the pipeline and hands back the two values the rest of the demo needs,
// the source's write key and the workspace's data plane URL.
//
// Opt-in, and only against a disposable workspace: this arm applies for real.
func TestInitWizardAppliesAndReportsWriteKey(t *testing.T) {
	if os.Getenv(initWizardE2EEnv) != "1" {
		t.Skipf("set %s=1 with a disposable live workspace to run the init wizard apply", initWizardE2EEnv)
	}

	allowManagedResidue(t)

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	// Start from an empty workspace so the deletion guard has nothing to refuse.
	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"wizard-e2e"}`), 0o600))

	// package.json answers the source question, so only the URL is passed.
	out, err = executor.Execute(cliBinPath, "init", "-l", dir,
		"--webhook-url", "https://webhooks.example.com/rudder", "--yes")
	require.NoError(t, err, "init failed: %s", out)

	assert.Contains(t, string(out), "Found package.json")
	assert.Contains(t, string(out), "write key")
	assert.Contains(t, string(out), "data plane URL")
	assert.NotContains(t, string(out), "could not read the source's write key",
		"the write key is mandatory in the report, output: %s", out)

	// Applying the committed project again changes nothing.
	out, err = executor.Execute(cliBinPath, "apply", "-l", dir, "--dry-run")
	require.NoError(t, err, "dry run after init failed: %s", out)
	assert.Contains(t, string(out), "No changes to apply", "output: %s", out)

	// The specs were committed, which is what stands in for a revert command.
	out, err = executor.Execute("git", "-C", dir, "log", "--oneline")
	require.NoError(t, err, "init did not leave a git repository: %s", out)
	assert.NotEmpty(t, string(out))
}

// TestInitWizardRefusesASecondRunInTheSameDirectory: a repeated init must say
// that the directory stopped it, not the workspace. The two refusals have
// different fixes, deleting the generated specs versus starting from an empty
// workspace, so the demo runbook needs to be able to tell them apart.
func TestInitWizardRefusesASecondRunInTheSameDirectory(t *testing.T) {
	dir := t.TempDir()

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	args := []string{"init", "-l", dir,
		"--source-type", "node",
		"--webhook-url", "https://webhooks.example.com/rudder",
		"--yes"}

	// The first run may be refused by the workspace, but it still writes.
	_, _ = executor.Execute(cliBinPath, args...)

	out, err := executor.Execute(cliBinPath, args...)
	require.Error(t, err, "a second init in the same directory must fail, output: %s", out)
	assert.Contains(t, string(out), "already been initialised")
	assert.NotContains(t, string(out), "refusing to remove",
		"the directory stopped this run, so it must not be reported as a workspace problem")
}
