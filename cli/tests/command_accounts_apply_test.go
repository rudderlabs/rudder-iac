package tests

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rawAccountSecret is the literal credentials value supplied via the var file.
// It must never surface in CLI output — the secret is write-only and the API
// never returns it.
const rawAccountSecret = "dummy-bq-service-account-key-12345"

// accountSnapshotIgnore are the volatile upstream fields excluded from the
// snapshot comparison: server-assigned id, workspace scoping, and timestamps.
// The secret is never returned by the API, so it never appears here.
var accountSnapshotIgnore = []string{"id", "workspaceId", "createdAt", "updatedAt"}

// TestAccountsApply drives the accounts provider end-to-end against a live stack:
// apply create → apply update → re-apply is a no-op. It needs a real backend with
// the SOURCE_BIGQUERY account definition deployed and a PAT, so it is skipped
// unless RUN_ACCOUNT_E2E=1 (mirrors how the other e2e tests assume a backend, but
// gated because accounts create real, credential-bearing resources).
func TestAccountsApply(t *testing.T) {
	if os.Getenv("RUN_ACCOUNT_E2E") != "1" {
		t.Skip("set RUN_ACCOUNT_E2E=1 with a live stack (SOURCE_BIGQUERY definition + PAT) to run")
	}

	// Accounts are gated behind an experimental flag, and the specs reference
	// secrets via {{ .VAR }} placeholders resolved at apply time.
	t.Setenv("RUDDERSTACK_X_ACCOUNT_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_ENABLE_VAR_SUBSTITUTION", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	projectDir := filepath.Join("testdata", "accounts")
	varFile := filepath.Join(projectDir, "credentials.vars.yaml")

	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	t.Run("apply create", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "apply", "-l",
			filepath.Join(projectDir, "create"), "--var-file", varFile, "--confirm=false")
		require.NoError(t, err, "create apply failed: %s", out)
		assert.NotContains(t, string(out), rawAccountSecret, "raw secret must never appear in CLI output")
		verifyAccountUpstream(t, "create")
	})

	t.Run("apply update", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "apply", "-l",
			filepath.Join(projectDir, "update"), "--var-file", varFile, "--confirm=false")
		require.NoError(t, err, "update apply failed: %s", out)
		assert.NotContains(t, string(out), rawAccountSecret, "raw secret must never appear in CLI output")
		verifyAccountUpstream(t, "update")
	})

	t.Run("re-apply leaves non-secret upstream state unchanged", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "apply", "-l",
			filepath.Join(projectDir, "update"), "--var-file", varFile, "--confirm=false")
		require.NoError(t, err, "re-apply failed: %s", out)
		// The credentials secret is always-unknown so it re-applies every time;
		// the snapshot (non-secret fields) proves nothing else churned.
		verifyAccountUpstream(t, "update")
	})
}

// verifyAccountUpstream fetches the managed accounts from the API and
// snapshot-compares them against testdata/expected/upstream/accounts/<dir>, the
// same snapshot-based approach the catalog and transformations e2e tests use
// (verifyState / verifyTestResults) rather than asserting on CLI stdout.
func verifyAccountUpstream(t *testing.T, dir string) {
	t.Helper()

	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)
	require.NoError(t, err)

	accounts, err := apiClient.Accounts.ListAll(context.Background(), client.WithHasExternalID(true))
	require.NoError(t, err, "listing managed accounts")
	require.Len(t, accounts, 1, "expected exactly one managed account")

	// Round-trip through JSON so the actual value is a map with the API's field
	// names, matching the snapshot file.
	raw, err := json.Marshal(accounts[0])
	require.NoError(t, err)
	var actual map[string]any
	require.NoError(t, json.Unmarshal(raw, &actual))

	expected := readJSONFile(t, filepath.Join(
		"testdata", "expected", "upstream", "accounts", dir, "account.json"))

	assert.NoError(t, helpers.CompareStates(actual, expected, accountSnapshotIgnore),
		"upstream account snapshot mismatch for %q", dir)
}
