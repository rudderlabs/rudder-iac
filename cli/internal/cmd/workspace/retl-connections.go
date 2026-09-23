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
	cmd.AddCommand(newCmdViewRetlConnection())

	return cmd
}

func newCmdViewRetlConnection() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "view <external-id>",
		Short: "Show one RETL connection in full",
		Long:  "Shows both endpoints by name and everything the connection carries. A connection whose endpoints are missing from the catalog still renders, with the names blank.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("workspace retl-connections view", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
				}...)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			retlProvider := d.Providers().RETL
			if !slices.Contains(retlProvider.SupportedTypes(), connection.ResourceType) {
				err = fmt.Errorf("RETL connections are experimental: set RUDDERSTACK_CLI_EXPERIMENTAL=true and RUDDERSTACK_X_RETL_CONNECTION_SUPPORT=true")
				return err
			}

			row, err := findByExternalID(cmd.Context(), retlProvider, connection.ResourceType, args[0], "rETL connection")
			if err != nil {
				return err
			}

			err = lister.New(oneResource(row), lister.WithFormat(viewFormat(jsonOutput))).
				List(cmd.Context(), connection.ResourceType, nil)
			return err
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
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
