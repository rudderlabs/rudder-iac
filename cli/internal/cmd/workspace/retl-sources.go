package workspace

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/spf13/cobra"
)

func NewCmdRetlSource() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "retl-sources",
		Short:   "Inspect RETL sources in the workspace",
		Long:    "Inspect RETL SQL model sources in the authenticated workspace. Use the list subcommand for table or JSON output.",
		Example: `  rudder-cli workspace retl-sources list --json`,
		Args:    cobra.NoArgs,
	}

	cmd.AddCommand(newCmdListRetlSources())

	return cmd
}

func newCmdListRetlSources() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List RETL sources in the workspace",
		Long:    "List RETL SQL model sources from the authenticated workspace. Results are printed as a table unless --json is supplied.",
		Example: "  rudder-cli workspace retl-sources list\n  rudder-cli workspace retl-sources list --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")

			var err error
			defer func() {
				telemetry.TrackCommand("workspace retl-source list", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			// Cast the RETL provider to access the List method
			retlProvider := d.Providers().RETL

			format := lister.TableFormat
			if jsonOutput {
				format = lister.JSONFormat
			}
			l := lister.New(retlProvider, lister.WithFormat(format))

			err = l.List(cmd.Context(), sqlmodel.ResourceType, nil)
			return err
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}
