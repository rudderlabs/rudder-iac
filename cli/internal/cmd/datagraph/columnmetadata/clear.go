package columnmetadata

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/spf13/cobra"
)

func newCmdClear() *cobra.Command {
	var (
		dataGraphID string
		modelID     string
	)

	cmd := &cobra.Command{
		Use:   "clear <columnName>",
		Short: "Remove a column display name alias",
		Args:  cobra.ExactArgs(1),
		Example: heredoc.Doc(`
			$ rudder-cli data-graphs column-metadata clear ltv --data-graph-id dg-1 --model-id m-1
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return requireDataGraphFlag()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("data-graphs column-metadata clear", err)
			}()

			columnName := args[0]

			deps, err := app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			dgClient := newDataGraphClient(deps.Client())
			if err := dgClient.DeleteColumnMetadata(cmd.Context(), dataGraphID, modelID, columnName); err != nil {
				return fmt.Errorf("clearing column metadata: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&dataGraphID, "data-graph-id", "", "Data graph ID")
	cmd.Flags().StringVar(&modelID, "model-id", "", "Model ID")
	_ = cmd.MarkFlagRequired("data-graph-id")
	_ = cmd.MarkFlagRequired("model-id")

	return cmd
}
