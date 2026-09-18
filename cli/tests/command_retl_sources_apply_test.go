package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// retlAccountExternalID and retlModelExternalID are the external ids the
// fixtures claim. A RETL source's externalId is its spec `id` (handler.Create
// passes the resource ID straight through), so these double as the lookup keys.
const (
	retlAccountExternalID = "retl-pg"
	retlModelExternalID   = "orders-model"
)

// retlAccountIDVar is the variable the SQL model fixtures resolve their
// account_id from. It cannot be a committed value: the id is assigned by the
// server when the account is applied, so the test writes this var file itself.
const retlAccountIDVar = "RETL_ACCOUNT_ID"

// TestRETLSourcesApply drives retl-source-sql-model end to end against a live
// stack: apply create → apply update → re-apply is a no-op.
//
// Deliberately ungated. DEX-901 found that RUN_CONNECTION_E2E was added to the
// e2e workflow but never defined as a repository variable, so that suite has
// never executed — a gate is only coverage if someone remembers to turn it on.
// This suite needs nothing TestAccountsApply does not already need (a live
// stack carrying SOURCE_POSTGRES), so it runs in the same lane instead.
func TestRETLSourcesApply(t *testing.T) {
	allowUnverifiedDestinationResidue(t)
	// Accounts are behind the experimental umbrella; the SQL model kind is not,
	// but the account this source hangs off is.
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")

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

	// The model's account_id must be the account's server-assigned id, which does
	// not exist until the account is applied. Apply the account alone, read the id
	// back, and feed it to the later applies as a generated var file. The
	// reference form (account: "#account:<name>") that would remove this step
	// lands with DEX-860; until then the literal id is the only shape available.
	applyRETLProject(t, executor, filepath.Join(projectDir, "bootstrap"), credentials)
	accountID := managedAccountID(t, retlAccountExternalID)
	accountVars := writeRETLAccountVarFile(t, accountID)

	t.Run("apply create", func(t *testing.T) {
		applyRETLProject(t, executor, filepath.Join(projectDir, "create"), credentials, accountVars)
		assertRETLModel(t, accountID, "Orders Model", "SELECT id, email FROM orders")
	})

	t.Run("apply update", func(t *testing.T) {
		applyRETLProject(t, executor, filepath.Join(projectDir, "update"), credentials, accountVars)
		assertRETLModel(t, accountID, "Orders Model - revised", "SELECT id, email, created_at FROM orders")
	})

	// Not assertable as "No changes to apply": the project carries the account
	// the model hangs off, and its password is write-only, so it maps to an
	// always-unknown secret that re-plans every run (see secret.String.Diff).
	// TestAccountsApply and TestConnectionsApply hit the same wall and settle for
	// the same thing — prove nothing else churned by reading the state back.
	t.Run("re-apply leaves the model unchanged", func(t *testing.T) {
		applyRETLProject(t, executor, filepath.Join(projectDir, "update"), credentials, accountVars)
		assertRETLModel(t, accountID, "Orders Model - revised", "SELECT id, email, created_at FROM orders")
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

// writeRETLAccountVarFile writes the server-assigned account id to a var file in
// the test's temp dir. The name must end in .vars.yaml for the CLI to accept it.
func writeRETLAccountVarFile(t *testing.T, accountID string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "account.vars.yaml")
	body := fmt.Sprintf("%s: %s\n", retlAccountIDVar, accountID)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
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

// assertRETLModel reads the managed SQL model back through the API and compares
// the whole source, with the server-assigned and volatile fields normalised out.
// Comparing the struct rather than a handful of fields is what catches a field
// the assertions never thought to name — an `enabled` that silently flipped, or
// a config shape decoded as the wrong union member.
func assertRETLModel(t *testing.T, accountID, displayName, sql string) {
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
		retlClient.WithSourceType(string(retlClient.ModelSourceType)),
		retlClient.WithHasExternalId(&hasExternalID),
	)
	require.NoError(t, err, "listing managed RETL sources")

	var actual *retlClient.RETLSource
	for i, source := range sources.Data {
		if source.ExternalID == retlModelExternalID {
			actual = &sources.Data[i]
			break
		}
	}
	require.NotNil(t, actual, "managed RETL source %q missing upstream", retlModelExternalID)

	// Server-assigned and time-varying fields carry no expectation; everything
	// else must match exactly.
	normalised := *actual
	normalised.ID = ""
	normalised.WorkspaceID = ""
	normalised.CreatedAt = nil
	normalised.UpdatedAt = nil

	assert.Equal(t, retlClient.RETLSource{
		Name:                 displayName,
		Config:               retlClient.RETLSQLModelConfig{PrimaryKey: "id", Sql: sql},
		IsEnabled:            true,
		SourceType:           retlClient.ModelSourceType,
		SourceDefinitionName: "postgres",
		AccountID:            accountID,
		ExternalID:           retlModelExternalID,
	}, normalised)
}
