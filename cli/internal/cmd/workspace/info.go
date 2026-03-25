package workspace

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

const (
	workspaceIDKey  = "WorkspaceID"
	nameKey         = "Name"
	environmentKey  = "Environment"
	statusKey       = "Status"
	regionKey       = "Region"
	dataPlaneURLKey = "Data plane URL"
)

func NewCmdInfo() *cobra.Command {
	var jsonOutput bool

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

			if jsonOutput {
				return printWorkspaceInfoJSON(cmd, workspace)
			}

			ui.PrintTable(workspaceInfoColumns(), workspaceInfoRows(workspace))
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output workspace information as JSON")

	return cmd
}

func printWorkspaceInfoJSON(cmd *cobra.Command, workspace *client.Workspace) error {
	camel := namer.StrategyCamelCase

	dataPlaneURL := ""
	if workspace.DataPlaneURL != nil {
		dataPlaneURL = *workspace.DataPlaneURL
	}

	out := map[string]string{
		camel.Name(workspaceIDKey):  workspace.ID,
		camel.Name(nameKey):         workspace.Name,
		camel.Name(environmentKey):  workspace.Environment,
		camel.Name(statusKey):       workspace.Status,
		camel.Name(regionKey):       workspace.Region,
		camel.Name(dataPlaneURLKey): dataPlaneURL,
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(out)
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
		{workspaceIDKey, workspace.ID},
		{nameKey, workspace.Name},
		{environmentKey, workspace.Environment},
		{statusKey, workspace.Status},
		{regionKey, workspace.Region},
		{dataPlaneURLKey, dataPlaneURL},
	}
}
