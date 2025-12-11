package workspace

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/telemetry"
	"github.com/spf13/cobra"
)

func NewCmdRetlSource() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retl-sources",
		Short: "Manage RETL sources in the workspace",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newCmdListRetlSources())

	return cmd
}

func newCmdListRetlSources() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List RETL sources in the workspace",
		Args:  cobra.NoArgs,
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
