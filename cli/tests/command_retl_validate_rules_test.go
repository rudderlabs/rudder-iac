package tests

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
)

// TestRETLValidationRules drives the RETL semantic rules through the binary.
// Each fixture is a project the control plane would refuse partway through an
// apply, after some of it had already been written; the rules exist to refuse
// it up front, and this checks they do, from both `validate` and `apply`.
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
	}

	for _, tc := range cases {
		t.Run(tc.name+": validate", func(t *testing.T) {
			out, err := executor.Execute(cliBinPath, "validate",
				"-l", filepath.Join(projectDir, tc.fixture), "--var-file", credentials)
			require.Error(t, err, "validate should fail, got: %s", out)
			for _, message := range tc.messages {
				assert.Contains(t, string(out), message)
			}
		})
	}

	// apply is workspace-wide, so the refusal is checked from an empty
	// workspace: had the rules let the project through, it would also have
	// deleted every other managed resource.
	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	for _, tc := range cases {
		t.Run(tc.name+": apply writes nothing", func(t *testing.T) {
			out, err := executor.Execute(cliBinPath, "apply",
				"-l", filepath.Join(projectDir, tc.fixture), "--var-file", credentials, "--confirm=false")
			require.Error(t, err, "apply should refuse the project, got: %s", out)
			for _, message := range tc.messages {
				assert.Contains(t, string(out), message)
			}

			assert.NotContains(t, managedAccountExternalIDs(t), "e2e-validate-pg")
			assert.NotContains(t, managedRETLSourceExternalIDs(t, retlClient.ModelSourceType), "e2e-validate-model")
			assert.NotContains(t, managedRETLSourceExternalIDs(t, retlClient.TableSourceType), "e2e-validate-table")
		})
	}
}
