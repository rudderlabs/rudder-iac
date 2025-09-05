package previewer

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

type PreviewerOpts func(*Previewer)

func WithLimit(limit int) PreviewerOpts {
	return func(p *Previewer) {
		p.Limit = limit
	}
}

func WithJson(json bool) PreviewerOpts {
	return func(p *Previewer) {
		p.Json = json
	}
}

func WithInteractive(interactive bool) PreviewerOpts {
	return func(p *Previewer) {
		p.Interactive = interactive
	}
}

type Previewer struct {
	Provider    PreviewProvider
	Limit       int
	Json        bool
	Interactive bool
}

type PreviewProvider interface {
	Preview(ctx context.Context, ID string, resourceType string, data resources.ResourceData, limit int) ([]map[string]any, error)
}

func New(provider PreviewProvider, opts ...PreviewerOpts) *Previewer {
	p := &Previewer{Provider: provider}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

type previewModel struct {
	table table.Model
	help  help.Model
	keys  keyMap
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

func (m previewModel) Init() tea.Cmd { return nil }

func (m previewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m previewModel) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.table.View(),
		m.help.View(m.keys),
	)
}

func (p *Previewer) Preview(ctx context.Context, ID string, resourceType string, data resources.ResourceData) error {
	spinner := ui.NewSpinner(fmt.Sprintf("Previewing %s...", ID))
	spinner.Start()
	rowsData, err := p.Provider.Preview(ctx, ID, resourceType, data, p.Limit)
	spinner.Stop()

	if err != nil {
		return err
	}

	// Handle empty results
	if len(rowsData) == 0 {
		fmt.Println("ℹ️  No preview data available. The query executed successfully but returned no rows.")
		return nil
	}

	if p.Json {
		return p.previewJson(rowsData)
	}
	return p.previewTable(rowsData)
}

func (p *Previewer) previewJson(rowsData []map[string]any) error {
	b, err := json.MarshalIndent(rowsData, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func (p *Previewer) previewTable(rowsData []map[string]any) error {
	// Calculate column widths based on content
	columns := maps.Keys(rowsData[0])
	tableColumns := make([]table.Column, 0)
	for col := range columns {
		width := len(col) // Start with column name length

		// Check all rows to find the maximum width for this column
		for _, row := range rowsData {
			if value, exists := row[col]; exists {
				valueStr := fmt.Sprintf("%v", value)
				if len(valueStr) > width {
					width = len(valueStr)
				}
			}
		}

		// Add some padding and ensure minimum/maximum width
		width += 2
		if width < 10 {
			width = 10
		}
		if width > 50 {
			width = 50
		}

		tableColumns = append(tableColumns, table.Column{Title: col, Width: width})
	}

	// Convert rows to table rows
	tableRows := make([]table.Row, len(rowsData))
	for i, row := range rowsData {
		tableRow := make(table.Row, len(row))
		for j, col := range tableColumns {
			tableRow[j] = ""
			if value, exists := row[col.Title]; exists {
				tableRow[j] = fmt.Sprintf("%v", value)
			}
		}
		tableRows[i] = tableRow
	}

	// Create table
	t := table.New(
		table.WithColumns(tableColumns),
		table.WithRows(tableRows),
		table.WithFocused(p.Interactive),
		table.WithHeight(len(tableRows)+1), // +1 for the header
	)

	// Apply styles
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

	// Create and run the model
	m := previewModel{
		table: t,
		help:  help.New(),
		keys:  keys,
	}

	if p.Interactive {
		program := tea.NewProgram(m)
		if _, err := program.Run(); err != nil {
			return err
		}
	} else {
		// Non-interactive mode: just print the table
		fmt.Println(m.View())
	}

	return nil
}
