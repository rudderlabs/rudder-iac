package accounts

import (
	"encoding/json"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler/export"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// enableVarSubstitution turns on the experimental gate so WithVariableName is
// honoured and secrets export as "{{ .VAR }}" tokens rather than masked literals.
func enableVarSubstitution(t *testing.T) {
	t.Helper()
	prevExp, prevFlag := viper.Get("experimental"), viper.Get("flags.enableVarSubstitution")
	viper.Set("experimental", true)
	viper.Set("flags.enableVarSubstitution", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.enableVarSubstitution", prevFlag)
	})
}

// exportHandler builds a handler with the declared credentials secret field
// injected, mirroring how NewHandler wires the strategy in production.
func exportHandler() *HandlerImpl {
	h := &HandlerImpl{}
	h.SingleSpecExportStrategy = &export.SingleSpecExportStrategy[accountsExportSpec, RemoteAccount]{Handler: h}
	h.SetSecretFields([]handler.SecretField{{JSONName: bigQuerySecretField}})
	return h
}

func remoteAccount(id, externalID, workspaceID string, options json.RawMessage) *RemoteAccount {
	a := &client.Account{ID: id, ExternalID: externalID, WorkspaceID: workspaceID, Options: options}
	a.Definition.Name = bigQueryDefinition
	return &RemoteAccount{Account: a}
}

func TestFormatForExportTokenizesCredentialsAndNeverLeaks(t *testing.T) {
	enableVarSubstitution(t)

	// The remote deliberately carries a "credentials" key in its options with a
	// sentinel value: even if a real secret ever rode along in options, the flat
	// config's credentials must always be the token, never that literal.
	h := exportHandler()
	remotes := map[string]*RemoteAccount{
		"lovable-prod-bq": remoteAccount("remote-1", "lovable-prod-bq", "ws-1",
			json.RawMessage(`{"project":"p1","credentials":"SUPER-SECRET-SENTINEL"}`)),
	}

	entities, entries, err := h.FormatForExport(remotes, nil, nil)
	require.NoError(t, err)
	require.Len(t, entities, 1)
	assert.Equal(t, "accounts/accounts.yaml", entities[0].RelativePath)

	require.Len(t, entries, 1)
	assert.Equal(t, "ws-1", entries[0].WorkspaceID)
	assert.Equal(t, "account:lovable-prod-bq", entries[0].URN)
	assert.Equal(t, "remote-1", entries[0].RemoteID)

	out, err := yaml.Marshal(entities[0].Content)
	require.NoError(t, err)
	rendered := string(out)

	assert.Contains(t, rendered, "{{ .ACCOUNT_LOVABLE_PROD_BQ_CREDENTIALS }}",
		"credentials must export as a per-account variable token")
	assert.NotContains(t, rendered, "SUPER-SECRET-SENTINEL",
		"a secret value carried in options must never reach the exported spec")
	assert.NotContains(t, rendered, "(unknown)",
		"the secret must be tokenized, not fall back to a masked literal")
	assert.Contains(t, rendered, "p1", "non-secret options must round-trip")

	// The generic var-file scaffolder derives placeholder keys by scanning the
	// generated content for tokens; assert the credentials variable is discoverable.
	assert.Contains(t, varsubst.ExtractVariableNames(out), "ACCOUNT_LOVABLE_PROD_BQ_CREDENTIALS")
}

func TestFormatForExportGivesEachAccountADistinctVariable(t *testing.T) {
	enableVarSubstitution(t)

	h := exportHandler()
	remotes := map[string]*RemoteAccount{
		"acct-a": remoteAccount("remote-a", "acct-a", "ws-1", json.RawMessage(`{}`)),
		"acct-b": remoteAccount("remote-b", "acct-b", "ws-1", json.RawMessage(`{}`)),
	}

	entities, _, err := h.FormatForExport(remotes, nil, nil)
	require.NoError(t, err)
	require.Len(t, entities, 1)

	out, err := yaml.Marshal(entities[0].Content)
	require.NoError(t, err)

	names := varsubst.ExtractVariableNames(out)
	assert.Contains(t, names, "ACCOUNT_ACCT_A_CREDENTIALS")
	assert.Contains(t, names, "ACCOUNT_ACCT_B_CREDENTIALS")
}
