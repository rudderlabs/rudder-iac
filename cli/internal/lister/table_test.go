package lister

import (
	"bytes"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/stretchr/testify/require"
)

func testResources(count int) []resources.ResourceData {
	rs := make([]resources.ResourceData, count)
	for i := range rs {
		rs[i] = resources.ResourceData{
			"id":   fmt.Sprintf("acc-%d", i),
			"name": fmt.Sprintf("Account %d", i),
		}
	}
	return rs
}

// renderAt renders the list as it appears on a terminal of the given height,
// after the WindowSizeMsg bubbletea emits on startup.
func renderAt(t *testing.T, rowCount, terminalHeight int) string {
	t.Helper()

	updated, _ := newModel(testResources(rowCount), nil).
		Update(tea.WindowSizeMsg{Width: 200, Height: terminalHeight})

	return updated.(model).View()
}

func TestPrintTableWithDetails_WhenNoResources_PrintsPlainMessage(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	t.Cleanup(ui.RestoreWriter)

	printErr := printTableWithDetails(nil, nil)
	require.NoError(t, printErr)

	require.Equal(t, "No resources found\n", buf.String())
}

func TestFitTableHeight(t *testing.T) {
	tests := []struct {
		name           string
		rowCount       int
		terminalHeight int
		expected       int
	}{
		{"caps to the terminal when the rows overflow it", 100, 40, 38},
		{"shrinks to the content when the rows fit", 3, 40, 5},
		{"keeps a row visible on a terminal with no room", 100, 2, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, fitTableHeight(tt.rowCount, tt.terminalHeight))
		})
	}
}

func TestModel_View_WhenResultSetIsLarge_FitsWithinTerminal(t *testing.T) {
	const terminalHeight = 40

	require.LessOrEqual(t, lipgloss.Height(renderAt(t, 100, terminalHeight)), terminalHeight)
}

func TestModel_View_WhenResultSetIsSmall_StaysCompact(t *testing.T) {
	// Three rows, the bordered header and the help footer: the table is not
	// padded out to the full terminal height.
	require.Equal(t, 6, lipgloss.Height(renderAt(t, 3, 40)))
}
