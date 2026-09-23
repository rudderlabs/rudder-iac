package lister

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

const noResourcesFoundMsg = "No resources found"

type model struct {
	table     table.Model
	help      help.Model
	keys      keyMap
	resources []resources.ResourceData
	width     int
	height    int
}

type keyMap struct {
	Up   key.Binding
	Down key.Binding
	Quit key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Quit},
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "move down"),
	),
	Quit: key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc", "quit"),
	),
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	// Details View
	var detailsView string
	if len(m.resources) > 0 {
		selected := m.resources[m.table.Cursor()]
		detailsView = ui.FormattedMap(selected)
	} else {
		detailsView = noResourcesFoundMsg
	}

	// Details View with Header
	detailsHeader := ui.Bold("Details")
	ruler := ui.RulerWithWidth(m.width - 66) // 66 is the width of the table (4 (# column) + 27 (ID) + 30 (Name) + 5 (padding))
	detailsContent := lipgloss.NewStyle().Padding(0, 2).Render(detailsView)
	fullDetailsView := lipgloss.JoinVertical(lipgloss.Top, detailsHeader, ruler, detailsContent)

	// Main Layout
	detailsStyle := lipgloss.NewStyle().
		Padding(0, 2)

	mainView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.table.View(),
		detailsStyle.Render(fullDetailsView),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		mainView,
		m.help.View(m.keys),
	)
}

func printTableWithDetails(rs []resources.ResourceData, columnWidths map[string]int) error {
	if len(rs) == 0 {
		ui.Println(noResourcesFoundMsg)
		return nil
	}

	// Default column widths
	idWidth := 27
	nameWidth := 30

	// Override with custom widths if provided
	if len(columnWidths) > 0 {
		if w, ok := columnWidths["id"]; ok {
			idWidth = w
		}
		if w, ok := columnWidths["name"]; ok {
			nameWidth = w
		}
	}

	columns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "ID", Width: idWidth},
		{Title: "Name", Width: nameWidth},
	}

	rows := make([]table.Row, len(rs))
	for i, resource := range rs {
		name := resource["name"]
		nameStr := ""
		if name == nil || name == "" {
			nameStr = "- not set -"
		} else {
			nameStr = name.(string)
		}

		rows[i] = table.Row{
			fmt.Sprintf("%d", i+1),
			resource["id"].(string),
			nameStr,
		}
	}

	// The TUI is only usable by someone who can press esc to leave it. Piped,
	// redirected, in CI or inside a recorded pty there is nobody to press it:
	// the program either fails to open /dev/tty or blocks forever. Print a
	// static table there instead, which is what every other CLI does when it
	// is not talking to a terminal.
	if !interactiveStdout() {
		printStaticTable(os.Stdout, columns, rows)
		return nil
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rows)+1), // +1 for the header
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := model{
		table:     t,
		help:      help.New(),
		keys:      keys,
		resources: rs,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

// interactiveStdout reports whether stdout is a terminal a person is watching
// AND that they have not asked for plain output.
//
// The terminal check catches pipes, redirects and CI. It cannot catch every
// case: a pty is a terminal whether or not a human is at the other end, so a
// session recorder or an automation harness driving a pty passes it and the TUI
// still blocks waiting for esc. Nothing can distinguish those from a real
// terminal, so RUDDERSTACK_CLI_NONINTERACTIVE exists to say so explicitly.
//
// bubbletea also opens /dev/tty rather than stdout, so even the terminal check
// is a proxy for what it will do rather than a guarantee.
func interactiveStdout() bool {
	if os.Getenv("RUDDERSTACK_CLI_NONINTERACTIVE") == "1" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// printStaticTable writes the same columns and rows as plain aligned text.
func printStaticTable(w io.Writer, columns []table.Column, rows []table.Row) {
	widths := make([]int, len(columns))
	for i, c := range columns {
		widths[i] = len(c.Title)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	line := func(cells []string) {
		parts := make([]string, 0, len(cells))
		for i, cell := range cells {
			if i < len(widths) {
				parts = append(parts, fmt.Sprintf("%-*s", widths[i], cell))
			}
		}
		fmt.Fprintln(w, strings.TrimRight(strings.Join(parts, "  "), " "))
	}

	titles := make([]string, len(columns))
	for i, c := range columns {
		titles[i] = c.Title
	}
	line(titles)
	for _, row := range rows {
		line(row)
	}
}
