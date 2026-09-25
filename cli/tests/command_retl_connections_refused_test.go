package tests

import (
	"os"
	"path/filepath"
	"testing"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRETLConnectionRefusedAtCreate is the end-to-end half of DEX-917.
//
// A destination whose definition declares no warehouse source type cannot carry
// a rETL connection, and the read path already knew that: remoteConnection
// rejects the row and eligible() skips it. Create did not, so it would write a
// connection the handler then refused to read — leaving no state, so every later
// apply planned the same create again and the backend refused the duplicate with
// "Destination does not support multiple connections", a message that names
// neither the cause nor anything the author can act on.
//
// The unit tests pin the guard. What they cannot show is the shape the author
// actually meets: one apply, one refusal, naming the destination type, with
// nothing written upstream. That is what this asserts.
//
// Since DEX-829 the refusal arrives one step earlier: retl/connection/semantic-valid
// reads the same destination definition locally and fails the project before any
// request is sent, so the message the author meets is the rule's rather than the
// handler's. The create-time guard still stands behind it — unit tests pin that
// path — and what this test now proves is that the local rule names the
// destination and its type, and that a refused project reaches the API with
// nothing.
func TestRETLConnectionRefusedAtCreate(t *testing.T) {
	if os.Getenv("RUN_RETL_E2E") != "1" {
		t.Skip("set RUN_RETL_E2E=1; this suite applies to a live workspace")
	}
	allowManagedResidue(t)

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	projectDir := filepath.Join("testdata", "retl_connections_refused")
	credentials := filepath.Join(projectDir, "credentials.vars.yaml")

	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		assert.NoError(t, err, "cleanup destroy failed: %s", out)
	})

	// Only the connection is at fault — the account, source and destination the
	// project also carries are all valid — and the reason has to name why.
	out, err = executor.Execute(cliBinPath,
		"apply", "-l", projectDir, "--var-file", credentials, "--confirm=false")
	require.Error(t, err, "apply should have refused the connection, got: %s", out)
	assert.Contains(t, string(out), `destination 'e2e-retl-archive' (type 'googleads') does not accept rETL sources`,
		"the refusal must name the destination and its type, not just fail")

	// Nothing was written: the point of refusing at create is that no
	// unreadable row is left behind for the next apply to trip over. Targeted
	// at this connection, across every page, so an unrelated managed connection
	// in the shared workspace cannot fail it and a leaked row cannot hide.
	assert.NotContains(t, lo.Map(managedRETLConnections(t), func(c retlClient.RETLConnection, _ int) string { return c.ExternalID }),
		"orders-to-archive", "a refused connection must leave nothing upstream")
}
