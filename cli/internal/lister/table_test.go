package lister

import (
	"bytes"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
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

// Without a terminal the interactive table cannot open /dev/tty, so the rows
// are rendered once and printed. Before this fell back, any pipe, CI job or
// script listing a non-empty resource type failed with "could not open a new
// TTY" as soon as there was a row to draw.
func TestPrintTableWithDetails_WhenNotATerminal_PrintsRowsInsteadOfFailing(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	t.Cleanup(ui.RestoreWriter)

	rs := []resources.ResourceData{
		{"id": "3MYZ9Yl1gHm2sUfbPH8KtIesEEX", "name": "google ads prod"},
	}

	require.NoError(t, printTableWithDetails(rs, nil))

	out := buf.String()
	require.Contains(t, out, "3MYZ9Yl1gHm2sUfbPH8KtIesEEX")
	require.Contains(t, out, "google ads prod")
	require.Contains(t, out, "ID")
}
