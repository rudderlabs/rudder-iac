package lister

import (
	"os"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableListWithNonTerminalInput(t *testing.T) {
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	oldStdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = reader.Close()
	})

	err = printTableWithDetails([]resources.ResourceData{{"id": "src-1", "name": "Example"}}, nil)
	assert.NoError(t, err, "workspace list should work when stdin is redirected")
}
