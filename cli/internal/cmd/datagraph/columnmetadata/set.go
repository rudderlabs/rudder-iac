package columnmetadata

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/spf13/cobra"
)

func newCmdSet() *cobra.Command {
	var (
		dataGraphID string
		modelID     string
		displayName string
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "set <columnName>",
		Short: "Set or update a column display name alias",
		Args:  cobra.ExactArgs(1),
		Example: heredoc.Doc(`
			$ rudder-cli data-graphs column-metadata set ltv --data-graph-id dg-1 --model-id m-1 --display-name "Lifetime Value"
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return requireDataGraphFlag()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("data-graphs column-metadata set", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
				}...)
			}()

			columnName := args[0]
			trimmedDisplayName := strings.TrimSpace(displayName)
			if trimmedDisplayName == "" {
				return fmt.Errorf("display name cannot be empty")
			}

			deps, err := app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			dgClient := newDataGraphClient(deps.Client())
			result, err := dgClient.UpsertColumnMetadata(
				cmd.Context(),
				dataGraphID,
				modelID,
				columnName,
				trimmedDisplayName,
			)
			if err != nil {
				return fmt.Errorf("setting column metadata: %w", err)
			}

			if jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			fmt.Fprintf(
				cmd.OutOrStdout(),
				"Set display name for column %q to %q\n",
				result.ColumnName,
				result.DisplayName,
			)
			return nil
		},
	}

	cmd.Flags().StringVar(&dataGraphID, "data-graph-id", "", "Data graph ID")
	cmd.Flags().StringVar(&modelID, "model-id", "", "Model ID")
	cmd.Flags().StringVar(&displayName, "display-name", "", "Display name alias for the column")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output the updated metadata as JSON")
	_ = cmd.MarkFlagRequired("data-graph-id")
	_ = cmd.MarkFlagRequired("model-id")
	_ = cmd.MarkFlagRequired("display-name")

	return cmd
}
