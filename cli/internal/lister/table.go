package lister

import (
	"fmt"

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

	columns, rows := tableData(rs, columnWidths)

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

	// The interactive table opens /dev/tty, which a pipe, a CI job or any
	// other non-interactive shell does not have. Render the rows once and
	// return instead of failing with "could not open a new TTY".
	if !ui.IsTerminal() {
		t.Blur()
		ui.Println(t.View())
		return nil
	}

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

// hasManaged reports whether this resource type carries adoption state. Keyed
// on the data rather than the resource type, so no lister has to opt in.
func hasManaged(rs []resources.ResourceData) bool {
	for _, r := range rs {
		if _, ok := r["managed"]; ok {
			return true
		}
	}
	return false
}

// tableData builds the table's columns and rows. Split out of
// printTableWithDetails so the column and cell decisions can be tested without
// a terminal to render into.
func tableData(rs []resources.ResourceData, columnWidths map[string]int) ([]table.Column, []table.Row) {
	width := func(key string, fallback int) int {
		if w, ok := columnWidths[key]; ok {
			return w
		}
		return fallback
	}

	columns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "ID", Width: width("id", 27)},
		{Title: "Name", Width: width("name", 30)},
	}
	// A resource type that carries "managed" gets the extra column; the others
	// are untouched, so this stays one table printer for every lister.
	managed := hasManaged(rs)
	if managed {
		columns = append(columns, table.Column{Title: "Managed as", Width: width("managed", 27)})
	}

	rows := make([]table.Row, len(rs))
	for i, resource := range rs {
		name := resource["name"]
		nameStr := "- not set -"
		if name != nil && name != "" {
			nameStr = name.(string)
		}

		rows[i] = table.Row{
			fmt.Sprintf("%d", i+1),
			resource["id"].(string),
			nameStr,
		}
		if managed {
			cell, _ := resource["managed"].(string)
			if cell == "" {
				cell = "-"
			}
			rows[i] = append(rows[i], cell)
		}
	}
	return columns, rows
}
