package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

// The upstream names of what TestRETLSourcesImportClaim seeds outside the CLI.
// They double as the needles that find a previous run's residue: seeds carry
// no externalId, so no destroy ever reaches them.
const (
	importSeedAccountName = "e2e-retl-import-pg"
	importSeedModelName   = "E2E Import Model"
	importSeedTableName   = "E2E Import Table"

	importModelLocalID = "e2e-import-model"
	importTableLocalID = "e2e-import-table"
)

// TestRETLSourcesImportClaim adopts RETL sources that were created outside the
// CLI — the webapp's path — and checks that apply claims the existing rows
// rather than creating copies.
//
//   - The SQL model goes through `import retl-sources`, which writes a spec
//     carrying the remote id as import metadata.
//   - The table source has no single-source import command; `import workspace`
//     is its only exporter. That command scaffolds every importable resource in
//     the workspace, so it is not run here (see TestAccountsImportWorkspace for
//     why that stays opt-in); the test writes the spec the exporter would, which
//     exercises the same claim on apply. The spec's table differs from the
//     remote one on purpose, so the claim is followed by an update.
//
// Everything seeded is unmanaged, the account included, so a run killed midway
// leaves nothing a later run's opening destroy has to delete. The claimed
// sources are managed on an unmanaged account, which destroy can delete. Seeds
// are found by name at the start of the next run and removed.
func TestRETLSourcesImportClaim(t *testing.T) {
	allowManagedResidue(t)

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	var (
		apiClient = newAccountsAPIClient(t)
		store     = retlClient.NewRudderRETLStore(apiClient)
	)

	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)
	removeImportSeeds(t, apiClient, store)

	// Registered before the destroy so it runs after it: the claimed sources
	// are destroy's to delete, and the unmanaged account they use cannot go
	// until they have.
	t.Cleanup(func() { removeImportSeeds(t, apiClient, store) })
	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		assert.NoError(t, err, "cleanup destroy failed: %s", out)
	})

	accountID := seedUnmanagedPostgresAccount(t, apiClient)
	model := seedUnmanagedRETLSource(t, apiClient, map[string]any{
		"name":                 importSeedModelName,
		"sourceType":           retlClient.ModelSourceType,
		"sourceDefinitionName": "postgres",
		"accountId":            accountID,
		"enabled":              true,
		"config":               map[string]any{"primaryKey": "id", "sql": "SELECT id, email FROM users"},
	})
	table := seedUnmanagedRETLSource(t, apiClient, map[string]any{
		"name":                 importSeedTableName,
		"sourceType":           retlClient.TableSourceType,
		"sourceDefinitionName": "postgres",
		"accountId":            accountID,
		"enabled":              true,
		"config":               map[string]any{"primaryKey": "id", "schema": "analytics", "table": "users"},
	})

	projectDir := t.TempDir()

	t.Run("import retl-sources writes a spec that names the remote source", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "import", "retl-sources",
			"--local-id", importModelLocalID, "--remote-id", model.ID, "--location", projectDir)
		require.NoError(t, err, "import retl-sources failed: %s", out)

		spec := readSpec(t, filepath.Join(projectDir, importModelLocalID+".yaml"))
		assert.Equal(t, &specs.Spec{
			Version:  specs.SpecVersionV1,
			Kind:     "retl-source-sql-model",
			Metadata: importMetadata(importModelLocalID, "retl-source-sql-model", model),
			Spec: map[string]any{
				"id":                importModelLocalID,
				"display_name":      importSeedModelName,
				"description":       "",
				"account_id":        accountID,
				"primary_key":       "id",
				"source_definition": "postgres",
				"enabled":           true,
				"sql":               "SELECT id, email FROM users",
			},
		}, spec)
	})

	writeSpec(t, filepath.Join(projectDir, importTableLocalID+".yaml"), &specs.Spec{
		Version:  specs.SpecVersionV1,
		Kind:     "retl-source-table",
		Metadata: importMetadata(importTableLocalID, "retl-source-table", table),
		Spec: map[string]any{
			"id":                importTableLocalID,
			"display_name":      importSeedTableName,
			"account_id":        accountID,
			"primary_key":       "id",
			"source_definition": "postgres",
			"schema":            "analytics",
			"table":             "users_claimed",
			"enabled":           true,
		},
	})

	wantModel := retlClient.RETLSource{
		ID:                   model.ID,
		Name:                 importSeedModelName,
		Config:               retlClient.RETLSQLModelConfig{PrimaryKey: "id", Sql: "SELECT id, email FROM users"},
		IsEnabled:            true,
		SourceType:           retlClient.ModelSourceType,
		SourceDefinitionName: "postgres",
		AccountID:            accountID,
		ExternalID:           importModelLocalID,
	}
	wantTable := retlClient.RETLSource{
		ID:                   table.ID,
		Name:                 importSeedTableName,
		Config:               retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "analytics", Table: "users_claimed"},
		IsEnabled:            true,
		SourceType:           retlClient.TableSourceType,
		SourceDefinitionName: "postgres",
		AccountID:            accountID,
		ExternalID:           importTableLocalID,
	}

	t.Run("apply claims the existing sources instead of creating new ones", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "apply", "-l", projectDir, "--confirm=false")
		require.NoError(t, err, "apply after import failed: %s", out)

		assert.Equal(t, wantModel, managedRETLSource(t, retlClient.ModelSourceType, importModelLocalID))
		assert.Equal(t, wantTable, managedRETLSource(t, retlClient.TableSourceType, importTableLocalID))

		importable := append(
			unmanagedRETLSourceIDs(t, store, retlClient.ModelSourceType),
			unmanagedRETLSourceIDs(t, store, retlClient.TableSourceType)...,
		)
		assert.NotContains(t, importable, model.ID, "a claimed model must leave the importable set")
		assert.NotContains(t, importable, table.ID, "a claimed table must leave the importable set")
	})

	// The project holds no account, so there is no always-unknown secret to
	// re-plan: a second apply must have nothing to do at all.
	t.Run("re-apply has no changes", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "apply", "-l", projectDir, "--dry-run", "--confirm=false")
		require.NoError(t, err, "dry run failed: %s", out)
		assert.Contains(t, string(out), "No changes to apply", "the claimed sources must converge")
	})
}

