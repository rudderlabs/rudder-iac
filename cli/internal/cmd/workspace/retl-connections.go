package workspace

import (
	"fmt"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/spf13/cobra"
)

func NewCmdRetlConnections() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retl-connections",
		Short: "Manage RETL connections in the workspace",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newCmdListRetlConnections())

	return cmd
}

func newCmdListRetlConnections() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List RETL connections in the workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")

			var err error
			defer func() {
				telemetry.TrackCommand("workspace retl-connections list", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			retlProvider := d.Providers().RETL
			// The connection handler is only registered behind its flag, and the
			// provider's error for a missing handler names an internal resource
			// type rather than the thing the reader has to turn on.
			if !slices.Contains(retlProvider.SupportedTypes(), connection.ResourceType) {
				err = fmt.Errorf("RETL connections are experimental: set RUDDERSTACK_CLI_EXPERIMENTAL=true and RUDDERSTACK_X_RETL_CONNECTION_SUPPORT=true")
				return err
			}

			format := lister.TableFormat
			if jsonOutput {
				format = lister.JSONFormat
			}
			l := lister.New(retlProvider, lister.WithFormat(format))

			err = l.List(cmd.Context(), connection.ResourceType, nil)
			return err
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}
