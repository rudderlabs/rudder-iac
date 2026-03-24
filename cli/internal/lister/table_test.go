package lister

import (
	"bytes"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/stretchr/testify/require"
)

func TestPrintTableWithDetails_WhenNoResources_PrintsPlainMessage(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	t.Cleanup(ui.RestoreWriter)

	printErr := printTableWithDetails(nil, nil)
	require.NoError(t, printErr)

	require.Equal(t, "No resources found\n", buf.String())
}

