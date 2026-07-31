package tests

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// destinationSnapshotIgnore are the volatile upstream fields excluded from the
// snapshot comparison: server-assigned id, workspace scoping, version, and
// timestamps. Write-only secrets (e.g. S3 access keys) are never returned by the
// API, so they never appear here.
var destinationSnapshotIgnore = []string{"id", "workspaceId", "version", "createdAt", "updatedAt"}

// destinationRawSecrets are literal secret values from the var file that must
// never surface in CLI output.
var destinationRawSecrets = []string{
	"AKIAIOSFODNN7EXAMPLE",
	"wJalrXUtnFEMI-K7MDENG-bPxRfiCYEXAMPLEKEY",
}

// TestDestinationsApply drives the destination provider end-to-end against a live
// stack: apply create → apply update → re-apply is a no-op. All destination specs
// live side by side under one project folder and are applied together (like the
// catalog e2e applies events, properties, etc. in one shot), then the managed
// destinations are snapshot-compared upstream via DestinationSnapshotTester — the
// same file-manager + count-guard + per-resource-compare structure verifyState
// uses. Destinations are double-gated — DestinationSupport enables the kind,
// UnverifiedDestinations registers the unverified ones (e.g. S3) — and key-auth
// specs reference secrets via {{ .VAR }} placeholders resolved at apply time.
func TestDestinationsApply(t *testing.T) {
	t.Setenv("RUDDERSTACK_X_DESTINATION_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_X_UNVERIFIED_DESTINATIONS", "true")
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_ENABLE_VAR_SUBSTITUTION", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	projectDir := filepath.Join("testdata", "project", "destinations", "s3")
	varFile := filepath.Join(projectDir, "s3.vars.yaml")

	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	// A leftover unverified destination breaks other e2e tests' destroy when
	// UnverifiedDestinations is off. Registered after t.Setenv so the flags still
	// hold when this cleanup runs.
	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		require.NoError(t, err, "cleanup destroy failed: %s", out)
	})

	apply := func(dir string) {
		out, err := executor.Execute(cliBinPath, "apply", "-l",
			filepath.Join(projectDir, dir), "--var-file", varFile, "--confirm=false")
		require.NoError(t, err, "%s apply failed: %s", dir, out)
		for _, secret := range destinationRawSecrets {
			assert.NotContains(t, string(out), secret, "raw secret must never appear in CLI output")
		}
	}

	t.Run("apply create", func(t *testing.T) {
		apply("create")
		verifyDestinationState(t, "create")
	})

	t.Run("apply update", func(t *testing.T) {
		apply("update")
		verifyDestinationState(t, "update")
	})

	t.Run("re-apply leaves upstream state unchanged", func(t *testing.T) {
		apply("update")
		verifyDestinationState(t, "update")
	})
}

// verifyDestinationState snapshot-compares the managed destinations against
// testdata/expected/upstream/destinations/<dir>, mirroring verifyState.
func verifyDestinationState(t *testing.T, dir string) {
	t.Helper()

	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)
	require.NoError(t, err)

	expectedStateDir := filepath.Join("testdata", "expected", "upstream", "destinations", dir)
	fileManager, err := helpers.NewSnapshotFileManager(expectedStateDir)
	require.NoError(t, err)

	tester := helpers.NewDestinationSnapshotTester(
		apiClient.Destinations,
		fileManager,
		destinationSnapshotIgnore,
	)
	assert.NoError(t, tester.SnapshotTest(context.Background()),
		"upstream destination state verification failed for %q", dir)
}
