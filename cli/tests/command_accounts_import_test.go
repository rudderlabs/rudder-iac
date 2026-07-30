package tests

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// varReference matches the "{{ .VAR }}" token an exported secret is masked to,
// capturing the variable name so the scaffolded var file can be checked for the
// matching placeholder.
var varReference = regexp.MustCompile(`\{\{\s*\.([A-Za-z0-9_]+)\s*\}\}`)

// TestAccountsImportWorkspace drives the import half of the accounts provider
// against a live stack: an unmanaged account is seeded upstream, discovered by
// `import workspace`, and adopted by `apply`.
//
// The assertion that earns this test is the on-disk one: export masks the
// write-only credential to a "{{ .VAR }}" reference, so no generated file may
// ever contain the real secret — that masking is all that stands between a user
// and a plaintext credential committed to version control.
//
// Same RUN_ACCOUNT_E2E=1 gate as TestAccountsApply: it needs a real backend with
// the SOURCE_BIGQUERY definition deployed and a PAT.
func TestAccountsImportWorkspace(t *testing.T) {
	if os.Getenv("RUN_ACCOUNT_E2E") != "1" {
		t.Skip("set RUN_ACCOUNT_E2E=1 with a live stack (SOURCE_BIGQUERY definition + PAT) to run")
	}

	t.Setenv("RUDDERSTACK_X_ACCOUNT_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_ENABLE_VAR_SUBSTITUTION", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	ctx := context.Background()
	apiClient := newAccountsAPIClient(t)

	// Start from a clean slate: a managed account left by a previous run would
	// be picked up by apply below and skew the importable-set assertion.
	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	seededID := seedUnmanagedAccount(t, apiClient)

	// The project starts empty — import scaffolds everything into <dir>/imported.
	projectDir := t.TempDir()

	out, err = executor.Execute(cliBinPath, "import", "workspace", "-l", projectDir)
	require.NoError(t, err, "import workspace failed: %s", out)
	assert.NotContains(t, string(out), rawAccountSecret, "raw secret must never appear in CLI output")

	importedDir := filepath.Join(projectDir, importer.ImportedDir)

	matches, err := filepath.Glob(filepath.Join(importedDir, "accounts", "*.yaml"))
	require.NoError(t, err)
	require.Len(t, matches, 1, "expected exactly one scaffolded account spec in %s", importedDir)

	spec, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Contains(t, string(spec), "SOURCE_BIGQUERY", "scaffolded spec must carry the account definition")

	reference := varReference.FindStringSubmatch(string(spec))
	require.NotNil(t, reference, "credentials must be exported as a {{ .VAR }} reference, got:\n%s", spec)
	varName := reference[1]

	varFile := filepath.Join(importedDir, importer.SecretsVarFileName)
	scaffoldedVars, err := os.ReadFile(varFile)
	require.NoError(t, err, "import must scaffold %s", importer.SecretsVarFileName)
	assert.Contains(t, string(scaffoldedVars), varName+":",
		"var file must hold a placeholder for every variable the specs reference")

	// Nothing written to disk may carry the credential — this covers the specs,
	// the var file and the import manifest in one sweep.
	assertNoSecretOnDisk(t, projectDir)

	// Fill in the placeholder and adopt the account. Import only scaffolds; the
	// remote is claimed (Update + SetExternalID) on apply.
	require.NoError(t, os.WriteFile(varFile, []byte(varName+": "+rawAccountSecret+"\n"), 0o600))

	out, err = executor.Execute(cliBinPath, "apply", "-l", projectDir, "--var-file", varFile, "--confirm=false")
	require.NoError(t, err, "apply after import failed: %s", out)
	assert.NotContains(t, string(out), rawAccountSecret, "raw secret must never appear in CLI output")

	// Adopted: the account carries an external id now, so it has left the
	// importable set and re-importing would no longer offer it.
	importable, err := apiClient.Accounts.ListAll(ctx, client.WithHasExternalID(false))
	require.NoError(t, err)
	for _, account := range importable {
		assert.NotEqual(t, seededID, account.ID, "imported account must no longer be importable")
	}

	managed, err := apiClient.Accounts.ListAll(ctx, client.WithHasExternalID(true))
	require.NoError(t, err)
	require.Len(t, managed, 1, "expected exactly one managed account after import")
	assert.Equal(t, seededID, managed[0].ID, "apply must adopt the seeded account, not create a new one")
	assert.NotContains(t, string(managed[0].Options), rawAccountSecret,
		"the credential is write-only — it must never come back in options")

	// Re-apply must stay clean: the always-unknown secret re-applies, nothing else.
	out, err = executor.Execute(cliBinPath, "apply", "-l", projectDir, "--var-file", varFile, "--confirm=false")
	require.NoError(t, err, "re-apply after import failed: %s", out)
}

// newAccountsAPIClient builds an API client from the same config the CLI binary
// uses, so the test and the CLI act on one workspace.
func newAccountsAPIClient(t *testing.T) *client.Client {
	t.Helper()

	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)
	require.NoError(t, err)
	return apiClient
}

// seedUnmanagedAccount creates the account the import is meant to discover: no
// externalId, so the provider sees it as importable. It is deleted on cleanup
// whether or not the adoption succeeded.
func seedUnmanagedAccount(t *testing.T, apiClient *client.Client) string {
	t.Helper()

	options, err := json.Marshal(map[string]any{"project": "acme-analytics-import", "location": "US"})
	require.NoError(t, err)
	secretPayload, err := json.Marshal(map[string]any{"credentials": rawAccountSecret})
	require.NoError(t, err)

	account, err := apiClient.Accounts.Create(context.Background(), &client.CreateAccountRequest{
		AccountDefinitionName: "SOURCE_BIGQUERY",
		Name:                  "e2e-import-bigquery",
		Options:               options,
		Secret:                secretPayload,
	})
	require.NoError(t, err, "seeding an unmanaged account")

	t.Cleanup(func() {
		if err := apiClient.Accounts.Delete(context.Background(), account.ID); err != nil {
			t.Logf("cleaning up seeded account %s: %v", account.ID, err)
		}
	})

	return account.ID
}

// assertNoSecretOnDisk walks every generated file and fails on the raw secret.
// A grep rather than a targeted field check on purpose: the point is that the
// credential is absent from the whole generated tree, wherever it might land.
func assertNoSecretOnDisk(t *testing.T, dir string) {
	t.Helper()

	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		assert.NotContains(t, string(content), rawAccountSecret,
			"raw secret leaked into %s", strings.TrimPrefix(path, dir))
		return nil
	})
	require.NoError(t, err)
}
