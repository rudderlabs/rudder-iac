package tests

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The external ids the fixtures claim. A RETL source's externalId is its spec
// `id` (handler.Create passes the resource ID straight through), so these
// double as the lookup keys.
const (
	retlAccountExternalID = "retl-pg"
	retlModelExternalID   = "orders-model"
	retlTableExternalID   = "customers-table"
)

// TestRETLSourcesApply drives both RETL source kinds end to end against a live
// stack: apply create → apply update → re-apply leaves them unchanged.
//
// Both kinds share one project because they share one account, and that is the
// point of the fixture: the account is named by reference
// (account: "#account:retl-pg") rather than by a server-assigned id, so the
// graph has to order the account ahead of both sources and substitute the id it
// gets back. Before #851 that reference did not exist and this suite had to
// apply the account alone, read its id back and feed it to the sources through
// a generated var file.
//
// Deliberately ungated. DEX-901 found that RUN_CONNECTION_E2E was added to the
// e2e workflow but never defined as a repository variable, so that suite has
// never executed — a gate is only coverage if someone remembers to turn it on.
// This suite needs nothing TestAccountsApply does not already need (a live
// stack carrying SOURCE_POSTGRES), so it runs in the same lane instead.
func TestRETLSourcesApply(t *testing.T) {
	allowUnverifiedDestinationResidue(t)
	// Accounts are behind the experimental umbrella and the table kind behind
	// its own flag; the SQL model kind is behind neither.
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_RETL_TABLE_SUPPORT", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	projectDir := filepath.Join("testdata", "retl_sources")
	credentials := filepath.Join(projectDir, "credentials.vars.yaml")

	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		assert.NoError(t, err, "cleanup destroy failed: %s", out)
	})

	t.Run("apply create", func(t *testing.T) {
		applyRETLProject(t, executor, filepath.Join(projectDir, "create"), credentials)

		accountID := managedAccountID(t, retlAccountExternalID)
		assertRETLModel(t, accountID, "Orders Model", "SELECT id, email FROM orders")
		assertRETLTable(t, accountID, "Customers Table", "customers")
	})

	t.Run("apply update", func(t *testing.T) {
		applyRETLProject(t, executor, filepath.Join(projectDir, "update"), credentials)

		accountID := managedAccountID(t, retlAccountExternalID)
		assertRETLModel(t, accountID, "Orders Model - revised", "SELECT id, email, created_at FROM orders")
		assertRETLTable(t, accountID, "Customers Table", "customers_v2")
	})

	// Not assertable as "No changes to apply": the project carries the account
	// both sources hang off, and its password is write-only, so it maps to an
	// always-unknown secret that re-plans every run (see secret.String.Diff).
	// TestAccountsApply and TestConnectionsApply hit the same wall and settle for
	// the same thing — prove nothing else churned by reading the state back.
	t.Run("re-apply leaves both sources unchanged", func(t *testing.T) {
		applyRETLProject(t, executor, filepath.Join(projectDir, "update"), credentials)

		accountID := managedAccountID(t, retlAccountExternalID)
		assertRETLModel(t, accountID, "Orders Model - revised", "SELECT id, email, created_at FROM orders")
		assertRETLTable(t, accountID, "Customers Table", "customers_v2")
	})
}

// applyRETLProject applies one fixture directory with the given var files.
func applyRETLProject(t *testing.T, executor *CmdExecutor, dir, credentials string, extraVarFiles ...string) {
	t.Helper()

	args := []string{"apply", "-l", dir, "--var-file", credentials}
	for _, varFile := range extraVarFiles {
		args = append(args, "--var-file", varFile)
	}
	args = append(args, "--confirm=false")

	out, err := executor.Execute(cliBinPath, args...)
	require.NoError(t, err, "apply %s failed: %s", dir, out)
}

// managedAccountID returns the server-assigned id of the managed account
// claiming the given externalId.
func managedAccountID(t *testing.T, externalID string) string {
	t.Helper()

	apiClient := newAccountsAPIClient(t)
	accounts, err := apiClient.Accounts.ListAll(context.Background(), client.WithHasExternalID(true))
	require.NoError(t, err, "listing managed accounts")

	for _, account := range accounts {
		if account.ExternalID == externalID {
			require.NotEmpty(t, account.ID, "managed account %q has no id", externalID)
			return account.ID
		}
	}

	t.Fatalf("managed account %q not found upstream", externalID)
	return ""
}

// managedRETLSource reads back the managed source of the given type claiming the
// given externalId, with the server-assigned and time-varying fields cleared so
// the caller can compare the whole struct. Comparing the struct rather than a
// handful of fields is what catches a field the assertions never thought to
// name — an `enabled` that silently flipped, or a config shape decoded as the
// wrong union member.
func managedRETLSource(t *testing.T, sourceType retlClient.SourceType, externalID string) retlClient.RETLSource {
	t.Helper()

	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)
	require.NoError(t, err)

	hasExternalID := true
	sources, err := retlClient.NewRudderRETLStore(apiClient).ListRetlSources(
		context.Background(),
		retlClient.WithSourceType(string(sourceType)),
		retlClient.WithHasExternalId(&hasExternalID),
	)
	require.NoError(t, err, "listing managed RETL sources")

	for _, source := range sources.Data {
		if source.ExternalID != externalID {
			continue
		}
		source.ID = ""
		source.WorkspaceID = ""
		source.CreatedAt = nil
		source.UpdatedAt = nil
		return source
	}

	t.Fatalf("managed RETL source %q missing upstream", externalID)
	return retlClient.RETLSource{}
}

func assertRETLModel(t *testing.T, accountID, displayName, sql string) {
	t.Helper()

	assert.Equal(t, retlClient.RETLSource{
		Name:                 displayName,
		Config:               retlClient.RETLSQLModelConfig{PrimaryKey: "id", Sql: sql},
		IsEnabled:            true,
		SourceType:           retlClient.ModelSourceType,
		SourceDefinitionName: "postgres",
		AccountID:            accountID,
		ExternalID:           retlModelExternalID,
	}, managedRETLSource(t, retlClient.ModelSourceType, retlModelExternalID))
}

func assertRETLTable(t *testing.T, accountID, displayName, table string) {
	t.Helper()

	assert.Equal(t, retlClient.RETLSource{
		Name:                 displayName,
		Config:               retlClient.RETLTableConfig{PrimaryKey: "id", Schema: "analytics", Table: table},
		IsEnabled:            true,
		SourceType:           retlClient.TableSourceType,
		SourceDefinitionName: "postgres",
		AccountID:            accountID,
		ExternalID:           retlTableExternalID,
	}, managedRETLSource(t, retlClient.TableSourceType, retlTableExternalID))
}
