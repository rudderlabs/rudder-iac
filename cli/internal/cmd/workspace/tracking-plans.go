package workspace

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/spf13/cobra"
)

func NewCmdTrackingPlans() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tracking-plans",
		Short:   "Inspect tracking plans in the workspace",
		Long:    "Inspect tracking plans in the authenticated workspace. Use the list subcommand for table or JSON output.",
		Example: `  rudder-cli workspace tracking-plans list --json`,
		Args:    cobra.NoArgs,
	}

	cmd.AddCommand(newCmdListTrackingPlans())

	return cmd
}

func newCmdListTrackingPlans() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List tracking plans in the workspace",
		Long:    "List tracking plans from the authenticated workspace, including their remote IDs and names. Results are printed as a table unless --json is supplied.",
		Example: "  rudder-cli workspace tracking-plans list\n  rudder-cli workspace tracking-plans list --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")

			var err error
			defer func() {
				telemetry.TrackCommand("workspace tracking-plans list", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			// Cast the DataCatalog provider to access the List method
			dcProvider := d.Providers().DataCatalog

			format := lister.TableFormat
			if jsonOutput {
				format = lister.JSONFormat
			}
			l := lister.New(dcProvider,
				lister.WithFormat(format),
				lister.WithColumnWidths(map[string]int{
					"id": 30,
				}),
			)

			err = l.List(cmd.Context(), types.TrackingPlanResourceType, nil)
			return err
		},
	}

	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}
