package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rawAccountSecret is the literal credentials value supplied via the var file.
// It must never surface in CLI output — the secret is write-only and the API
// never returns it.
const rawAccountSecret = "dummy-bq-service-account-key-12345"

// TestAccountsApply drives the accounts provider end-to-end against a live stack:
// apply create → apply update → re-apply is a no-op. It needs a real backend with
// the SOURCE_BIGQUERY account definition deployed and a PAT, so it is skipped
// unless RUN_ACCOUNT_E2E=1 (mirrors how the other e2e tests assume a backend, but
// gated because accounts create real, credential-bearing resources).
//
// Note: the secret (credentials) is unknown on read, so it always diffs — the
// "no changes" assertion below therefore targets the non-secret fields via the
// dry-run summary. When a stack is wired in, tighten this to snapshot the
// upstream options the way command_apply_test.go does for the catalog.
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
	})

	t.Run("apply update", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "apply", "-l",
			filepath.Join(projectDir, "update"), "--var-file", varFile, "--confirm=false")
		require.NoError(t, err, "update apply failed: %s", out)
		assert.NotContains(t, string(out), rawAccountSecret, "raw secret must never appear in CLI output")
	})

	t.Run("re-apply is a no-op for non-secret fields", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "apply", "-l",
			filepath.Join(projectDir, "update"), "--var-file", varFile,
			"--dry-run", "--confirm=false")
		require.NoError(t, err, "dry-run failed: %s", out)
		// Only the credentials secret should be flagged (always-unknown); the
		// non-secret option fields (project, location) must be in sync.
		assert.NotContains(t, string(out), "project", "non-secret fields should not diff on re-apply")
		assert.NotContains(t, string(out), "location", "non-secret fields should not diff on re-apply")
	})
}
