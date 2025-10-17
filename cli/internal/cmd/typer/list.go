package typer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

func newCmdList() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all tracking plans",
		Long:  "List all tracking plans available in the workspace",
		Args:  cobra.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !config.GetConfig().ExperimentalFlags.RudderTyper {
				return fmt.Errorf("typer commands are disabled")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() {
				telemetry.TrackCommand("typer list", nil, []telemetry.KV{
					{K: "json", V: jsonOutput},
				}...)
			}()

			deps, err := app.NewDeps()
			if err != nil {
				return fmt.Errorf("failed to initialize dependencies: %w", err)
			}

			client := deps.Client()
			dataCatalogClient := catalog.NewRudderDataCatalog(client)

			ctx := context.Background()

			var trackingPlans []*catalog.TrackingPlanWithIdentifiers
			if !jsonOutput {
				spinner := ui.NewSpinner("Fetching tracking plans...")
				spinner.Start()
				trackingPlans, err = dataCatalogClient.GetTrackingPlans(ctx)
				spinner.Stop()
			} else {
				trackingPlans, err = dataCatalogClient.GetTrackingPlans(ctx)
			}

			if err != nil {
				return fmt.Errorf("failed to fetch tracking plans: %w", err)
			}

			if len(trackingPlans) == 0 {
				fmt.Println("No tracking plans found")
				return nil
			}

			if jsonOutput {
				return printTrackingPlansAsJSON(trackingPlans)
			}

			return printTrackingPlansTable(trackingPlans)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func printTrackingPlansAsJSON(trackingPlans []*catalog.TrackingPlanWithIdentifiers) error {
	for _, tp := range trackingPlans {
		output := map[string]any{
			"id":          tp.ID,
			"name":        tp.Name,
			"version":     tp.Version,
			"description": tp.Description,
			"workspaceId": tp.WorkspaceID,
			"eventCount":  len(tp.Events),
			"createdAt":   tp.CreatedAt,
			"updatedAt":   tp.UpdatedAt,
		}
		b, err := json.Marshal(output)
		if err != nil {
			return fmt.Errorf("failed to marshal tracking plan: %w", err)
		}
		fmt.Println(string(b))
	}
	return nil
}

type trackingPlanModel struct {
	table         table.Model
	help          help.Model
	keys          keyMap
	trackingPlans []*catalog.TrackingPlanWithIdentifiers
	width         int
	height        int
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

var trackingPlanKeys = keyMap{
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

func (m trackingPlanModel) Init() tea.Cmd { return nil }

func (m trackingPlanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m trackingPlanModel) View() string {
	var detailsView string
	if len(m.trackingPlans) > 0 {
		selected := m.trackingPlans[m.table.Cursor()]
		details := map[string]any{
			"id":          selected.ID,
			"name":        selected.Name,
			"version":     selected.Version,
			"description": formatDescription(selected.Description),
			"workspaceId": selected.WorkspaceID,
			"eventCount":  len(selected.Events),
			"createdAt":   selected.CreatedAt.Format("2006-01-02 15:04:05"),
			"updatedAt":   selected.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		detailsView = ui.FormattedMap(details)
	} else {
		detailsView = "No tracking plans found."
	}

	detailsHeader := ui.Bold("Details")
	ruler := ui.RulerWithWidth(m.width - 115)
	detailsContent := lipgloss.NewStyle().Padding(0, 2).Render(detailsView)
	fullDetailsView := lipgloss.JoinVertical(lipgloss.Top, detailsHeader, ruler, detailsContent)

	detailsStyle := lipgloss.NewStyle().Padding(0, 2)

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

func formatDescription(desc *string) string {
	if desc == nil || *desc == "" {
		return "- not set -"
	}
	return *desc
}

func printTrackingPlansTable(trackingPlans []*catalog.TrackingPlanWithIdentifiers) error {
	columns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "ID", Width: 27},
		{Title: "Name", Width: 30},
		{Title: "Version", Width: 8},
		{Title: "Description", Width: 40},
	}

	rows := make([]table.Row, len(trackingPlans))
	for i, tp := range trackingPlans {
		name := tp.Name
		if name == "" {
			name = "- not set -"
		}

		description := formatDescription(tp.Description)
		if len(description) > 37 {
			description = description[:37] + "..."
		}

		rows[i] = table.Row{
			fmt.Sprintf("%d", i+1),
			tp.ID,
			name,
			fmt.Sprintf("%d", tp.Version),
			description,
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rows)+1),
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

	m := trackingPlanModel{
		table:         t,
		help:          help.New(),
		keys:          trackingPlanKeys,
		trackingPlans: trackingPlans,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run table UI: %w", err)
	}

	return nil
}
