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

// testResources carries enough fields for the details pane to be taller than a
// short table, which is what makes truncation visible.
func testResources(count int) []resources.ResourceData {
	rs := make([]resources.ResourceData, count)
	for i := range rs {
		rs[i] = resources.ResourceData{
			"id":          fmt.Sprintf("acc-%d", i),
			"name":        fmt.Sprintf("Account %d", i),
			"createdAt":   "2026-01-01T00:00:00Z",
			"updatedAt":   "2026-02-01T00:00:00Z",
			"type":        "snowflake",
			"workspaceId": "ws-42",
		}
	}
	return rs
}

// renderAt renders the list as it appears on a terminal of the given height,
// after the WindowSizeMsg that bubbletea emits on startup.
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

func TestAvailableHeight(t *testing.T) {
	tests := []struct {
		name           string
		terminalHeight int
		expected       int
	}{
		{"reserves the help footer and a spare line", 40, 38},
		{"keeps a row visible on a terminal with no room", 2, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, availableHeight(tt.terminalHeight))
		})
	}
}

func TestTableHeight(t *testing.T) {
	tests := []struct {
		name      string
		rowCount  int
		available int
		expected  int
	}{
		{"caps to the available rows when the content overflows", 100, 38, 38},
		{"shrinks to the content when it fits, rather than padding", 3, 38, 5},
		{"keeps every row when no terminal height is known", 100, 0, 102},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tableHeight(tt.rowCount, tt.available))
		})
	}
}

func TestModel_View_WhenResultSetIsLarge_FitsWithinTerminal(t *testing.T) {
	const terminalHeight = 40

	view := renderAt(t, 100, terminalHeight)

	require.LessOrEqual(t, lipgloss.Height(view), terminalHeight)
	require.Contains(t, view, "ws-42", "details pane should still be rendered")
}

// The details pane is taller than a two-row table, so holding it to the table's
// height would cut the fields off the bottom.
func TestModel_View_WhenResultSetIsSmall_KeepsDetailsIntact(t *testing.T) {
	view := renderAt(t, 1, 40)

	require.Contains(t, view, "snowflake")
	require.Contains(t, view, "ws-42")
	require.LessOrEqual(t, lipgloss.Height(view), 40)
}

func TestModel_View_WhenTerminalIsShort_CapsDetailsPane(t *testing.T) {
	const terminalHeight = 8

	view := renderAt(t, 100, terminalHeight)

	require.LessOrEqual(t, lipgloss.Height(view), terminalHeight)
	require.NotContains(t, view, "ws-42", "the last details field should be cut, not the layout")
}

// go test runs with stdout on a pipe, which is the redirected-output case.
func TestNewModel_WithoutTerminal_KeepsEveryRow(t *testing.T) {
	m := newModel(testResources(100), nil)

	require.Equal(t, 0, m.maxHeight)
	require.Equal(t, 100, m.table.Height())
}
