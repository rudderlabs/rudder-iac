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
	allowUnverifiedDestinationResidue(t)
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_DESTINATION_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_X_RETL_CONNECTION_SUPPORT", "true")

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
	// unreadable row is left behind for the next apply to trip over.
	assert.Empty(t, managedRETLConnectionExternalIDs(t),
		"a refused connection must leave nothing upstream")
}

// managedRETLConnectionExternalIDs lists the external ids of every managed
// connection in the workspace.
func managedRETLConnectionExternalIDs(t *testing.T) []string {
	t.Helper()

	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)
	require.NoError(t, err)

	page, err := retlClient.NewRudderRETLStore(apiClient).ListConnections(
		context.Background(), &retlClient.ListRETLConnectionsRequest{Page: 1})
	require.NoError(t, err, "listing RETL connections")

	ids := make([]string, 0, len(page.Data))
	for _, conn := range page.Data {
		if conn.ExternalID != "" {
			ids = append(ids, conn.ExternalID)
		}
	}
	return ids
}
