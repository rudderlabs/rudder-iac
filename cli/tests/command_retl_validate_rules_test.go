package tests

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRETLValidationRules checks that a project the control plane would refuse
// partway through an apply is refused up front instead, and that nothing
// reaches the workspace when it is.
//
// The refusal messages themselves are pinned in the rule modules and in the
// runnable docs fragments, so they are not what this suite is for. What only an
// apply can show is the consequence: the rules run before the syncer writes
// anything, so a project carrying one bad source does not leave the account and
// the good source behind it. Every case is therefore driven through `apply`
// only — a `validate` invocation here would re-assert the module's own strings
// against a live lane and prove nothing further.
func TestRETLValidationRules(t *testing.T) {
	allowManagedResidue(t)

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	var (
		projectDir  = filepath.Join("testdata", "retl_validate_rules")
		credentials = filepath.Join(projectDir, "credentials.vars.yaml")
	)

	cases := []struct {
		name     string
		fixture  string
		messages []string
	}{
		{
			// The control plane compares RETL source names case-insensitively
			// across both kinds.
			name:    "display_name clash across source kinds",
			fixture: "display_name_clash",
			messages: []string{
				"error[retl/table/semantic-valid]: duplicate display_name 'e2e validate users' (case-insensitive match with 'E2E Validate Users') across RETL sources",
			},
		},
		{
			name:    "account reference to a missing or mismatched account",
			fixture: "account_reference",
			messages: []string{
				"error[retl/sqlmodel/semantic-valid]: account 'e2e-validate-missing' not found in the project; reference an account spec in the project, or set account_id instead",
				"error[retl/table/semantic-valid]: account 'e2e-validate-pg' is a 'postgres' account (SOURCE_POSTGRES) and cannot back source_definition 'snowflake'",
			},
		},
		{
			// The connection rules were the one family with no apply-cycle
			// coverage: a topology the backend would reject, and a sync
			// behaviour the JSON mapper flow does not offer.
			name:    "connection topology the destination cannot serve",
			fixture: "connection_topology",
			messages: []string{
				"error[retl/connection/semantic-valid]: destination 'e2e-validate-http' config has no 'connection_mode' entry for source type 'warehouse'",
				"error[retl/connection/semantic-valid]: 'sync_behaviour' must be one of [upsert full] for source definition 'postgres' and destination 'e2e-validate-http' on the json_mapper flow",
			},
		},
	}

	// apply is workspace-wide, so the refusal is checked from an empty
	// workspace: had the rules let the project through, it would also have
	// deleted every other managed resource.
	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := executor.Execute(cliBinPath, "apply",
				"-l", filepath.Join(projectDir, tc.fixture), "--var-file", credentials, "--confirm=false")
			require.Error(t, err, "apply should refuse the project, got: %s", out)
			for _, message := range tc.messages {
				assert.Contains(t, string(out), message)
			}

			// The account is the load-bearing one: it has no dependency of its
			// own, so it is what a syncer that started before validating would
			// have created first.
			assert.NotContains(t, managedAccountExternalIDs(t), "e2e-validate-pg")
			sources := managedRETLSourceExternalIDs(t)
			assert.NotContains(t, sources, "e2e-validate-model")
			assert.NotContains(t, sources, "e2e-validate-table")
			assert.NotContains(t, managedDestinationExternalIDs(t), "e2e-validate-http")
			assert.NotContains(t, managedRETLConnectionExternalIDs(t), "e2e-validate-connection")
		})
	}
}
