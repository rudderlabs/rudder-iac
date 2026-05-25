package columnmetadata

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/charmbracelet/bubbles/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

func newCmdList() *cobra.Command {
	var (
		dataGraphID string
		modelID     string
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List column display name aliases for a model",
		Example: heredoc.Doc(`
			$ rudder-cli data-graphs column-metadata list --data-graph-id dg-1 --model-id m-1
			$ rudder-cli data-graphs column-metadata list --data-graph-id dg-1 --model-id m-1 --json
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return requireDataGraphFlag()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("data-graphs column-metadata list", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
				}...)
			}()

			deps, err := app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			dgClient := newDataGraphClient(deps.Client())
			result, err := dgClient.ListColumnMetadata(cmd.Context(), dataGraphID, modelID)
			if err != nil {
				return fmt.Errorf("listing column metadata: %w", err)
			}

			if jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			if len(result.ColumnMetadata) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No column metadata found.")
				return nil
			}

			rows := make([]table.Row, len(result.ColumnMetadata))
			items := result.ColumnMetadata
			sort.Slice(items, func(i, j int) bool {
				return items[i].ColumnName < items[j].ColumnName
			})
			for i, item := range items {
				rows[i] = table.Row{item.ColumnName, item.DisplayName, item.UpdatedAt}
			}

			ui.PrintTable([]table.Column{
				{Title: "Column", Width: 30},
				{Title: "Display Name", Width: 40},
				{Title: "Updated At", Width: 30},
			}, rows)

			return nil
		},
	}

	cmd.Flags().StringVar(&dataGraphID, "data-graph-id", "", "Data graph ID")
	cmd.Flags().StringVar(&modelID, "model-id", "", "Model ID")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	_ = cmd.MarkFlagRequired("data-graph-id")
	_ = cmd.MarkFlagRequired("model-id")

	return cmd
}
