package tests

import (
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
func TestRETLConnectionRefusedAtCreate(t *testing.T) {
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

	// The project is otherwise valid — account, source and destination all apply.
	// Only the connection is refused, and the reason has to name why.
	out, err = executor.Execute(cliBinPath,
		"apply", "-l", projectDir, "--var-file", credentials, "--confirm=false")
	require.Error(t, err, "apply should have refused the connection, got: %s", out)
	assert.Contains(t, string(out), `destination type "WEBHOOK" does not accept warehouse sources`,
		"the refusal must name the destination type, not just fail")

	// Nothing was written: the point of refusing at create is that no
	// unreadable row is left behind for the next apply to trip over. Targeted
	// at this connection, across every page, so an unrelated managed connection
	// in the shared workspace cannot fail it and a leaked row cannot hide.
	assert.NotContains(t, lo.Map(managedRETLConnections(t), func(c retlClient.RETLConnection, _ int) string { return c.ExternalID }),
		"orders-to-archive", "a refused connection must leave nothing upstream")
}