// seedUnmanagedPostgresAccount creates the account the seeded sources hang off,
// without an externalId, and returns its id.
func seedUnmanagedPostgresAccount(t *testing.T, apiClient *client.Client) string {
	t.Helper()

	options, err := json.Marshal(map[string]any{
		"host": "db.acme-analytics.internal", "dbname": "analytics", "user": "rudder", "port": 5432, "sslMode": "disable",
	})
	require.NoError(t, err)
	secret, err := json.Marshal(map[string]any{"password": "dummy-retl-import-pg-password-12345"})
	require.NoError(t, err)

	account, err := apiClient.Accounts.Create(context.Background(), &client.CreateAccountRequest{
		AccountDefinitionName: "SOURCE_POSTGRES",
		Name:                  importSeedAccountName,
		Options:               options,
		Secret:                secret,
	})
	require.NoError(t, err, "seeding an unmanaged account")
	return account.ID
}

// seedUnmanagedRETLSource posts the body as is rather than through
// CreateRetlSource, whose request always carries an externalId: the point of a
// seed is to look like a source the webapp made, which has none.
func seedUnmanagedRETLSource(t *testing.T, apiClient *client.Client, body map[string]any) *retlClient.RETLSource {
	t.Helper()

	payload, err := json.Marshal(body)
	require.NoError(t, err)
	resp, err := apiClient.Do(context.Background(), "POST", "/v2/retl-sources", bytes.NewReader(payload))
	require.NoError(t, err, "seeding RETL source %q", body["name"])

	var source retlClient.RETLSource
	require.NoError(t, json.Unmarshal(resp, &source))
	require.NotEmpty(t, source.ID, "seeded RETL source %q has no id", body["name"])
	require.Empty(t, source.ExternalID, "a seed must be unmanaged")
	// Import metadata only applies within its own workspace, so an empty id
	// would turn the claim into a silent create.
	require.NotEmpty(t, source.WorkspaceID, "seeded RETL source %q has no workspace id", body["name"])
	return &source
}

// removeImportSeeds deletes whatever a previous or the current run seeded:
// unmanaged sources and accounts matched by name, sources first because an
// account in use cannot be deleted. A claimed source is managed and left to
// destroy. Deletion errors are logged rather than failed, so cleanup always
// gets as far as it can.
func removeImportSeeds(t *testing.T, apiClient *client.Client, store retlClient.RETLStore) {
	t.Helper()

	ctx := context.Background()
	sources, err := store.ListRetlSources(ctx, retlClient.WithHasExternalId(lo.ToPtr(false)))
	require.NoError(t, err, "listing unmanaged RETL sources")
	for _, source := range sources.Data {
		if source.Name != importSeedModelName && source.Name != importSeedTableName {
			continue
		}
		if err := store.DeleteRetlSource(ctx, source.ID); err != nil {
			t.Logf("deleting seeded RETL source %s: %v", source.ID, err)
		}
	}

	accounts, err := apiClient.Accounts.ListAll(ctx, client.WithHasExternalID(false))
	require.NoError(t, err, "listing unmanaged accounts")
	for _, account := range accounts {
		if account.Name != importSeedAccountName {
			continue
		}
		if err := apiClient.Accounts.Delete(ctx, account.ID); err != nil {
			t.Logf("deleting seeded account %s: %v", account.ID, err)
		}
	}
}

func unmanagedRETLSourceIDs(t *testing.T, store retlClient.RETLStore, sourceType retlClient.SourceType) []string {
	t.Helper()

	sources, err := store.ListRetlSources(context.Background(),
		retlClient.WithSourceType(string(sourceType)), retlClient.WithHasExternalId(lo.ToPtr(false)))
	require.NoError(t, err, "listing unmanaged RETL sources")
	return lo.Map(sources.Data, func(s retlClient.RETLSource, _ int) string { return s.ID })
}

// importMetadata is the metadata block that ties a local spec to the remote
// source, in the map shape a spec file decodes to.
func importMetadata(localID, resourceType string, remote *retlClient.RETLSource) map[string]any {
	return map[string]any{
		"name": localID,
		"import": map[string]any{
			"workspaces": []any{map[string]any{
				"workspace_id": remote.WorkspaceID,
				"resources": []any{map[string]any{
					"urn":       fmt.Sprintf("%s:%s", resourceType, localID),
					"remote_id": remote.ID,
				}},
			}},
		},
	}
}

func readSpec(t *testing.T, path string) *specs.Spec {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err, "reading %s", path)
	var spec specs.Spec
	require.NoError(t, yaml.Unmarshal(raw, &spec), "decoding %s", path)
	return &spec
}

func writeSpec(t *testing.T, path string, spec *specs.Spec) {
	t.Helper()

	raw, err := yaml.Marshal(spec)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o644))
}
