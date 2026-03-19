package workspace

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show information about the authenticated workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("workspace info", err)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			workspace, err := d.Client().Workspaces.GetByAuthToken(cmd.Context())
			if err != nil {
				return fmt.Errorf("fetching workspace information: %w", err)
			}

			ui.PrintTable(workspaceInfoColumns(), workspaceInfoRows(workspace))
			return nil
		},
	}

	return cmd
}

func workspaceInfoColumns() []table.Column {
	return []table.Column{
		{Title: "Field", Width: 20},
		{Title: "Value", Width: 80},
	}
}

func workspaceInfoRows(workspace *client.Workspace) []table.Row {
	dataPlaneURL := ""
	if workspace.DataPlaneURL != nil {
		dataPlaneURL = *workspace.DataPlaneURL
	}

	return []table.Row{
		{"WorkspaceID", workspace.ID},
		{"Name", workspace.Name},
		{"Environment", workspace.Environment},
		{"Status", workspace.Status},
		{"Region", workspace.Region},
		{"Data plane URL", dataPlaneURL},
	}
}
