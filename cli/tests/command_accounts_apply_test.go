package tests

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rawAccountSecret is the literal BigQuery credentials value supplied via the var
// file. It must never surface in CLI output — the secret is write-only and the API
// never returns it.
const rawAccountSecret = "dummy-bq-service-account-key-12345"

// accountExternalIDs are the accounts the fixtures manage, one per supported
// warehouse definition. Each has a testdata/expected/upstream/accounts/<dir>/<id>.json
// snapshot.
var accountExternalIDs = []string{"prod-bq", "prod-pg", "prod-sf"}

// fixtureAccountSecrets is every secret value in credentials.vars.yaml. Snowflake
// carries more than one, so the leak assertion has to cover the whole set. These
// are fixture placeholders, not credentials -- hence the gitleaks exemption.
var fixtureAccountSecrets = []string{
	rawAccountSecret,
	"dummy-pg-password-12345",
	"dummy-sf-key-12345",
	"dummy-sf-phrase-12345", //gitleaks:allow
}

// accountSnapshotIgnore are the volatile upstream fields excluded from the
// snapshot comparison: server-assigned id, workspace scoping, and timestamps.
// The secret is never returned by the API, so it never appears here.
var accountSnapshotIgnore = []string{"id", "workspaceId", "createdAt", "updatedAt"}

// TestAccountsApply drives the accounts provider end-to-end against a live stack:
// apply create → apply update → re-apply is a no-op. It needs a real backend with
// the SOURCE_BIGQUERY, SOURCE_POSTGRES and SOURCE_SNOWFLAKE account definitions
// deployed and a PAT.
func TestAccountsApply(t *testing.T) {
	// Accounts are gated behind an experimental flag, and the specs reference
	// secrets via {{ .VAR }} placeholders resolved at apply time.
	allowUnverifiedDestinationResidue(t)
	t.Setenv("RUDDERSTACK_X_ACCOUNT_SUPPORT", "true")
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
		assertNoSecretsInOutput(t, out)
		verifyAccountUpstream(t, "create")
	})

	t.Run("apply update", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "apply", "-l",
			filepath.Join(projectDir, "update"), "--var-file", varFile, "--confirm=false")
		require.NoError(t, err, "update apply failed: %s", out)
		assertNoSecretsInOutput(t, out)
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

// assertNoSecretsInOutput fails if any fixture secret shows up in the given bytes —
// CLI output, or any file the CLI generated.
func assertNoSecretsInOutput(t *testing.T, out []byte) {
	t.Helper()
	for _, s := range fixtureAccountSecrets {
		assert.NotContains(t, string(out), s, "raw secret must never appear in CLI output")
	}
}

// verifyAccountUpstream fetches the managed accounts from the API and
// snapshot-compares each against testdata/expected/upstream/accounts/<dir>/<externalId>.json,
// the same snapshot-based approach the catalog and transformations e2e tests use
// (verifyState / verifyTestResults) rather than asserting on CLI stdout.
func verifyAccountUpstream(t *testing.T, dir string) {
	t.Helper()

	apiClient := newAccountsAPIClient(t)

	accounts, err := apiClient.Accounts.ListAll(context.Background(), client.WithHasExternalID(true))
	require.NoError(t, err, "listing managed accounts")
	require.Len(t, accounts, len(accountExternalIDs), "expected one managed account per fixture")

	// Round-trip through JSON so the actual values are maps with the API's field
	// names, matching the snapshot files.
	byExternalID := make(map[string]map[string]any, len(accounts))
	for _, account := range accounts {
		raw, err := json.Marshal(account)
		require.NoError(t, err)
		var actual map[string]any
		require.NoError(t, json.Unmarshal(raw, &actual))
		byExternalID[actual["externalId"].(string)] = actual
	}

	for _, externalID := range accountExternalIDs {
		actual, ok := byExternalID[externalID]
		require.True(t, ok, "managed account %q missing upstream", externalID)

		expected := readJSONFile(t, filepath.Join(
			"testdata", "expected", "upstream", "accounts", dir, externalID+".json"))

		assert.NoError(t, helpers.CompareStates(actual, expected, accountSnapshotIgnore),
			"upstream account snapshot mismatch for %s/%s", dir, externalID)
	}
}
