package lister

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

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
	if m.width == 0 {
		return "Initializing..."
	}

	// Details View
	var detailsView string
	if len(m.resources) > 0 {
		selected := m.resources[m.table.Cursor()]
		detailsView = ui.RenderDetails(selected)
	} else {
		detailsView = "No resources found."
	}

	// Details View with Header
	detailsHeader := ui.Bold("Details")
	ruler := ui.RulerWithWidth(m.width - (4 + 27 + 30 + 6) - 4)
	detailsContent := lipgloss.NewStyle().Padding(0, 2).Render(detailsView)
	fullDetailsView := lipgloss.JoinVertical(lipgloss.Left, detailsHeader, ruler, detailsContent)

	// Main Layout
	detailsStyle := lipgloss.NewStyle().
		Padding(0, 2).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderForeground(lipgloss.Color("240"))

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

func printTableWithDetails(rs []resources.ResourceData) error {
	columns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "ID", Width: 27}, // KSUID length
		{Title: "Name", Width: 30},
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

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rows)),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
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
