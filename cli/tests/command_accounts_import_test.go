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
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// importedAccountName is the upstream name of the unmanaged account seeded for
// the import to discover. It doubles as the needle that finds the right spec
// among however many other accounts the target workspace happens to hold.
const importedAccountName = "e2e-import-bigquery"

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
// `import workspace` refuses to run while the project has changes to sync, so
// the test starts from the same `destroy` the rest of the accounts e2e uses:
// it needs a workspace with no managed resources. Like TestAccountsApply, this
// therefore requires a DISPOSABLE workspace — never point it at one whose
// contents you care about.
//
// After import, the generated project is pruned to the seeded account (specs and
// manifest alike) before applying. `import workspace` necessarily scaffolds every
// importable resource in the workspace; without the prune, apply would reconcile
// resources the test never asked for, and any one of them failing validation
// upstream would fail this test for reasons unrelated to accounts.
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

	// Import is refused while anything is out of sync, so clear the managed
	// resources first — the same precondition TestAccountsApply establishes.
	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	seededID := seedUnmanagedAccount(t, apiClient)

	// The project starts empty — import scaffolds everything into <dir>/imported.
	projectDir := t.TempDir()

	out, err = executor.Execute(cliBinPath, "import", "workspace", "-l", projectDir)
	require.NoError(t, err, "import workspace failed: %s", out)
	assertNoSecretsInOutput(t, out)

	importedDir := filepath.Join(projectDir, importer.ImportedDir)

	specPath, spec := findAccountSpec(t, importedDir)
	assert.Contains(t, spec, "SOURCE_BIGQUERY", "scaffolded spec must carry the account definition")

	reference := varReference.FindStringSubmatch(spec)
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

	// Reduce the generated project to the seeded account before applying.
	pruneToAccount(t, importedDir, specPath, seededID)

	// Fill in the placeholder and adopt the account. Import only scaffolds; the
	// remote is claimed (Update + SetExternalID) on apply.
	require.NoError(t, os.WriteFile(varFile, []byte(varName+": "+rawAccountSecret+"\n"), 0o600))

	out, err = executor.Execute(cliBinPath, "apply", "-l", projectDir, "--var-file", varFile, "--confirm=false")
	require.NoError(t, err, "apply after import failed: %s", out)
	assertNoSecretsInOutput(t, out)

	// Adopted rather than recreated: the same upstream account now carries an
	// external id, so it has left the importable set.
	adopted, err := apiClient.Accounts.Get(ctx, seededID)
	require.NoError(t, err, "the seeded account must still exist after apply")
	assert.NotEmpty(t, adopted.ExternalID, "apply must claim the account with an external id")
	assert.NotContains(t, string(adopted.Options), rawAccountSecret,
		"the credential is write-only — it must never come back in options")

	importable, err := apiClient.Accounts.ListAll(ctx, client.WithHasExternalID(false))
	require.NoError(t, err)
	for _, account := range importable {
		assert.NotEqual(t, seededID, account.ID, "imported account must no longer be importable")
	}

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
		Name:                  importedAccountName,
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

// findAccountSpec returns the scaffolded spec for the seeded account. The
// workspace may hold other importable accounts, so the spec is matched by name
// rather than by being the only one present.
func findAccountSpec(t *testing.T, importedDir string) (string, string) {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(importedDir, "accounts", "*.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, matches, "expected a scaffolded account spec in %s", importedDir)

	for _, path := range matches {
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		if strings.Contains(string(content), importedAccountName) {
			return path, string(content)
		}
	}

	t.Fatalf("no scaffolded spec found for account %q among %v", importedAccountName, matches)
	return "", ""
}

// pruneToAccount reduces the generated project to the seeded account: every
// other scaffolded spec is removed, and the import manifest is filtered to the
// matching URN. Both halves are needed — a manifest entry whose spec is gone is
// an orphaned URN and fails validation.
func pruneToAccount(t *testing.T, importedDir, keepSpec, remoteID string) {
	t.Helper()

	entries, err := os.ReadDir(importedDir)
	require.NoError(t, err)
	for _, entry := range entries {
		path := filepath.Join(importedDir, entry.Name())
		switch entry.Name() {
		case "accounts", importer.SecretsVarFileName, importmanifest.FileName:
			continue
		default:
			require.NoError(t, os.RemoveAll(path))
		}
	}

	accountSpecs, err := filepath.Glob(filepath.Join(importedDir, "accounts", "*.yaml"))
	require.NoError(t, err)
	for _, path := range accountSpecs {
		if path != keepSpec {
			require.NoError(t, os.Remove(path))
		}
	}

	filterManifest(t, filepath.Join(importedDir, importmanifest.FileName), remoteID)
}

// filterManifest rewrites the import manifest in place, keeping only the entry
// whose remote id is the seeded account. It decodes and re-emits through the
// production types (specs.WorkspacesImportMetadata, importmanifest.BuildNode)
// rather than a hand-rolled struct, so the test cannot drift from the real
// manifest shape.
func filterManifest(t *testing.T, path, remoteID string) {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var manifest struct {
		Spec specs.WorkspacesImportMetadata `yaml:"spec"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &manifest))

	var kept []importmanifest.ImportEntry
	for _, workspace := range manifest.Spec.Workspaces {
		for _, resource := range workspace.Resources {
			if resource.RemoteID == remoteID {
				kept = append(kept, importmanifest.ImportEntry{
					WorkspaceID: workspace.WorkspaceID,
					URN:         resource.URN,
					RemoteID:    resource.RemoteID,
				})
			}
		}
	}
	require.Len(t, kept, 1, "manifest must map the seeded account %s to a URN", remoteID)

	node, err := importmanifest.BuildNode(kept)
	require.NoError(t, err)
	filtered, err := yaml.Marshal(node)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, filtered, 0o644))
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
