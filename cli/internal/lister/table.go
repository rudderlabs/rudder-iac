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

const (
	noResourcesFoundMsg = "No resources found"

	// Header row plus the border rendered underneath it.
	tableHeaderHeight = 2
	// Help footer, plus one line so the first row is not scrolled out of view.
	reservedHeight = 2
)

type model struct {
	table     table.Model
	help      help.Model
	keys      keyMap
	resources []resources.ResourceData
	width     int
	// Rows the table and the details pane may each occupy. Zero means the
	// terminal height is unknown and neither pane is constrained.
	maxHeight int
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
		m.help.Width = msg.Width
		m.maxHeight = availableHeight(msg.Height)
		m.table.SetHeight(tableHeight(len(m.resources), m.maxHeight))
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

	// Main Layout. The details pane is held to the same budget as the table, so
	// that neither side of the join can outgrow the terminal.
	detailsStyle := lipgloss.NewStyle().Padding(0, 2)
	if m.maxHeight > 0 {
		detailsStyle = detailsStyle.MaxHeight(m.maxHeight)
	}

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

// availableHeight is the room the panes have on a terminal of the given height,
// once the help footer and one spare line are reserved.
func availableHeight(terminalHeight int) int {
	return max(terminalHeight-reservedHeight, tableHeaderHeight+1)
}

// tableHeight sizes the table to its content without letting it outgrow the
// rows available to it, so that large result sets scroll inside the table
// viewport instead of pushing the details pane and help footer off screen. A
// budget of zero leaves every row in place, for output that is not going to a
// terminal and will never receive a WindowSizeMsg.
func tableHeight(rowCount, available int) int {
	if available <= 0 {
		return rowCount + tableHeaderHeight
	}
	return min(rowCount+tableHeaderHeight, available)
}

func newModel(rs []resources.ResourceData, columnWidths map[string]int) model {
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

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
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

	// Piped output never receives a WindowSizeMsg to correct an initial guess,
	// so it keeps every row rather than silently losing some of them.
	maxHeight := 0
	if h := ui.GetTerminalHeight(); h > 0 {
		maxHeight = availableHeight(h)
	}

	// Set after the styles, because the header border changes how a height maps
	// onto rendered lines.
	t.SetHeight(tableHeight(len(rows), maxHeight))

	return model{
		table:     t,
		help:      help.New(),
		keys:      keys,
		resources: rs,
		maxHeight: maxHeight,
	}
}

func printTableWithDetails(rs []resources.ResourceData, columnWidths map[string]int) error {
	if len(rs) == 0 {
		ui.Println(noResourcesFoundMsg)
		return nil
	}

	p := tea.NewProgram(newModel(rs, columnWidths))
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
